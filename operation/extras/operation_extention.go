package extras

import (
	"encoding/json"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state"
	didstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	estate "github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
)

type Authentication interface {
	hint.Hinter
	util.IsValider
	util.Byter
	Contract() base.Address
	AuthenticationID() string
	ProofData() string
}

type Settlement interface {
	hint.Hinter
	util.IsValider
	util.Byter
	OpSender() base.Address
}

type ProxyPayer interface {
	hint.Hinter
	util.IsValider
	util.Byter
	ProxyPayer() base.Address
}

var BaseAuthenticationHint = hint.MustNewHint("mitum-extension-base-authentication-v0.0.1")
var AuthenticationExtensionType string = "authentication"

type BaseAuthentication struct {
	hint.BaseHinter
	contract         base.Address
	authenticationID string
	proofData        string
}

func NewBaseAuthentication(contract base.Address, authenticationID, proofData string) BaseAuthentication {
	return BaseAuthentication{
		BaseHinter:       hint.NewBaseHinter(BaseAuthenticationHint),
		contract:         contract,
		authenticationID: authenticationID,
		proofData:        proofData,
	}
}

func (ba BaseAuthentication) Contract() base.Address {
	return ba.contract
}

func (ba BaseAuthentication) AuthenticationID() string {
	return ba.authenticationID
}

func (ba BaseAuthentication) ProofData() string {
	return ba.proofData
}

func (ba BaseAuthentication) ExtType() string {
	return AuthenticationExtensionType
}

func (ba BaseAuthentication) Bytes() []byte {
	if ba.Equal(BaseAuthentication{}) {
		return []byte{}
	}
	var bs [][]byte
	bs = append(bs, ba.contract.Bytes())
	bs = append(bs, []byte(ba.authenticationID))
	bs = append(bs, []byte(ba.proofData))
	return util.ConcatBytesSlice(bs...)
}

func (ba BaseAuthentication) IsValid([]byte) error {
	if err := util.CheckIsValiders(nil, false, ba.contract); err != nil {
		return common.ErrValueInvalid.Wrap(err)
	}

	if len(ba.authenticationID) < 1 {
		return common.ErrValueInvalid.Wrap(errors.Errorf("empty authentication id"))
	}

	if len(ba.proofData) < 1 {
		return common.ErrValueInvalid.Wrap(errors.Errorf("empty proof data"))
	}

	return nil
}

func (ba BaseAuthentication) Equal(b BaseAuthentication) bool {
	if !ba.contract.Equal(b.contract) {
		return false
	}

	if ba.authenticationID != b.authenticationID {
		return false
	}

	if ba.proofData != b.proofData {
		return false
	}

	return true
}

