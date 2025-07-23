package plugin

import (
	"context"
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	cstate "github.com/ProtoconNet/mitum-currency/v3/state"
	cestate "github.com/ProtoconNet/mitum-currency/v3/state/extension"
	pstate "github.com/ProtoconNet/mitum-currency/v3/state/plugin"
	currencytypes "github.com/ProtoconNet/mitum-currency/v3/types"
	ptypes "github.com/ProtoconNet/mitum-currency/v3/types/plugin"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/pkg/errors"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"os"
	"plugin"
	"reflect"
	"sync"
	"time"
)

var registerModelProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(RegisterModelProcessor)
	},
}

const ContractMemoryLimit = 32

type RegisterModelProcessor struct {
	*base.BaseOperationProcessor
	encs *encoder.Encoders
}

func NewRegisterModelProcessor(encs encoder.Encoders) currencytypes.GetNewProcessor {
	return func(
		height base.Height,
		getStateFunc base.GetStateFunc,
		newPreProcessConstraintFunc base.NewOperationProcessorProcessFunc,
		newProcessConstraintFunc base.NewOperationProcessorProcessFunc,
	) (base.OperationProcessor, error) {
		e := util.StringError("failed to create new RegisterModelProcessor")

		nopp := registerModelProcessorPool.Get()
		opp, ok := nopp.(*RegisterModelProcessor)
		if !ok {
			return nil, errors.Errorf("expected RegisterModelProcessor, not %T", nopp)
		}

		b, err := base.NewBaseOperationProcessor(
			height, getStateFunc, newPreProcessConstraintFunc, newProcessConstraintFunc)
		if err != nil {
			return nil, e.Wrap(err)
		}

		if opp.encs == nil {
			opp.encs = &encs
		}

		opp.BaseOperationProcessor = b

		return opp, nil
	}
}

func (opp *RegisterModelProcessor) PreProcess(
	ctx context.Context, op base.Operation, getStateFunc base.GetStateFunc,
) (context.Context, base.OperationProcessReasonError, error) {
	fact, ok := op.Fact().(RegisterModelFact)
	if !ok {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMTypeMismatch).
				Errorf("expected %T, not %T", RegisterModelFact{}, op.Fact())), nil
	}

	if err := fact.IsValid(nil); err != nil {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("%v", err)), nil
	}

	if found, _ := cstate.CheckNotExistsState(pstate.DesignStateKey(fact.Contract()), getStateFunc); found {
		return ctx, base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Wrap(common.ErrMServiceE).Errorf("wasm service for contract account %v",
				fact.Contract(),
			)), nil
	}

	return ctx, nil, nil
}

func (opp *RegisterModelProcessor) Process(
	_ context.Context, op base.Operation, getStateFunc base.GetStateFunc) (
	[]base.StateMergeValue, base.OperationProcessReasonError, error,
) {
	fact, _ := op.Fact().(RegisterModelFact)

	var sts []base.StateMergeValue

	//contractCode, err := base64.StdEncoding.DecodeString(fact.ContractCode())
	//if err != nil {
	//	return nil, base.NewBaseOperationProcessReasonError("failed to decode contract code, %v: %w", fact.Contract(), err), nil
	//}

	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	i.Use(interp.Exports{
		"github.com/ProtoconNet/mitum-currency/v3/operation/plugin": map[string]reflect.Value{
			"GetAccountFunc": reflect.ValueOf(reflect.TypeOf((*GetAccountFunc)(nil)).Elem()),
		},
	})

	_, err := i.EvalPath(fact.ContractCode())
	if err != nil {
		fmt.Println(err)
		return nil, base.NewBaseOperationProcessReasonError("failed to load plugin, %v: %w", fact.Contract(), err), nil
	}

	symRun, err := i.Eval("main.Run")
	if err != nil {
		return nil, base.NewBaseOperationProcessReasonError("failed to lookup function, %v: %w", fact.Contract(), err), nil
	}

	runFunc, ok := symRun.Interface().(func(map[string]string, GetAccountFunc) string)
	if !ok {
		return nil, base.NewBaseOperationProcessReasonError("failed to cast function, %v: %w", fact.Contract(), err), nil
	}

	callData := map[string]string{
		"sender": fact.Sender().String(),
	}

	getAccount := NewGetAccountFunc(*opp.encs, getStateFunc)

	result := runFunc(callData, getAccount)
	fmt.Println(result)

	design := ptypes.NewDesign(fact.ContractCode())
	if err := design.IsValid(nil); err != nil {
		return nil, base.NewBaseOperationProcessReasonError("invalid plugin design, %q; %w", fact.Contract(), err), nil
	}

	sts = append(sts, cstate.NewStateMergeValue(
		pstate.DesignStateKey(fact.Contract()),
		pstate.NewDesignStateValue(design),
	))

	st, _ := cstate.ExistsState(cestate.StateKeyContractAccount(fact.Contract()), "contract account", getStateFunc)
	ca, _ := cestate.StateContractAccountValue(st)
	nca := ca.SetActive(true)

	sts = append(sts, cstate.NewStateMergeValue(
		cestate.StateKeyContractAccount(fact.Contract()),
		cestate.NewContractAccountStateValue(nca),
	))

	return sts, nil, nil
}

func (opp *RegisterModelProcessor) Close() error {
	registerModelProcessorPool.Put(opp)

	return nil
}

func LoadPluginFromBytes(data []byte) (*plugin.Plugin, error) {
	fmt.Println("start", time.Now())
	tmpFile, err := os.CreateTemp("", "*.so")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	time.Sleep(6 * time.Second)
	fmt.Println("end", time.Now())
	return plugin.Open(tmpFile.Name())
}

type ContractFunc func(context.Context) (string, error)

func RunWithTimeout(fn ContractFunc, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		res, err := fn(ctx)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	select {
	case <-ctx.Done():
		return "", errors.New("execution timed out")
	case err := <-errCh:
		return "", err
	case res := <-resultCh:
		return res, nil
	}
}

func safeOpenPlugin(path string) (*plugin.Plugin, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	return p, nil
}
