package cmds

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	isaacdatabase "github.com/ProtoconNet/mitum2/isaac/database"
	isaacnetwork "github.com/ProtoconNet/mitum2/isaac/network"
	isaacstates "github.com/ProtoconNet/mitum2/isaac/states"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/network/quicmemberlist"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/ProtoconNet/mitum2/util/ps"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	PNameDigestDesign                   = ps.Name("digest-design")
	PNameGenerateGenesis                = ps.Name("mitum-currency-generate-genesis")
	PNameDigestAPIHandlers              = ps.Name("mitum-currency-digest-api-handlers")
	PNameDigesterFollowUp               = ps.Name("mitum-currency-followup_digester")
	BEncoderContextKey                  = util.ContextKey("bson-encoder")
	ProposalOperationFactHintContextKey = util.ContextKey("proposal-operation-fact-hint")
	OperationProcessorContextKey        = util.ContextKey("mitum-currency-operation-processor")
)

type ProposalOperationFactHintFunc func() func(hint.Hint) bool

func LoadFromStdInput() ([]byte, error) {
	var b []byte
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			b = append(b, sc.Bytes()...)
			b = append(b, []byte("\n")...)
		}

		if err := sc.Err(); err != nil {
			return nil, err
		}
	}

	return bytes.TrimSpace(b), nil
}

type NetworkIDFlag []byte

func (v *NetworkIDFlag) UnmarshalText(b []byte) error {
	*v = b

	return nil
}

func (v NetworkIDFlag) NetworkID() base.NetworkID {
	return base.NetworkID(v)
}

func PrettyPrint(out io.Writer, i interface{}) {
	var b []byte
	b, err := enc.Marshal(i)
	if err != nil {
		panic(err)
	}

	_, _ = fmt.Fprintln(out, string(b))
}

func AttachHandlerSendOperation(pctx context.Context) error {
	var log *logging.Logging
	var params *launch.LocalParams
	var db isaac.Database
	var pool *isaacdatabase.TempPool
	var states *isaacstates.States
	var svVoteF isaac.SuffrageVoteFunc
	var memberList *quicmemberlist.Memberlist

	if err := util.LoadFromContext(pctx,
		launch.LoggingContextKey, &log,
		launch.LocalParamsContextKey, &params,
		launch.CenterDatabaseContextKey, &db,
		launch.PoolDatabaseContextKey, &pool,
		launch.StatesContextKey, &states,
		launch.SuffrageVotingVoteFuncContextKey, &svVoteF,
		launch.MemberlistContextKey, &memberList,
	); err != nil {
		return err
	}

	sendOperationFilterF, err := SendOperationFilterFunc(pctx)
	if err != nil {
		return err
	}

	var gerror error

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameSendOperation,
		isaacnetwork.QuicstreamHandlerSendOperation(
			params.ISAAC.NetworkID(),
			pool,
			db.ExistsInStateOperation,
			sendOperationFilterF,
			svVoteF,
			func(ctx context.Context, id string, op base.Operation, b []byte) error {
				if broker := states.HandoverXBroker(); broker != nil {
					if err := broker.SendData(ctx, isaacstates.HandoverMessageDataTypeOperation, op); err != nil {
						log.Log().Error().Err(err).
							Interface("operation", op.Hash()).
							Msg("send operation data to handover y broker; ignored")
					}
				}

				return memberList.CallbackBroadcast2(b, id, nil)
			},
			params.MISC.MaxMessageSize,
		),
		nil,
	)

	return gerror
}

func SendOperationFilterFunc(ctx context.Context) (
	func(base.Operation) (bool, error),
	error,
) {
	var db isaac.Database
	var oprs *hint.CompatibleSet[isaac.NewOperationProcessorInternalFunc]
	var oprsB *hint.CompatibleSet[NewOperationProcessorInternalWithProposalFunc]
	var f ProposalOperationFactHintFunc

	if err := util.LoadFromContextOK(ctx,
		launch.CenterDatabaseContextKey, &db,
		launch.OperationProcessorsMapContextKey, &oprs,
		OperationProcessorsMapBContextKey, &oprsB,
		ProposalOperationFactHintContextKey, &f,
	); err != nil {
		return nil, err
	}

	operationFilterF := f()

	return func(op base.Operation) (bool, error) {
		switch hinter, ok := op.Fact().(hint.Hinter); {
		case !ok:
			return false, nil
		case !operationFilterF(hinter.Hint()):
			return false, errors.Errorf("Not supported operation")
		}
		var height base.Height

		switch m, found, err := db.LastBlockMap(); {
		case err != nil:
			return false, err
		case !found:
			return true, nil
		default:
			height = m.Manifest().Height()
		}

		f, closeF, err := OperationPreProcess(db, oprs, oprsB, op, height)
		if err != nil {
			return false, err
		}

		defer func() {
			_ = closeF()
		}()

		_, reason, err := f(context.Background(), db.State)
		if err != nil {
			return false, err
		}

		return reason == nil, reason
	}, nil
}

