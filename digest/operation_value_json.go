package digest

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/localtime"
)

type ExpendedOperationMarshaler struct {
	common.BaseOperationJSONMarshaler
	extras.BaseOperationExtensionsJSONMarshaler
}

type ExpendedOperationValueJSONMarshaler struct {
	hint.BaseHinter
	Hash        util.Hash                  `json:"hash"`
	Operation   ExpendedOperationMarshaler `json:"operation"`
	Height      base.Height                `json:"height"`
	ConfirmedAt localtime.Time             `json:"confirmed_at"`
	Reason      string                     `json:"reason"`
	InState     bool                       `json:"in_state"`
	Index       uint64                     `json:"index"`
}

type OperationValueJSONMarshaler struct {
	hint.BaseHinter
	Hash        util.Hash                         `json:"hash"`
	Operation   common.BaseOperationJSONMarshaler `json:"operation"`
	Height      base.Height                       `json:"height"`
	ConfirmedAt localtime.Time                    `json:"confirmed_at"`
	Reason      string                            `json:"reason"`
	InState     bool                              `json:"in_state"`
	Index       uint64                            `json:"index"`
}

func (va OperationValue) MarshalJSON() ([]byte, error) {
	var op base.Operation = va.op
	eo, ok := op.(extras.ExtendedOperation)
	if ok && len(eo.Extensions()) > 0 {
		return util.MarshalJSON(ExpendedOperationValueJSONMarshaler{
			BaseHinter: va.BaseHinter,
			Hash:       va.op.Fact().Hash(),
			Operation: ExpendedOperationMarshaler{
				BaseOperationJSONMarshaler: common.BaseOperationJSONMarshaler{
					BaseHinter: hint.NewBaseHinter(va.op.Hint()),
					Hash:       va.op.Hash(),
					Fact:       va.op.Fact(),
					Signs:      va.op.Signs(),
				},
				BaseOperationExtensionsJSONMarshaler: extras.BaseOperationExtensionsJSONMarshaler{
					Extension: eo.Extensions(),
				},
			},
			Height:      va.height,
			ConfirmedAt: localtime.New(va.confirmedAt),
			Reason:      va.reason,
			InState:     va.inState,
			Index:       va.index,
		})
	} else {
		return util.MarshalJSON(OperationValueJSONMarshaler{
			BaseHinter: va.BaseHinter,
			Hash:       va.op.Fact().Hash(),
			Operation: common.BaseOperationJSONMarshaler{
				BaseHinter: hint.NewBaseHinter(va.op.Hint()),
				Hash:       va.op.Hash(),
				Fact:       va.op.Fact(),
				Signs:      va.op.Signs(),
			},
			Height:      va.height,
			ConfirmedAt: localtime.New(va.confirmedAt),
			Reason:      va.reason,
			InState:     va.inState,
			Index:       va.index,
		})
	}

}
