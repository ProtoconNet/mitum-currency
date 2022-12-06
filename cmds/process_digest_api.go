package cmds

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	isaacnetwork "github.com/spikeekips/mitum/isaac/network"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/network/quicmemberlist"
	"github.com/spikeekips/mitum/network/quicstream"
	"github.com/spikeekips/mitum/util"
	mitumutil "github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

const (
	ProcessNameDigestAPI      = "digest_api"
	ProcessNameStartDigestAPI = "start_digest_api"
	HookNameSetLocalChannel   = "set_local_channel"
)

func ProcessStartDigestAPI(ctx context.Context) (context.Context, error) {
	var nt *digest.HTTP2Server
	if err := mitumutil.LoadFromContext(ctx, ContextValueDigestNetwork, &nt); err != nil {
		return ctx, err
	}
	if nt == nil {
		return ctx, nil
	}

	return ctx, nt.Start()
}

func ProcessDigestAPI(ctx context.Context) (context.Context, error) {
	var design DigestDesign
	if err := mitumutil.LoadFromContext(ctx, ContextValueDigestDesign, &design); err != nil {
		return ctx, err
	}

	var log *logging.Logging
	if err := mitumutil.LoadFromContextOK(ctx, launch.LoggingContextKey, &log); err != nil {
		return ctx, err
	}

	if design.Network() == nil {
		log.Log().Debug().Msg("digest api disabled; empty network")

		return ctx, nil
	}

	var st *digest.Database
	if err := mitumutil.LoadFromContextOK(ctx, ContextValueDigestDatabase, &st); err != nil {
		log.Log().Debug().Err(err).Msg("digest api disabled; empty database")

		return ctx, nil
	} else if st == nil {
		log.Log().Debug().Msg("digest api disabled; empty database")

		return ctx, nil
	}

	log.Log().Info().
		Str("bind", design.Network().Bind().String()).
		Str("publish", design.Network().ConnInfo().String()).
		Msg("trying to start http2 server for digest API")

	var nt *digest.HTTP2Server
	var certs []tls.Certificate
	if design.Network().Bind().Scheme == "https" {
		certs = design.Network().Certs()
	}

	if sv, err := digest.NewHTTP2Server(
		design.Network().Bind().Host,
		design.Network().ConnInfo().URL().Host,
		certs,
	); err != nil {
		return ctx, err
	} else if err := sv.Initialize(); err != nil {
		return ctx, err
	} else {
		nt = sv
	}

	return context.WithValue(ctx, ContextValueDigestNetwork, nt), nil
}

// func NewSendHandler(
// 	priv base.Privatekey,
// 	networkID base.NetworkID,
// 	f func() (*isaacnetwork.QuicstreamClient, *quicmemberlist.Memberlist, error),
// ) func(interface{}) (base.Operation, error) {
// 	return func(v interface{}) (base.Operation, error) {
// 		fmt.Println(time.Now(), " >>>> digest NewSendHandler 1")
// 		op, ok := v.(base.Operation)
// 		if !ok {
// 			return nil, util.ErrWrongType.Errorf("expected Operation, not %T", v)
// 		}
// 		fmt.Println(time.Now(), " >>>> digest NewSendHandler 2", op.Hash())
// 		buf := bytes.NewBuffer(nil)
// 		if err := json.NewEncoder(buf).Encode(op); err != nil {
// 			return nil, err
// 		} else if buf == nil {
// 			return nil, errors.Errorf("buffer from json encoding operation is nil")
// 		}

// 		var header = isaacnetwork.NewSendOperationRequestHeader()

// 		client, memberlist, err := f()

// 		errchan := make(chan error, memberlist.MembersLen())
// 		switch {
// 		case err != nil:
// 			return nil, err

// 			// ci, ok := connInfo.(quicstream.UDPConnInfo)
// 			// if !ok {
// 			// 	return nil, util.ErrWrongType.Errorf("expected quicstream.UDPConnInfo, not %T", v)
// 			// }

// 		default:
// 			// memberlist.Broadcast(quicmemberlist.NewBroadcast(i, id, notifych)
// 			// memberlist.Members(func(node quicmemberlist.Node) bool {
// 			// 	client.SendOperation()
// 			// 	ci = node.UDPConnInfo()
// 			// 	return true
// 			// })

// 			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
// 			defer cancel()

// 			worker := util.NewErrgroupWorker(ctx, int64(memberlist.MembersLen()))
// 			defer worker.Close()
// 			go func() {
// 				defer worker.Done()

