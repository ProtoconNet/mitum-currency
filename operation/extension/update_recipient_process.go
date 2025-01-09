package extension

import (
	"context"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

var UpdateRecipientProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(UpdateRecipientProcessor)
	},
}

func (UpdateRecipient) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	// NOTE Process is nil func
	return nil, nil, nil
}

type UpdateRecipientProcessor struct {
	*base.BaseOperationProcessor
	ca  base.StateMergeValue
	sb  base.StateMergeValue
	fee common.Big
}

func NewUpdateRecipientProcessor() types.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("create new UpdateRecipientProcessor")

		nopp := UpdateRecipientProcessorPool.Get()
		opp, ok := nopp.(*UpdateRecipientProcessor)
		if !ok {
			return nil, errors.Errorf("expected UpdateRecipientProcessor, not %T", nopp)
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

func (opp *UpdateRecipientProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(UpdateRecipientFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected UpdateRecipientFact, not %T", op.Fact())), nil
	}

	for i := range fact.Recipients() {
		if _, _, _, cErr := state.ExistsCAccount(
			fact.Recipients()[i], "recipient", true, false, getStateFunc); cErr != nil {
			return ctx, base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMCAccountNA).
					Errorf("%v: recipient %v is contract account", cErr, fact.Recipients()[i])), nil
		}
	}

	return ctx, nil, nil
}

func (opp *UpdateRecipientProcessor) Process( // nolint:dupl
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, _ := op.Fact().(UpdateRecipientFact)

	var ctAccSt base.State
	var err error
	ctAccSt, err = state.ExistsState(extension.StateKeyContractAccount(fact.Contract()), "contract account status", getStateFunc)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("check existence of contract account status %v ; %w", fact.Contract(), err), nil
	}

	var stmvs []base.StateMergeValue // nolint:prealloc

	for _, recipient := range fact.Recipients() {
		smv, err := state.CreateNotExistAccount(recipient, getStateFunc)
		if err != nil {
			return nil, base.NewBaseOperationProcessReasonError("%w", err), nil
		} else if smv != nil {
			stmvs = append(stmvs, smv)
		}
	}

	ctsv := ctAccSt.Value()
	if ctsv == nil {
		return nil, nil, util.ErrNotFound.Errorf("contract account status not found in State")
	}

	sv, ok := ctsv.(extension.ContractAccountStateValue)
	if !ok {
		return nil, nil, errors.Errorf("invalid contract account value found, %T", ctsv)
	}

	status := sv.Status()
	err = status.SetRecipients(fact.Recipients())
	if err != nil {
		return nil, nil, err
	}

	stmvs = append(stmvs, state.NewStateMergeValue(ctAccSt.Key(), extension.NewContractAccountStateValue(status)))

	return stmvs, nil, nil
}

func (opp *UpdateRecipientProcessor) Close() error {
	UpdateRecipientProcessorPool.Put(opp)

	return nil
}
