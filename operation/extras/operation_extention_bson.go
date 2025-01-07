package extras

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

func (ba BaseAuthentication) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":             ba.Hint().String(),
			"contract":          ba.contract.String(),
			"authentication_id": ba.authenticationID,
			"proof_data":        ba.proofData,
		},
	)
}

type BaseAuthenticationBSONUnmarshaler struct {
	Hint             string `bson:"_hint"`
	Contract         string `bson:"contract"`
	AuthenticationID string `bson:"authentication_id"`
	ProofData        string `bson:"proof_data"`
}

func (ba *BaseAuthentication) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	if len(b) < 1 {
		ba.contract = nil
		ba.authenticationID = ""
		ba.proofData = ""

		return nil
	}
	var u BaseAuthenticationBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *ba)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *ba)
	}

	ba.BaseHinter = hint.NewBaseHinter(ht)

	a, err := base.DecodeAddress(u.Contract, enc)
	if err != nil {
		if err != nil {
			return common.DecorateError(err, common.ErrDecodeBson, *ba)
		}
	}

	ba.contract = a
	ba.authenticationID = u.AuthenticationID
	ba.proofData = u.ProofData

	return nil
}

func (bs BaseSettlement) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"_hint":     bs.Hint().String(),
			"op_sender": bs.opSender.String(),
		},
	)
}

type BaseSettlementBSONUnmarshaler struct {
	Hint     string `bson:"_hint"`
	OpSender string `bson:"op_sender"`
}

func (bs *BaseSettlement) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	if len(b) < 1 {
		bs.opSender = nil

		return nil
	}
	var u BaseSettlementBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *bs)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *bs)
	}

	bs.BaseHinter = hint.NewBaseHinter(ht)

	a, err := base.DecodeAddress(u.OpSender, enc)
	if err != nil {
		if err != nil {
			return common.DecorateError(err, common.ErrDecodeBson, *bs)
		}
	}
	bs.opSender = a

	return nil
}

func (bs BaseProxyPayer) MarshalBSON() ([]byte, error) {
	if bs.proxyPayer == nil {
		return bsonenc.Marshal(
			bson.M{
				"_hint":       bs.Hint().String(),
				"proxy_payer": "",
			},
		)
	}
	return bsonenc.Marshal(
		bson.M{
			"_hint":       bs.Hint().String(),
			"proxy_payer": bs.proxyPayer.String(),
		},
	)
}

type BaseProxyPayerBSONUnmarshaler struct {
	Hint       string `bson:"_hint"`
	ProxyPayer string `bson:"proxy_payer"`
}

func (bs *BaseProxyPayer) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	if len(b) < 1 {
		bs.proxyPayer = nil

		return nil
	}
	var u BaseProxyPayerBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *bs)
	}

	ht, err := hint.ParseHint(u.Hint)
	if err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *bs)
	}

	bs.BaseHinter = hint.NewBaseHinter(ht)

	a, err := base.DecodeAddress(u.ProxyPayer, enc)
	if err != nil {
		if err != nil {
			return common.DecorateError(err, common.ErrDecodeBson, *bs)
		}
	}
	bs.proxyPayer = a

	return nil
}

func (be BaseOperationExtensions) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bson.M{
			"extension": be.extension,
		},
	)
}

type BaseOperationExtensionsBSONUnmarshaler struct {
	Extension bson.Raw `bson:"extension"`
}

func (be *BaseOperationExtensions) DecodeBSON(b []byte, enc *bsonenc.Encoder) error {
	var u BaseOperationExtensionsBSONUnmarshaler

	if err := enc.Unmarshal(b, &u); err != nil {
		return common.DecorateError(err, common.ErrDecodeBson, *be)
	}

	extensions := make(map[string]OperationExtension)
	m, err := enc.DecodeMap(u.Extension)
	if err != nil {
		return err
	}

	for k, v := range m {
		extension, ok := v.(OperationExtension)
		if !ok {
			return errors.Errorf("expected OperationExtension, not %T", v)
		}
		extensions[k] = extension
	}

	be.extension = extensions

	return nil
}
