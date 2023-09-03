package cmds

import (
	"context"
	"io"
	"os"
	"path/filepath"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	"github.com/ProtoconNet/mitum-currency/v3/operation/processor"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	isaacblock "github.com/ProtoconNet/mitum2/isaac/block"
	isaacdatabase "github.com/ProtoconNet/mitum2/isaac/database"
	isaacnetwork "github.com/ProtoconNet/mitum2/isaac/network"
	isaacoperation "github.com/ProtoconNet/mitum2/isaac/operation"
	isaacstates "github.com/ProtoconNet/mitum2/isaac/states"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/network/quicmemberlist"
	"github.com/ProtoconNet/mitum2/network/quicstream"
	"github.com/ProtoconNet/mitum2/storage"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	jsonenc "github.com/ProtoconNet/mitum2/util/encoder/json"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/logging"
	"gopkg.in/yaml.v3"
)

func POperationProcessorsMap(pctx context.Context) (context.Context, error) {
	var isaacParams *isaac.Params
	var db isaac.Database

	if err := util.LoadFromContextOK(pctx,
		launch.ISAACParamsContextKey, &isaacParams,
		launch.CenterDatabaseContextKey, &db,
	); err != nil {
		return pctx, err
	}

	limiterF, err := launch.NewSuffrageCandidateLimiterFunc(pctx)
	if err != nil {
		return pctx, err
	}

	set := hint.NewCompatibleSet[isaac.NewOperationProcessorInternalFunc](1 << 9)

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
		currency.CreateAccountsHint,
		currency.NewCreateAccountsProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.KeyUpdaterHint,
		currency.NewKeyUpdaterProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.TransfersHint,
		currency.NewTransfersProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.CurrencyRegisterHint,
		currency.NewCurrencyRegisterProcessor(isaacParams.Threshold()),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.CurrencyPolicyUpdaterHint,
		currency.NewCurrencyPolicyUpdaterProcessor(isaacParams.Threshold()),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		currency.SuffrageInflationHint,
		currency.NewSuffrageInflationProcessor(isaacParams.Threshold()),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		extension.CreateContractAccountHint,
		extension.NewCreateContractAccountProcessor(),
	); err != nil {
		return pctx, err
	} else if err := opr.SetProcessor(
		extension.WithdrawHint,
		extension.NewWithdrawProcessor(),
	); err != nil {
		return pctx, err
	}

	_ = set.Add(currency.CreateAccountsHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(currency.KeyUpdaterHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(currency.TransfersHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(currency.CurrencyRegisterHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(currency.CurrencyPolicyUpdaterHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(currency.SuffrageInflationHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(extension.CreateContractAccountHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(extension.WithdrawHint, func(height base.Height) (base.OperationProcessor, error) {
		return opr.New(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(isaacoperation.SuffrageCandidateHint, func(height base.Height) (base.OperationProcessor, error) {
		policy := db.LastNetworkPolicy()
		if policy == nil { // NOTE Usually it means empty block data
			return nil, nil
		}

		return isaacoperation.NewSuffrageCandidateProcessor(
			height,
			db.State,
			limiterF,
			nil,
			policy.SuffrageCandidateLifespan(),
		)
	})

	_ = set.Add(isaacoperation.SuffrageJoinHint, func(height base.Height) (base.OperationProcessor, error) {
		policy := db.LastNetworkPolicy()
		if policy == nil { // NOTE Usually it means empty block data
			return nil, nil
		}

		return isaacoperation.NewSuffrageJoinProcessor(
			height,
			isaacParams.Threshold(),
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(isaac.SuffrageExpelOperationHint, func(height base.Height) (base.OperationProcessor, error) {
		policy := db.LastNetworkPolicy()
		if policy == nil { // NOTE Usually it means empty block data
			return nil, nil
		}

		return isaacoperation.NewSuffrageExpelProcessor(
			height,
			db.State,
			nil,
			nil,
		)
	})

	_ = set.Add(isaacoperation.SuffrageDisjoinHint, func(height base.Height) (base.OperationProcessor, error) {
		return isaacoperation.NewSuffrageDisjoinProcessor(
			height,
			db.State,
			nil,
			nil,
		)
	})

	var f ProposalOperationFactHintFunc = IsSupportedProposalOperationFactHintFunc

	pctx = context.WithValue(pctx, OperationProcessorContextKey, opr)
	pctx = context.WithValue(pctx, launch.OperationProcessorsMapContextKey, set) //revive:disable-line:modifies-parameter
	pctx = context.WithValue(pctx, ProposalOperationFactHintContextKey, f)

	return pctx, nil
}

func PGenerateGenesis(pctx context.Context) (context.Context, error) {
	e := util.StringError("generate genesis block")

	var log *logging.Logging
	var design launch.NodeDesign
	var genesisDesign launch.GenesisDesign
	var enc encoder.Encoder
	var local base.LocalNode
	var isaacParams *isaac.Params
	var db isaac.Database

	if err := util.LoadFromContextOK(pctx,
		launch.LoggingContextKey, &log,
		launch.DesignContextKey, &design,
		launch.GenesisDesignContextKey, &genesisDesign,
		launch.EncoderContextKey, &enc,
		launch.LocalContextKey, &local,
		launch.ISAACParamsContextKey, &isaacParams,
		launch.CenterDatabaseContextKey, &db,
	); err != nil {
		return pctx, e.Wrap(err)
	}

	g := NewGenesisBlockGenerator(
		local,
		isaacParams.NetworkID(),
		enc,
		db,
		launch.LocalFSDataDirectory(design.Storage.Base),
		genesisDesign.Facts,
	)
	_ = g.SetLogging(log)

	if _, err := g.Generate(); err != nil {
		return pctx, e.Wrap(err)
	}

	return pctx, nil
}

func PEncoder(pctx context.Context) (context.Context, error) {
	e := util.StringError("prepare encoders")

	encs := encoder.NewEncoders()
	jenc := jsonenc.NewEncoder()
	benc := bsonenc.NewEncoder()

	if err := encs.AddHinter(jenc); err != nil {
		return pctx, e.Wrap(err)
	}
	if err := encs.AddHinter(benc); err != nil {
		return pctx, e.Wrap(err)
	}

	return util.ContextWithValues(pctx, map[util.ContextKey]interface{}{
		launch.EncodersContextKey: encs,
		launch.EncoderContextKey:  jenc,
		BEncoderContextKey:        benc,
	}), nil
}

func PLoadDigestDesign(pctx context.Context) (context.Context, error) {
	e := util.StringError("load design")

	var log *logging.Logging
	var flag launch.DesignFlag
	var enc *jsonenc.Encoder

	if err := util.LoadFromContextOK(pctx,
		launch.LoggingContextKey, &log,
		launch.DesignFlagContextKey, &flag,
		launch.EncoderContextKey, &enc,
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
			Digest *DigestDesign
		}

		if err := yaml.Unmarshal(b, &m); err != nil {
			return pctx, err
		} else if m.Digest == nil {
			return pctx, nil
		} else if i, err := m.Digest.Set(pctx); err != nil {
			return pctx, err
		} else {
			pctx = i
		}

		pctx = context.WithValue(pctx, ContextValueDigestDesign, *m.Digest)

		log.Log().Debug().Object("design", *m.Digest).Msg("digest design loaded")
	default:
		return pctx, e.Errorf("unknown digest design uri, %q", flag.URL())
	}

	return pctx, nil
}

func PNetworkHandlers(pctx context.Context) (context.Context, error) {
	e := util.StringError("prepare network handlers")

	var log *logging.Logging
	var encs *encoder.Encoders
	var enc encoder.Encoder
	var design launch.NodeDesign
	var local base.LocalNode
	var params *launch.LocalParams
	var db isaac.Database
	var pool *isaacdatabase.TempPool
	var proposalMaker *isaac.ProposalMaker
	var m *quicmemberlist.Memberlist
	var syncSourcePool *isaac.SyncSourcePool
	var handlers *quicstream.PrefixHandler
	var nodeInfo *isaacnetwork.NodeInfoUpdater
	var svVoteF isaac.SuffrageVoteFunc
	var ballotBox *isaacstates.Ballotbox
	var filterNotifyMsg quicmemberlist.FilterNotifyMsgFunc

	if err := util.LoadFromContextOK(pctx,
		launch.LoggingContextKey, &log,
		launch.EncodersContextKey, &encs,
		launch.EncoderContextKey, &enc,
		launch.DesignContextKey, &design,
		launch.LocalContextKey, &local,
		launch.LocalParamsContextKey, &params,
		launch.CenterDatabaseContextKey, &db,
		launch.PoolDatabaseContextKey, &pool,
		launch.ProposalMakerContextKey, &proposalMaker,
		launch.MemberlistContextKey, &m,
		launch.SyncSourcePoolContextKey, &syncSourcePool,
		launch.QuicstreamHandlersContextKey, &handlers,
		launch.NodeInfoContextKey, &nodeInfo,
		launch.SuffrageVotingVoteFuncContextKey, &svVoteF,
		launch.BallotboxContextKey, &ballotBox,
		launch.FilterMemberlistNotifyMsgFuncContextKey, &filterNotifyMsg,
	); err != nil {
		return pctx, e.Wrap(err)
	}

	isaacParams := params.ISAAC

	lastBlockMapF := launch.QuicstreamHandlerLastBlockMapFunc(db)
	suffrageNodeConnInfoF := launch.QuicstreamHandlerSuffrageNodeConnInfoFunc(db, m)

	var gerror error

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixLastSuffrageProofString,
		isaacnetwork.QuicstreamHandlerLastSuffrageProof(
			func(last util.Hash) (string, []byte, []byte, bool, error) {
				enchint, metabytes, body, found, lastheight, err := db.LastSuffrageProofBytes()

				switch {
				case err != nil:
					return enchint, nil, nil, false, err
				case !found:
					return enchint, nil, nil, false, storage.ErrNotFound.Errorf("last SuffrageProof not found")
				}

				switch h, err := isaacdatabase.ReadHashRecordMeta(metabytes); {
				case err != nil:
					return enchint, nil, nil, true, err
				case last != nil && last.Equal(h):
					nbody, _ := util.NewLengthedBytesSlice(0x01, [][]byte{lastheight.Bytes(), nil})

					return enchint, nil, nbody, false, nil
				default:
					nbody, _ := util.NewLengthedBytesSlice(0x01, [][]byte{lastheight.Bytes(), body})

					return enchint, metabytes, nbody, true, nil
				}
			},
		), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixSuffrageProofString,
		isaacnetwork.QuicstreamHandlerSuffrageProof(db.SuffrageProofBytes), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixLastBlockMapString,
		isaacnetwork.QuicstreamHandlerLastBlockMap(lastBlockMapF), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixBlockMapString,
		isaacnetwork.QuicstreamHandlerBlockMap(db.BlockMapBytes), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixBlockMapItemString,
		isaacnetwork.QuicstreamHandlerBlockMapItem(
			func(height base.Height, item base.BlockMapItemType) (io.ReadCloser, bool, error) {
				e := util.StringError("get BlockMapItem")

				var menc encoder.Encoder

				switch m, found, err := db.BlockMap(height); {
				case err != nil:
					return nil, false, e.Wrap(err)
				case !found:
					return nil, false, e.Wrap(storage.ErrNotFound.Errorf("BlockMap not found"))
				default:
					i, found := encs.Find(m.Encoder())
					if !found {
						return nil, false, e.Wrap(storage.ErrNotFound.Errorf("encoder of BlockMap not found"))
					}

					menc = i
				}

				reader, err := isaacblock.NewLocalFSReaderFromHeight(
					launch.LocalFSDataDirectory(design.Storage.Base), height, menc,
				)
				if err != nil {
					return nil, false, e.Wrap(err)
				}
				defer func() {
					_ = reader.Close()
				}()

				return reader.Reader(item)
			},
		), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixNodeChallengeString,
		isaacnetwork.QuicstreamHandlerNodeChallenge(isaacParams.NetworkID(), local), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixSuffrageNodeConnInfoString,
		isaacnetwork.QuicstreamHandlerSuffrageNodeConnInfo(suffrageNodeConnInfoF), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixSyncSourceConnInfoString,
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
		isaacnetwork.HandlerPrefixStateString,
		isaacnetwork.QuicstreamHandlerState(db.StateBytes), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixExistsInStateOperationString,
		isaacnetwork.QuicstreamHandlerExistsInStateOperation(db.ExistsInStateOperation), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixNodeInfoString,
		isaacnetwork.QuicstreamHandlerNodeInfo(launch.QuicstreamHandlerGetNodeInfoFunc(enc, nodeInfo)), nil)

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixSendBallotsString,
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

	if err := launch.AttachMemberlistNetworkHandlers(pctx); err != nil {
		return pctx, err
	}

	return pctx, nil
}

func PStatesNetworkHandlers(pctx context.Context) (context.Context, error) {
	var log *logging.Logging
	var local base.LocalNode
	var params *launch.LocalParams
	var states *isaacstates.States

	if err := util.LoadFromContext(pctx,
		launch.LoggingContextKey, &log,
		launch.LocalContextKey, &local,
		launch.LocalParamsContextKey, &params,
		launch.StatesContextKey, &states,
	); err != nil {
		return pctx, err
	}

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

	var gerror error

	launch.EnsureHandlerAdd(pctx, &gerror,
		isaacnetwork.HandlerPrefixSetAllowConsensusString,
		isaacnetwork.QuicstreamHandlerSetAllowConsensus(
			local.Publickey(),
			params.ISAAC.NetworkID(),
			states.SetAllowConsensus,
		),
		nil,
	)

	return pctx, gerror
}
