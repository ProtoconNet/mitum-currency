package plugin

import (
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"strings"

	types "github.com/ProtoconNet/mitum-currency/v3/types/plugin"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

var (
	DesignStateValueHint = hint.MustNewHint("mitum-plugin-design-state-value-v0.0.1")
	PluginStateKeyPrefix = "plugin"
	DesignStateKeySuffix = "design"
)

func PluginStateKey(addr base.Address) string {
	return fmt.Sprintf("%s:%s", PluginStateKeyPrefix, addr.String())
}

type DesignStateValue struct {
	hint.BaseHinter
	Design types.Design
}

func NewDesignStateValue(design types.Design) DesignStateValue {
	return DesignStateValue{
		BaseHinter: hint.NewBaseHinter(DesignStateValueHint),
		Design:     design,
	}
}

func (sv DesignStateValue) Hint() hint.Hint {
	return sv.BaseHinter.Hint()
}

func (sv DesignStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("invalid DesignStateValue")

	if err := sv.BaseHinter.IsValid(DesignStateValueHint.Type().Bytes()); err != nil {
		return e.Wrap(err)
	}

	if err := sv.Design.IsValid(nil); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (sv DesignStateValue) HashBytes() []byte {
	return sv.Design.Bytes()
}

func GetDesignFromState(st base.State) (types.Design, error) {
	v := st.Value()
	if v == nil {
		return types.Design{}, errors.Errorf("state value is nil")
	}

	d, ok := v.(DesignStateValue)
	if !ok {
		return types.Design{}, errors.Errorf("expected DesignStateValue but %T", v)
	}

	return d.Design, nil
}

func IsDesignStateKey(key string) bool {
	return strings.HasPrefix(key, PluginStateKeyPrefix) && strings.HasSuffix(key, DesignStateKeySuffix)
}

func DesignStateKey(addr base.Address) string {
	return fmt.Sprintf("%s:%s", PluginStateKey(addr), DesignStateKeySuffix)
}

var (
	DataStateValueHint = hint.MustNewHint("mitum-plugin-data-state-value-v0.0.1")
	DataStateKeySuffix = "data"
)

type DataStateValue struct {
	hint.BaseHinter
	Data []byte
}

func NewDataStateValue(data []byte) DataStateValue {
	return DataStateValue{
		BaseHinter: hint.NewBaseHinter(DataStateValueHint),
		Data:       data,
	}
}

func (sv DataStateValue) Hint() hint.Hint {
	return sv.BaseHinter.Hint()
}

func (sv DataStateValue) IsValid([]byte) error {
	e := util.ErrInvalid.Errorf("invalid DataStateValue")

	if err := sv.BaseHinter.IsValid(DataStateValueHint.Type().Bytes()); err != nil {
		return e.Wrap(err)
	}

	return nil
}

func (sv DataStateValue) HashBytes() []byte {
	return sv.Data
}

func GetDataFromState(st base.State) ([]byte, error) {
	v := st.Value()
	if v == nil {
		return nil, errors.Errorf("State value is nil")
	}

	ts, ok := v.(DataStateValue)
	if !ok {
		return nil, common.ErrTypeMismatch.Wrap(errors.Errorf("expected DataStateValue found, %T", v))
	}

	return ts.Data, nil
}

func IsDataStateKey(key string) bool {
	return strings.HasPrefix(key, PluginStateKeyPrefix) && strings.HasSuffix(key, DataStateKeySuffix)
}

func DataStateKey(addr base.Address, key string) string {
	return fmt.Sprintf("%s:%s:%s", PluginStateKey(addr), key, DataStateKeySuffix)
}
