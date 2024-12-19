package extension

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	UpdateHandlerFactHint = hint.MustNewHint("mitum-extension-update-handler-operation-fact-v0.0.1")
	UpdateHandlerHint     = hint.MustNewHint("mitum-extension-update-handler-operation-v0.0.1")
)

const MaxHandlers = 20

type UpdateHandlerFact struct {
	base.BaseFact
	sender   base.Address
	contract base.Address
	handlers []base.Address
	currency types.CurrencyID
}

func NewUpdateHandlerFact(
	token []byte,
	sender,
	contract base.Address,
	handlers []base.Address,
	currency types.CurrencyID,
) UpdateHandlerFact {
	fact := UpdateHandlerFact{
		BaseFact: base.NewBaseFact(UpdateHandlerFactHint, token),
		sender:   sender,
		contract: contract,
		handlers: handlers,
		currency: currency,
	}

	fact.SetHash(fact.GenerateHash())

	return fact
}

func (fact UpdateHandlerFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact UpdateHandlerFact) Bytes() []byte {
	bs := make([][]byte, len(fact.handlers)+4)
	bs[0] = fact.Token()
	bs[1] = fact.sender.Bytes()
	bs[2] = fact.contract.Bytes()
	bs[3] = fact.currency.Bytes()
	for i := range fact.handlers {
		bs[4+i] = fact.handlers[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (fact UpdateHandlerFact) IsValid(b []byte) error {
	if err := fact.BaseHinter.IsValid(nil); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if err := util.CheckIsValiders(nil, false, fact.sender, fact.contract, fact.currency); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if len(fact.handlers) > MaxHandlers {
		return common.ErrFactInvalid.Wrap(common.ErrArrayLen.Wrap(errors.Errorf("number of handlers, %d, exceeds maximum limit, %d", len(fact.handlers), MaxHandlers)))
	}

	handlersMap := make(map[string]struct{})
	for i := range fact.handlers {
		_, found := handlersMap[fact.handlers[i].String()]
		if found {
			return common.ErrFactInvalid.Wrap(common.ErrDupVal.Wrap(errors.Errorf("handler %v", fact.handlers[i])))
		} else {
			handlersMap[fact.handlers[i].String()] = struct{}{}
		}
		if err := fact.handlers[i].IsValid(nil); err != nil {
			return common.ErrFactInvalid.Wrap(common.ErrValOOR.Wrap(errors.Errorf("invalid handler address %v", err)))
		}
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact UpdateHandlerFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact UpdateHandlerFact) Token() base.Token {
	return fact.BaseFact.Token()
}

func (fact UpdateHandlerFact) Currency() types.CurrencyID {
	return fact.currency
}

func (fact UpdateHandlerFact) Sender() base.Address {
	return fact.sender
}

func (fact UpdateHandlerFact) Signer() base.Address {
	return fact.sender
}

func (fact UpdateHandlerFact) Contract() base.Address {
	return fact.contract
}

func (fact UpdateHandlerFact) Handlers() []base.Address {
	return fact.handlers
}

func (fact UpdateHandlerFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.handlers)+2)

	oprs := fact.handlers
	copy(as, oprs)

	as[len(fact.handlers)] = fact.sender
	as[len(fact.handlers)+1] = fact.contract

	return as, nil
}

func (fact UpdateHandlerFact) FeeBase() map[types.CurrencyID][]common.Big {
	required := make(map[types.CurrencyID][]common.Big)
	required[fact.Currency()] = []common.Big{common.ZeroBig}

	return required
}

func (fact UpdateHandlerFact) FeePayer() base.Address {
	return fact.sender
}

func (fact UpdateHandlerFact) FactUser() base.Address {
	return fact.sender
}

func (fact UpdateHandlerFact) ContractOwnerOnly() [][2]base.Address {
	return [][2]base.Address{{fact.contract, fact.sender}}
}

type UpdateHandler struct {
	extras.ExtendedOperation
}

func NewUpdateHandler(fact UpdateHandlerFact) (UpdateHandler, error) {
	return UpdateHandler{
		ExtendedOperation: extras.NewExtendedOperation(UpdateHandlerHint, fact),
	}, nil
}
