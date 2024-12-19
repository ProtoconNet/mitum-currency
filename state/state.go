package state

import (
	"strings"

	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/pkg/errors"
)

type StateValueMerger struct {
	*common.BaseStateValueMerger
}

func NewStateValueMerger(height base.Height, key string, st base.State) *StateValueMerger {
	s := &StateValueMerger{
		BaseStateValueMerger: common.NewBaseStateValueMerger(height, key, st),
	}

	return s
}

func NewStateMergeValue(key string, stv base.StateValue) base.StateMergeValue {
	StateValueMergerFunc := func(height base.Height, st base.State) base.StateValueMerger {
		nst := st
		if st == nil {
			nst = common.NewBaseState(base.NilHeight, key, nil, nil, nil)
		}
		return NewStateValueMerger(height, nst.Key(), nst)
	}

	return common.NewBaseStateMergeValue(
		key,
		stv,
		StateValueMergerFunc,
	)
}

// CheckNotExistsState returns found, error. Using CheckNotExistsState, check found first and check err for reason.
func CheckNotExistsState(
	key string,
	getState base.GetStateFunc,
) (found bool, err error) {
	switch _, found, err = getState(key); {
	case found:
		return found, base.NewBaseOperationProcessReasonError("State, %v already exists", key)
	default:
		return found, err
	}
}

// CheckExistsState returns found, error. Using CheckExistsState, check found first and check err for reason.
func CheckExistsState(
	key string,
	getState base.GetStateFunc,
) error {
	switch _, found, err := getState(key); {
	case !found:
		return base.NewBaseOperationProcessReasonError("State, %v does not exist", key)
	default:
		return err
	}
}

func ExistsState(
	k,
	name string,
	getState base.GetStateFunc,
) (base.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case !found:
		return nil, errors.Errorf("%v does not exist", name)
	default:
		return st, nil
	}
}

func NotExistsState(
	k,
	name string,
	getState base.GetStateFunc,
) (base.State, error) {
	var st base.State
	switch _, found, err := getState(k); {
	case err != nil:
		return nil, err
	case found:
		return nil, errors.Errorf("%v already exists", name)
	case !found:
		st = common.NewBaseState(base.NilHeight, k, nil, nil, nil)
	}
	return st, nil
}

func ExistsCurrencyPolicy(cid types.CurrencyID, getStateFunc base.GetStateFunc) (*types.CurrencyPolicy, error) {
	var policy types.CurrencyPolicy
	switch st, found, err := getStateFunc(currency.DesignStateKey(cid)); {
	case err != nil:
		return nil, err
	case !found:
		return nil, common.ErrCurrencyNF.Wrap(errors.Errorf("currency id, %v", cid))
	default:
		cd, ok := st.Value().(currency.DesignStateValue)
		if !ok {
			return nil, common.ErrTypeMismatch.Wrap(errors.Errorf("expected CurrencyDesignStateValue, not %T", st.Value()))

		}
		policy = cd.Design.Policy()
	}
	return &policy, nil
}

func ExistsAccount(addr base.Address, name string, isExist bool, getStateFunc base.GetStateFunc) (base.State, error) {
	var st base.State
	var found bool
	var err error
	k := currency.AccountStateKey(addr)
	switch st, found, err = getStateFunc(k); {
	case err != nil:
		return st, common.ErrStateValInvalid.Wrap(errors.Errorf("%s account, %v: %v", name, addr, err))
	case !found:
		if isExist {
			return st, common.ErrAccountNF.Wrap(errors.Errorf("%s account, %v", name, addr))
		} else {
			return common.NewBaseState(base.NilHeight, k, nil, nil, nil), nil
		}
	default:
		if !isExist {
			return st, common.ErrAccountE.Wrap(errors.Errorf("%s account, %v", name, addr))
		}
		//account, err = currency.LoadAccountStateValue(st)
		//if err != nil {
		//	return st, common.ErrStateValInvalid.Wrap(errors.Errorf("%s account, %v: %v", name, addr, err))
		//}
	}
	return st, nil
}

