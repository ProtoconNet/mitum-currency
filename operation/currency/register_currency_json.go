package currency

import (
	"encoding/json"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	jsonenc "github.com/ProtoconNet/mitum2/util/encoder/json"
)

type RegisterCurrencyFactJSONMarshaler struct {
	base.BaseFactJSONMarshaler
	Currency types.CurrencyDesign `json:"currency"`
}

func (fact RegisterCurrencyFact) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(RegisterCurrencyFactJSONMarshaler{
		BaseFactJSONMarshaler: fact.BaseFact.JSONMarshaler(),
		Currency:              fact.currency,
	})
}

type RegisterCurrencyFactJSONUnMarshaler struct {
	base.BaseFactJSONUnmarshaler
	Currency json.RawMessage `json:"currency"`
}

func (fact *RegisterCurrencyFact) DecodeJSON(b []byte, enc *jsonenc.Encoder) error {
	e := util.StringError("failed to decode RegisterCurrencyFact")

	var uf RegisterCurrencyFactJSONUnMarshaler
	if err := enc.Unmarshal(b, &uf); err != nil {
		return e.Wrap(err)
	}

	fact.BaseFact.SetJSONUnmarshaler(uf.BaseFactJSONUnmarshaler)

	return fact.unpack(enc, uf.Currency)
}

type registerCurrencyMarshaler struct {
	common.BaseOperationJSONMarshaler
}

func (op RegisterCurrency) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(registerCurrencyMarshaler{
		BaseOperationJSONMarshaler: op.BaseOperation.JSONMarshaler(),
	})
}

func (op *RegisterCurrency) DecodeJSON(b []byte, enc *jsonenc.Encoder) error {
	e := util.StringError("failed to decode RegisterCurrency")

	var ubo common.BaseNodeOperation
	if err := ubo.DecodeJSON(b, enc); err != nil {
		return e.Wrap(err)
	}

	op.BaseNodeOperation = ubo

	return nil
}
