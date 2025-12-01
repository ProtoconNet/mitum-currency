package types

import (
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

func (fa NilFeeer) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(hint.BaseHinter{
		HT: fa.Hint(),
	})
}

func (fa *NilFeeer) UnmarsahlJSON(b []byte) error {
	e := util.StringError("unmarshal json of NilFeeer")

	var ht hint.BaseHinter
	if err := util.UnmarshalJSON(b, &ht); err != nil {
		return e.Wrap(err)
	}

	fa.BaseHinter = ht

	return nil
}

type FixedFeeerJSONMarshaler struct {
	hint.BaseHinter
	Receiver base.Address `json:"receiver"`
	Amount   string       `json:"amount"`
}

func (fa FixedFeeer) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(FixedFeeerJSONMarshaler{
		BaseHinter: fa.BaseHinter,
		Receiver:   fa.receiver,
		Amount:     fa.amount.String(),
	})
}

type FixedFeeerJSONUnmarshaler struct {
	Hint     hint.Hint `json:"_hint"`
	Receiver string    `json:"receiver"`
	Amount   string    `json:"amount"`
}

func (fa *FixedFeeer) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode json of FixedFeeer")

	var ufa FixedFeeerJSONUnmarshaler
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return e.Wrap(err)
	}

	return fa.unpack(enc, ufa.Hint, ufa.Receiver, ufa.Amount)
}

type FixedItemFeeerJSONMarshaler struct {
	hint.BaseHinter
	Receiver      base.Address `json:"receiver"`
	Amount        string       `json:"amount"`
	ItemFeeAmount string       `json:"item_fee_amount"`
}

func (fa FixedItemFeeer) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(FixedItemFeeerJSONMarshaler{
		BaseHinter:    fa.BaseHinter,
		Receiver:      fa.receiver,
		Amount:        fa.amount.String(),
		ItemFeeAmount: fa.itemFeeAmount.String(),
	})
}

type FixedItemFeeerJSONUnmarshaler struct {
	Hint          hint.Hint `json:"_hint"`
	Receiver      string    `json:"receiver"`
	Amount        string    `json:"amount"`
	ItemFeeAmount string    `json:"item_fee_amount"`
}

func (fa *FixedItemFeeer) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode json of FixedItemFeeer")

	var ufa FixedItemFeeerJSONUnmarshaler
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return e.Wrap(err)
	}

	return fa.unpack(enc, ufa.Hint, ufa.Receiver, ufa.Amount, ufa.ItemFeeAmount)
}

type RatioFeeerJSONMarshaler struct {
	hint.BaseHinter
	Receiver base.Address `json:"receiver"`
	Ratio    float64      `json:"ratio"`
	Min      string       `json:"min"`
	Max      string       `json:"max"`
}

func (fa RatioFeeer) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(RatioFeeerJSONMarshaler{
		BaseHinter: fa.BaseHinter,
		Receiver:   fa.receiver,
		Ratio:      fa.ratio,
		Min:        fa.min.String(),
		Max:        fa.max.String(),
	})
}

type RatioFeeerJSONUnmarshaler struct {
	Hint     hint.Hint `json:"_hint"`
	Receiver string    `json:"receiver"`
	Ratio    float64   `json:"ratio"`
	Min      string    `json:"min"`
	Max      string    `json:"max"`
}

func (fa *RatioFeeer) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("Decode json of RatioFeeer")

	var ufa RatioFeeerJSONUnmarshaler
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return e.Wrap(err)
	}

	return fa.unpack(enc, ufa.Hint, ufa.Receiver, ufa.Ratio, ufa.Min, ufa.Max)
}
