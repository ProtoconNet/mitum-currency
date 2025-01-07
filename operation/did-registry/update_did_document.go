package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	ctypes "github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

var (
	UpdateDIDDocumentFactHint = hint.MustNewHint("mitum-did-update-did-document-operation-fact-v0.0.1")
	UpdateDIDDocumentHint     = hint.MustNewHint("mitum-did-update-did-document-operation-v0.0.1")
)

type UpdateDIDDocumentFact struct {
	base.BaseFact
	sender   base.Address
	contract base.Address
	did      string
	document types.DIDDocument
	currency ctypes.CurrencyID
}

func NewUpdateDIDDocumentFact(
	token []byte, sender, contract base.Address,
	did string, doc types.DIDDocument, currency ctypes.CurrencyID) UpdateDIDDocumentFact {
	bf := base.NewBaseFact(UpdateDIDDocumentFactHint, token)
	fact := UpdateDIDDocumentFact{
		BaseFact: bf,
		sender:   sender,
		contract: contract,
		did:      did,
		document: doc,
		currency: currency,
	}

	fact.SetHash(fact.GenerateHash())
	return fact
}

func (fact UpdateDIDDocumentFact) IsValid(b []byte) error {
	if fact.sender.Equal(fact.contract) {
		return common.ErrFactInvalid.Wrap(
			common.ErrSelfTarget.Wrap(errors.Errorf("sender %v is same with contract account", fact.sender)))
	}

	if err := util.CheckIsValiders(nil, false,
		fact.BaseHinter,
		fact.sender,
		fact.contract,
		fact.currency,
		fact.document,
	); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	if _, adrStr, err := types.ParseDIDScheme(fact.did); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	} else if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	} else if fact.Sender().String() != adrStr {
		return common.ErrFactInvalid.Wrap(
			errors.Errorf("sender %v is not controller of did %v", fact.sender, fact.did))
	}

	if err := common.IsValidOperationFact(fact, b); err != nil {
		return common.ErrFactInvalid.Wrap(err)
	}

	return nil
}

func (fact UpdateDIDDocumentFact) Hash() util.Hash {
	return fact.BaseFact.Hash()
}

func (fact UpdateDIDDocumentFact) GenerateHash() util.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact UpdateDIDDocumentFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.Token(),
		fact.sender.Bytes(),
		fact.contract.Bytes(),
		[]byte(fact.did),
		fact.document.Bytes(),
		fact.currency.Bytes(),
	)
}

func (fact UpdateDIDDocumentFact) Token() base.Token {
	return fact.BaseFact.Token()
}

func (fact UpdateDIDDocumentFact) Sender() base.Address {
	return fact.sender
}

func (fact UpdateDIDDocumentFact) Signer() base.Address {
	return fact.sender
}

func (fact UpdateDIDDocumentFact) Contract() base.Address {
	return fact.contract
}

func (fact UpdateDIDDocumentFact) DID() string {
	return fact.did
}

func (fact UpdateDIDDocumentFact) Document() types.DIDDocument {
	return fact.document
}

func (fact UpdateDIDDocumentFact) Currency() ctypes.CurrencyID {
	return fact.currency
}

func (fact UpdateDIDDocumentFact) Addresses() ([]base.Address, error) {
	as := []base.Address{fact.sender}

	return as, nil
}

func (fact UpdateDIDDocumentFact) FeeBase() map[types.CurrencyID][]common.Big {
	required := make(map[types.CurrencyID][]common.Big)
	required[fact.Currency()] = []common.Big{common.ZeroBig}

	return required
}

func (fact UpdateDIDDocumentFact) FeePayer() base.Address {
	return fact.sender
}

func (fact UpdateDIDDocumentFact) FactUser() base.Address {
	return fact.sender
}

func (fact UpdateDIDDocumentFact) ActiveContract() []base.Address {
	return []base.Address{fact.contract}
}

type UpdateDIDDocument struct {
	extras.ExtendedOperation
}

func NewUpdateDIDDocument(fact UpdateDIDDocumentFact) (UpdateDIDDocument, error) {
	return UpdateDIDDocument{
		ExtendedOperation: extras.NewExtendedOperation(UpdateDIDDocumentHint, fact),
	}, nil
}
