package isaacoperation

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact SuffrageJoinFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":     fact.Hint().String(),
			"candidate": fact.candidate.String(),
			"start":     fact.start,
			"hash":      fact.BaseFact.Hash().String(),
			"token":     fact.BaseFact.Token(),
		},
	)
}

type SuffrageJoinFactBSONUnMarshaler struct {
	Hint      string      `bson:"_hint"`
	Candidate string      `bson:"candidate"`
	Start     base.Height `bson:"start"`
}

func (fact *SuffrageJoinFact) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of SuffrageJoinFact")

	var u common.BaseFactBSONUnmarshaler

	err := enc.Unmarshal(b, &u)
	if err != nil {
		return e.Wrap(err)
	}

	h := valuehash.NewBytesFromString(u.Hash)

	fact.BaseFact.SetHash(h)
	fact.BaseFact.SetToken(u.Token)

	var uf SuffrageJoinFactBSONUnMarshaler
	if err := bson.Unmarshal(b, &uf); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(uf.Hint)
	if err != nil {
		return e.Wrap(err)
	}
	fact.BaseHinter = hint.NewBaseHinter(ht)

	return fact.unpack(enc, uf.Candidate, uf.Start)
}

func (op *SuffrageJoin) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of SuffrageJoin")
	var ubo common.BaseNodeOperation

	err := ubo.DecodeBSON(b, enc)
	if err != nil {
		return e.Wrap(err)
	}

	op.BaseNodeOperation = ubo

	return nil
}

func (fact SuffrageGenesisJoinFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint": fact.Hint().String(),
			"nodes": fact.nodes,
			"hash":  fact.BaseFact.Hash().String(),
			"token": fact.BaseFact.Token(),
		},
	)
}

type SuffrageGenesisJoinFactBSONUnMarshaler struct {
	Hint  string   `bson:"_hint"`
	Nodes bson.Raw `bson:"nodes"`
}

func (fact *SuffrageGenesisJoinFact) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of SuffrageGenesisJoinFact")

	var u common.BaseFactBSONUnmarshaler

	err := enc.Unmarshal(b, &u)
	if err != nil {
		return e.Wrap(err)
	}

	h := valuehash.NewBytesFromString(u.Hash)

	fact.BaseFact.SetHash(h)
	fact.BaseFact.SetToken(u.Token)

	var uf SuffrageGenesisJoinFactBSONUnMarshaler
	if err := bson.Unmarshal(b, &uf); err != nil {
		return e.Wrap(err)
	}

	ht, err := hint.ParseHint(uf.Hint)
	if err != nil {
		return e.Wrap(err)
	}
	fact.BaseHinter = hint.NewBaseHinter(ht)

	r, err := uf.Nodes.Values()
	if err != nil {
		return err
	}

	nodes := make([]common.BaseNode, len(r))
	for i := range r {
		node := common.BaseNode{}
		if err := node.DecodeBSON(r[i].Value, enc); err != nil {
			return err
		}
		nodes[i] = node
	}

	return nil
}

func (op *SuffrageGenesisJoin) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	e := util.StringError("Decode bson of SuffrageGenesisJoin")
	var ubo common.BaseOperation

	err := ubo.DecodeBSON(b, enc)
	if err != nil {
		return e.Wrap(err)
	}

	op.BaseOperation = ubo

	return nil
}
