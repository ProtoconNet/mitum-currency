package currency

import (
	"context"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/isaac"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var createAccountItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateAccountItemProcessor)
	},
}

var createAccountProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateAccountProcessor)
	},
}

func (CreateAccount) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type CreateAccountItemProcessor struct {
	h    util.Hash
	item CreateAccountItem
	ns   base.StateMergeValue
	nb   map[types.CurrencyID]base.StateMergeValue
}

func (opp *CreateAccountItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	e := util.StringError("preprocess for CreateAccountItemProcessor")

	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		policy, err := state.ExistsCurrencyPolicy(am.Currency(), getStateFunc)
		if err != nil {
			return err
		}

		if am.Big().Compare(policy.NewAccountMinBalance()) < 0 {
			return base.NewBaseOperationProcessReasonError(
				"amount should be over minimum balance, %v < %v", am.Big(), policy.NewAccountMinBalance())
		}
	}

	target, err := opp.item.Address()
	if err != nil {
		return e.Wrap(err)
	}

	st, err := state.NotExistsState(currency.StateKeyAccount(target), "keys of target", getStateFunc)
	if err != nil {
		return err
	}
	opp.ns = state.NewStateMergeValue(st.Key(), st.Value())

	nb := map[types.CurrencyID]base.StateMergeValue{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		switch _, found, err := getStateFunc(currency.StateKeyBalance(target, am.Currency())); {
		case err != nil:
			return e.Wrap(err)
		case found:
			return e.Wrap(isaac.ErrStopProcessingRetry.Errorf("target balance already exists"))
		default:
			nb[am.Currency()] = state.NewStateMergeValue(currency.StateKeyBalance(target, am.Currency()), currency.NewBalanceStateValue(types.NewZeroAmount(am.Currency())))
		}
	}
	opp.nb = nb

	return nil
}

func (opp *CreateAccountItemProcessor) Process(
	_ context.Context, _ base.Operation, _ base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	e := util.StringError("preprocess for CreateAccountItemProcessor")

	var (
		nac types.Account
		err error
	)

	if opp.item.AddressType() == types.EthAddressHint.Type() {
		nac, err = types.NewEthAccountFromKeys(opp.item.Keys())
	} else {
		nac, err = types.NewAccountFromKeys(opp.item.Keys())
	}
	if err != nil {
		return nil, e.Wrap(err)
	}

	sts := make([]base.StateMergeValue, len(opp.item.Amounts())+1)
	sts[0] = state.NewStateMergeValue(opp.ns.Key(), currency.NewAccountStateValue(nac))

	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		v, ok := opp.nb[am.Currency()].Value().(currency.BalanceStateValue)
		if !ok {
			return nil, e.Wrap(errors.Errorf("not BalanceStateValue, %T", opp.nb[am.Currency()].Value()))
		}
		stv := currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Add(am.Big())))
		sts[i+1] = state.NewStateMergeValue(opp.nb[am.Currency()].Key(), stv)
	}

	return sts, nil
}

func (opp *CreateAccountItemProcessor) Close() {
	opp.h = nil
	opp.item = nil
	opp.ns = nil
	opp.nb = nil

	createAccountItemProcessorPool.Put(opp)
}

type CreateAccountProcessor struct {
	*base.BaseOperationProcessor
	ns       []*CreateAccountItemProcessor
	required map[types.CurrencyID][2]common.Big // required[0] : amount + fee, required[1] : fee
}

func NewCreateAccountProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new CreateAccountProcessor")

		nopp := createAccountProcessorPool.Get()
		opp, ok := nopp.(*CreateAccountProcessor)
		if !ok {
			return nil, errors.Errorf("expected CreateAccountProcessor, not %T", nopp)
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b
		opp.ns = nil
		opp.required = nil

		return opp, nil
	}
}

func (opp *CreateAccountProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(CreateAccountFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError("expected CreateAccountFact, not %T", op.Fact()), nil
	}

	if err := state.CheckExistsState(currency.StateKeyAccount(fact.sender), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("failed to check existence of sender %v; %w", fact.sender, err), nil
	}

	if err := state.CheckNotExistsState(extension.StateKeyContractAccount(fact.Sender()), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("contract account cannot be create-account sender, %v; %w", fact.Sender(), err), nil
	}

	if err := state.CheckFactSignsByState(fact.sender, op.Signs(), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("invalid signing; %w", err), nil
	}

	for i := range fact.items {
		cip := createAccountItemProcessorPool.Get()
		c, ok := cip.(*CreateAccountItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected CreateAccountItemProcessor, not %T", cip), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess CreateAccountItem; %w", err), nil
		}

		c.Close()
	}

	return ctx, nil, nil
}

