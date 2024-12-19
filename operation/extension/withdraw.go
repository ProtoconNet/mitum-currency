package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	WithdrawFactHint = hint.MustNewHint("mitum-extension-withdraw-operation-fact-v0.0.1")
	WithdrawHint     = hint.MustNewHint("mitum-extension-withdraw-operation-v0.0.1")
)

var MaxWithdrawItems uint = 1000

type WithdrawItem interface {
	hint.Hinter
	util.IsValider
	currency.AmountsItem
	Bytes() []byte
	Target() base.Address
	Rebuild() WithdrawItem
}

type WithdrawFact struct {
	base.BaseFact
	sender base.Address
	items  []WithdrawItem
}

func NewWithdrawFact(token []byte, sender base.Address, items []WithdrawItem) WithdrawFact {
	bf := base.NewBaseFact(WithdrawFactHint, token)
	fact := WithdrawFact{
		BaseFact: bf,
		sender:   sender,
		items:    items,
	}
	fact.SetHash(fact.GenerateHash())

	return fact
}

func (fact WithdrawFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact WithdrawFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact WithdrawFact) Token() base.Token {
	return fact.BaseFact.Token()
}

func (fact WithdrawFact) Bytes() []byte {
	its := make([][]byte, len(fact.items))
	for i := range fact.items {
		its[i] = fact.items[i].Bytes()
	}

	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		util.ConcatBytesSlice(its...),
	)
}

func (fact WithdrawFact) IsValid(b []byte) error {
	if err := fact.BaseHinter.IsValid(nil); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if n := len(fact.items); n < 1 {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("empty items")))
	} else if n > int(MaxWithdrawItems) {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("items, %d over max, %d", n, MaxWithdrawItems)))
	}

	if err := util.CheckIsValiders(nil, false, fact.sender); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	foundTargets := map[string]struct{}{}
	for i := range fact.items {
		it := fact.items[i]
		if err := util.CheckIsValiders(nil, false, it); err != nil {
			return common.ErrFactInvalid.Wrap(err)
		}

		k := it.Target().String()
		switch _, found := foundTargets[k]; {
		case found:
			return common.ErrFactInvalid.Wrap(common.ErrDupVal.Wrap(errors.Errorf("target account, %v", it.Target())))
		case fact.sender.Equal(it.Target()):
			return common.ErrFactInvalid.Wrap(common.ErrSelfTarget.Wrap(errors.Errorf("target account is same with sender account, %v", fact.sender)))
		default:
			foundTargets[k] = struct{}{}
		}
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact WithdrawFact) Sender() base.Address {
	return fact.sender
}

func (fact WithdrawFact) Signer() base.Address {
	return fact.sender
}

func (fact WithdrawFact) Items() []WithdrawItem {
	return fact.items
}

func (fact WithdrawFact) Rebuild() WithdrawFact {
	items := make([]WithdrawItem, len(fact.items))
	for i := range fact.items {
		it := fact.items[i]
		items[i] = it.Rebuild()
	}

	fact.items = items
	fact.SetHash(fact.GenerateHash())

	return fact
}

func (fact WithdrawFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items)+1)
	for i := range fact.items {
		as[i] = fact.items[i].Target()
	}

	as[len(fact.items)] = fact.Sender()

	return as, nil
}

func (fact WithdrawFact) FeeBase() map[types.CurrencyID][]common.Big {
	required := make(map[types.CurrencyID][]common.Big)
	items := make([]currency.AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	for i := range items {
		it := items[i]
		amounts := it.Amounts()
		for j := range amounts {
			am := amounts[j]
			cid := am.Currency()
			big := am.Big()
			var k []common.Big
			if arr, found := required[cid]; found {
				arr = append(arr, big)
				copy(k, arr)
			} else {
				k = append(k, big)
			}
			required[cid] = k
		}
	}

	return required
}

func (fact WithdrawFact) FeePayer() base.Address {
	return fact.sender
}

func (fact WithdrawFact) FactUser() base.Address {
	return fact.sender
}

func (fact WithdrawFact) ContractOwnerOnly() [][2]base.Address {
	var arr [][2]base.Address
	for i := range fact.items {
		arr = append(arr, [2]base.Address{fact.items[i].Target(), fact.sender})
	}
	return arr
}

type Withdraw struct {
	extras.ExtendedOperation
}

func NewWithdraw(fact WithdrawFact) (Withdraw, error) {
	return Withdraw{
		ExtendedOperation: extras.NewExtendedOperation(WithdrawHint, fact),
	}, nil
}
