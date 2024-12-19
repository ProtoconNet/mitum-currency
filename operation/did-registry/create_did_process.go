package did_registry

import (
	"context"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	dstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	ctypes "github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"sync"
)

var createDIDProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(CreateDIDProcessor)
	},
}

func (CreateDID) Process(
	_ context.Context, _ base.GetStateFunc,
) ([]base.StateMergeValue, base.OperationProcessReasonError, error) {
	return nil, nil, nil
}

type CreateDIDProcessor struct {
	*base.BaseOperationProcessor
}

func NewCreateDIDProcessor() ctypes.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("failed to create new CreateDIDProcessor")

		nOpp := createDIDProcessorPool.Get()
		opp, ok := nOpp.(*CreateDIDProcessor)
		if !ok {
			return nil, e.Errorf("expected %T, not %T", CreateDIDProcessor{}, nOpp)
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

func (opp *CreateDIDProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(CreateDIDFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected %T, not %T", CreateDIDFact{}, op.Fact())), nil
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
				Wrap(common.ErrMServiceNF).Errorf("did service in contract account %v",
				fact.Contract(),
			)), nil
	}

	if found, _ := state.CheckNotExistsState(dstate.DataStateKey(fact.Contract(), fact.Sender().String()), getStateFunc); found {
		return nil, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMStateE).Errorf("did data for address %v in contract account %v",
				fact.Sender(), fact.Contract(),
			)), nil
	}

	return ctx, nil, nil
}

func (opp *CreateDIDProcessor) Process( // nolint:dupl
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, _ := op.Fact().(CreateDIDFact)

	st, _ := state.ExistsState(dstate.DesignStateKey(fact.Contract()), "did design", getStateFunc)

	design, err := dstate.GetDesignFromState(st)
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("service design value not found, %q; %w", fact.Contract(), err), nil
	}

	didData := types.NewData(
		fact.Sender(), design.DIDMethod(),
	)
	if err := didData.IsValid(nil); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("invalid did data; %w", err), nil
	}

	var sts []base.StateMergeValue // nolint:prealloc
	sts = append(sts, state.NewStateMergeValue(
		dstate.DataStateKey(fact.Contract(), fact.Sender().String()),
		dstate.NewDataStateValue(didData),
	))

	didr := didData.DIDResource()
	didr.SetFragment("auth_key")
	authentication := types.NewAsymmetricKeyAuthentication(didr.DIDUrl(), fact.authType, didData.DID(), fact.PublicKey())
	svc := types.NewService(didData.DID(), fact.ServiceType(), fact.ServiceEndpoint())

	didDocument := types.NewDIDDocument(didData.DID(),
		[]types.IAuthentication{authentication}, []types.IVerificationMethod{}, svc)
	if err := didDocument.IsValid(nil); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("invalid did document; %w", err), nil
	}
	sts = append(sts, state.NewStateMergeValue(
		dstate.DocumentStateKey(fact.Contract(), didData.DID()),
		dstate.NewDocumentStateValue(didDocument),
	))

	return sts, nil, nil
}

func (opp *CreateDIDProcessor) Close() error {
	createDIDProcessorPool.Put(opp)

	return nil
}
