package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type CreateDIDFactJSONMarshaler struct {
	base.BaseFactJSONMarshaler
	Sender          base.Address     `json:"sender"`
	Contract        base.Address     `json:"contract"`
	AuthType        string           `json:"authType"`
	PublicKey       string           `json:"publicKey"`
	ServiceType     string           `json:"serviceType"`
	ServiceEndpoint string           `json:"serviceEndpoints"`
	Currency        types.CurrencyID `json:"currency"`
}

func (fact CreateDIDFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(CreateDIDFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Sender:                fact.sender,
		Contract:              fact.contract,
		AuthType:              fact.authType,
		PublicKey:             fact.publicKey.String(),
		ServiceType:           fact.serviceType,
		ServiceEndpoint:       fact.serviceEndpoint,
		Currency:              fact.currency,
	})
}

type CreateDIDFactJSONUnmarshaler struct {
	base.BaseFactJSONUnmarshaler
	Sender          string `json:"sender"`
	Contract        string `json:"contract"`
	AuthType        string `json:"authType"`
	PublicKey       string `json:"publicKey"`
	ServiceType     string `json:"serviceType"`
	ServiceEndpoint string `json:"serviceEndpoints"`
	Currency        string `json:"currency"`
}

func (fact *CreateDIDFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u CreateDIDFactJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	fact.BaseFact.SetJSONUnmarshaler(u.BaseFactJSONUnmarshaler)

	pubKey, err := base.DecodePublickeyFromString(u.PublicKey, enc)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	if err := fact.unpack(enc, u.Sender, u.Contract, u.AuthType, pubKey, u.ServiceType, u.ServiceEndpoint, u.Currency); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	return nil
}

type OperationMarshaler struct {
	common.BaseOperationJSONMarshaler
	extras.BaseOperationExtensionsJSONMarshaler
}

func (op CreateDID) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(OperationMarshaler{
		BaseOperationJSONMarshaler:           op.BaseOperation.JSONMarshaler(),
		BaseOperationExtensionsJSONMarshaler: op.BaseOperationExtensions.JSONMarshaler(),
	})
}

func (op *CreateDID) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var ubo common.BaseOperation
	if err := ubo.DecodeJSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *op)
	}

	op.BaseOperation = ubo

	var ueo extras.BaseOperationExtensions
	if err := ueo.DecodeJSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *op)
	}

	op.BaseOperationExtensions = &ueo

	return nil
}
