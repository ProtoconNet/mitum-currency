package types

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
)

func (ky BaseAccountKey) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":  ky.Hint().String(),
			"weight": ky.w,
			"key":    ky.k.String(),
		},
	)
}

type KeyBSONUnmarshaler struct {
	Hint   string `bson:"_hint"`
	Weight uint   `bson:"weight"`
	Keys   string `bson:"key"`
}

func (ky *BaseAccountKey) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("failed to decode bson of BaseAccountKey")

	var uk KeyBSONUnmarshaler
	if err := bson.Unmarshal(b, &uk); err != nil {
		return e.Wrap(err)
	}
	ht, err := hint.ParseHint(uk.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	return ky.unpack(enc, ht, uk.Weight, uk.Keys)
}

func (ks BaseAccountKeys) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":        ks.Hint().String(),
			"hash":         ks.Hash().String(),
			"keys":         ks.keys,
			"threshold":    ks.threshold,
			"address_type": ks.addressType,
		},
	)
}

func (ks ContractAccountKeys) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":     ks.Hint().String(),
			"hash":      ks.Hash().String(),
			"keys":      ks.keys,
			"threshold": ks.threshold,
		},
	)
}

type KeysBSONUnmarshaler struct {
	Hint        string   `bson:"_hint"`
	Hash        string   `bson:"hash"`
	Keys        bson.Raw `bson:"keys"`
	Threshold   uint     `bson:"threshold"`
	AddressType string   `bson:"address_type"`
}

func (ks *BaseAccountKeys) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("failed to decode bson of BaseAccountKeys")

	var uks KeysBSONUnmarshaler
	if err := bson.Unmarshal(b, &uks); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(uks.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	ks.BaseHinter = hint.NewBaseHinter(ht)

	hks, err := enc.DecodeSlice(uks.Keys)
	if err != nil {
		return e.Wrap(err)
	}

	keys := make([]AccountKey, len(hks))
	for i := range hks {
		j, ok := hks[i].(BaseAccountKey)
		if !ok {
			return errors.Errorf("expected BaseAccountKey, not %T", hks[i])
		}

		keys[i] = j
	}
	ks.keys = keys
	ks.threshold = uks.Threshold
	ks.addressType = hint.Type(uks.AddressType)

	if ks.addressType == AddressHint.Type() {
		ks.h = valuehash.NewBytesFromString(uks.Hash)
	} else {
		ks.h = common.NewBytesFromString(uks.Hash)
	}

	return nil
}

func (ks *ContractAccountKeys) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("failed to decode bson of ContractAccountKeys")

	var uks KeysBSONUnmarshaler
	if err := bson.Unmarshal(b, &uks); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(uks.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	ks.BaseHinter = hint.NewBaseHinter(ht)

	hks, err := enc.DecodeSlice(uks.Keys)
	if err != nil {
		return e.Wrap(err)
	}

	keys := make([]AccountKey, len(hks))
	for i := range hks {
		j, ok := hks[i].(BaseAccountKey)
		if !ok {
			return errors.Errorf("expected BaseAccountKey, not %T", hks[i])
		}

		keys[i] = j
	}
	ks.keys = keys
	ks.threshold = uks.Threshold

	ks.h = valuehash.NewBytesFromString(uks.Hash)

	return nil
}
