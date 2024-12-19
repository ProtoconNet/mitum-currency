package digest

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

func (hd *Handlers) handleAccount(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	var address base.Address
	if a, err := base.DecodeAddress(strings.TrimSpace(mux.Vars(r)["address"]), hd.enc); err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)

		return
	} else if err := a.IsValid(nil); err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)
		return
	} else {
		address = a
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleAccountInGroup(address)
	}); err != nil {
		//if errors.Is(err, mongo.ErrNoDocuments) {
		//	err = util.ErrNotFound.Errorf("account, %v in handleAccount", address.String())
		//} else {
		//	hd.Log().Err(err).Str("address", address.String()).Msg("get account")
		//}

		hd.Log().Err(err).Str("address", address.String()).Msg("get account")

		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Millisecond*100)
		}
	}
}

func (hd *Handlers) handleAccountInGroup(address base.Address) (interface{}, error) {
	switch va, _, err := hd.database.Account(address); {
	case err != nil:
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
		hal, err := hd.buildAccountHal(va)
		if err != nil {
			return nil, err
		}
		return hd.enc.Marshal(hal)
	//case !found:
	//return nil, util.ErrNotFound
	default:
		hal, err := hd.buildAccountHal(va)
		if err != nil {
			return nil, err
		}
		return hd.enc.Marshal(hal)
	}
}

func (hd *Handlers) buildAccountHal(va AccountValue) (Hal, error) {
	var hal Hal

	if va.IsZeroValue() {
		hal = NewEmptyHal()
		hal = hal.
			AddLink("operationsByAccount:{address,offset}", NewHalLink("/account/{address}/operations"+"?offset={offset}", nil).SetTemplated()).
			AddLink("operationsByAccount:{address,offset,reverse}", NewHalLink("/account/{address}/operations"+"?offset={offset}&reverse=1", nil).SetTemplated())
		return hal, nil
	}

	hinted := va.Account().Address().String()
	h, err := hd.combineURL(HandlerPathAccount, "address", hinted)
	if err != nil {
		return nil, err
	}

	hal = NewBaseHal(va, NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathAccountOperations, "address", hinted)
	if err != nil {
		return nil, err
	}
	hal = hal.
		AddLink("operations", NewHalLink(h, nil)).
		AddLink("operations:{offset}", NewHalLink(h+"?offset={offset}", nil).SetTemplated()).
		AddLink("operations:{offset,reverse}", NewHalLink(h+"?offset={offset}&reverse=1", nil).SetTemplated())

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	return hal, nil
}

func (hd *Handlers) handleAccountOperations(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var address base.Address
	if a, err := base.DecodeAddress(strings.TrimSpace(mux.Vars(r)["address"]), hd.enc); err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)

		return
	} else if err := a.IsValid(nil); err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)
		return
	} else {
		address = a
	}

	limit := ParseLimitQuery(r.URL.Query().Get("limit"))
	offset := ParseStringQuery(r.URL.Query().Get("offset"))
	reverse := ParseBoolQuery(r.URL.Query().Get("reverse"))

	cachekey := CacheKey(r.URL.Path, StringOffsetQuery(offset), StringBoolQuery("reverse", reverse))
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		i, filled, err := hd.handleAccountOperationsInGroup(address, offset, reverse, limit)

		return []interface{}{i, filled}, err
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		var b []byte
		var filled bool
		{
			l := v.([]interface{})
			b = l[0].([]byte)
			filled = l[1].(bool)
		}

		HTTP2WriteHalBytes(hd.enc, w, b, http.StatusOK)

		if !shared {
			expire := hd.expireNotFilled
			if len(offset) > 0 && filled {
				expire = time.Hour * 30
			}

			HTTP2WriteCache(w, cachekey, expire)
		}
	}
}

func (hd *Handlers) handleAccountOperationsInGroup(
	address base.Address,
	offset string,
	reverse bool,
	l int64,
) ([]byte, bool, error) {
	var limit int64
	if l < 0 {
		limit = hd.itemsLimiter("account-operations")
	} else {
		limit = l
	}

	var vas []Hal
	if err := hd.database.OperationsByAddress(
		address, true, reverse, offset, limit,
		func(_ util.Hash, va OperationValue) (bool, error) {
			hal, err := hd.buildOperationHal(va)
			if err != nil {
				return false, err
			}
			vas = append(vas, hal)

			return true, nil
		},
	); err != nil {
		return nil, false, err
	}
	//} else if len(vas) < 1 {
	//	return nil, false, util.ErrNotFound.Errorf("operations in handleAccountsOperations")
	//}

	i, err := hd.buildAccountOperationsHal(address, vas, offset, reverse)
	if err != nil {
		return nil, false, err
	}

	b, err := hd.enc.Marshal(i)
	return b, int64(len(vas)) == limit, err
}

func (hd *Handlers) buildAccountOperationsHal(
	address base.Address,
	vas []Hal,
	offset string,
	reverse bool,
) (Hal, error) {
	var hal Hal

	if len(vas) < 1 {
		hal = NewEmptyHal()
		return hal, nil
	}

	baseSelf, err := hd.combineURL(HandlerPathAccountOperations, "address", address.String())
	if err != nil {
		return nil, err
	}

	self := baseSelf
	if len(offset) > 0 {
		self = AddQueryValue(baseSelf, StringOffsetQuery(offset))
	}
	if reverse {
		self = AddQueryValue(baseSelf, StringBoolQuery("reverse", reverse))
	}

	hal = NewBaseHal(vas, NewHalLink(self, nil))

	h, err := hd.combineURL(HandlerPathAccount, "address", address.String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("account", NewHalLink(h, nil))

	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(OperationValue)
		nextoffset = buildOffset(va.Height(), va.Index())
	}

	if len(nextoffset) > 0 {
		next := baseSelf
		if len(nextoffset) > 0 {
			next = AddQueryValue(next, StringOffsetQuery(nextoffset))
		}

		if reverse {
			next = AddQueryValue(next, StringBoolQuery("reverse", reverse))
		}

		hal = hal.AddLink("next", NewHalLink(next, nil))
	}

	hal = hal.AddLink("reverse", NewHalLink(AddQueryValue(baseSelf, StringBoolQuery("reverse", !reverse)), nil))

	return hal, nil
}