func (ba BaseAuthentication) Verify(op base.Operation, getStateFunc base.GetStateFunc) error {
	var authentication types.IAuthentication
	var doc types.DIDDocument
	dr, err := types.NewDIDResourceFromString(ba.AuthenticationID())
	if err != nil {
		return err
	}

	switch i := op.Fact().(type) {
	case FeeAble:
		factSender := i.FeePayer()
		if dr.MethodSpecificID() != factSender.String() {
			return errors.Errorf("fact sender is not matched with authentication id controller")
		}
	}

	contract := ba.Contract()
	if contract == nil {
		return errors.Errorf("empty contract address")
	}
	if st, err := state.ExistsState(didstate.DocumentStateKey(contract, dr.DID()), "did document", getStateFunc); err != nil {
		return err
	} else if doc, err = didstate.GetDocumentFromState(st); err != nil {
		return err
	}

	authentication, err = doc.Authentication(ba.AuthenticationID())
	if err != nil {
		return err
	}

	if authentication.Controller() != dr.DID() {
		return errors.Errorf(
			"Controller of authentication id, %v is not matched with DID in document, %v",
			authentication.Controller(),
			dr.DID(),
		)
	}

	switch authentication.AuthType() {
	case types.AuthTypeECDSASECP:
		details := authentication.Details()
		pubKey, ok := details.(base.Publickey)
		if !ok {
			return errors.Errorf("expected PublicKey, but %T", details)
		}

		signature := base58.Decode(ba.ProofData())
		err := pubKey.Verify(op.Fact().Hash().Bytes(), signature)
		if err != nil {
			return err
		}
	case types.AuthTypeVC:
		details := authentication.Details()
		m, ok := details.(map[string]interface{})
		if !ok {
			return errors.Errorf("get authentication details")
		}
		p := m["proof"]
		proof, ok := p.(types.Proof)
		if !ok {
			return errors.Errorf("get vc proof")
		}
		vm := proof.VerificationMethod()
		dr, err := types.NewDIDResourceFromString(vm)
		if err != nil {
			return err
		}

		var doc types.DIDDocument
		if st, err := state.ExistsState(didstate.DocumentStateKey(contract, dr.DID()), "did document", getStateFunc); err != nil {
			return err
		} else if doc, err = didstate.GetDocumentFromState(st); err != nil {
			return err
		}

		sAuthentication, err := doc.Authentication(vm)
		if err != nil {
			return err
		}

		if sAuthentication.Controller() != dr.DID() {
			return errors.Errorf(
				"Controller of authentication id, %v is not matched with DID in document, %v", authentication.Controller(), dr.DID())
		}

		if sAuthentication.AuthType() != types.AuthTypeECDSASECP {
			return errors.Errorf("auth type must be EcdsaSecp256k1VerificationKey2019")
		}

		sDetails := sAuthentication.Details()
		pubKey, ok := sDetails.(base.Publickey)
		if !ok {
			return errors.Errorf("expected PublicKey, but %T", details)
		}

		signature := base58.Decode(ba.ProofData())

		err = pubKey.Verify(op.Fact().Hash().Bytes(), signature)
		if err != nil {
			return errors.Errorf("signature verification failed, %v", err)
		}
	default:
	}

	return nil
}

var BaseSettlementHint = hint.MustNewHint("mitum-extension-base-settlement-v0.0.1")
var SettlementExtensionType string = "settlement"

type BaseSettlement struct {
	hint.BaseHinter
	opSender base.Address
}

func NewBaseSettlement(opSender base.Address) BaseSettlement {
	return BaseSettlement{
		BaseHinter: hint.NewBaseHinter(BaseSettlementHint),
		opSender:   opSender,
	}
}

func (bs BaseSettlement) OpSender() base.Address {
	return bs.opSender
}

func (bs BaseSettlement) Bytes() []byte {
	if bs.Equal(BaseSettlement{}) {
		return []byte{}
	}
	var b [][]byte
	b = append(b, bs.opSender.Bytes())
	return util.ConcatBytesSlice(b...)
}

func (bs BaseSettlement) IsValid([]byte) error {
	if err := util.CheckIsValiders(nil, false, bs.opSender); err != nil {
		return common.ErrValueInvalid.Wrap(err)
	}

	return nil
}

func (bs BaseSettlement) ExtType() string {
	return SettlementExtensionType
}

func (bs BaseSettlement) Verify(op base.Operation, getStateFunc base.GetStateFunc) error {
	opSender := bs.OpSender()
	if opSender == nil {
		return errors.Errorf("empty op sender")
	}
	if err := state.CheckFactSignsByState(opSender, op.Signs(), getStateFunc); err != nil {
		return err
	}

	if _, _, aErr, cErr := state.ExistsCAccount(opSender, "op sender", true, false, getStateFunc); aErr != nil {
		return aErr
	} else if cErr != nil {
		return cErr
	}

	return nil
}

func (bs BaseSettlement) Equal(b BaseSettlement) bool {
	if !bs.opSender.Equal(b.opSender) {
		return false
	}

	return true
}

var BaseProxyPayerHint = hint.MustNewHint("mitum-extension-base-proxy-payer-v0.0.1")
var ProxyPayerExtensionType string = "proxy_payer"

