package plugin

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/pkg/errors"
)

type GetAccountFunc func(string) (*types.Account, error)

func NewGetAccountFunc(encs encoder.Encoders, getStateFunc base.GetStateFunc) GetAccountFunc {
	return func(addr string) (*types.Account, error) {
		return GetAccount(addr, encs, getStateFunc)
	}
}

func GetAccount(addr string, encs encoder.Encoders, getStateFunc base.GetStateFunc) (*types.Account, error) {
	address, err := base.DecodeAddress(addr, encs.JSON())
	if err != nil {
		return nil, common.ErrValueInvalid.Wrap(errors.Errorf("failed to decode address, %v", addr))
	}
	var account *types.Account
	var st base.State
	var found bool
	k := currency.AccountStateKey(address)
	switch st, found, err = getStateFunc(k); {
	case err != nil:
		return nil, common.ErrStateValInvalid.Wrap(errors.Errorf("account, %v: %v", addr, err))
	case !found:
		return nil, common.ErrAccountNF.Wrap(errors.Errorf("account, %v", addr))
	default:
		account, err = currency.LoadAccountStateValue(st)
		if err != nil {
			return nil, common.ErrStateValInvalid.Wrap(errors.Errorf("account, %v: %v", addr, err))
		}
	}
	return account, nil
}
