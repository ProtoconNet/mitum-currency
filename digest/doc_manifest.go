package digest

import (
	"time"

	mongodbst "github.com/ProtoconNet/mitum-currency/v3/digest/mongodb"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util/encoder"
)

type ManifestDoc struct {
	mongodbst.BaseDoc
	va          base.Manifest
	operations  []base.Operation
	height      base.Height
	confirmedAt time.Time
	proposer    base.Address
	round       base.Round
	gitInfo     string
}

func NewManifestDoc(
	manifest base.Manifest,
	enc encoder.Encoder,
	height base.Height,
	operations []base.Operation,
	confirmedAt time.Time,
	proposer base.Address,
	round base.Round,
	gitInfo string,
) (ManifestDoc, error) {
	b, err := mongodbst.NewBaseDoc(nil, manifest, enc)
	if err != nil {
		return ManifestDoc{}, err
	}

	return ManifestDoc{
		BaseDoc:     b,
		va:          manifest,
		operations:  operations,
		height:      height,
		confirmedAt: confirmedAt,
		proposer:    proposer,
		round:       round,
		gitInfo:     gitInfo,
	}, nil
}

func (doc ManifestDoc) MarshalBSON() ([]byte, error) {
	m, err := doc.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["block"] = doc.va.Hash()
	m["operations"] = len(doc.operations)
	m["height"] = doc.height
	m["confirmed_at"] = doc.confirmedAt.String()
	m["proposer"] = doc.proposer.String()
	m["round"] = doc.round.Uint64()
	m["buildInfo"] = doc.gitInfo

	return bsonenc.Marshal(m)
}