type BaseProxyPayer struct {
	hint.BaseHinter
	proxyPayer base.Address
}

func NewBaseProxyPayer(proxyPayer base.Address) BaseProxyPayer {
	return BaseProxyPayer{
		BaseHinter: hint.NewBaseHinter(BaseProxyPayerHint),
		proxyPayer: proxyPayer,
	}
}

func (bs BaseProxyPayer) ProxyPayer() base.Address {
	return bs.proxyPayer
}

func (bs BaseProxyPayer) Bytes() []byte {
	if bs.Equal(BaseProxyPayer{}) {
		return []byte{}
	}
	var b [][]byte
	if bs.proxyPayer != nil {
		b = append(b, bs.proxyPayer.Bytes())
	}
	return util.ConcatBytesSlice(b...)
}

func (bs BaseProxyPayer) IsValid([]byte) error {
	if err := util.CheckIsValiders(nil, false, bs.proxyPayer); err != nil {
		return common.ErrValueInvalid.Wrap(err)
	}

	return nil
}

func (bs BaseProxyPayer) ExtType() string {
	return ProxyPayerExtensionType
}

func (bs BaseProxyPayer) Verify(op base.Operation, getStateFunc base.GetStateFunc) error {
	proxyPayer := bs.ProxyPayer()
	if proxyPayer == nil {
		return errors.Errorf("empty proxy payer")
	}
	feeBaser, ok := op.Fact().(FeeAble)
	if !ok {
		return errors.Errorf("failed to get fact sender from operation")
	}

	sender := feeBaser.FeePayer()
	if sender == nil {
		return errors.Errorf("empty fact sender")
	}

	if _, cSt, aErr, cErr := state.ExistsCAccount(proxyPayer, "proxy payer", true, true, getStateFunc); aErr != nil {
		return aErr
	} else if cErr != nil {
		return cErr
	} else if ca, err := estate.LoadCAStateValue(cSt); err != nil {
		return err
	} else if !ca.IsRecipients(sender) {
		return errors.Errorf("user is not recipient of proxy payer")
	}

	return nil
}

func (bs BaseProxyPayer) Equal(b BaseProxyPayer) bool {
	if !bs.proxyPayer.Equal(b.proxyPayer) {
		return false
	}

	return true
}

type ExtendedOperation struct {
	common.BaseOperation
	*BaseOperationExtensions
}

func NewExtendedOperation(hint hint.Hint, fact base.Fact) ExtendedOperation {
	return ExtendedOperation{
		BaseOperation:           common.NewBaseOperation(hint, fact),
		BaseOperationExtensions: NewBaseOperationExtensions(),
	}
}

func (op ExtendedOperation) IsValid(networkID []byte) error {
	if err := op.BaseOperation.IsValid(networkID); err != nil {
		return err
	}
	if err := op.BaseOperationExtensions.IsValid(networkID); err != nil {
		return err
	}

	return nil
}

func (op *ExtendedOperation) HashSign(priv base.Privatekey, networkID base.NetworkID) error {
	err := op.Sign(priv, networkID)
	if err != nil {
		return err
	}

	op.SetHash(op.hash())

	return nil
}

func (op ExtendedOperation) hash() util.Hash {
	return valuehash.NewSHA256(op.HashBytes())
}

func (op ExtendedOperation) HashBytes() []byte {
	var bs [][]byte
	bs = append(bs, op.BaseOperation.HashBytes())

	if op.BaseOperationExtensions != nil {
		bs = append(bs, op.BaseOperationExtensions.Bytes())
	}

	return util.ConcatBytesSlice(bs...)
}

type OperationExtension interface {
	ExtType() string
	Verify(base.Operation, base.GetStateFunc) error
	util.IsValider
	util.Byter
}

type OperationExtensions interface {
	util.IsValider
	util.Byter
	Verify(base.Operation, base.GetStateFunc) error
	Extension(string) OperationExtension
	Extensions() map[string]OperationExtension
	AddExtension(OperationExtension) error
}

