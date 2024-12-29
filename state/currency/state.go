package currency

import (
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
	"strings"
)

var (
	AccountStateValueHint = hint.MustNewHint("account-state-value-v0.0.1")
	BalanceStateValueHint = hint.MustNewHint("balance-state-value-v0.0.1")
	DesignStateValueHint  = hint.MustNewHint("currency-design-state-value-v0.0.1")
)

var (
	AccountStateKeySuffix = ":account"
	BalanceStateKeySuffix = ":balance"
	DesignStateKeyPrefix  = "currencydesign:"
)

type AccountStateValue struct {
	hint.BaseHinter
	Account types.Account
}

func NewAccountStateValue(account types.Account) AccountStateValue {
	return AccountStateValue{
		BaseHinter: hint.NewBaseHinter(AccountStateValueHint),
		Account:    account,
	}
}

func (a AccountStateValue) Hint() hint.Hint {
	return a.BaseHinter.Hint()
}

func (a AccountStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("Invalid AccountStateValue")

	if err := a.BaseHinter.IsValid(AccountStateValueHint.Type().Bytes()); err != nil {
		return e.Wrap(err)
	}

	if err := util.CheckIsValiders(nil, false, a.Account); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (a AccountStateValue) HashBytes() []byte {
	return a.Account.Bytes()
}

func GetAccountKeysFromState(st base.State) (types.AccountKeys, error) {
	ac, err := LoadAccountStateValue(st)
	if err != nil {
		return nil, err
	}
	return ac.Keys(), nil
}

type BalanceStateValue struct {
	hint.BaseHinter
	Amount types.Amount
}

func NewBalanceStateValue(amount types.Amount) BalanceStateValue {
	return BalanceStateValue{
		BaseHinter: hint.NewBaseHinter(BalanceStateValueHint),
		Amount:     amount,
	}
}

func (b BalanceStateValue) Hint() hint.Hint {
	return b.BaseHinter.Hint()
}

func (b BalanceStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("Invalid BalanceStateValue")

	if err := b.BaseHinter.IsValid(BalanceStateValueHint.Type().Bytes()); err != nil {
		return e.Wrap(err)
	}

	if err := util.CheckIsValiders(nil, false, b.Amount); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (b BalanceStateValue) HashBytes() []byte {
	return b.Amount.Bytes()
}

func StateBalanceValue(st base.State) (types.Amount, error) {
	v := st.Value()
	if v == nil {
		return types.Amount{}, util.ErrNotFound.Errorf("balance not found in State")
	}

	a, ok := v.(BalanceStateValue)
	if !ok {
		return types.Amount{}, errors.Errorf("invalid balance value found, %T", v)
	}

	return a.Amount, nil
}

type AddBalanceStateValue struct {
	Amount types.Amount
}

func NewAddBalanceStateValue(amount types.Amount) AddBalanceStateValue {
	return AddBalanceStateValue{
		Amount: amount,
	}
}

func (b AddBalanceStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("Invalid AddBalanceStateValue")

	if err := util.CheckIsValiders(nil, false, b.Amount); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (b AddBalanceStateValue) HashBytes() []byte {
	return b.Amount.Bytes()
}

type DeductBalanceStateValue struct {
	Amount types.Amount
}

func NewDeductBalanceStateValue(amount types.Amount) DeductBalanceStateValue {
	return DeductBalanceStateValue{
		Amount: amount,
	}
}

func (b DeductBalanceStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("Invalid DeductBalanceStateValue")

	if err := util.CheckIsValiders(nil, false, b.Amount); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (b DeductBalanceStateValue) HashBytes() []byte {
	return b.Amount.Bytes()
}

type DesignStateValue struct {
	hint.BaseHinter
	Design types.CurrencyDesign
}

func NewCurrencyDesignStateValue(currencyDesign types.CurrencyDesign) DesignStateValue {
	return DesignStateValue{
		BaseHinter: hint.NewBaseHinter(DesignStateValueHint),
		Design:     currencyDesign,
	}
}

func (c DesignStateValue) Hint() hint.Hint {
	return c.BaseHinter.Hint()
}

func (c DesignStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("Invalid DesignStateValue")

	if err := c.BaseHinter.IsValid(DesignStateValueHint.Type().Bytes()); err != nil {
		return e.Wrap(err)
	}

	if err := util.CheckIsValiders(nil, false, c.Design); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (c DesignStateValue) HashBytes() []byte {
	return c.Design.Bytes()
}

func GetDesignFromState(st base.State) (types.CurrencyDesign, error) {
	v := st.Value()
	if v == nil {
		return types.CurrencyDesign{}, errors.Errorf("state value is nil")
	}

	de, ok := v.(DesignStateValue)
	if !ok {
		return types.CurrencyDesign{}, errors.Errorf("expected DesignStateValue, but %T", v)
	}

	return de.Design, nil
}

func BalanceStateKeyPrefix(a base.Address, cid types.CurrencyID) string {
	return fmt.Sprintf("%s-%s", a.String(), cid)
}

func AccountStateKey(a base.Address) string {
	return fmt.Sprintf("%s%s", a.String(), AccountStateKeySuffix)
}

func IsAccountStateKey(key string) bool {
	return strings.HasSuffix(key, AccountStateKeySuffix)
}

func LoadAccountStateValue(st base.State) (*types.Account, error) {
	v := st.Value()
	if v == nil {
		return nil, util.ErrNotFound.Errorf("state value is nil")
	}

	s, ok := v.(AccountStateValue)
	if !ok {
		return nil, errors.Errorf("expected %T, but %T", AccountStateValue{}, v)
	}
	return &(s.Account), nil

}

func BalanceStateKey(a base.Address, cid types.CurrencyID) string {
	return fmt.Sprintf("%s%s", BalanceStateKeyPrefix(a, cid), BalanceStateKeySuffix)
}

func IsBalanceStateKey(key string) bool {
	return strings.HasSuffix(key, BalanceStateKeySuffix)
}

func IsDesignStateKey(key string) bool {
	return strings.HasPrefix(key, DesignStateKeyPrefix)
}

func DesignStateKey(cid types.CurrencyID) string {
	return fmt.Sprintf("%s%s", DesignStateKeyPrefix, cid)
}
