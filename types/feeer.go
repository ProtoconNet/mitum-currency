package types

import (
	"bytes"
	"encoding/binary"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
)

const (
	FeeerNil       = "nil"
	FeeerFixed     = "fixed"
	ItemFeeerFixed = "fixed-item"
	FeeerRatio     = "ratio"
)

var (
	NilFeeerHint       = hint.MustNewHint("mitum-currency-nil-feeer-v0.0.1")
	FixedFeeerHint     = hint.MustNewHint("mitum-currency-fixed-feeer-v0.0.1")
	FixedItemFeeerHint = hint.MustNewHint("mitum-currency-fixed-item-feeer-v0.0.1")
	RatioFeeerHint     = hint.MustNewHint("mitum-currency-ratio-feeer-v0.0.1")
)

var UnlimitedMaxFeeAmount = common.NewBig(-1)

type Feeer interface {
	util.IsValider
	hint.Hinter
	Type() string
	Bytes() []byte
	Receiver() base.Address
	Min() common.Big
	Fee(common.Big) (common.Big, error)
}

type ItemFeeer interface {
	Feeer
	ItemFee(common.Big) (common.Big, error)
}

type NilFeeer struct {
	hint.BaseHinter
}

func NewNilFeeer() NilFeeer {
	return NilFeeer{BaseHinter: hint.NewBaseHinter(NilFeeerHint)}
}

func (NilFeeer) Type() string {
	return FeeerNil
}

func (NilFeeer) Bytes() []byte {
	return nil
}

func (NilFeeer) Receiver() base.Address {
	return nil
}

func (NilFeeer) Min() common.Big {
	return common.ZeroBig
}

func (NilFeeer) Fee(common.Big) (common.Big, error) {
	return common.ZeroBig, nil
}

func (fa NilFeeer) IsValid([]byte) error {
	return fa.BaseHinter.IsValid(nil)
}

type FixedFeeer struct {
	hint.BaseHinter
	receiver base.Address
	amount   common.Big
}

func NewFixedFeeer(receiver base.Address, amount common.Big) FixedFeeer {
	return FixedFeeer{
		BaseHinter: hint.NewBaseHinter(FixedFeeerHint),
		receiver:   receiver,
		amount:     amount,
	}
}

func (FixedFeeer) Type() string {
	return FeeerFixed
}

func (fa FixedFeeer) Bytes() []byte {
	return util.ConcatBytesSlice(fa.receiver.Bytes(), fa.amount.Bytes())
}

func (fa FixedFeeer) Receiver() base.Address {
	return fa.receiver
}

func (fa FixedFeeer) Min() common.Big {
	return fa.amount
}

func (fa FixedFeeer) Fee(common.Big) (common.Big, error) {
	if fa.isZero() {
		return common.ZeroBig, nil
	}

	return fa.amount, nil
}

func (fa FixedFeeer) IsValid([]byte) error {
	if err := fa.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if err := util.CheckIsValiders(nil, false, fa.receiver); err != nil {
		return util.ErrInvalid.Errorf("invalid receiver for fixed feeer: %v", err)
	}

	if !fa.amount.OverNil() {
		return util.ErrInvalid.Errorf("fixed feeer amount under zero")
	}

	return nil
}

func (fa FixedFeeer) isZero() bool {
	return fa.amount.IsZero()
}

type FixedItemFeeer struct {
	hint.BaseHinter
	receiver      base.Address
	amount        common.Big
	itemFeeAmount common.Big
}

func NewFixedItemFeeer(receiver base.Address, amount, itemFeeAmount common.Big) FixedItemFeeer {
	return FixedItemFeeer{
		BaseHinter:    hint.NewBaseHinter(FixedItemFeeerHint),
		receiver:      receiver,
		amount:        amount,
		itemFeeAmount: itemFeeAmount,
	}
}

func (FixedItemFeeer) Type() string {
	return ItemFeeerFixed
}

func (fa FixedItemFeeer) Bytes() []byte {
	return util.ConcatBytesSlice(fa.receiver.Bytes(), fa.amount.Bytes())
}

