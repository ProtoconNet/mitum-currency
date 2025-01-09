package currency

import (
	"context"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var updateKeyProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(UpdateKeyProcessor)
	},
}

func (UpdateKey) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type UpdateKeyProcessor struct {
	*base.BaseOperationProcessor
}

func NewUpdateKeyProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new UpdateKeyProcessor")

		nopp := updateKeyProcessorPool.Get()
		opp, ok := nopp.(*UpdateKeyProcessor)
		if !ok {
			return nil, errors.Errorf("expected %T, not %T", &UpdateKeyProcessor{}, nopp)
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

func (opp *UpdateKeyProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(UpdateKeyFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMTypeMismatch).Errorf("expected %T, not %T", UpdateKeyFact{}, op.Fact())), nil
	}

	if aState, _, aErr, cErr := state.ExistsCAccount(fact.Sender(), "sender", true, false, getStateFunc); aErr != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Errorf("%v", aErr)), nil
	} else if cErr != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCAccountNA).Errorf("%v", cErr)), nil
	} else if ac, err := currency.LoadAccountStateValue(aState); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMStateValInvalid).Errorf("%v: sender %v", err, fact.Sender())), nil
	} else if _, ok := ac.Keys().(types.NilAccountKeys); ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMValueInvalid).Errorf("sender %v must be multi-sig account", fact.Sender())), nil
	} else if ac.Keys().Equal(fact.Keys()) {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMValueInvalid).Errorf("sender keys is same with keys to update, keys hash %v", fact.keys.Hash())), nil
	}

	return ctx, nil, nil
}

func (opp *UpdateKeyProcessor) Process( // nolint:dupl
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, _ := op.Fact().(UpdateKeyFact)

	var stmvs []base.StateMergeValue // nolint:prealloc
	var tgAccSt base.State
	var err error
	if tgAccSt, err = state.ExistsState(currency.AccountStateKey(fact.Sender()), "sender keys", getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("sender account not found, %v; %w", fact.Sender(), err), nil
	}

	ac, err := currency.LoadAccountStateValue(tgAccSt)
	if err != nil {
		return nil, nil, err
	}
	uac, err := ac.SetKeys(fact.keys)
	if err != nil {
		return nil, nil, err
	}
	stmvs = append(stmvs, state.NewStateMergeValue(tgAccSt.Key(), currency.NewAccountStateValue(uac)))

	return stmvs, nil, nil
}

func (opp *UpdateKeyProcessor) Close() error {
	updateKeyProcessorPool.Put(opp)

	return nil
}
