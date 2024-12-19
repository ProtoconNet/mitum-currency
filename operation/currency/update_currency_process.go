package currency

import (
	"context"
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/state"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var updateCurrencyProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(UpdateCurrencyProcessor)
	},
}

func (UpdateCurrency) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type UpdateCurrencyProcessor struct {
	*base.BaseOperationProcessor
	suffrage  base.Suffrage
	threshold base.Threshold
}

func NewUpdateCurrencyProcessor(threshold base.Threshold) types.GetNewProcessor {
	return func(height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new UpdateCurrencyProcessor")

		nopp := updateCurrencyProcessorPool.Get()
		opp, ok := nopp.(*UpdateCurrencyProcessor)
		if !ok {
			return nil, e.Wrap(errors.Errorf("expected %T, not %T", &UpdateCurrencyProcessor{}, nopp))
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		opp.BaseOperationProcessor = b
		opp.threshold = threshold

		switch i, found, err := getStateFunc(isaac.SuffrageStateKey); {
		case err != nil:
			return nil, e.Wrap(err)
		case !found, i == nil:
			return nil, e.Errorf("Empty state")
		default:
			sufstv := i.Value().(base.SuffrageNodesStateValue) //nolint:forcetypeassert //...

			suf, err := sufstv.Suffrage()
			if err != nil {
				return nil, e.Errorf("get suffrage from state")
			}

			opp.suffrage = suf
		}

		return opp, nil
	}
}

func (opp *UpdateCurrencyProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	nop, ok := op.(UpdateCurrency)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMTypeMismatch).Errorf("expected %T, not %T", UpdateCurrency{}, op)), nil
	}

	if err := base.CheckFactSignsBySuffrage(opp.suffrage, opp.threshold, nop.NodeSigns()); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMSignInvalid).Errorf("%v", common.ErrSignNE)), nil
	}

	fact, ok := op.Fact().(UpdateCurrencyFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMTypeMismatch).Errorf("expected UpdateCurrencyFact, not %T", op.Fact())), nil
	}

	err := state.CheckExistsState(ccstate.DesignStateKey(fact.Currency()), getStateFunc)
	if err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCurrencyNF).Errorf("currency id %q", fact.Currency())), nil
	}

	if receiver := fact.Policy().Feeer().Receiver(); receiver != nil {
		if _, err := state.ExistsAccount(receiver, "feeer receiver", true, getStateFunc); err != nil {
			return ctx, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.Errorf("%v", err)), nil
		}
	}

	if err := state.CheckExistsState(ccstate.DesignStateKey(fact.Currency()), getStateFunc); err != nil {
		return ctx, nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCurrencyNF).Errorf("currency id %q", fact.Currency()))
	}

	return ctx, nil, nil
}

func (opp *UpdateCurrencyProcessor) Process(
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, ok := op.Fact().(UpdateCurrencyFact)
	if !ok {
		return nil, nil, errors.Errorf("expected %T, not %T", UpdateCurrencyFact{}, op.Fact())
	}

	sts := make([]base.StateMergeValue, 1)

	st, err := state.ExistsState(ccstate.DesignStateKey(fact.Currency()), fmt.Sprintf("currency design, %v", fact.Currency()), getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of currency id %q; %w", fact.Currency(), err), nil
	}

	de, err := ccstate.GetDesignFromState(st)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("get currency design of %v; %w", fact.Currency(), err), nil
	}

	de.SetPolicy(fact.Policy())

	c := state.NewStateMergeValue(
		st.Key(),
		ccstate.NewCurrencyDesignStateValue(de),
	)
	sts[0] = c

	return sts, nil, nil
}

func (opp *UpdateCurrencyProcessor) Close() error {
	opp.suffrage = nil
	opp.threshold = 0

	updateCurrencyProcessorPool.Put(opp)

	return nil
}
