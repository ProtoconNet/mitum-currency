package did_registry

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact CreateDIDFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":           fact.Hint().String(),
			"hash":            fact.BaseFact.Hash().String(),
			"token":           fact.BaseFact.Token(),
			"sender":          fact.sender,
			"contract":        fact.contract,
			"authType":        fact.authType,
			"publicKey":       fact.publicKey.String(),
			"serviceType":     fact.serviceType,
			"serviceEndpoint": fact.serviceEndpoint,
			"currency":        fact.currency,
		},
	)
}

type CreateDIDFactBSONUnmarshaler struct {
	Hint            string `bson:"_hint"`
	Sender          string `bson:"sender"`
	Contract        string `bson:"contract"`
	AuthType        string `bson:"authType"`
	PublicKey       string `bson:"publicKey"`
	ServiceType     string `bson:"serviceType"`
	ServiceEndpoint string `bson:"serviceEndpoints"`
	Currency        string `bson:"currency"`
}

func (fact *CreateDIDFact) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var u common.BaseFactBSONUnmarshaler

	err := enc.Unmarshal(b, &u)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *fact)
	}

	fact.BaseFact.SetHash(valuehash.NewBytesFromString(u.Hash))
	fact.BaseFact.SetToken(u.Token)

	var uf CreateDIDFactBSONUnmarshaler
	if err := bson.Unmarshal(b, &uf); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *fact)
	}

	ht, err := hint.ParseHint(uf.Hint)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *fact)
	}
	fact.BaseHinter = hint.NewBaseHinter(ht)

	pubKey, err := base.DecodePublickeyFromString(uf.PublicKey, enc)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeJson, *fact)
	}

	if err := fact.unpack(enc, uf.Sender, uf.Contract, uf.AuthType, pubKey, uf.ServiceType, uf.ServiceEndpoint, uf.Currency); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *fact)
	}

	return nil
}

func (op CreateDID) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint": op.Hint().String(),
			"hash":  op.Hash().String(),
			"fact":  op.Fact(),
			"signs": op.Signs(),
		})
}

func (op *CreateDID) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo common.BaseOperation
	if err := ubo.DecodeBSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *op)
	}

	op.BaseOperation = ubo

	var ueo extras.BaseOperationExtensions
	if err := ueo.DecodeBSON(b, enc); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *op)
	}

	op.BaseOperationExtensions = &ueo

	return nil
}
