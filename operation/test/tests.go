package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/isaac"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type TestOperationProcessorNoItem interface {
	SetCurrency(string, int64, base.Address, []types.CurrencyID, bool) *TestOperationProcessorNoItem
	SetAmount(int64, types.CurrencyID, []types.Amount) *TestOperationProcessorNoItem
	SetContractAccount(base.Address, string, int64, types.CurrencyID, []Account, bool) *TestOperationProcessorNoItem
	SetAccount(string, int64, types.CurrencyID, []Account, bool) *TestOperationProcessorNoItem
	Print(string) *TestOperationProcessorNoItem
	RunPreProcess()
	RunProcess()
	IsValid()
}

type BaseTestOperationProcessorNoItem[To any] struct {
	*TestProcessor
	Op To
}

func NewBaseTestOperationProcessorNoItem[To any](tp *TestProcessor) BaseTestOperationProcessorNoItem[To] {
	t := BaseTestOperationProcessorNoItem[To]{
		TestProcessor: tp,
	}

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) SetCurrency(
	cid string, am int64, addr base.Address, target []types.CurrencyID, instate bool) *BaseTestOperationProcessorNoItem[To] {
	t.TestProcessor.SetCurrency2(cid, am, addr, target, instate)

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) SetAmount(am int64, cid types.CurrencyID, target []types.Amount) *BaseTestOperationProcessorNoItem[To] {
	t.TestProcessor.SetAmount(am, cid, target)

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) SetContractAccount(
	owner base.Address, priv string, amount int64, cid types.CurrencyID, target []Account, inState bool,
) *BaseTestOperationProcessorNoItem[To] {
	t.TestProcessor.SetContractAccount(owner, priv, amount, cid, target, inState)

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) SetAccount(
	priv string, amount int64, cid types.CurrencyID, target []Account, inState bool) *BaseTestOperationProcessorNoItem[To] {
	t.TestProcessor.SetAccount(priv, amount, cid, target, inState)

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) LoadOperation(fileName string) *BaseTestOperationProcessorNoItem[To] {
	op := t.TestProcessor.LoadOperation(fileName)
	nop, ok := op.(To)
	if !ok {
		panic(fmt.Sprintf("operation type is not %T\n", t.Op))
	}
	t.Op = nop

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) Print(fileName string) *BaseTestOperationProcessorNoItem[To] {
	t.TestProcessor.Print(fileName, t.Op)

	return t
}

func (t *BaseTestOperationProcessorNoItem[To]) RunPreProcess() {
	//t.MockGetter.On("Get", mock.Anything).Return(nil, false, nil)
	op, _ := any(t.Op).(base.Operation)
	_, err, _ := t.Opr.PreProcess(context.Background(), op, t.GetStateFunc)
	t.err = err

	return
}

func (t *BaseTestOperationProcessorNoItem[To]) RunProcess() {
	if t.err != nil {
		panic(t.err)
	}
	op, _ := any(t.Op).(base.Operation)
	stmv, err, _ := t.Opr.Process(context.Background(), op, t.GetStateFunc)
	for i := range stmv {
		st, found, _ := t.MockGetter.Get(stmv[i].Key())
		var merger base.StateValueMerger
		if !found {
			merger = stmv[i].Merger(base.Height(1), nil)
		} else {
			merger = stmv[i].Merger(base.Height(1), st)
		}
		merger.Merge(stmv[i].Value(), op.Fact().Hash())
		state, _ := merger.CloseValue()
		t.SetState(state, true)
	}
	t.err = err

	return
}

func (t *BaseTestOperationProcessorNoItem[To]) IsValid() {
	op, _ := any(t.Op).(base.Operation)
	err := op.IsValid(t.NetworkID)
	t.err = err

	return
}

type TestOperationProcessorWithItem[Tim any] interface {
	Items() []Tim
	SetCurrency(string, int64, base.Address, []types.CurrencyID, bool) *TestOperationProcessorWithItem[Tim]
	SetAmount(int64, types.CurrencyID, []types.Amount) *TestOperationProcessorWithItem[Tim]
	SetContractAccount(base.Address, string, int64, types.CurrencyID, []Account, bool) *TestOperationProcessorWithItem[Tim]
	SetAccount(string, int64, types.CurrencyID, []Account, bool) *TestOperationProcessorWithItem[Tim]
	Print(string) *TestOperationProcessorWithItem[Tim]
	RunPreProcess()
	RunProcess()
	IsValid()
}