type BaseOperationExtensions struct {
	extension map[string]OperationExtension
}

func NewBaseOperationExtensions() *BaseOperationExtensions {
	return &BaseOperationExtensions{
		extension: make(map[string]OperationExtension),
	}

}

func (be BaseOperationExtensions) Bytes() []byte {
	var bs [][]byte
	if be.extension != nil {
		extension, _ := json.Marshal(be.extension)
		bs = append(bs, valuehash.NewSHA256(extension).Bytes())
	}

	return util.ConcatBytesSlice(bs...)
}

func (be BaseOperationExtensions) Verify(op base.Operation, getStateFunc base.GetStateFunc) error {
	auth := be.Extension(AuthenticationExtensionType)
	if auth != nil {
		if err := auth.IsValid(nil); err != nil {
			return err
		}

		if err := auth.Verify(op, getStateFunc); err != nil {
			return err
		}
	}
	settlement := be.Extension(SettlementExtensionType)
	if settlement != nil {
		if err := settlement.IsValid(nil); err != nil {
			return err
		}

		if err := settlement.Verify(op, getStateFunc); err != nil {
			return err
		}
	}
	proxyPayer := be.Extension(ProxyPayerExtensionType)
	if proxyPayer != nil {
		if err := proxyPayer.IsValid(nil); err != nil {
			return err
		}

		if err := proxyPayer.Verify(op, getStateFunc); err != nil {
			return err
		}
	}

	return nil
}

func (be BaseOperationExtensions) IsValid(networkID []byte) error {
	for _, ext := range be.extension {
		if err := ext.IsValid(networkID); err != nil {
			return err
		}
	}
	return nil
}

func (be BaseOperationExtensions) Extension(extType string) OperationExtension {
	if len(be.extension) < 1 {
		return nil
	}

	extension, ok := be.extension[extType]
	if !ok {
		return nil
	}

	return extension
}

func (be BaseOperationExtensions) Extensions() map[string]OperationExtension {
	return be.extension
}

func (be *BaseOperationExtensions) AddExtension(extension OperationExtension) error {
	if err := util.CheckIsValiders(nil, false, extension); err != nil {
		return err
	}

	_, ok := be.extension[extension.ExtType()]
	if ok {
		return errors.Errorf("%s is already added", extension.ExtType())
	}

	be.extension[extension.ExtType()] = extension

	return nil
}

type FeeAble interface {
	FeeBase() map[types.CurrencyID][]common.Big
	FeePayer() base.Address
}

// VerifyFeeAble function checks
// existence of currency id
func VerifyFeeAble(fact FeeAble, getStateFunc base.GetStateFunc) base.OperationProcessReasonError {
	if len(fact.FeeBase()) < 1 {
		return base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("empty Fee Base"))
	}

	for cid := range fact.FeeBase() {
		_, err := state.ExistsCurrencyPolicy(cid, getStateFunc)
		if err != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("fail to get currency %v", cid))
		}
	}

	return nil
}

type FactUser interface {
	FactUser() base.Address
}

// VerifyFactUser function checks
// existence of user account
// it is not a contract account
func VerifyFactUser(fact FactUser, getStateFunc base.GetStateFunc) base.OperationProcessReasonError {
	user := fact.FactUser()

	if user == nil {
		return base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.
				Errorf("empty user account"))
	}
	if _, _, aErr, cErr := state.ExistsCAccount(user, "sender", true, false, getStateFunc); aErr != nil {
		return base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Errorf("%v", aErr))
	} else if cErr != nil {
		return base.NewBaseOperationProcessReasonError(
			common.ErrMPreProcess.Wrap(common.ErrMCAccountNA).Errorf("%v", cErr))
	}

	return nil
}

type ContractOwnerOnly interface {
	ContractOwnerOnly() [][2]base.Address // contract, sender
}

