package digest

import (
	"fmt"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"

	"github.com/ProtoconNet/mitum2/base"
	mitumutil "github.com/ProtoconNet/mitum2/util"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

func (hd *Handlers) handleOperation(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	h, err := parseHashFromPath(mux.Vars(r)["hash"])
	if err != nil {
		HTTP2ProblemWithError(w, errors.Wrap(err, "invalid hash for operation by hash"), http.StatusBadRequest)

		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleOperationInGroup(h)
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Millisecond*500)
		}
	}
}

func (hd *Handlers) handleOperationInGroup(h mitumutil.Hash) ([]byte, error) {
	var (
		va  OperationValue
		err error
	)
	switch va, _, err = hd.database.Operation(h, true); {
	case err != nil:
		return nil, err
	//case !found:
	//return nil, mitumutil.ErrNotFound.Errorf("operation %v in handleOperation", h)
	default:
		hal, err := hd.buildOperationHal(va)
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("operation:{hash}", NewHalLink(HandlerPathOperation, nil).SetTemplated())
		hal = hal.AddLink("block:{height}", NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated())

		return hd.enc.Marshal(hal)
	}
}

func (hd *Handlers) handleOperations(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	limit := ParseLimitQuery(r.URL.Query().Get("limit"))
	offset := ParseStringQuery(r.URL.Query().Get("offset"))
	reverse := ParseBoolQuery(r.URL.Query().Get("reverse"))

	cachekey := CacheKey(r.URL.Path, StringOffsetQuery(offset), StringBoolQuery("reverse", reverse))
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		i, filled, err := hd.handleOperationsInGroup(offset, reverse, limit)

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

func (hd *Handlers) handleOperationsInGroup(offset string, reverse bool, l int64) ([]byte, bool, error) {
	filter, err := buildOperationsFilterByOffset(offset, reverse)
	if err != nil {
		return nil, false, err
	}

	var vas []Hal
	var opsCount int64
	switch l, count, e := hd.loadOperationsHALFromDatabase(filter, reverse, l); {
	case e != nil:
		return nil, false, e
	case len(l) < 1:
		return nil, false, mitumutil.ErrNotFound.Errorf("Operations in handleOperations")
	default:
		vas = l
		opsCount = count
	}

	h, err := hd.combineURL(HandlerPathOperations)
	if err != nil {
		return nil, false, err
	}
	hal := hd.buildOperationsHal(h, vas, offset, reverse)
	if next := nextOffsetOfOperations(h, vas, reverse); len(next) > 0 {
		hal = hal.AddLink("next", NewHalLink(next, nil))
	}
	hal.AddExtras("total_operations", opsCount)

	b, err := hd.enc.Marshal(hal)
	return b, int64(len(vas)) == hd.itemsLimiter("operations"), err
}

func (hd *Handlers) handleOperationsByHeight(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	limit := ParseLimitQuery(r.URL.Query().Get("limit"))
	offset := ParseStringQuery(r.URL.Query().Get("offset"))
	reverse := ParseBoolQuery(r.URL.Query().Get("reverse"))

	cachekey := CacheKey(r.URL.Path, StringOffsetQuery(offset), StringBoolQuery("reverse", reverse))
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	var height base.Height
	switch h, err := parseHeightFromPath(mux.Vars(r)["height"]); {
	case err != nil:
		HTTP2ProblemWithError(w, errors.Errorf("Invalid height found for manifest by height"), http.StatusBadRequest)

		return
	case h <= base.NilHeight:
		HTTP2ProblemWithError(w, errors.Errorf("Invalid height, %v", h), http.StatusBadRequest)
		return
	default:
		height = h
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		i, filled, err := hd.handleOperationsByHeightInGroup(height, offset, reverse, limit)
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

func (hd *Handlers) handleOperationsByHeightInGroup(
	height base.Height,
	offset string,
	reverse bool,
	l int64,
) ([]byte, bool, error) {
	filter, err := buildOperationsByHeightFilterByOffset(height, offset, reverse)
	if err != nil {
		return nil, false, err
	}

	var vas []Hal
	var opsCount int64
	switch l, count, e := hd.loadOperationsHALFromDatabase(filter, reverse, l); {
	case e != nil:
		return nil, false, e
	case len(l) < 1:
		return nil, false, mitumutil.ErrNotFound.Errorf("Operations in handleOperationsByHeight")
	default:
		vas = l
		opsCount = count
	}

	h, err := hd.combineURL(HandlerPathOperationsByHeight, "height", height.String())
	if err != nil {
		return nil, false, err
	}
	hal := hd.buildOperationsHal(h, vas, offset, reverse)
	if next := nextOffsetOfOperationsByHeight(h, vas, reverse); len(next) > 0 {
		hal = hal.AddLink("next", NewHalLink(next, nil))
	}
	hal.AddExtras("total_operations", opsCount)

	b, err := hd.enc.Marshal(hal)
	return b, int64(len(vas)) == hd.itemsLimiter("operations"), err
}

func (hd *Handlers) handleOperationsByHash(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	hashes := ParseStringQuery(r.URL.Query().Get("hashes"))

	cacheKey := CacheKey(r.URL.Path, stringHashesQuery(hashes))
	if err := LoadFromCache(hd.cache, cacheKey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cacheKey, func() (interface{}, error) {
		i, err := hd.handleOperationsByHashInGroup(hashes)
		return i, err
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		b := v.([]byte)

		HTTP2WriteHalBytes(hd.enc, w, b, http.StatusOK)

		if !shared {
			expire := hd.expireNotFilled
			HTTP2WriteCache(w, cacheKey, expire)
		}
	}
}

func (hd *Handlers) handleOperationsByHashInGroup(
	hashes string,
) ([]byte, error) {
	filter, err := buildOperationsByHashesFilter(hashes)
	if err != nil {
		return nil, err
	}

	var vas []Hal
	var opsCount int64
	switch l, count, e := hd.loadOperationsHALFromDatabaseByHash(filter); {
	case e != nil:
		return nil, e
	case len(l) < 1:
		return nil, mitumutil.ErrNotFound.Errorf("Operations in handleOperationsByHash")
	default:
		vas = l
		opsCount = count
	}

	hal := hd.buildOperationsByHashHal(vas)
	hal.AddExtras("total_operations", opsCount)

	b, err := hd.enc.Marshal(hal)
	return b, err
}

func (hd *Handlers) buildOperationHal(va OperationValue) (Hal, error) {
	var hal Hal
	var h string
	var err error

	if va.IsZeroValue() {
		hal = NewEmptyHal()
	} else {
		h, err = hd.combineURL(HandlerPathOperation, "hash", va.Operation().Fact().Hash().String())
		if err != nil {
			return nil, err
		}

		hal = NewBaseHal(va, NewHalLink(h, nil))

		h, err = hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("block", NewHalLink(h, nil))
	}

	// h, err = hd.combineURL(HandlerPathManifestByHeight, "height", va.Height().String())
	// if err != nil {
	// 	return nil, err
	// }

	// hal = hal.AddLink("manifest", NewHalLink(h, nil))

	if va.InState() {
		if t, ok := va.Operation().(currency.CreateAccount); ok {
			items := t.Fact().(currency.CreateAccountFact).Items()
			for i := range items {
				a, err := items[i].Address()
				if err != nil {
					return nil, err
				}
				address := a.String()

				h, err := hd.combineURL(HandlerPathAccount, "address", address)
				if err != nil {
					return nil, err
				}
				keyHash := items[i].Keys().Hash().String()
				hal = hal.AddLink(
					fmt.Sprintf("new_account:%s", keyHash),
					NewHalLink(h, nil).
						SetProperty("key", keyHash).
						SetProperty("address", address),
				)
			}
		}
	}

	return hal, nil
}

func (*Handlers) buildOperationsHal(baseSelf string, vas []Hal, offset string, reverse bool) Hal {
	var hal Hal

	self := baseSelf
	if len(offset) > 0 {
		self = AddQueryValue(baseSelf, StringOffsetQuery(offset))
	}
	if reverse {
		self = AddQueryValue(self, StringBoolQuery("reverse", reverse))
	}
	hal = NewBaseHal(vas, NewHalLink(self, nil))

	hal = hal.AddLink("reverse", NewHalLink(AddQueryValue(baseSelf, StringBoolQuery("reverse", !reverse)), nil))

	return hal
}

func (*Handlers) buildOperationsByHashHal(vas []Hal) Hal {
	var hal Hal
	hal = NewBaseHal(vas, NewHalLink("", nil))

	return hal
}

func buildOperationsFilterByOffset(offset string, reverse bool) (bson.M, error) {
	filter := bson.M{}
	if len(offset) > 0 {
		height, index, err := parseOffset(offset)
		if err != nil {
			return nil, err
		}

		if reverse {
			filter["$or"] = []bson.M{
				{"height": bson.M{"$lt": height}},
				{"$and": []bson.M{
					{"height": height},
					{"index": bson.M{"$lt": index}},
				}},
			}
		} else {
			filter["$or"] = []bson.M{
				{"height": bson.M{"$gt": height}},
				{"$and": []bson.M{
					{"height": height},
					{"index": bson.M{"$gt": index}},
				}},
			}
		}
	}

	return filter, nil
}

func buildOperationsByHeightFilterByOffset(height base.Height, offset string, reverse bool) (bson.M, error) {
	var filter bson.M
	if len(offset) < 1 {
		return bson.M{"height": height}, nil
	}

	index, err := strconv.ParseUint(offset, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid index of offset")
	}

	if reverse {
		filter = bson.M{
			"height": height,
			"index":  bson.M{"$lt": index},
		}
	} else {
		filter = bson.M{
			"height": height,
			"index":  bson.M{"$gt": index},
		}
	}

	return filter, nil
}

const maxHashCount = 40

func buildOperationsByHashesFilter(hashes string) (bson.M, error) {
	var filter bson.M
	if len(hashes) < 1 {
		return nil, errors.Errorf("empty hashes")
	}

	hashStrArr := strings.Split(hashes, ",")
	if len(hashStrArr) > maxHashCount {
		return nil, errors.Errorf("total hash count, %v is over max hash count, %v", len(hashStrArr), maxHashCount)
	}

	var hashArr []mitumutil.Hash
	for i := range hashStrArr {
		h := valuehash.NewBytesFromString(hashStrArr[i])

		err := h.IsValid(nil)
		if err != nil {
			return nil, err
		}
		hashArr = append(hashArr, h)
	}

	filter = bson.M{
		"fact": bson.M{
			"$in": hashArr,
		},
	}

	return filter, nil
}

func nextOffsetOfOperations(baseSelf string, vas []Hal, reverse bool) string {
	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(OperationValue)
		nextoffset = buildOffset(va.Height(), va.Index())
	}

	if len(nextoffset) < 1 {
		return ""
	}

	next := baseSelf
	if len(nextoffset) > 0 {
		next = AddQueryValue(next, StringOffsetQuery(nextoffset))
	}

	if reverse {
		next = AddQueryValue(next, StringBoolQuery("reverse", reverse))
	}

	return next
}

func nextOffsetOfOperationsByHeight(baseSelf string, vas []Hal, reverse bool) string {
	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(OperationValue)
		nextoffset = fmt.Sprintf("%d", va.Index())
	}

	if len(nextoffset) < 1 {
		return ""
	}

	next := baseSelf
	if len(nextoffset) > 0 {
		next = AddQueryValue(next, StringOffsetQuery(nextoffset))
	}

	if reverse {
		next = AddQueryValue(next, StringBoolQuery("reverse", reverse))
	}

	return next
}

func (hd *Handlers) loadOperationsHALFromDatabase(filter bson.M, reverse bool, l int64) ([]Hal, int64, error) {
	var limit int64
	if l < 0 {
		limit = hd.itemsLimiter("operations")
	} else {
		limit = l
	}

	var vas []Hal
	var opsCount int64
	if err := hd.database.Operations(
		filter, true, reverse, limit,
		func(_ mitumutil.Hash, va OperationValue, count int64) (bool, error) {
			hal, err := hd.buildOperationHal(va)
			if err != nil {
				return false, err
			}
			vas = append(vas, hal)
			opsCount = count
			return true, nil
		},
	); err != nil {
		return nil, opsCount, err
	} else if len(vas) < 1 {
		return nil, opsCount, nil
	}

	return vas, opsCount, nil
}

func (hd *Handlers) loadOperationsHALFromDatabaseByHash(filter bson.M) ([]Hal, int64, error) {
	var vas []Hal
	var opsCount int64
	if err := hd.database.OperationsByHash(
		filter,
		func(_ mitumutil.Hash, va OperationValue, count int64) (bool, error) {
			hal, err := hd.buildOperationHal(va)
			if err != nil {
				return false, err
			}
			vas = append(vas, hal)
			opsCount = count
			return true, nil
		},
	); err != nil {
		return nil, opsCount, err
	} else if len(vas) < 1 {
		return nil, opsCount, nil
	}

	return vas, opsCount, nil
}
