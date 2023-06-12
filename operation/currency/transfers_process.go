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

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var transfersItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(TransfersItemProcessor)
	},
}

var transfersProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(TransfersProcessor)
	},
}

func (Transfers) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type TransfersItemProcessor struct {
	h    util.Hash
	item TransfersItem
	rb   map[types.CurrencyID]base.StateMergeValue
}

func (opp *TransfersItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	e := util.StringErrorFunc("failed to preprocess for TransfersItemProcessor")

	if _, err := state.ExistsState(currency.StateKeyAccount(opp.item.Receiver()), "receiver", getStateFunc); err != nil {
		return e(err, "")
	}

	rb := map[types.CurrencyID]base.StateMergeValue{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		_, err := state.ExistsCurrencyPolicy(am.Currency(), getStateFunc)
		if err != nil {
			return err
		}

		st, _, err := getStateFunc(currency.StateKeyBalance(opp.item.Receiver(), am.Currency()))
		if err != nil {
			return err
		}

		var balance types.Amount
		if st == nil {
			balance = types.NewZeroAmount(am.Currency())
		} else {
			balance, err = currency.StateBalanceValue(st)
			if err != nil {
				return err
			}
		}

		rb[am.Currency()] = state.NewStateMergeValue(currency.StateKeyBalance(opp.item.Receiver(), am.Currency()), currency.NewBalanceStateValue(balance))
	}

	opp.rb = rb

	return nil
}

func (opp *TransfersItemProcessor) Process(
	_ context.Context, _ base.Operation, _ base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	e := util.StringErrorFunc("failed to preprocess for TransfersItemProcessor")

	sts := make([]base.StateMergeValue, len(opp.item.Amounts()))
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		v, ok := opp.rb[am.Currency()].Value().(currency.BalanceStateValue)
		if !ok {
			return nil, e(errors.Errorf("not BalanceStateValue, %T", opp.rb[am.Currency()].Value()), "")
		}
		stv := currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Add(am.Big())))
		sts[i] = state.NewStateMergeValue(opp.rb[am.Currency()].Key(), stv)
	}

	return sts, nil
}

func (opp *TransfersItemProcessor) Close() error {
	opp.h = nil
	opp.item = nil
	opp.rb = nil

	transfersItemProcessorPool.Put(opp)

	return nil
}

type TransfersProcessor struct {
	*base.BaseOperationProcessor
	ns       []*TransfersItemProcessor
	required map[types.CurrencyID][2]common.Big
}

func NewTransfersProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringErrorFunc("failed to create new TransfersProcessor")

		nopp := transfersProcessorPool.Get()
		opp, ok := nopp.(*TransfersProcessor)
		if !ok {
			return nil, e(errors.Errorf("expected TransfersProcessor, not %T", nopp), "")
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e(err, "")
		}

		opp.BaseOperationProcessor = b
		opp.ns = nil
		opp.required = nil

		return opp, nil
	}
}

func (opp *TransfersProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(TransfersFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError("expected TransfersFact, not %T", op.Fact()), nil
	}

	if err := state.CheckExistsState(currency.StateKeyAccount(fact.sender), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("failed to check existence of sender %v : %w", fact.sender, err), nil
	}

	if err := state.CheckNotExistsState(extension.StateKeyContractAccount(fact.Sender()), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("contract account cannot transfer amounts, %q: %w", fact.Sender(), err), nil
	}

	if err := state.CheckFactSignsByState(fact.sender, op.Signs(), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError("invalid signing :  %w", err), nil
	}

	for i := range fact.items {
		cip := transfersItemProcessorPool.Get()
		c, ok := cip.(*TransfersItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected TransfersItemProcessor, not %T", cip), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess transfer item: %w", err), nil
		}
		c.Close()
	}

	return ctx, nil, nil
}

func (opp *TransfersProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(TransfersFact)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("expected TransfersFact, not %T", op.Fact()), nil
	}

	var (
		senderBalSts, feeReceiveBalSts map[types.CurrencyID]base.State
		required                       map[types.CurrencyID][2]common.Big
		err                            error
	)

	if feeReceiveBalSts, required, err = opp.calculateItemsFee(op, getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("failed to calculate fee: %w", err), nil
	} else if senderBalSts, err = CheckEnoughBalance(fact.sender, required, getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("failed to check enough balance: %w", err), nil
	} else {
		opp.required = required
	}

	ns := make([]*TransfersItemProcessor, len(fact.items))
	for i := range fact.items {
		cip := transfersItemProcessorPool.Get()
		c, ok := cip.(*TransfersItemProcessor)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected TransfersItemProcessor, not %T", cip), nil
		}

		c.h = op.Hash()
		c.item = fact.items[i]

		if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
			return nil, base.NewBaseOperationProcessReasonError("fail to preprocess transfer item: %w", err), nil
		}

		ns[i] = c
	}
	opp.ns = ns

	var stmvs []base.StateMergeValue // nolint:prealloc
	for i := range opp.ns {
		s, err := opp.ns[i].Process(ctx, op, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("failed to process transfer item: %w", err), nil
		}
		stmvs = append(stmvs, s...)
	}

	for cid := range senderBalSts {
		v, ok := senderBalSts[cid].Value().(currency.BalanceStateValue)
		if !ok {
			return nil, base.NewBaseOperationProcessReasonError("expected BalanceStateValue, not %T", senderBalSts[cid].Value()), nil
		}

		var stmv base.StateMergeValue
		if senderBalSts[cid].Key() == feeReceiveBalSts[cid].Key() {
			stmv = state.NewStateMergeValue(
				senderBalSts[cid].Key(),
				currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Sub(opp.required[cid][0]).Add(opp.required[cid][1]))),
			)
		} else {
			stmv = state.NewStateMergeValue(
				senderBalSts[cid].Key(),
				currency.NewBalanceStateValue(v.Amount.WithBig(v.Amount.Big().Sub(opp.required[cid][0]))),
			)
			r, ok := feeReceiveBalSts[cid].Value().(currency.BalanceStateValue)
			if !ok {
				return nil, base.NewBaseOperationProcessReasonError("expected BalanceStateValue, not %T", feeReceiveBalSts[cid].Value()), nil
			}
			stmvs = append(
				stmvs,
				state.NewStateMergeValue(
					feeReceiveBalSts[cid].Key(),
					currency.NewBalanceStateValue(r.Amount.WithBig(r.Amount.Big().Add(opp.required[cid][1]))),
				),
			)
		}
		stmvs = append(stmvs, stmv)
	}

	return stmvs, nil, nil
}

func (opp *TransfersProcessor) Close() error {
	for i := range opp.ns {
		_ = opp.ns[i].Close()
	}

	opp.ns = nil
	opp.required = nil

	transfersProcessorPool.Put(opp)

	return nil
}

func (opp *TransfersProcessor) calculateItemsFee(op base.Operation, getStateFunc base.GetStateFunc) (map[types.CurrencyID]base.State, map[types.CurrencyID][2]common.Big, error) {
	fact, ok := op.Fact().(TransfersFact)
	if !ok {
		return nil, nil, errors.Errorf("expected TransfersFact, not %T", op.Fact())
	}
	items := make([]AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	return CalculateItemsFee(getStateFunc, items)
}