// VerifyContractOwnerOnly function checks
// existence of contract account
// sender is owner of contract account
func VerifyContractOwnerOnly(fact ContractOwnerOnly, getStateFunc base.GetStateFunc) base.OperationProcessReasonError {
	for _, addresses := range fact.ContractOwnerOnly() {
		contract := addresses[0]
		sender := addresses[1]
		if contract == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty contract account"))
		}
		if sender == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty sender account"))
		}

		if _, cSt, aErr, cErr := state.ExistsCAccount(contract, "contract", true, true, getStateFunc); aErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", aErr))
		} else if cErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", cErr))
		} else if status, err := estate.StateContractAccountValue(cSt); err != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMStateValInvalid).
					Errorf("%v", cErr))
		} else if !status.Owner().Equal(sender) {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMAccountNAth).
					Errorf("sender %v is not owner of contract account", sender))
		}
	}

	return nil
}

type InActiveContractOwnerHandlerOnly interface {
	InActiveContractOwnerHandlerOnly() [][2]base.Address // contract, sender
}

// VerifyInActiveContractOwnerHandlerOnly function checks
// existence of contract account
// sender is owner of contract account
// inactive contract account
func VerifyInActiveContractOwnerHandlerOnly(fact InActiveContractOwnerHandlerOnly, getStateFunc base.GetStateFunc) base.OperationProcessReasonError {
	for _, addresses := range fact.InActiveContractOwnerHandlerOnly() {
		contract := addresses[0]
		sender := addresses[1]

		if contract == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty contract account"))
		}
		if sender == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty sender account"))
		}

		_, cSt, aErr, cErr := state.ExistsCAccount(contract, "contract", true, true, getStateFunc)
		if aErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", aErr))
		} else if cErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", cErr))
		}

		ca, err := estate.CheckCAAuthFromState(cSt, sender)
		if err != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", err))
		}

		if ca.IsActive() {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMServiceE).Errorf(
					"contract account %v has already been activated", contract))
		}
	}

	return nil
}

type ActiveContractOwnerHandlerOnly interface {
	ActiveContractOwnerHandlerOnly() [][2]base.Address // contract, sender
}

// VerifyActiveContractOwnerHandlerOnly function checks
// existence of contract account
// sender is owner of contract account
func VerifyActiveContractOwnerHandlerOnly(fact ActiveContractOwnerHandlerOnly, getStateFunc base.GetStateFunc) base.OperationProcessReasonError {
	for _, addresses := range fact.ActiveContractOwnerHandlerOnly() {
		contract := addresses[0]
		sender := addresses[1]

		if contract == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty contract account"))
		}
		if sender == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty sender account"))
		}

		_, cSt, aErr, cErr := state.ExistsCAccount(contract, "contract", true, true, getStateFunc)
		if aErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", aErr))
		} else if cErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", cErr))
		}

		ca, err := estate.CheckCAAuthFromState(cSt, sender)
		if err != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", err))
		}

		if !ca.IsActive() {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMServiceNF).Errorf(
					"contract account %v has not been activated", contract))
		}
	}

	return nil
}

type ActiveContract interface {
	ActiveContract() []base.Address
}

// VerifyActiveContract function checks
// existence of contract account
// active contract account
func VerifyActiveContract(fact ActiveContract, getStateFunc base.GetStateFunc) base.OperationProcessReasonError {
	for _, contract := range fact.ActiveContract() {
		if contract == nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("empty contract account"))
		}

		_, cSt, aErr, cErr := state.ExistsCAccount(contract, "contract", true, true, getStateFunc)
		if aErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", aErr))
		} else if cErr != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", cErr))
		}

		ca, err := estate.LoadCAStateValue(cSt)
		if err != nil {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Errorf("%v", err))
		}

		if !ca.IsActive() {
			return base.NewBaseOperationProcessReasonError(
				common.ErrMPreProcess.
					Wrap(common.ErrMServiceNF).Errorf(
					"contract account %v has not been activated", contract))
		}
	}

	return nil
}