func (opp *CreateAccountProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(CreateAccountFact)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("expected CreateAccountFact, not %T", op.Fact()), nil
	}

	var (
		senderBalSts, feeReceiveBalSts map[types.CurrencyID]base.State
		required                       map[types.CurrencyID][2]common.Big
		err                            error
	)

	if feeReceiveBalSts, required, err = opp.calculateItemsFee(op, getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("failed to calculate fee; %w", err), nil
	} else if senderBalSts, err = CheckEnoughBalance(fact.sender, required, getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("not enough balance of sender %v ; %w", fact.sender, err), nil
	} else {
		opp.required = required
	}

	ns := make([]*CreateAccountItemProcessor, len(fact.items))
	for i := range fact.items {
		cip := createAccountItemProcessorPool.Get()
		c, ok := cip.(*CreateAccountItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected CreateAccountItemProcessor, not %T", cip), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess CreateAccountItem; %w", err), nil
		}

		ns[i] = c
	}
	opp.ns = ns

	var stateMergeValues []base.StateMergeValue // nolint:prealloc
	for i := range opp.ns {
		s, err := opp.ns[i].Process(ctx, op, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("failed to process CreateAccountItem; %w", err), nil
		}
		stateMergeValues = append(stateMergeValues, s...)
	}

	for cid := range senderBalSts {
		v, ok := senderBalSts[cid].Value().(currency.BalanceStateValue)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected BalanceStateValue, not %T", senderBalSts[cid].Value()), nil
		}

		var stateMergeValue base.StateMergeValue
		if senderBalSts[cid].Key() == feeReceiveBalSts[cid].Key() {
			stateMergeValue = state.NewStateMergeValue(
				senderBalSts[cid].Key(),
				currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Sub(opp.required[cid][0]).Add(opp.required[cid][1]))),
			)
		} else {
			stateMergeValue = state.NewStateMergeValue(
				senderBalSts[cid].Key(),
				currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Sub(opp.required[cid][0]))),
			)
			r, ok := feeReceiveBalSts[cid].Value().(currency.BalanceStateValue)
			if !ok {
				return nil, base.NewBaseOperationProcessReasonError("expected BalanceStateValue, not %T", feeReceiveBalSts[cid].Value()), nil
			}
			stateMergeValues = append(
				stateMergeValues,
				state.NewStateMergeValue(
					feeReceiveBalSts[cid].Key(),
					currency.NewBalanceStateValue(r.Amount.WithBig(r.Amount.Big().Add(opp.required[cid][1]))),
				),
			)
		}
		stateMergeValues = append(stateMergeValues, stateMergeValue)
	}

	return stateMergeValues, nil, nil
}

func (opp *CreateAccountProcessor) Close() error {
	for i := range opp.ns {
		opp.ns[i].Close()
	}

	opp.ns = nil
	opp.required = nil

	createAccountProcessorPool.Put(opp)

	return nil
}

func (opp *CreateAccountProcessor) calculateItemsFee(
	op base.Operation,
	getStateFunc base.GetStateFunc,
) (map[types.CurrencyID]base.State, map[types.CurrencyID][2]common.Big, error) {
	fact, ok := op.Fact().(CreateAccountFact)
	if !ok {
		return nil, nil, errors.Errorf("expected CreateAccountFact, not %T", op.Fact())
	}

	items := make([]AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	return CalculateItemsFee(getStateFunc, items)
}

func CalculateItemsFee(getStateFunc base.GetStateFunc, items []AmountsItem) (map[types.CurrencyID]base.State, map[types.CurrencyID][2]common.Big, error) {
	feeReceiveSts := map[types.CurrencyID]base.State{}
	required := map[types.CurrencyID][2]common.Big{}

	for i := range items {
		it := items[i]

		for j := range it.Amounts() {
			am := it.Amounts()[j]

			rq := [2]common.Big{common.ZeroBig, common.ZeroBig}
			if k, found := required[am.Currency()]; found {
				rq = k
			}

			policy, err := state.ExistsCurrencyPolicy(am.Currency(), getStateFunc)
			if err != nil {
				return nil, nil, err
			}

			var k common.Big
			switch k, err = policy.Feeer().Fee(am.Big()); {
			case err != nil:
				return nil, nil, err
			case !k.OverZero():
				required[am.Currency()] = [2]common.Big{rq[0].Add(am.Big()), rq[1]}
			default:
				required[am.Currency()] = [2]common.Big{rq[0].Add(am.Big()).Add(k), rq[1].Add(k)}
			}

			if policy.Feeer().Receiver() == nil {
				continue
			}

			if err := state.CheckExistsState(currency.StateKeyAccount(policy.Feeer().Receiver()), getStateFunc); err != nil {
				return nil, nil, err
			} else if st, found, err := getStateFunc(currency.StateKeyBalance(policy.Feeer().Receiver(), am.Currency())); err != nil {
				return nil, nil, err
			} else if !found {
				return nil, nil, errors.Errorf("feeer receiver %s not found", policy.Feeer().Receiver())
			} else {
				feeReceiveSts[am.Currency()] = st
			}
		}
	}

	return feeReceiveSts, required, nil
}

func CheckEnoughBalance(
	holder base.Address,
	required map[types.CurrencyID][2]common.Big,
	getStateFunc base.GetStateFunc,
) (map[types.CurrencyID]base.State, error) {
	sbSts := map[types.CurrencyID]base.State{}

	for cid := range required {
		rq := required[cid]

		st, err := state.ExistsState(currency.StateKeyBalance(holder, cid), "currency of holder", getStateFunc)
		if err != nil {
			return nil, err
		}

		am, err := currency.StateBalanceValue(st)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("insufficient balance of sender; %w", err)
		}

		if am.Big().Compare(rq[0]) < 0 {
			return nil, base.NewBaseOperationProcessReasonError(
				"insufficient balance of sender, %s; %d !> %d", holder.String(), am.Big(), rq[0])
		}
		sbSts[cid] = st
	}

	return sbSts, nil
}
