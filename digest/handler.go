package digest

import (
	"context"
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/network/quicmemberlist"
	"github.com/ProtoconNet/mitum2/network/quicstream"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
	"time"

	"github.com/ProtoconNet/mitum-currency/v3/digest/network"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"
)

var (
	HTTP2EncoderHintHeader = http.CanonicalHeaderKey("x-mitum-encoder-hint")
	HALMimetype            = "application/hal+json; charset=utf-8"
)

var (
	HandlerPathNodeInfo                   = `/`
	HandlerPathCurrencies                 = `/currency`
	HandlerPathCurrency                   = `/currency/{currency_id:` + types.ReCurrencyID + `}`
	HandlerPathManifests                  = `/block/manifests`
	HandlerPathOperations                 = `/block/operations`
	HandlerPathOperationsByHash           = `/block/operations/facts`
	HandlerPathOperation                  = `/block/operation/{hash:(?i)[0-9a-z][0-9a-z]+}`
	HandlerPathBlockByHeight              = `/block/{height:[0-9]+}`
	HandlerPathBlockByHash                = `/block/{hash:(?i)[0-9a-z][0-9a-z]+}`
	HandlerPathOperationsByHeight         = `/block/{height:[0-9]+}/operations`
	HandlerPathManifestByHeight           = `/block/{height:[0-9]+}/manifest`
	HandlerPathManifestByHash             = `/block/{hash:(?i)[0-9a-z][0-9a-z]+}/manifest`
	HandlerPathAccount                    = `/account/{address:(?i)` + types.REStringAddressString + `}`            // revive:disable-line:line-length-limit
	HandlerPathAccountOperations          = `/account/{address:(?i)` + types.REStringAddressString + `}/operations` // revive:disable-line:line-length-limit
	HandlerPathAccounts                   = `/accounts`
	HandlerPathOperationBuildFactTemplate = `/builder/operation/fact/template/{fact:[\w][\w\-]*}`
	HandlerPathOperationBuildFact         = `/builder/operation/fact`
	HandlerPathOperationBuildSign         = `/builder/operation/sign`
	HandlerPathOperationBuild             = `/builder/operation`
	HandlerPathSend                       = `/builder/send`
	HandlerPathQueueSend                  = `/builder/send/queue`
	HandelrPathEventOperation             = `/event/operation/{hash:(?i)[0-9a-z][0-9a-z]+}`
	HandelrPathEventAccount               = `/event/account/{address:(?i)` + types.REStringAddressString + `}`
	HandlerPathEventContract              = `/event/contract/{address:(?i)` + types.REStringAddressString + `}`
)

var (
	UnknownProblem     = NewProblem(DefaultProblemType, "unknown problem occurred")
	UnknownProblemJSON []byte
)

var GlobalItemsLimit int64 = 10

func init() {
	if b, err := JSON.Marshal(UnknownProblem); err != nil {
		panic(err)
	} else {
		UnknownProblemJSON = b
	}
}

type Handlers struct {
	*zerolog.Logger
	networkID       base.NetworkID
	encs            *encoder.Encoders
	enc             encoder.Encoder
	database        *Database
	cache           Cache
	queue           chan RequestWrapper
	nodeInfoHandler NodeInfoHandler
	send            func(interface{}) (base.Operation, error)
	client          func() (*launch.LocalParams, *quicmemberlist.Memberlist, []quicstream.ConnInfo, error)
	router          *mux.Router
	routes          map[ /* path */ string]*mux.Route
	itemsLimiter    func(string /* request type */) int64
	rg              *singleflight.Group
	expireNotFilled time.Duration
}

func NewHandlers(
	ctx context.Context,
	networkID base.NetworkID,
	encs *encoder.Encoders,
	enc encoder.Encoder,
	st *Database,
	cache Cache,
	router *mux.Router,
	queue chan RequestWrapper,
) *Handlers {
	var log *logging.Logging
	if err := util.LoadFromContextOK(ctx, launch.LoggingContextKey, &log); err != nil {
		return nil
	}

	return &Handlers{
		Logger:          log.Log(),
		networkID:       networkID,
		encs:            encs,
		enc:             enc,
		database:        st,
		cache:           cache,
		queue:           queue,
		router:          router,
		routes:          map[string]*mux.Route{},
		itemsLimiter:    DefaultItemsLimiter,
		rg:              &singleflight.Group{},
		expireNotFilled: time.Second * 3,
	}
}

