package did_registry

import (
	"context"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	dstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	crtypes "github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
)

var updateDIDDocumentProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(UpdateDIDDocumentProcessor)
	},
}

func (UpdateDIDDocument) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	return nil, nil, nil
}

type UpdateDIDDocumentProcessor struct {
	*base.BaseOperationProcessor
}

func NewUpdateDIDDocumentProcessor() crtypes.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("failed to create new UpdateDIDDocumentProcessor")

		nopp := updateDIDDocumentProcessorPool.Get()
		opp, ok := nopp.(*UpdateDIDDocumentProcessor)
		if !ok {
			return nil, e.Errorf("expected %T, not %T", UpdateDIDDocumentProcessor{}, nopp)
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

func (opp *UpdateDIDDocumentProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(UpdateDIDDocumentFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected %T, not %T", UpdateDIDDocumentFact{}, op.Fact())), nil
	}

	if err := fact.IsValid(nil); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", err)), nil
	}

	if err := state.CheckExistsState(ccstate.DesignStateKey(fact.Currency()), getStateFunc); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCurrencyNF).Errorf("currency id %v", fact.Currency())), nil
	}

	if err := state.CheckExistsState(dstate.DesignStateKey(fact.Contract()), getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMServiceNF).Errorf("DID service for contract account %v",
				fact.Contract(),
			)), nil
	}

	_, id, err := types.ParseDIDScheme(fact.DID())
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMValueInvalid).Errorf("did scheme is invalid %v",
				fact.DID(),
			)), nil
	}

	if st, err := state.ExistsState(dstate.DataStateKey(fact.Contract(), id), "did data", getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateNF).Errorf("DID Data for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	} else if d, err := dstate.GetDataFromState(st); err != nil {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateValInvalid).Errorf(
				"DID Data for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	} else if !d.Address().Equal(fact.Sender()) {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateValInvalid).Errorf(
				"sender %v not matched with DID account address for DID %v in contract account %v", fact.Sender(), fact.DID(), fact.Contract(),
			)), nil
	}

	if _, err := state.ExistsState(dstate.DocumentStateKey(fact.Contract(), fact.DID()), "did document", getStateFunc); err != nil {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateNF).Errorf("DID document for DID %v in contract account %v", fact.DID(),
				fact.Contract(),
			)), nil
	}

	return ctx, nil, nil
}

func (opp *UpdateDIDDocumentProcessor) Process( // nolint:dupl
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	e := util.StringError("failed to process DeleteData")

	fact, ok := op.Fact().(UpdateDIDDocumentFact)
	if !ok {
		return nil, nil, e.Errorf("expected DeleteDataFact, not %T", op.Fact())
	}

	var sts []base.StateMergeValue // nolint:prealloc
	sts = append(sts, state.NewStateMergeValue(
		dstate.DocumentStateKey(fact.Contract(), fact.DID()),
		dstate.NewDocumentStateValue(fact.Document()),
	))

	return sts, nil, nil
}

func (opp *UpdateDIDDocumentProcessor) Close() error {
	updateDIDDocumentProcessorPool.Put(opp)

	return nil
}