func (fa FixedItemFeeer) Receiver() base.Address {
	return fa.receiver
}

func (fa FixedItemFeeer) Min() common.Big {
	return fa.amount
}

func (fa FixedItemFeeer) Fee(common.Big) (common.Big, error) {
	if fa.isZero(fa.amount) {
		return common.ZeroBig, nil
	}

	return fa.amount, nil
}

func (fa FixedItemFeeer) ItemFee(common.Big) (common.Big, error) {
	if fa.isZero(fa.itemFeeAmount) {
		return common.ZeroBig, nil
	}

	return fa.itemFeeAmount, nil
}

func (fa FixedItemFeeer) IsValid([]byte) error {
	if err := fa.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if err := util.CheckIsValiders(nil, false, fa.receiver); err != nil {
		return util.ErrInvalid.Errorf("invalid receiver for fixed item feeer: %v", err)
	}

	if !fa.amount.OverNil() {
		return util.ErrInvalid.Errorf("fixed item feeer amount under zero")
	}

	return nil
}

func (fa FixedItemFeeer) isZero(am common.Big) bool {
	return am.IsZero()
}

type RatioFeeer struct {
	hint.BaseHinter
	receiver base.Address
	ratio    float64 // 0 >=, or <= 1.0
	min      common.Big
	max      common.Big
}

func NewRatioFeeer(receiver base.Address, ratio float64, min, max common.Big) RatioFeeer {
	return RatioFeeer{
		BaseHinter: hint.NewBaseHinter(RatioFeeerHint),
		receiver:   receiver,
		ratio:      ratio,
		min:        min,
		max:        max,
	}
}

func (RatioFeeer) Type() string {
	return FeeerRatio
}

func (fa RatioFeeer) Bytes() []byte {
	var rb bytes.Buffer
	_ = binary.Write(&rb, binary.BigEndian, fa.ratio)

	return util.ConcatBytesSlice(fa.receiver.Bytes(), rb.Bytes(), fa.min.Bytes(), fa.max.Bytes())
}

func (fa RatioFeeer) Receiver() base.Address {
	return fa.receiver
}

func (fa RatioFeeer) Min() common.Big {
	return fa.min
}

func (fa RatioFeeer) Fee(a common.Big) (common.Big, error) {
	if fa.isZero() {
		return common.ZeroBig, nil
	} else if a.IsZero() {
		return fa.min, nil
	}

	if fa.isOne() {
		return a, nil
	} else if f := a.MulFloat64(fa.ratio); f.Compare(fa.min) < 0 {
		return fa.min, nil
	} else {
		if !fa.isUnlimited() && f.Compare(fa.max) > 0 {
			return fa.max, nil
		}
		return f, nil
	}
}

func (fa RatioFeeer) IsValid([]byte) error {
	if err := fa.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if err := util.CheckIsValiders(nil, false, fa.receiver); err != nil {
		return util.ErrInvalid.Errorf("Invalid receiver for ratio feeer: %v", err)
	}

	if fa.ratio < 0 || fa.ratio > 1 {
		return util.ErrInvalid.Errorf("Invalid ratio, %v; it should be 0 >=, <= 1", fa.ratio)
	}

	if !fa.min.OverNil() {
		return util.ErrInvalid.Errorf("Ratio feeer min amount under zero")
	} else if !fa.max.Equal(UnlimitedMaxFeeAmount) {
		if !fa.max.OverNil() {
			return util.ErrInvalid.Errorf("Ratio feeer max amount under zero")
		} else if fa.min.Compare(fa.max) > 0 {
			return util.ErrInvalid.Errorf("ratio feeer min should over max")
		}
	}

	return nil
}

func (fa RatioFeeer) isUnlimited() bool {
	return fa.max.Equal(UnlimitedMaxFeeAmount)
}

func (fa RatioFeeer) isZero() bool {
	return fa.ratio == 0
}

func (fa RatioFeeer) isOne() bool {
	return fa.ratio == 1
}
