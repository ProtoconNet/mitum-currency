package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/digest"
	did "github.com/ProtoconNet/mitum-currency/v3/operation/did-registry"
	"os"
	"path/filepath"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	isaacoperation "github.com/ProtoconNet/mitum-currency/v3/operation/isaac"
	"github.com/ProtoconNet/mitum-currency/v3/operation/plugin"
	"github.com/ProtoconNet/mitum-currency/v3/operation/processor"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	isaacdatabase "github.com/ProtoconNet/mitum2/isaac/database"
	isaacnetwork "github.com/ProtoconNet/mitum2/isaac/network"
	isaacstates "github.com/ProtoconNet/mitum2/isaac/states"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/network/quicmemberlist"
	"github.com/ProtoconNet/mitum2/storage"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	jsonenc "github.com/ProtoconNet/mitum2/util/encoder/json"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type NewOperationProcessorInternalWithProposalFunc func(base.Height, base.ProposalSignFact, base.GetStateFunc) (base.OperationProcessor, error)

func POperationProcessorsMap(pctx context.Context) (context.Context, error) {
	var isaacParams *isaac.Params
	var db isaac.Database
	var encs *encoder.Encoders

	if err := util.LoadFromContextOK(pctx,
		launch.ISAACParamsContextKey, &isaacParams,
		launch.CenterDatabaseContextKey, &db,
		launch.EncodersContextKey, &encs,
	); err != nil {
		return pctx, err
	}

	limiterF, err := launch.NewSuffrageCandidateLimiterFunc(pctx)
	if err != nil {
		return pctx, err
	}

	setA := hint.NewCompatibleSet[isaac.NewOperationProcessorInternalFunc](1 << 9)
	setB := hint.NewCompatibleSet[NewOperationProcessorInternalWithProposalFunc](1 << 9)

	opr := processor.NewOperationProcessor()
	err = opr.SetCheckDuplicationFunc(processor.CheckDuplication)
	if err != nil {
		return pctx, err
	}
	err = opr.SetGetNewProcessorFunc(processor.GetNewProcessor)
	if err != nil {
		return pctx, err
	}
	if err := opr.SetProcessor(
		currency.CreateAccountHint,
		currency.NewCreateAccountProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.UpdateKeyHint,
		currency.NewUpdateKeyProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.TransferHint,
		currency.NewTransferProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.RegisterCurrencyHint,
		currency.NewRegisterCurrencyProcessor(isaacParams.Threshold()),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.UpdateCurrencyHint,
		currency.NewUpdateCurrencyProcessor(isaacParams.Threshold()),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.MintHint,
		currency.NewMintProcessor(isaacParams.Threshold()),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		extension.CreateContractAccountHint,
		extension.NewCreateContractAccountProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		extension.UpdateHandlerHint,
		extension.NewUpdateHandlerProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		extension.UpdateRecipientHint,
		extension.NewUpdateRecipientProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		extension.WithdrawHint,
		extension.NewWithdrawProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		did.RegisterModelHint,
		did.NewRegisterModelProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		did.CreateDIDHint,
		did.NewCreateDIDProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		did.UpdateDIDDocumentHint,
		did.NewUpdateDIDDocumentProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		plugin.RegisterModelHint,
		plugin.NewRegisterModelProcessor(*encs),
	); err != nil {
		return pctx, err
	}

	_ = setA.Add(currency.CreateAccountHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(currency.UpdateKeyHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(currency.TransferHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(currency.RegisterCurrencyHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(currency.UpdateCurrencyHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(currency.MintHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(extension.CreateContractAccountHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(extension.UpdateHandlerHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(extension.UpdateRecipientHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(extension.WithdrawHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return opr.New(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(did.CreateDIDHint, func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
		return opr.New(
			height,
			getStatef,
			nil,
			nil,
		)
	})

	_ = setA.Add(did.RegisterModelHint, func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
		return opr.New(
			height,
			getStatef,
			nil,
			nil,
		)
	})

	_ = setA.Add(did.UpdateDIDDocumentHint, func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
		return opr.New(
			height,
			getStatef,
			nil,
			nil,
		)
	})

	_ = setA.Add(isaacoperation.SuffrageCandidateHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			policy := db.LastNetworkPolicy()
			if policy == nil { // NOTE Usually it means empty block data
				return nil, nil
			}

			return isaacoperation.NewSuffrageCandidateProcessor(
				height,
				getStatef,
				limiterF,
				nil,
				policy.SuffrageCandidateLifespan(),
			)
		})

	_ = setA.Add(isaacoperation.SuffrageJoinHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			policy := db.LastNetworkPolicy()
			if policy == nil { // NOTE Usually it means empty block data
				return nil, nil
			}

			return isaacoperation.NewSuffrageJoinProcessor(
				height,
				isaacParams.Threshold(),
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(isaac.SuffrageExpelOperationHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			policy := db.LastNetworkPolicy()
			if policy == nil { // NOTE Usually it means empty block data
				return nil, nil
			}

			return isaacoperation.NewSuffrageExpelProcessor(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(isaacoperation.SuffrageDisjoinHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return isaacoperation.NewSuffrageDisjoinProcessor(
				height,
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(isaacoperation.NetworkPolicyHint,
		func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
			return isaacoperation.NewNetworkPolicyProcessor(
				height,
				isaacParams.Threshold(),
				getStatef,
				nil,
				nil,
			)
		})

	_ = setA.Add(plugin.RegisterModelHint, func(height base.Height, getStatef base.GetStateFunc) (base.OperationProcessor, error) {
		return opr.New(
			height,
			getStatef,
			nil,
			nil,
		)
	})

	//var f ProposalOperationFactHintFunc = IsSupportedProposalOperationFactHintFunc

	pctx = context.WithValue(pctx, OperationProcessorContextKey, opr)
	pctx = context.WithValue(pctx, launch.OperationProcessorsMapContextKey, setA) //revive:disable-line:modifies-parameter
	pctx = context.WithValue(pctx, OperationProcessorsMapBContextKey, setB)       //revive:disable-line:modifies-parameter
	//pctx = context.WithValue(pctx, ProposalOperationFactHintContextKey, f)

	return pctx, nil
}

func PGenerateGenesis(pctx context.Context) (context.Context, error) {
	e := util.StringError("generate genesis block")

	var log *logging.Logging
	var design launch.NodeDesign
	var genesisDesign launch.GenesisDesign
	var encs *encoder.Encoders
	var local base.LocalNode
	var isaacParams *isaac.Params
	var db isaac.Database
	var fsnodeinfo launch.NodeInfo
	var eventLogging *launch.EventLogging
	var newReaders func(context.Context, string, *isaac.BlockItemReadersArgs) (*isaac.BlockItemReaders, error)

	if err := util.LoadFromContextOK(pctx,
		launch.LoggingContextKey, &log,
		launch.DesignContextKey, &design,
		launch.GenesisDesignContextKey, &genesisDesign,
		launch.EncodersContextKey, &encs,
		launch.LocalContextKey, &local,
		launch.ISAACParamsContextKey, &isaacParams,
		launch.CenterDatabaseContextKey, &db,
		launch.FSNodeInfoContextKey, &fsnodeinfo,
		launch.EventLoggingContextKey, &eventLogging,
		launch.NewBlockItemReadersFuncContextKey, &newReaders,
	); err != nil {
		return pctx, e.Wrap(err)
	}

	var el zerolog.Logger

	switch i, found := eventLogging.Logger(launch.NodeEventLogger); {
	case !found:
		return pctx, errors.Errorf("Node event logger not found")
	default:
		el = i
	}

	root := launch.LocalFSDataDirectory(design.Storage.Base)

	var readers *isaac.BlockItemReaders

	switch i, err := newReaders(pctx, root, nil); {
	case err != nil:
		return pctx, err
	default:
		defer i.Close()

		readers = i
	}

	g := NewGenesisBlockGenerator(
		local,
		isaacParams.NetworkID(),
		encs,
		db,
		root,
		genesisDesign.Facts,
		func() (base.BlockMap, bool, error) {
			return isaac.BlockItemReadersDecode[base.BlockMap](
				readers.Item,
				base.GenesisHeight,
				base.BlockItemMap,
				nil,
			)
		},
		pctx,
	)
	_ = g.SetLogging(log)

	if _, err := g.Generate(); err != nil {
		return pctx, e.Wrap(err)
	}

	el.Debug().Interface("node_info", fsnodeinfo).Msg("node initialized")

	return pctx, nil
}

func PEncoder(pctx context.Context) (context.Context, error) {
	e := util.StringError("prepare encoders")

	jenc := jsonenc.NewEncoder()
	encs := encoder.NewEncoders(jenc, jenc)
	benc := bsonenc.NewEncoder()

	if err := encs.AddEncoder(benc); err != nil {
		return pctx, e.Wrap(err)
	}

	return util.ContextWithValues(pctx, map[util.ContextKey]interface{}{
		launch.EncodersContextKey: encs,
		BEncoderContextKey:        benc,
	}), nil
}

func PLoadDigestDesign(pctx context.Context) (context.Context, error) {
	e := util.StringError("load design")

	var log *logging.Logging
	var flag launch.DesignFlag

	if err := util.LoadFromContextOK(pctx,
		launch.LoggingContextKey, &log,
		launch.DesignFlagContextKey, &flag,
	); err != nil {
		return pctx, e.Wrap(err)
	}

	switch flag.Scheme() {
	case "file":
		b, err := os.ReadFile(filepath.Clean(flag.URL().Path))
		if err != nil {
			return pctx, e.Wrap(err)
		}

		var m struct {
			API *digest.YamlDigestDesign `yaml:"api"`
			//Sequencer *digest.YamlSequencerDesign `yaml:"sequencer"`
		}

		nb, err := util.ReplaceEnvVariables(b)
		if err != nil {
			return pctx, e.Wrap(err)
		}

		if err := yaml.Unmarshal(nb, &m); err != nil {
			return pctx, e.Wrap(err)
		} else if m.API == nil {
			return pctx, nil
		} else if i, err := m.API.Set(pctx); err != nil {
			return pctx, e.Wrap(err)
		} else {
			pctx = i
		}

		pctx = context.WithValue(pctx, digest.ContextValueDigestDesign, *m.API)
		//pctx = context.WithValue(pctx, digest.ContextValueSequencerDesign, *m.Sequencer)

		log.Log().Debug().Object("design", *m.API).Msg("digest design loaded")
	default:
		return pctx, e.Errorf("Unknown digest design uri, %q", flag.URL())
	}

	return pctx, nil
}

func PNetworkHandlers(pctx context.Context) (context.Context, error) {
	e := util.StringError("prepare network handlers")

	var log *logging.Logging
	var encs *encoder.Encoders
	var design launch.NodeDesign
	var local base.LocalNode
	var params *launch.LocalParams
	var db isaac.Database
	var pool *isaacdatabase.TempPool
	var proposalMaker *isaac.ProposalMaker
	var m *quicmemberlist.Memberlist
	var syncSourcePool *isaac.SyncSourcePool
	var nodeinfo *isaacnetwork.NodeInfoUpdater
	var svVoteF isaac.SuffrageVoteFunc
	var ballotBox *isaacstates.Ballotbox
	var filterNotifyMsg quicmemberlist.FilterNotifyMsgFunc
	var lvps *isaac.LastVoteproofsHandler

	if err := util.LoadFromContextOK(pctx,
		launch.LoggingContextKey, &log,
		launch.EncodersContextKey, &encs,
		launch.DesignContextKey, &design,
		launch.LocalContextKey, &local,
		launch.LocalParamsContextKey, &params,
		launch.CenterDatabaseContextKey, &db,
		launch.PoolDatabaseContextKey, &pool,
		launch.ProposalMakerContextKey, &proposalMaker,
		launch.MemberlistContextKey, &m,
		launch.SyncSourcePoolContextKey, &syncSourcePool,
		launch.NodeInfoContextKey, &nodeinfo,
		launch.SuffrageVotingVoteFuncContextKey, &svVoteF,
		launch.BallotboxContextKey, &ballotBox,
		launch.FilterMemberlistNotifyMsgFuncContextKey, &filterNotifyMsg,
		launch.LastVoteproofsHandlerContextKey, &lvps,
	); err != nil {
		return pctx, e.Wrap(err)
	}

	isaacParams := params.ISAAC

	lastBlockMapF := launch.QuicstreamHandlerLastBlockMapFunc(db)
	suffrageNodeConnInfoF := launch.QuicstreamHandlerSuffrageNodeConnInfoFunc(db, m)

	var gerror error

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameLastSuffrageProof,
		isaacnetwork.QuicstreamHandlerLastSuffrageProof(
			func(last util.Hash) (string, []byte, []byte, bool, error) {
				enchint, metab, body, found, lastheight, err := db.LastSuffrageProofBytes()

				switch {
				case err != nil:
					return enchint, nil, nil, false, err
				case !found:
					return enchint, nil, nil, false, storage.ErrNotFound.Errorf("Last SuffrageProof not found")
				}

				switch {
				case last != nil && len(metab) > 0 && valuehash.NewBytes(metab).Equal(last):
					nbody, _ := util.NewLengthedBytesSlice([][]byte{lastheight.Bytes(), nil})

					return enchint, nil, nbody, false, nil
				default:
					nbody, _ := util.NewLengthedBytesSlice([][]byte{lastheight.Bytes(), body})

					return enchint, metab, nbody, true, nil
				}
			},
		), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameSuffrageProof,
		isaacnetwork.QuicstreamHandlerSuffrageProof(db.SuffrageProofBytes), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameLastBlockMap,
		isaacnetwork.QuicstreamHandlerLastBlockMap(lastBlockMapF), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameBlockMap,
		isaacnetwork.QuicstreamHandlerBlockMap(db.BlockMapBytes), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameNodeChallenge,
		isaacnetwork.QuicstreamHandlerNodeChallenge(isaacParams.NetworkID(), local), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameSuffrageNodeConnInfo,
		isaacnetwork.QuicstreamHandlerSuffrageNodeConnInfo(suffrageNodeConnInfoF), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameSyncSourceConnInfo,
		isaacnetwork.QuicstreamHandlerSyncSourceConnInfo(
			func() ([]isaac.NodeConnInfo, error) {
				members := make([]isaac.NodeConnInfo, syncSourcePool.Len()*2)

				var i int
				syncSourcePool.Actives(func(nci isaac.NodeConnInfo) bool {
					members[i] = nci
					i++

					return true
				})

				return members[:i], nil
			},
		), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameState,
		isaacnetwork.QuicstreamHandlerState(db.StateBytes), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameExistsInStateOperation,
		isaacnetwork.QuicstreamHandlerExistsInStateOperation(db.ExistsInStateOperation), nil)

	if vp := lvps.Last().Cap(); vp != nil {
		_ = nodeinfo.SetLastVote(vp.Point(), vp.Result())
	}

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameNodeInfo,
		isaacnetwork.QuicstreamHandlerNodeInfo(launch.QuicstreamHandlerGetNodeInfoFunc(enc, nodeinfo)), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerNameSendBallots,
		isaacnetwork.QuicstreamHandlerSendBallots(isaacParams.NetworkID(),
			func(bl base.BallotSignFact) error {
				switch passed, err := filterNotifyMsg(bl); {
				case err != nil:
					log.Log().Trace().
						Str("module", "filter-notify-msg-send-ballots").
						Err(err).
						Interface("handover_message", bl).
						Msg("filter error")

					fallthrough
				case !passed:
					log.Log().Trace().
						Str("module", "filter-notify-msg-send-ballots").
						Interface("handover_message", bl).
						Msg("filtered")

					return nil
				}

				_, err := ballotBox.VoteSignFact(bl)

				return err
			},
			params.MISC.MaxMessageSize,
		), nil)

	if gerror != nil {
		return pctx, gerror
	}

	if err := launch.AttachBlockItemsNetworkHandlers(pctx); err != nil {
		return pctx, err
	}

	if err := launch.AttachMemberlistNetworkHandlers(pctx); err != nil {
		return pctx, err
	}

	return pctx, nil
}

func PStatesNetworkHandlers(pctx context.Context) (context.Context, error) {
	if err := launch.AttachHandlerOperation(pctx); err != nil {
		return pctx, err
	}

	if err := AttachHandlerSendOperation(pctx); err != nil {
		return pctx, err
	}

	if err := launch.AttachHandlerStreamOperations(pctx); err != nil {
		return pctx, err
	}

	if err := launch.AttachHandlerProposals(pctx); err != nil {
		return pctx, err
	}

	return pctx, nil
}

func PSuffrageCandidateLimiterSet(pctx context.Context) (context.Context, error) {
	e := util.StringError("prepare SuffrageCandidateLimiterSet")

	var db isaac.Database
	if err := util.LoadFromContextOK(pctx, launch.CenterDatabaseContextKey, &db); err != nil {
		return pctx, e.Wrap(err)
	}

	set := hint.NewCompatibleSet[base.SuffrageCandidateLimiterFunc](8) //nolint:gomnd //...

	if err := set.Add(
		isaacoperation.FixedSuffrageCandidateLimiterRuleHint,
		base.SuffrageCandidateLimiterFunc(FixedSuffrageCandidateLimiterFunc()),
	); err != nil {
		return pctx, e.Wrap(err)
	}

	if err := set.Add(
		isaacoperation.MajoritySuffrageCandidateLimiterRuleHint,
		base.SuffrageCandidateLimiterFunc(MajoritySuffrageCandidateLimiterFunc(db)),
	); err != nil {
		return pctx, e.Wrap(err)
	}

	return context.WithValue(pctx, launch.SuffrageCandidateLimiterSetContextKey, set), nil
}

func FixedSuffrageCandidateLimiterFunc() func(
	base.SuffrageCandidateLimiterRule,
) (base.SuffrageCandidateLimiter, error) {
	return func(rule base.SuffrageCandidateLimiterRule) (base.SuffrageCandidateLimiter, error) {
		switch i, err := util.AssertInterfaceValue[isaacoperation.FixedSuffrageCandidateLimiterRule](rule); {
		case err != nil:
			return nil, err
		default:
			return isaacoperation.NewFixedSuffrageCandidateLimiter(i), nil
		}
	}
}

func MajoritySuffrageCandidateLimiterFunc(
	db isaac.Database,
) func(base.SuffrageCandidateLimiterRule) (base.SuffrageCandidateLimiter, error) {
	return func(rule base.SuffrageCandidateLimiterRule) (base.SuffrageCandidateLimiter, error) {
		var i isaacoperation.MajoritySuffrageCandidateLimiterRule
		if err := util.SetInterfaceValue(rule, &i); err != nil {
			return nil, err
		}

		proof, found, err := db.LastSuffrageProof()

		switch {
		case err != nil:
			return nil, errors.WithMessagef(err, "get last suffrage for MajoritySuffrageCandidateLimiter")
		case !found:
			return nil, errors.Errorf("last suffrage not found for MajoritySuffrageCandidateLimiter")
		}

		suf, err := proof.Suffrage()
		if err != nil {
			return nil, errors.WithMessagef(err, "get suffrage for MajoritySuffrageCandidateLimiter")
		}

		return isaacoperation.NewMajoritySuffrageCandidateLimiter(
			i,
			func() (uint64, error) {
				return uint64(suf.Len()), nil
			},
		), nil
	}
}