func IsSupportedProposalOperationFactHintFunc() func(hint.Hint) bool {
	return func(ht hint.Hint) bool {
		for i := range SupportedProposalOperationFactHinters {
			s := SupportedProposalOperationFactHinters[i].Hint
			if ht.Type() != s.Type() {
				continue
			}

			return ht.IsCompatible(s)
		}

		return false
	}
}

func OperationPreProcess(
	db isaac.Database,
	oprsA *hint.CompatibleSet[isaac.NewOperationProcessorInternalFunc],
	oprsB *hint.CompatibleSet[NewOperationProcessorInternalWithProposalFunc],
	op base.Operation,
	height base.Height,
) (
	preprocess func(context.Context, base.GetStateFunc) (context.Context, base.OperationProcessReasonError, error),
	cancel func() error,
	_ error,
) {
	fA, foundA := oprsA.Find(op.Hint())
	fB, foundB := oprsB.Find(op.Hint())
	if !foundA && !foundB {
		return op.PreProcess, util.EmptyCancelFunc, nil
	}

	if foundA {
		switch opp, err := fA(height, db.State); {
		case err != nil:
			return nil, nil, err
		default:
			return func(pctx context.Context, getStateFunc base.GetStateFunc) (
				context.Context, base.OperationProcessReasonError, error,
			) {
				return opp.PreProcess(pctx, op, getStateFunc)
			}, opp.Close, nil
		}
	}
	switch opp, err := fB(height, nil, db.State); {
	case err != nil:
		return nil, nil, err
	default:
		return func(pctx context.Context, getStateFunc base.GetStateFunc) (
			context.Context, base.OperationProcessReasonError, error,
		) {
			return opp.PreProcess(pctx, op, getStateFunc)
		}, opp.Close, nil
	}

}

func parseNodeValueDuration(value string) (time.Duration, error) {
	var s string
	if err := yaml.Unmarshal([]byte(value), &s); err != nil {
		return 0, errors.WithStack(err)
	}

	return util.ParseDuration(s)
}

func updateDesignMap(src []byte, dotPath string, newVal any) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if len(root.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("top-level must be a mapping node")
	}

	parts := strings.Split(dotPath, ".")
	cur := doc

	for i, p := range parts {
		last := i == len(parts)-1
		ki, val, found := findPair(cur, p)

		if !found {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: p,
			}
			if last {
				var nv *yaml.Node
				switch v := newVal.(type) {
				case *yaml.Node:
					nv = v
				default:
					var err error
					nv, err = coerceForExistingScalar(nil, newVal)
					if err != nil {
						return nil, fmt.Errorf("set %q: %w", p, err)
					}
				}
				cur.Content = append(cur.Content, keyNode, nv)
				break
			} else {
				mapNode := &yaml.Node{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
				}
				cur.Content = append(cur.Content, keyNode, mapNode)
				cur = mapNode
				continue
			}
		}

		if last {
			nv, err := coerceForExistingScalar(val, newVal)
			if err != nil {
				return nil, fmt.Errorf("set %q: %w", p, err)
			}
			cur.Content[ki+1] = nv
			break
		}

		if val.Kind != yaml.MappingNode {
			val.Kind, val.Tag, val.Content = yaml.MappingNode, "!!map", nil
		}
		cur = val
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	_ = enc.Close()
	return out.Bytes(), nil
}

func findPair(m *yaml.Node, key string) (int, *yaml.Node, bool) {
	if m.Kind != yaml.MappingNode {
		return -1, nil, false
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if k := m.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			return i, m.Content[i+1], true
		}
	}
	return -1, nil, false
}

func coerceForExistingScalar(existing *yaml.Node, v any) (*yaml.Node, error) {
	if existing == nil {
		switch val := v.(type) {
		case bool:
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!bool",
				Value: strconv.FormatBool(val),
			}, nil
		case int:
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!int",
				Value: strconv.Itoa(val),
			}, nil
		case int64:
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!int",
				Value: strconv.FormatInt(val, 10),
			}, nil
		case float64:
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!float",
				Value: strconv.FormatFloat(val, 'f', -1, 64),
			}, nil
		default:
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: fmt.Sprint(v),
			}, nil
		}
	}

	if existing.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("existing value is not a scalar (kind=%v)", existing.Kind)
	}

	switch existing.Tag {
	case "!!str":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprint(v)}, nil
	case "!!int":
		iv, err := coerceInt(v)
		if err != nil {
			return nil, err
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(iv)}, nil
	case "!!bool":
		bv, err := coerceBool(v)
		if err != nil {
			return nil, err
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(bv)}, nil
	default:
		return nil, fmt.Errorf("unsupported scalar tag %q at value %q", existing.Tag, existing.Value)
	}
}

func coerceInt(v any) (int, error) {
	switch t := v.(type) {
	case int:
		return t, nil
	case int64:
		return int(t), nil
	case string:
		x, err := strconv.Atoi(t)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %q as int", t)
		}
		return x, nil
	default:
		return 0, fmt.Errorf("type %T cannot be coerced to int", v)
	}
}

func coerceBool(v any) (bool, error) {
	switch t := v.(type) {
	case bool:
		return t, nil
	case string:
		switch strings.ToLower(t) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf("cannot parse %q as bool", t)
		}
	default:
		return false, fmt.Errorf("type %T cannot be coerced to bool", v)
	}
}
