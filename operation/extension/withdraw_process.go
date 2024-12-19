package extension

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var withdrawItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(WithdrawItemProcessor)
	},
}

var withdrawProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(WithdrawProcessor)
	},
}

func (Withdraw) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type WithdrawItemProcessor struct {
	h      util.Hash
	sender base.Address
	item   WithdrawItem
}

func (opp *WithdrawItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	e := util.StringError("preprocess WithdrawItemProcessor")

	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		_, err := state.ExistsCurrencyPolicy(am.Currency(), getStateFunc)
		if err != nil {
			return e.Wrap(err)
		}

		st, _, err := getStateFunc(ccstate.BalanceStateKey(opp.item.Target(), am.Currency()))
		if err != nil {
			return e.Wrap(err)
		}

		balance, err := ccstate.StateBalanceValue(st)
		if err != nil {
			return e.Wrap(err)
		}

		if balance.Big().Compare(am.Big()) < 0 {
			return errors.Errorf("insufficient contract account balance")
		}
	}

	return nil
}

func (opp *WithdrawItemProcessor) Process(
	_ context.Context, _ base.Operation, _ base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	sts := make([]base.StateMergeValue, len(opp.item.Amounts()))
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		key := ccstate.BalanceStateKey(opp.item.Target(), am.Currency())
		sts[i] = common.NewBaseStateMergeValue(
			key,
			ccstate.NewDeductBalanceStateValue(am),
			func(height base.Height, st base.State) base.StateValueMerger {
				return ccstate.NewBalanceStateValueMerger(height, key, am.Currency(), st)
			},
		)
	}

	return sts, nil
}

func (opp *WithdrawItemProcessor) Close() {
	opp.h = nil
	opp.sender = nil
	opp.item = nil

	withdrawItemProcessorPool.Put(opp)
}

type WithdrawProcessor struct {
	*base.BaseOperationProcessor
	ns       []*WithdrawItemProcessor
	required map[types.CurrencyID][2]common.Big // required[0] : amount + fee, required[1] : fee
}

func NewWithdrawProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new WithdrawProcessor")

		nopp := withdrawProcessorPool.Get()
		opp, ok := nopp.(*WithdrawProcessor)
		if !ok {
			return nil, e.WithMessage(nil, "expected WithdrawProcessor, not %T", nopp)
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b

		return opp, nil
	}
}

func (opp *WithdrawProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(WithdrawFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected %T, not %T", WithdrawFact{}, op.Fact())), nil
	}

	for i := range fact.items {
		cip := withdrawItemProcessorPool.Get()
		c, ok := cip.(*WithdrawItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMTypeMismatch).
					Errorf("expected %T, not %T", WithdrawItemProcessor{}, cip)), nil
		}

		c.h = op.Hash()
		c.sender = fact.Sender()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", err)), nil
		}

		c.Close()
	}

	return ctx, nil, nil
}

func (opp *WithdrawProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, _ := op.Fact().(WithdrawFact)

	ns := make([]*WithdrawItemProcessor, len(fact.items))
	for i := range fact.items {
		cip := withdrawItemProcessorPool.Get()
		c, _ := cip.(*WithdrawItemProcessor)
		c.h = op.Hash()
		c.sender = fact.Sender()
		c.item = fact.items[i]

		ns[i] = c
	}

	var stateMergeValues []base.StateMergeValue // nolint:prealloc
	for i := range ns {
		s, err := ns[i].Process(ctx, op, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("process WithdrawItem: %v", err), nil
		}
		stateMergeValues = append(stateMergeValues, s...)
	}

	var required map[types.CurrencyID][]common.Big
	switch i := op.Fact().(type) {
	case extras.FeeAble:
		required = i.FeeBase()
	default:
	}

	totalAmounts := map[string]types.Amount{}
	for cid, rqs := range required {
		total := common.ZeroBig
		for i := range rqs {
			total = total.Add(rqs[i])
		}

		totalAmounts[ccstate.BalanceStateKey(fact.Sender(), cid)] = types.NewAmount(total, cid)
	}

	for key, total := range totalAmounts {
		stateMergeValues = append(
			stateMergeValues,
			common.NewBaseStateMergeValue(
				key,
				ccstate.NewAddBalanceStateValue(total),
				func(height base.Height, st base.State) base.StateValueMerger {
					return ccstate.NewBalanceStateValueMerger(height, key, total.Currency(), st)
				}),
		)
	}

	return stateMergeValues, nil, nil
}

func (opp *WithdrawProcessor) Close() error {
	for i := range opp.ns {
		opp.ns[i].Close()
	}

	opp.required = nil
	withdrawProcessorPool.Put(opp)

	return nil
}