func (hd *Handlers) handleAccounts(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	offset := ParseStringQuery(r.URL.Query().Get("offset"))

	var pub base.Publickey
	offsetHeight := base.NilHeight
	var offsetAddress string
	switch i, h, a, err := hd.parseAccountsQueries(r.URL.Query().Get("publickey"), offset); {
	case err != nil:
		HTTP2ProblemWithError(w, fmt.Errorf("Invalue accounts query: %v", err), http.StatusBadRequest)

		return
	default:
		pub = i
		offsetHeight = h
		offsetAddress = a
	}

	cachekey := CacheKey(r.URL.Path, pub.String(), offset)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	var lastaddress base.Address
	i, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		switch h, items, a, err := hd.accountsByPublickey(pub, offsetAddress); {
		case err != nil:
			return nil, err
		case h == base.NilHeight:
			return nil, nil
		default:
			if offsetHeight <= base.NilHeight {
				offsetHeight = h
			} else if offsetHeight > h {
				offsetHeight = h
			}

			lastaddress = a

			return items, nil
		}
	})
	if err != nil {
		hd.Log().Err(err).Stringer("publickey", pub).Msg("get accounts")

		HTTP2HandleError(w, err)

		return
	}

	var items []Hal
	if i != nil {
		items = i.([]Hal)
	}

	switch hal, err := hd.buildAccountsHal(url.Values{
		"publickey": []string{pub.String()},
	}, items, offset, offsetHeight, lastaddress); {
	case err != nil:
		HTTP2HandleError(w, err)

		return
	default:
		b, err := hd.enc.Marshal(hal)
		if err != nil {
			HTTP2HandleError(w, err)

			return
		}
		HTTP2WriteHalBytes(hd.enc, w, b, http.StatusOK)
	}

	if !shared {
		expire := hd.expireNotFilled
		if offsetHeight > base.NilHeight && len(offsetAddress) > 0 {
			expire = time.Minute
		}

		HTTP2WriteCache(w, cachekey, expire)
	}
}

func (*Handlers) buildAccountsHal(
	queries url.Values,
	vas []Hal,
	offset string,
	topHeight base.Height,
	lastaddress base.Address,
) (Hal, error) { // nolint:unparam
	baseSelf := HandlerPathAccounts
	if len(queries) > 0 {
		baseSelf += "?" + queries.Encode()
	}

	self := baseSelf
	if len(offset) > 0 {
		self = AddQueryValue(baseSelf, StringOffsetQuery(offset))
	}

	var hal Hal
	hal = NewBaseHal(vas, NewHalLink(self, nil))

	var nextoffset string
	if len(vas) > 0 {
		nextoffset = buildOffsetByString(topHeight, lastaddress.String())
	}

	if len(nextoffset) > 0 {
		next := baseSelf
		if len(nextoffset) > 0 {
			next = AddQueryValue(next, StringOffsetQuery(nextoffset))
		}

		hal = hal.AddLink("next", NewHalLink(next, nil))
	}

	return hal, nil
}

func (hd *Handlers) parseAccountsQueries(s, offset string) (base.Publickey, base.Height, string, error) {
	var pub base.Publickey
	switch ps := strings.TrimSpace(s); {
	case len(ps) < 1:
		return nil, base.NilHeight, "", errors.Errorf("Empty query")
	default:
		i, err := base.DecodePublickeyFromString(ps, hd.enc)
		if err == nil {
			err = i.IsValid(nil)
		}

		if err != nil {
			return nil, base.NilHeight, "", err
		}

		pub = i
	}

	offset = strings.TrimSpace(offset)
	if len(offset) < 1 {
		return pub, base.NilHeight, "", nil
	}

	switch h, a, err := parseOffsetByString(offset); {
	case err != nil:
		return nil, base.NilHeight, "", err
	case len(a) < 1:
		return nil, base.NilHeight, "", errors.Errorf("Empty address in offset of accounts")
	default:
		return pub, h, a, nil
	}
}

func (hd *Handlers) accountsByPublickey(
	pub base.Publickey,
	offsetAddress string,
) (base.Height, []Hal, base.Address, error) {
	offsetHeight := base.NilHeight
	var lastaddress base.Address

	switch h, err := hd.database.topHeightByPublickey(pub); {
	case err != nil:
		return offsetHeight, nil, nil, err
	case h == base.NilHeight:
		return offsetHeight, nil, nil, nil
	default:
		if offsetHeight <= base.NilHeight {
			offsetHeight = h
		} else if offsetHeight > h {
			offsetHeight = h
		}
	}

	var items []Hal
	if err := hd.database.AccountsByPublickey(pub, false, offsetHeight, offsetAddress, hd.itemsLimiter("accounts"),
		func(va AccountValue) (bool, error) {
			hal, err := hd.buildAccountHal(va)
			if err != nil {
				return false, err
			}
			items = append(items, hal)
			lastaddress = va.Account().Address()

			return true, nil
		}); err != nil {
		return offsetHeight, nil, nil, err
	}

	return offsetHeight, items, lastaddress, nil
}
