package currency

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
	"sync"
)

var transferItemProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(TransferItemProcessor)
	},
}

var transferProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(TransferProcessor)
	},
}

func (Transfer) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type TransferItemProcessor struct {
	h    util.Hash
	item TransferItem
}

func (opp *TransferItemProcessor) PreProcess(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) error {
	return nil
}

func (opp *TransferItemProcessor) Process(
	_ context.Context, _ base.Operation, getStateFunc base.GetStateFunc,
) ([]base.StateMergeValue, error) {
	e := util.StringError("preprocess TransferItemProcessor")

	var sts []base.StateMergeValue
	receiver := opp.item.Receiver()
	smv, err := state.CreateNotExistAccount(receiver, getStateFunc)
	if err != nil {
		return nil, e.Wrap(err)
	} else if smv != nil {
		sts = append(sts, smv)
	}

	amounts := opp.item.Amounts()
	for i := range amounts {
		am := amounts[i]
		cid := am.Currency()

		st, _, err := getStateFunc(currency.BalanceStateKey(receiver, cid))
		if err != nil {
			return nil, err
		}

		var balance types.Amount
		if st == nil {
			balance = types.NewZeroAmount(cid)
		} else {
			balance, err = currency.StateBalanceValue(st)
			if err != nil {
				return nil, err
			}
		}

		sts = append(sts, common.NewBaseStateMergeValue(
			currency.BalanceStateKey(receiver, cid),
			currency.NewAddBalanceStateValue(balance.WithBig(am.Big())),
			func(height base.Height, st base.State) base.StateValueMerger {
				return currency.NewBalanceStateValueMerger(height,
					currency.BalanceStateKey(receiver, cid),
					cid,
					st,
				)
			},
		))
	}

	return sts, nil
}

func (opp *TransferItemProcessor) Close() {
	opp.h = nil
	opp.item = nil

	transferItemProcessorPool.Put(opp)
}

type TransferProcessor struct {
	*base.BaseOperationProcessor
	required map[types.CurrencyID][2]common.Big
}

func NewTransferProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new TransferProcessor")

		nopp := transferProcessorPool.Get()
		opp, ok := nopp.(*TransferProcessor)
		if !ok {
			return nil, e.Wrap(errors.Errorf("expected TransferProcessor, not %T", nopp))
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b
		opp.required = nil

		return opp, nil
	}
}

func (opp *TransferProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(TransferFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMTypeMismatch).Errorf("expected %T, not %T", TransferFact{}, op.Fact()),
		), nil
	}

	var wg sync.WaitGroup
	errChan := make(chan *base.BaseOperationProcessReasonError, len(fact.items))
	for i := range fact.items {
		wg.Add(1)
		go func(item TransferItem) {
			defer wg.Done()
			tip := transferItemProcessorPool.Get()
			t, ok := tip.(*TransferItemProcessor)
			if !ok {
				err := base.NewBaseOperationProcessReasonError(
					common.ErrMPreProcess.Wrap(
						common.ErrMTypeMismatch).Errorf("expected %T, not %T", &TransferItemProcessor{}, tip))
				errChan <- &err
				return
			}

			t.h = op.Hash()
			t.item = item

			if err := t.PreProcess(ctx, op, getStateFunc); err != nil {
				err := base.NewBaseOperationProcessReasonError(common.ErrMPreProcess.Errorf("%v", err))
				errChan <- &err
				return
			}
			t.Close()
		}(fact.items[i])
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return nil, *err, nil
		}
	}

	return ctx, nil, nil
}

func (opp *TransferProcessor) Process( // nolint:dupl
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(TransferFact)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("expected %T, not %T", TransferFact{}, op.Fact()), nil
	}

	var stateMergeValues []base.StateMergeValue // nolint:prealloc
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan *base.BaseOperationProcessReasonError, len(fact.items))
	for i := range fact.items {
		wg.Add(1)
		go func(item TransferItem) {
			defer wg.Done()
			cip := transferItemProcessorPool.Get()
			c, ok := cip.(*TransferItemProcessor)
			if !ok {
				err := base.NewBaseOperationProcessReasonError("expected %T, not %T", &TransferItemProcessor{}, cip)
				errChan <- &err
				return
			}

			c.h = op.Hash()
			c.item = item

			if err := c.PreProcess(ctx, op, getStateFunc); err != nil {
				err := base.NewBaseOperationProcessReasonError("fail to preprocess transfer item: %w", err)
				errChan <- &err
				return
			}

			s, err := c.Process(ctx, op, getStateFunc)
			if err != nil {
				err := base.NewBaseOperationProcessReasonError("process transfer item: %w", err)
				errChan <- &err
				return
			}
			mu.Lock()
			stateMergeValues = append(stateMergeValues, s...)
			mu.Unlock()
			c.Close()
		}(fact.items[i])
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()
	for err := range errChan {
		if err != nil {
			return nil, *err, nil
		}
	}

	var required map[types.CurrencyID][]common.Big
	switch i := op.Fact().(type) {
	case extras.FeeAble:
		required = i.FeeBase()
	default:
	}

	totalAmounts, err := PrepareSenderTotalAmounts(fact.Sender(), required, getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("process CreateAccount; %w", err), nil
	}

	for key, total := range totalAmounts {
		stateMergeValues = append(
			stateMergeValues,
			common.NewBaseStateMergeValue(
				key,
				currency.NewDeductBalanceStateValue(total),
				func(height base.Height, st base.State) base.StateValueMerger {
					return currency.NewBalanceStateValueMerger(height, key, total.Currency(), st)
				}),
		)
	}

	return stateMergeValues, nil, nil
}

func (opp *TransferProcessor) Close() error {
	opp.required = nil

	transferProcessorPool.Put(opp)

	return nil
}
