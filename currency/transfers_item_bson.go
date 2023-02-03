package currency // nolint:dupl

import (
	bsonenc "github.com/spikeekips/mitum-currency/digest/util/bson"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson"
)

func (it BaseTransfersItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":    it.Hint().String(),
			"receiver": it.receiver,
			"amounts":  it.amounts,
		},
	)
}

type TransfersItemBSONUnmarshaler struct {
	HT string   `bson:"_hint"`
	RC string   `bson:"receiver"`
	AM bson.Raw `bson:"amounts"`
}

func (it *BaseTransfersItem) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringErrorFunc("failed to decode bson of BaseTransfersItem")

	var uit TransfersItemBSONUnmarshaler
	if err := bson.Unmarshal(b, &uit); err != nil {
		return e(err, "")
	}

	ht, err := hint.ParseHint(uit.HT)
	if err != nil {
		return e(err, "")
	}

	return it.unpack(enc, ht, uit.RC, uit.AM)
}