type BaseTestOperationProcessorWithItem[To any, Tim any] struct {
	*TestProcessor
	Op    To
	items []Tim
}

func NewBaseTestOperationProcessorWithItem[To any, Tim any](tp *TestProcessor) BaseTestOperationProcessorWithItem[To, Tim] {
	t := BaseTestOperationProcessorWithItem[To, Tim]{
		TestProcessor: tp,
		items:         make([]Tim, 1),
	}

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) Items() []Tim {
	return t.items
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) SetCurrency(
	cid string, am int64, addr base.Address, target []types.CurrencyID, instate bool) *BaseTestOperationProcessorWithItem[To, Tim] {
	t.TestProcessor.SetCurrency2(cid, am, addr, target, instate)

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) SetAmount(am int64, cid types.CurrencyID, target []types.Amount) *BaseTestOperationProcessorWithItem[To, Tim] {
	t.TestProcessor.SetAmount(am, cid, target)

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) SetContractAccount(
	owner base.Address, priv string, amount int64, cid types.CurrencyID, target []Account, inState bool,
) *BaseTestOperationProcessorWithItem[To, Tim] {
	t.TestProcessor.SetContractAccount(owner, priv, amount, cid, target, inState)

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) SetAccount(
	priv string, amount int64, cid types.CurrencyID, target []Account, inState bool) *BaseTestOperationProcessorWithItem[To, Tim] {
	t.TestProcessor.SetAccount(priv, amount, cid, target, inState)

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) LoadOperation(fileName string) *BaseTestOperationProcessorWithItem[To, Tim] {
	op := t.TestProcessor.LoadOperation(fileName)
	nop, ok := op.(To)
	if !ok {
		panic(fmt.Sprintf("operation type is not %T\n", t.Op))
	}
	t.Op = nop

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) Print(fileName string) *BaseTestOperationProcessorWithItem[To, Tim] {
	t.TestProcessor.Print(fileName, t.Op)

	return t
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) RunPreProcess() {
	//t.MockGetter.On("Get", mock.Anything).Return(nil, false, nil)
	op, _ := any(t.Op).(base.Operation)
	_, err, _ := t.Opr.PreProcess(context.Background(), op, t.GetStateFunc)
	t.err = err

	return
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) RunProcess() {
	if t.err != nil {
		panic(t.err)
	}
	op, _ := any(t.Op).(base.Operation)
	stmv, err, _ := t.Opr.Process(context.Background(), op, t.GetStateFunc)
	for i := range stmv {
		st, found, _ := t.MockGetter.Get(stmv[i].Key())
		var merger base.StateValueMerger
		if !found {
			merger = stmv[i].Merger(base.Height(1), nil)
		} else {
			merger = stmv[i].Merger(base.Height(1), st)
		}
		merger.Merge(stmv[i].Value(), op.Fact().Hash())
		state, _ := merger.CloseValue()
		t.SetState(state, true)
	}
	t.err = err

	return
}

func (t *BaseTestOperationProcessorWithItem[To, Tim]) IsValid() {
	op, _ := any(t.Op).(base.Operation)
	err := op.IsValid(t.NetworkID)
	t.err = err

	return
}

type Account struct {
	adr  base.Address
	priv base.Privatekey
	keys types.AccountKeys
}

func (a *Account) SetAddress(adr base.Address) {
	a.adr = adr
}

func (a *Account) Address() base.Address {
	return a.adr
}

func (a *Account) SetPriv(priv base.Privatekey) {
	a.priv = priv
}

func (a *Account) Priv() base.Privatekey {
	return a.priv
}

func (a *Account) SetKeys(keys types.AccountKeys) {
	a.keys = keys
}

func (a *Account) Keys() types.AccountKeys {
	return a.keys
}

type TestProcessor struct {
	NetworkID       base.NetworkID
	GenesisPriv     base.Privatekey
	GenesisAddr     base.Address
	GenesisCurrency types.CurrencyID
	NodeAddr        base.Address
	NodePriv        base.Privatekey
	Opr             base.OperationProcessor
	GetStateFunc    func(string) (base.State, bool, error)
	MockGetter      *MockStateGetter
	Encoders        *encoder.Encoders
	err             error
}

