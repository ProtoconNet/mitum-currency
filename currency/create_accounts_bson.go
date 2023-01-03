package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

func (fact CreateAccountsFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"sender": fact.sender,
				"items":  fact.items,
			},
			fact.BaseFact.BSONM(),
		))
}

type CreateAccountsFactBSONUnmarshaler struct {
	HT hint.Hint `bson:"_hint"`
	SD string    `bson:"sender"`
	IT bson.Raw  `bson:"items"`
}

func (fact *CreateAccountsFact) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringErrorFunc("failed to decode bson of CreateAccountsFact")

	var ubf base.BaseFact
	if err := ubf.DecodeBSON(b, enc); err != nil {
		return err
	}

	fact.BaseFact = ubf

	var uf CreateAccountsFactBSONUnmarshaler
	if err := bson.Unmarshal(b, &uf); err != nil {
		return e(err, "")
	}

	fact.BaseHinter = hint.NewBaseHinter(uf.HT)

	return fact.unpack(enc, uf.SD, uf.IT)
}

func (op *CreateAccounts) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.DecodeBSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
