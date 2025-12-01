package types

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
)

func (fa NilFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.NewHintedDoc(fa.Hint()))
}

func (fa *NilFeeer) UnmarsahlBSON(b []byte) error {
	e := util.StringError("unmarshal bson of NilFeeer")

	var head bsonenc.HintedHead
	if err := bsonenc.Unmarshal(b, &head); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(head.H)
	if err != nil {
		return e.Wrap(err)
	}

	fa.BaseHinter = hint.NewBaseHinter(ht)

	return nil
}

func (fa FixedFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":    fa.Hint().String(),
			"receiver": fa.receiver,
			"amount":   fa.amount.String(),
		},
	)

}

type FixedFeeerBSONUnmarshaler struct {
	Hint     string `bson:"_hint"`
	Receiver string `bson:"receiver"`
	Amount   string `bson:"amount"`
}

func (fa *FixedFeeer) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of FixedFeeer")

	var ufa FixedFeeerBSONUnmarshaler
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(ufa.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	return fa.unpack(enc, ht, ufa.Receiver, ufa.Amount)
}

func (fa FixedItemFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":           fa.Hint().String(),
			"receiver":        fa.receiver,
			"amount":          fa.amount.String(),
			"item_fee_amount": fa.itemFeeAmount.String(),
		},
	)
}

type FixedItemFeeerBSONUnmarshaler struct {
	Hint          string `bson:"_hint"`
	Receiver      string `bson:"receiver"`
	Amount        string `bson:"amount"`
	ItemFeeAmount string `bson:"item_fee_amount"`
}

func (fa *FixedItemFeeer) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of FixedItemFeeer")

	var ufa FixedItemFeeerBSONUnmarshaler
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(ufa.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	return fa.unpack(enc, ht, ufa.Receiver, ufa.Amount, ufa.ItemFeeAmount)
}

func (fa RatioFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":    fa.Hint().String(),
			"receiver": fa.receiver,
			"ratio":    fa.ratio,
			"min":      fa.min.String(),
			"max":      fa.max.String(),
		},
	)
}

type RatioFeeerBSONUnmarshaler struct {
	Hint     string  `bson:"_hint"`
	Receiver string  `bson:"receiver"`
	Ratio    float64 `bson:"ratio"`
	Min      string  `bson:"min"`
	Max      string  `bson:"max"`
}

func (fa *RatioFeeer) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of RatioFeeer")

	var ufa RatioFeeerBSONUnmarshaler
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return e.Wrap(err)
	}
	ht, err := hint.ParseHint(ufa.Hint)
	if err != nil {
		return e.Wrap(err)
	}

	return fa.unpack(enc, ht, ufa.Receiver, ufa.Ratio, ufa.Min, ufa.Max)
}
