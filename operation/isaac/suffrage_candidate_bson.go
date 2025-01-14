package isaacoperation

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact SuffrageCandidateFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":     fact.Hint().String(),
			"address":   fact.address.String(),
			"publickey": fact.publickey.String(),
			"hash":      fact.BaseFact.Hash().String(),
			"token":     fact.BaseFact.Token(),
		},
	)
}

type SuffrageCandidateFactBSONUnMarshaler struct {
	Hint      string `bson:"_hint"`
	Address   string `bson:"address"`
	Publickey string `bson:"publickey"`
}

func (fact *SuffrageCandidateFact) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of SuffrageCandidateFact")

	var u common.BaseFactBSONUnmarshaler

	err := enc.Unmarshal(b, &u)
	if err != nil {
		return e.Wrap(err)
	}

	h := valuehash.NewBytesFromString(u.Hash)

	fact.BaseFact.SetHash(h)
	fact.BaseFact.SetToken(u.Token)

	var uf SuffrageCandidateFactBSONUnMarshaler
	if err := bson.Unmarshal(b, &uf); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(uf.Hint)
	if err != nil {
		return e.Wrap(err)
	}
	fact.BaseHinter = hint.NewBaseHinter(ht)

	return fact.unpack(enc, uf.Address, uf.Publickey)
}

func (op SuffrageCandidate) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(op.BaseOperation)
}

func (op *SuffrageCandidate) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of SuffrageCandidate")
	var ubo common.BaseNodeOperation

	err := ubo.DecodeBSON(b, enc)
	if err != nil {
		return e.Wrap(err)
	}

	op.BaseNodeOperation = ubo

	return nil
}