func (t *TestProcessor) Setup(getter *MockStateGetter) {
	t.NetworkID = []byte("network_id")
	t.MockGetter = getter
	t.GetStateFunc = func(key string) (base.State, bool, error) {
		return t.MockGetter.Get(key)
	}
	t.NodeAddr, t.NodePriv = t.NewTestSuffrageState("Goq9aEmL9GD7votJPyh2LuTGPByUK62SUyAjJRykp2J7mpr", "nodesas", true)
	t.GenesisAddr, _, t.GenesisPriv = t.NewTestAccountState("7ddEcRfHjZszPDqQFRY57Xa7XBJp95kgrf7EE3p3V7oumpr", true)
	t.GenesisCurrency = t.NewTestCurrencyState("MCC", t.GenesisAddr, true)
	t.NewTestBalanceState(t.GenesisAddr, types.CurrencyID(t.GenesisCurrency), 100000000, true)

}

func (t *TestProcessor) NewPrivateKey(seed string) string {
	if len(seed) < 36 {
		seed = seed + strings.Repeat("*", 36)
	}
	k, _ := base.NewMPrivatekeyFromSeed(seed)
	return k.String()
}

func (t *TestProcessor) NewTestAccount(priv string) (types.Account, base.Address, types.AccountKeys, base.Privatekey) {
	privateKey, err := base.ParseMPrivatekey(priv)
	if err != nil {
		panic(err)
	}
	publicKey := privateKey.Publickey()
	key, _ := types.NewBaseAccountKey(publicKey, 100)
	keys, _ := types.NewBaseAccountKeys([]types.AccountKey{key}, 100)
	address, _ := types.NewAddressFromKeys(keys)
	account, _ := types.NewAccount(address, keys)

	return account, address, keys, privateKey
}

func (t *TestProcessor) SetState(state base.State, inState bool) {
	if inState {
		t.MockGetter.Set(state.Key(), state)
	}
}

// NewTestAccountState returns address, keys, private key, account state
func (t *TestProcessor) NewTestAccountState(priv string, inState bool) (base.Address, types.AccountKeys, base.Privatekey) {
	account, address, keys, privateKey := t.NewTestAccount(priv)
	state := common.NewBaseState(base.Height(1), ccstate.AccountStateKey(address), ccstate.NewAccountStateValue(account), nil, []util.Hash{})

	t.SetState(state, inState)

	return address, keys, privateKey
}

func (t *TestProcessor) NewTestContractAccountState(owner base.Address, priv string, inState bool) (base.Address, base.Privatekey) {
	account, address, _, privateKey := t.NewTestAccount(priv)
	cKeys, _ := types.NewContractAccountKeys()
	naccount, _ := account.SetKeys(cKeys)
	state := common.NewBaseState(base.Height(1), ccstate.AccountStateKey(address), ccstate.NewAccountStateValue(naccount), nil, []util.Hash{})

	//_, ownerAddress, _, _ := t.NewTestAccount(owner.String())
	status := types.NewContractAccountStatus(owner, []base.Address{})
	cState := common.NewBaseState(base.Height(1), extension.StateKeyContractAccount(address), extension.NewContractAccountStateValue(status), nil, []util.Hash{})

	t.SetState(state, inState)
	t.SetState(cState, inState)

	return address, privateKey
}

func (t *TestProcessor) NewTestBalanceState(addr base.Address, cid types.CurrencyID, am int64, inState bool) {
	if err := cid.IsValid(nil); err != nil {
		panic(err)
	}
	state := common.NewBaseState(
		base.Height(1),
		ccstate.BalanceStateKey(addr, cid),
		ccstate.NewBalanceStateValue(types.NewAmount(common.NewBig(am), cid)),
		nil,
		[]util.Hash{},
	)

	t.SetState(state, inState)

	return
}

// NewTestCurrencyState returns currency id, currency state
func (t *TestProcessor) NewTestCurrencyState(cid string, addr base.Address, inState bool) types.CurrencyID {
	if len(cid) < 3 {
		panic(cid)
	}

	currencyID := types.CurrencyID(cid)
	design := types.NewCurrencyDesign(common.ZeroBig, currencyID, common.NewBig(9), addr, types.NewCurrencyPolicy(common.ZeroBig, types.NewNilFeeer()))
	state := common.NewBaseState(base.Height(1), ccstate.DesignStateKey(currencyID), ccstate.NewCurrencyDesignStateValue(design), nil, []util.Hash{})

	t.SetState(state, inState)

	return currencyID
}

