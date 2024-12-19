package digest

import (
	mongodbst "github.com/ProtoconNet/mitum-currency/v3/digest/mongodb"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum-currency/v3/state/currency"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/pkg/errors"
)

type CurrencyDoc struct {
	mongodbst.BaseDoc
	st base.State
	cd types.CurrencyDesign
}

func NewCurrencyDoc(st base.State, enc encoder.Encoder) (CurrencyDoc, error) {
	cd, err := currency.GetDesignFromState(st)
	if err != nil {
		return CurrencyDoc{}, errors.Wrap(err, "CurrencyDoc needs CurrencyDesign state")
	}

	b, err := mongodbst.NewBaseDoc(nil, st, enc)
	if err != nil {
		return CurrencyDoc{}, err
	}

	return CurrencyDoc{
		BaseDoc: b,
		st:      st,
		cd:      cd,
	}, nil
}

func (doc CurrencyDoc) MarshalBSON() ([]byte, error) {
	m, err := doc.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["currency"] = doc.cd.Currency().String()
	m["height"] = doc.st.Height()

	return bsonenc.Marshal(m)
}
