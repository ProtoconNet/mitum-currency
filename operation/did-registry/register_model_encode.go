package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

func (fact *RegisterModelFact) unpack(
	enc encoder.Encoder,
	sa, ta, didMethod, cid string,
) error {
	fact.currency = types.CurrencyID(cid)

	sender, err := base.DecodeAddress(sa, enc)
	if err != nil {
		return err
	}
	fact.sender = sender
	contract, err := base.DecodeAddress(ta, enc)
	if err != nil {
		return err
	}
	fact.contract = contract
	fact.didMethod = didMethod

	return nil
}