func (hd *Handlers) Initialize() error {
	cors := handlers.CORS(
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"content-type"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	hd.router.Use(cors)

	hd.setHandlers()

	return nil
}

func (hd *Handlers) SetLimiter(f func(string) int64) *Handlers {
	hd.itemsLimiter = f

	return hd
}

func (hd *Handlers) Cache() Cache {
	return hd.cache
}

func (hd *Handlers) Router() *mux.Router {
	return hd.router
}

func (hd *Handlers) Routes() map[string]*mux.Route {
	return hd.routes
}

func (hd *Handlers) Handler() http.Handler {
	return network.HTTPLogHandler(hd.router, hd.Logger)
}

func (hd *Handlers) setHandlers() {
	post := 5
	get := 1000
	_ = hd.setHandler(HandlerPathCurrencies, hd.handleCurrencies, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathCurrency, hd.handleCurrency, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathManifests, hd.handleManifests, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperations, hd.handleOperations, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperationsByHash, hd.handleOperationsByHash, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperation, hd.handleOperation, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperationsByHeight, hd.handleOperationsByHeight, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathManifestByHeight, hd.handleManifestByHeight, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathManifestByHash, hd.handleManifestByHash, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathBlockByHeight, hd.handleBlock, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathBlockByHash, hd.handleBlock, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathAccount, hd.handleAccount, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathAccountOperations, hd.handleAccountOperations, true, get, get).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathAccounts, hd.handleAccounts, true, get, get).
		Methods(http.MethodOptions, "GET")
	// _ = hd.setHandler(HandlerPathOperationBuildFactTemplate, hd.handleOperationBuildFactTemplate, true).
	// 	Methods(http.MethodOptions, "GET")
	// _ = hd.setHandler(HandlerPathOperationBuildFact, hd.handleOperationBuildFact, false).
	// 	Methods(http.MethodOptions, http.MethodPost)
	// _ = hd.setHandler(HandlerPathOperationBuildSign, hd.handleOperationBuildSign, false).
	// 	Methods(http.MethodOptions, http.MethodPost)
	// _ = hd.setHandler(HandlerPathOperationBuild, hd.handleOperationBuild, true).
	// 	Methods(http.MethodOptions, http.MethodGet, http.MethodPost)
	_ = hd.setHandler(HandlerPathSend, hd.handleSend, false, post, post).
		Methods(http.MethodOptions, http.MethodPost)
	_ = hd.setHandler(HandlerPathQueueSend, hd.handleQueueSend, false, post, post).
		Methods(http.MethodOptions, http.MethodPost)
	_ = hd.setHandler(HandlerPathNodeInfo, hd.handleNodeInfo, true, get, get).
		Methods(http.MethodOptions, "GET")
}

func (hd *Handlers) setHandler(prefix string, h network.HTTPHandlerFunc, useCache bool, rps, burst int) *mux.Route {
	var handler http.Handler
	if !useCache {
		handler = http.HandlerFunc(h)
	} else {
		ch := NewCachedHTTPHandler(hd.cache, h)

		handler = ch
	}

	var name string
	if prefix == "" || prefix == "/" {
		name = "root"
	} else {
		name = prefix
	}

	var route *mux.Route
	if r := hd.router.Get(name); r != nil {
		route = r
	} else {
		route = hd.router.Name(name)
	}

	handler = RateLimiter(rps, burst)(handler)

	/*
		if rules, found := hd.rateLimit[prefix]; found {
			handler = process.NewRateLimitMiddleware(
				process.NewRateLimit(rules, limiter.Rate{Limit: -1}), // NOTE by default, unlimited
				hd.rateLimitStore,
			).Middleware(handler)

			hd.Log().Debug().Str("prefix", prefix).Msg("ratelimit middleware attached")
		}
	*/

	route = route.
		Path(prefix).
		Handler(handler)

	hd.routes[prefix] = route

	return route
}

func (hd *Handlers) combineURL(path string, pairs ...string) (string, error) {
	if n := len(pairs); n%2 != 0 {
		return "", errors.Errorf("Combine url; uneven pairs to combine url")
	} else if n < 1 {
		u, err := hd.routes[path].URL()
		if err != nil {
			return "", errors.Wrap(err, "combine url")
		}
		return u.String(), nil
	}

	u, err := hd.routes[path].URLPath(pairs...)
	if err != nil {
		return "", errors.Wrap(err, "combine url")
	}
	return u.String(), nil
}

func CacheKeyPath(r *http.Request) string {
	return r.URL.Path
}

func CacheKey(key string, s ...string) string {
	var l []string
	var notempty bool
	for i := len(s) - 1; i >= 0; i-- {
		a := s[i]

		if !notempty {
			if len(strings.TrimSpace(a)) < 1 {
				continue
			}
			notempty = true
		}

		l = append(l, a)
	}

	r := make([]string, len(l))
	for i := range l {
		r[len(l)-1-i] = l[i]
	}

	return fmt.Sprintf("%s-%s", key, strings.Join(r, ","))
}

func DefaultItemsLimiter(string) int64 {
	return GlobalItemsLimit
}

func RateLimiter(rps int, burst int) func(http.Handler) http.Handler {
	if rps <= 0 {
		// Rate limiting is disabled
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
