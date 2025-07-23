package plugin

import (
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
)

func (de *Design) unmarshal(
	_ encoder.Encoder,
	ht hint.Hint, cc string,
) error {
	de.BaseHinter = hint.NewBaseHinter(ht)
	de.contractCode = cc

	return nil
}
