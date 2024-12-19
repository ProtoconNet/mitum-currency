package did_registry

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	dtypes "github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/pkg/errors"
)

type UpdateDIDDocumentFactJSONMarshaler struct {
	base.BaseFactJSONMarshaler
	Sender   base.Address       `json:"sender"`
	Contract base.Address       `json:"contract"`
	DID      string             `json:"did"`
	Document dtypes.DIDDocument `json:"document"`
	Currency types.CurrencyID   `json:"currency"`
}

func (fact UpdateDIDDocumentFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(UpdateDIDDocumentFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Sender:                fact.sender,
		Contract:              fact.contract,
		DID:                   fact.did,
		Document:              fact.document,
		Currency:              fact.currency,
	})
}

type UpdateDIDDocumentFactJSONUnmarshaler struct {
	base.BaseFactJSONUnmarshaler
	Sender   string          `json:"sender"`
	Contract string          `json:"contract"`
	DID      string          `json:"did"`
	Document json.RawMessage `json:"document"`
	Currency string          `json:"currency"`
}

func (fact *UpdateDIDDocumentFact) DecodeJSON(b []byte, enc encoder.Encoder) error {
	var u UpdateDIDDocumentFactJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	fact.BaseFact.SetJSONUnmarshaler(u.BaseFactJSONUnmarshaler)

	if t, err := enc.Decode(u.Document); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	} else if v, ok := t.(dtypes.DIDDocument); !ok {
		return common.DecorateError(errors.Errorf("expected DIDDocument, but %T", t), common.ErrDecodeJson, *fact)
	} else {
		fact.document = v
	}

	if err := fact.unpack(enc, u.Sender, u.Contract, u.DID, u.Currency); err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	return nil
}

func (op UpdateDIDDocument) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(OperationMarshaler{
		BaseOperationJSONMarshaler:           op.BaseOperation.JSONMarshaler(),
		BaseOperationExtensionsJSONMarshaler: op.BaseOperationExtensions.JSONMarshaler(),
	})
}

func (op *UpdateDIDDocument) DecodeJSON(b []byte, enc encoder.Encoder) error {
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
