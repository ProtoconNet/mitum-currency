package types

import (
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var StringAddressHint = hint.MustNewHint("sas-v2")

type StringAddress struct {
	BaseStringAddress
}

func NewStringAddress(s string) StringAddress {
	return StringAddress{
		BaseStringAddress: NewBaseStringAddressWithHint(StringAddressHint, s),
	}
}

func ParseStringAddress(s string) (StringAddress, error) {
	b, t, err := hint.ParseFixedTypedString(s, base.AddressTypeSize)

	switch {
	case err != nil:
		return StringAddress{}, errors.Wrap(err, "parse StringAddress")
	case t != StringAddressHint.Type():
		return StringAddress{}, util.ErrInvalid.Errorf("wrong hint type in StringAddress")
	}

	return NewStringAddress(b), nil
}

func (ad StringAddress) IsValid([]byte) error {
	if err := ad.BaseHinter.IsValid(StringAddressHint.Type().Bytes()); err != nil {
		return util.ErrInvalid.WithMessage(err, "wrong hint in StringAddress")
	}

	if err := ad.BaseStringAddress.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid StringAddress")
	}

	return nil
}

func (ad *StringAddress) UnmarshalText(b []byte) error {
	ad.s = string(b) + StringAddressHint.Type().String()

	return nil
}

func (ad StringAddress) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.TypeString, bsoncore.AppendString(nil, ad.s), nil
}

func (ad *StringAddress) DecodeBSON(b []byte, _ *bsonenc.Encoder) error {
	*ad = NewStringAddress(string(b))

	return nil
}