func (t *TestProcessor) NewTestSuffrageState(priv, node string, inState bool) (base.Address, base.Privatekey) {
	privateKey, _ := base.ParseMPrivatekey(priv)
	nodeAddr, _ := base.ParseStringAddress(node)
	n := isaac.NewNode(privateKey.Publickey(), nodeAddr)
	ns := isaac.NewSuffrageNodeStateValue(n, base.GenesisHeight)

	state := common.NewBaseState(base.Height(1), isaac.SuffrageStateKey, isaac.NewSuffrageNodesStateValue(base.GenesisHeight, []base.SuffrageNodeStateValue{ns}), nil, []util.Hash{})

	t.SetState(state, inState)

	return nodeAddr, privateKey
}

func (t *TestProcessor) SetCurrency(cid string, am int64, receiverPriv string, target *types.CurrencyID, instate bool) {
	receiverAddr, _, _ := t.NewTestAccountState(receiverPriv, instate)
	t.NewTestCurrencyState(cid, receiverAddr, instate)
	t.NewTestBalanceState(receiverAddr, types.CurrencyID(cid), am, instate)
	if len(cid) < 3 {
		panic(cid)
	}
	c := types.CurrencyID(cid)
	target = &c

	return
}

func (t *TestProcessor) SetAccount(priv string, amount int64, cid types.CurrencyID, target []Account, inState bool) {
	addr, keys, privateKey := t.NewTestAccountState(priv, inState)
	ac := Account{addr, privateKey, keys}
	UpdateSlice[Account](ac, target)
	t.NewTestBalanceState(addr, cid, amount, inState)

	return
}

func (t *TestProcessor) SetContractAccount(owner base.Address, priv string, amount int64, cid types.CurrencyID, target []Account, inState bool) {
	addr, privateKey := t.NewTestContractAccountState(owner, priv, inState)
	ac := Account{addr, privateKey, nil}
	UpdateSlice[Account](ac, target)
	t.NewTestBalanceState(addr, cid, amount, inState)

	return
}

func (t *TestProcessor) SetAmount(am int64, cid types.CurrencyID, target []types.Amount) {
	a := types.NewAmount(common.NewBig(am), cid)
	UpdateSlice[types.Amount](a, target)

	return
}

func (t *TestProcessor) SetCurrency2(
	cid string, am int64, addr base.Address, target []types.CurrencyID, instate bool) {
	t.NewTestCurrencyState(cid, addr, instate)
	t.NewTestBalanceState(addr, types.CurrencyID(cid), am, instate)
	c := types.CurrencyID(cid)

	UpdateSlice[types.CurrencyID](c, target)

	return
}

func (t *TestProcessor) LoadOperation(fileName string) base.Operation {
	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var v json.RawMessage
	var op base.Operation
	var ok bool
	enc := t.Encoders.JSON()
	if err = json.Unmarshal(bytes, &v); err != nil {
		panic(err)
	} else if hinter, err := enc.Decode(bytes); err != nil {
		panic(err)
	} else if op, ok = hinter.(base.Operation); !ok {
		panic("decoded object is not Operation")
	}

	return op
}

func (t *TestProcessor) Decode(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var v json.RawMessage
	enc := t.Encoders.JSON()
	if err = json.Unmarshal(bytes, &v); err != nil {
		panic(err)
	}

	_, err = enc.Decode(bytes)
	t.err = err

	return
}

func (t *TestProcessor) Print(fileName string, i interface{}) {
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var b []byte
	enc := t.Encoders.JSON()
	b, err = enc.Marshal(i)
	if err != nil {
		panic(err)
	}

	_, _ = fmt.Fprintf(file, string(b))

	return
}

func (t *TestProcessor) Error() error {
	return t.err
}

func UpdateSlice[T any](a T, target []T) {
	var n []T
	copy(n, target)
	n = append(n, a)
	copy(target, n)
}

type MockStateGetter struct {
	store map[string]base.State
	mu    sync.RWMutex
}

func NewMockStateGetter() *MockStateGetter {
	return &MockStateGetter{store: make(map[string]base.State)}
}

func (m *MockStateGetter) Get(key string) (base.State, bool, error) {
	v, found := m.store[key]
	if !found {
		return nil, found, nil
	}

	return v, found, nil
}

func (m *MockStateGetter) Set(key string, st base.State) {
	m.store[key] = st
}