// 				memberlist.Members(func(node quicmemberlist.Node) bool {
// 					ci := node.UDPConnInfo()
// 					return worker.NewJob(func(ctx context.Context, _ uint64) error {
// 						cctx, cancel := context.WithTimeout(ctx, time.Second*2) //nolint:gomnd //...
// 						defer cancel()
// 						if err := util.Retry(ctx, func() (bool, error) {
// 							fmt.Println(time.Now(), " >>>> digest NewSendHandler request", ci)
// 							response, _, cancelrequest, err := client.Request(cctx, ci, header, buf)
// 							switch {
// 							case err != nil:
// 								fmt.Println(time.Now(), " >>>> digest NewSendHandler 3 Client Request response", ci, err)
// 								return false, err
// 							case response.Err() != nil:
// 								fmt.Println(time.Now(), " >>>> digest NewSendHandler 7", ci, response.Err())
// 								return true, response.Err()
// 							}

// 							defer func() {
// 								_ = cancelrequest()
// 							}()

// 							return false, nil
// 						}, 1, time.Second*1); err != nil {
// 							fmt.Println(time.Now(), " >>>> digest NewSendHandler 8 errchan used")
// 							errchan <- err
// 						}

// 						return nil
// 					}) == nil
// 				})
// 			}()

// 			worker.Wait()
// 			close(errchan)

// 		}
// 		fmt.Println(time.Now(), " >>>> digest NewSendHandler 9")
// 		var success bool
// 		var failed error
// 		for err := range errchan {
// 			if !success && err == nil {
// 				success = true
// 			} else {
// 				fmt.Println(time.Now(), " >>>> digest NewSendHandler 10", err)
// 				failed = err
// 			}
// 		}

// 		if success {
// 			fmt.Println(time.Now(), " >>>> digest NewSendHandler 11")
// 			return op, nil
// 		}
// 		fmt.Println(time.Now(), " >>>> digest NewSendHandler 12")
// 		return op, failed
// 	}
// }

func NewSendHandler(
	priv base.Privatekey,
	networkID base.NetworkID,
	f func() ([]*isaacnetwork.QuicstreamClient, *quicmemberlist.Memberlist, error),
) func(interface{}) (base.Operation, error) {
	return func(v interface{}) (base.Operation, error) {
		op, ok := v.(base.Operation)
		if !ok {
			return nil, util.ErrWrongType.Errorf("expected Operation, not %T", v)
		}

		var header = isaacnetwork.NewSendOperationRequestHeader()

		clientpool, memberlist, err := f()

		switch {
		case err != nil:
			return nil, err

		default:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			var nodelist []quicstream.UDPConnInfo
			memberlist.Members(func(node quicmemberlist.Node) bool {
				nodelist = append(nodelist, node.UDPConnInfo())
				return true
			})

			for i := range nodelist {
				buf := bytes.NewBuffer(nil)
				if err := json.NewEncoder(buf).Encode(op); err != nil {
					return nil, err
				} else if buf == nil {
					return nil, errors.Errorf("buffer from json encoding operation is nil")
				}

				response, _, cancelrequest, err := clientpool[i].Request(ctx, nodelist[i], header, buf)
				if err != nil {
					return op, err
				}
				if response.Err() != nil {
					return op, response.Err()
				}
				defer func() {
					_ = cancelrequest()
				}()
			}
		}

		return op, nil
	}
}

/*
func SignSeal(sl seal.Seal, priv base.Privatekey, networkID base.NetworkID) (seal.Seal, error) {
	p := reflect.New(reflect.TypeOf(sl))
	p.Elem().Set(reflect.ValueOf(sl))

	signer := p.Interface().(seal.Signer)

	if err := signer.Sign(priv, networkID); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(seal.Seal), nil
}

func HookSetLocalChannel(ctx context.Context) (context.Context, error) {
	var conf config.LocalNetwork
	if err := mitumutil.LoadFromContext(ctx, ContextValueLocalNetwork, &conf); err != nil {
		return ctx, err
	}

	var local base.LocalNode
	if err := mitumutil.LoadFromContext(ctx, launch.LocalContextKey, &local); err != nil {
		return nil, err
	}

		var nodepool *network.Nodepool
		if err := process.LoadNodepoolContextValue(ctx, &nodepool); err != nil {
			return nil, err
		}

		ch, err := process.LoadNodeChannel(conf.ConnInfo(), encs, time.Second*30)
		if err != nil {
			return ctx, err
		}

		if err := nodepool.SetChannel(local.Address(), ch); err != nil {
			return ctx, err
		}

	return ctx, nil
}

func makeSendingSeal(priv base.Privatekey, networkID base.NetworkID, v interface{}) (seal.Seal, error) {
	switch t := v.(type) {

		case operation.Seal, seal.Seal:
			s, err := SignSeal(v.(seal.Seal), priv, networkID)
			if err != nil {
				return nil, err
			}

			if err := s.IsValid(networkID); err != nil {
				return nil, err
			}

			return s, nil

	case base.Operation:
		bs, err := operation.NewBaseSeal(priv, []base.Operation{t}, networkID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create operation.Seal")
		}

		if err := bs.IsValid(networkID); err != nil {
			return nil, err
		}

		return bs, nil
	default:
		return nil, errors.Errorf("unsupported message type, %T", t)
	}
}
*/
