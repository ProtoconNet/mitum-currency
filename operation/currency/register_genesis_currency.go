package currency

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	RegisterGenesisCurrencyFactHint = hint.MustNewHint("mitum-currency-register-genesis-currency-operation-fact-v0.0.1")
	RegisterGenesisCurrencyHint     = hint.MustNewHint("mitum-currency-register-genesis-currency-operation-v0.0.1")
)

type RegisterGenesisCurrencyFact struct {
	base.BaseFact
	genesisNodeKey base.Publickey
	keys           types.AccountKeys
	cs             []types.CurrencyDesign
}

func NewRegisterGenesisCurrencyFact(
	token []byte,
	genesisNodeKey base.Publickey,
	keys types.AccountKeys,
	cs []types.CurrencyDesign,
) RegisterGenesisCurrencyFact {
	fact := RegisterGenesisCurrencyFact{
		BaseFact:       base.NewBaseFact(RegisterGenesisCurrencyFactHint, token),
		genesisNodeKey: genesisNodeKey,
		keys:           keys,
		cs:             cs,
	}
	fact.SetHash(fact.GenerateHash())

	return fact
}

func (fact RegisterGenesisCurrencyFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact RegisterGenesisCurrencyFact) Bytes() []byte {
	bs := make([][]byte, len(fact.cs)+3)
	bs[0] = fact.Token()
	bs[1] = []byte(fact.genesisNodeKey.String())
	bs[2] = fact.keys.Bytes()

	for i := range fact.cs {
		bs[i+3] = fact.cs[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (fact RegisterGenesisCurrencyFact) IsValid(b []byte) error {
	if err := fact.BaseHinter.IsValid(nil); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if len(fact.cs) < 1 {
		return common.ErrFactInvalid.Wrap(common.ErrValOOR.Wrap(errors.Errorf("empty GenesisCurrency for RegisterGenesisCurrencyFact")))
	}

	if err := util.CheckIsValiders(nil, false, fact.genesisNodeKey, fact.keys); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	founds := map[types.CurrencyID]struct{}{}
	for i := range fact.cs {
		c := fact.cs[i]
		if err := c.IsValid(nil); err != nil {
			return common.ErrFactInvalid.Wrap(err)
		} else if !c.InitialSupply().Big().Equal(c.TotalSupply()) {
			return common.ErrFactInvalid.Wrap(
				errors.Errorf(
					"currency initial supply, %v not matched with total supply, %v",
					c.InitialSupply().Big(),
					c.TotalSupply(),
				))
		} else if _, found := founds[c.Currency()]; found {
			return common.ErrFactInvalid.Wrap(common.ErrDupVal.Wrap(errors.Errorf("currency id, %v", c.Currency())))
		} else {
			founds[c.Currency()] = struct{}{}
		}
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact RegisterGenesisCurrencyFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact RegisterGenesisCurrencyFact) Token() base.Token {
	return fact.BaseFact.Token()
}

func (fact RegisterGenesisCurrencyFact) GenesisNodeKey() base.Publickey {
	return fact.genesisNodeKey
}

func (fact RegisterGenesisCurrencyFact) Keys() types.AccountKeys {
	return fact.keys
}

func (fact RegisterGenesisCurrencyFact) Address() (base.Address, error) {
	return types.NewAddressFromKeys(fact.keys)
}

func (fact RegisterGenesisCurrencyFact) Currencies() []types.CurrencyDesign {
	return fact.cs
}

type RegisterGenesisCurrency struct {
	common.BaseOperation
}

func NewRegisterGenesisCurrency(
	fact RegisterGenesisCurrencyFact,
) RegisterGenesisCurrency {
	return RegisterGenesisCurrency{BaseOperation: common.NewBaseOperation(RegisterGenesisCurrencyHint, fact)}
}

func (op RegisterGenesisCurrency) IsValid(networkID []byte) error {
	if err := op.BaseOperation.IsValid(networkID); err != nil {
		return err
	}

	if len(op.Signs()) != 1 {
		return util.ErrInvalid.Errorf("Genesis currencies should be signed only by genesis node key")
	}

	fact, ok := op.Fact().(RegisterGenesisCurrencyFact)
	if !ok {
		return errors.Errorf("expected RegisterGenesisCurrencyFact, not %T", op.Fact())
	}

	if !fact.genesisNodeKey.Equal(op.Signs()[0].Signer()) {
		return util.ErrInvalid.Errorf("Not signed by genesis node key")
	}

	return nil
}
