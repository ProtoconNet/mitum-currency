package plugin

import (
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

type DesignJSONMarshaler struct {
	hint.BaseHinter
	ContractCode string `json:"contract_code"`
}

func (de Design) MarshalJSON() ([]byte, error) {
	return util.MarshalJSON(DesignJSONMarshaler{
		BaseHinter:   de.BaseHinter,
		ContractCode: de.contractCode,
	})
}

type DesignJSONUnmarshaler struct {
	Hint         hint.Hint `json:"_hint"`
	ContractCode string    `json:"contract_code"`
}

func (de *Design) DecodeJSON(b []byte, enc encoder.Encoder) error {
	e := util.StringError("failed to decode json of Design")

	var u DesignJSONUnmarshaler
	if err := enc.Unmarshal(b, &u); err != nil {
		return e.Wrap(err)
	}

	return de.unmarshal(enc, u.Hint, u.ContractCode)
}