func ExistsCAccount(addr base.Address, name string, isExist, isContract bool, getStateFunc base.GetStateFunc) (
	accountState, caccountState base.State, accountErr, caccountErr error) {
	var accountFound, caccountFound bool
	ak := currency.AccountStateKey(addr)
	cak := extension.StateKeyContractAccount(addr)
	accountState, accountFound, accountErr = getStateFunc(ak)
	caccountState, caccountFound, caccountErr = getStateFunc(cak)

	switch {
	case accountErr != nil:
		accountErr = common.ErrStateValInvalid.Wrap(errors.Errorf("%s account, %v: %v", name, addr, accountErr))
		return accountState, caccountState, accountErr, caccountErr
	case !accountFound:
		if isExist {
			accountErr = common.ErrAccountNF.Wrap(errors.Errorf("%s account, %v", name, addr))
			return accountState, caccountState, accountErr, caccountErr
		} else {
			accountState = common.NewBaseState(base.NilHeight, ak, nil, nil, nil)
		}
	case accountFound:
		if !isExist {
			accountErr = common.ErrAccountE.Wrap(errors.Errorf("%s account, %v", name, addr))
			return accountState, caccountState, accountErr, caccountErr
		}
	}
	switch {
	case caccountErr != nil:
		caccountErr = common.ErrStateValInvalid.Wrap(errors.Errorf("%s account, %v: %v", name, addr, caccountErr))
		return accountState, caccountState, accountErr, caccountErr
	case !caccountFound:
		if isContract {
			caccountErr = common.ErrCAccountNF.Wrap(errors.Errorf("%s account, %v", name, addr))
			return accountState, caccountState, accountErr, caccountErr
		} else {
			caccountState = common.NewBaseState(base.NilHeight, cak, nil, nil, nil)
		}
	default:
		if !isContract {
			caccountErr = common.ErrCAccountE.Wrap(errors.Errorf("%s account, %v", name, addr))
			return accountState, caccountState, accountErr, caccountErr
		}
	}

	return accountState, caccountState, accountErr, caccountErr
}

func CheckFactSignsByState(
	address base.Address,
	fs []base.Sign,
	getState base.GetStateFunc,
) error {
	st, err := ExistsState(currency.AccountStateKey(address), "signer account", getState)
	if err != nil {
		return common.ErrAccountNF.Wrap(err)
	}
	keys, err := currency.GetAccountKeysFromState(st)
	switch {
	case err != nil:
		return common.ErrStateValInvalid.Wrap(errors.Errorf("signer account; %v", err))
	case keys == nil:
		return common.ErrStateValInvalid.Wrap(errors.Errorf("empty keys found"))
	}

	if err := types.CheckThreshold(fs, keys); err != nil {
		return common.ErrSignInvalid.Wrap(errors.Errorf("threshold; %v", err))
	}

	return nil
}

func CreateNotExistAccount(address base.Address, getStateFunc base.GetStateFunc) (base.StateMergeValue, error) {
	var smv base.StateMergeValue
	k := currency.AccountStateKey(address)
	switch _, found, err := getStateFunc(k); {
	case err != nil:
		return nil, errors.Errorf("failed to get state: %v", err)
	case !found:
		nilKys, err := types.NewNilAccountKeysFromAddress(address)
		if err != nil {
			return nil, errors.Errorf(
				"failed to get single sig account key from address %v: %v", address, err)
		}
		acc, err := types.NewAccount(address, nilKys)
		if err != nil {
			return nil, errors.Errorf(
				"failed to get account from address and keys, %v: %v", address, err)
		}
		smv = NewStateMergeValue(k, currency.NewAccountStateValue(acc))

		return smv, nil
	default:
	}
	return nil, nil
}

func ParseStateKey(key string, Prefix string, expected int) ([]string, error) {
	parsedKey := strings.Split(key, ":")
	if parsedKey[0] != Prefix {
		return nil, errors.Errorf("State Key not include Prefix, %s", parsedKey)
	}
	if len(parsedKey) < expected {
		return nil, errors.Errorf("Parsed State Key length under %v", expected)
	} else {
		return parsedKey, nil
	}
}

func ParseStateKeyBySuffix(key string, suffix string, expected int) ([]string, error) {
	parsedKey := strings.Split(key, ":")
	if parsedKey[len(parsedKey)-1] != suffix {
		return nil, errors.Errorf("State Key, %v not include Suffix, %s", parsedKey, suffix)
	}
	if len(parsedKey) < expected {
		return nil, errors.Errorf("Parsed State Key length under %v", expected)
	} else {
		return parsedKey, nil
	}
}
