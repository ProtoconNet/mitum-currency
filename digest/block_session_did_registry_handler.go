package digest

import (
	dstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	"github.com/ProtoconNet/mitum2/base"
	"go.mongodb.org/mongo-driver/mongo"
)

func (bs *BlockSession) prepareDIDRegistry() error {
	if len(bs.sts) < 1 {
		return nil
	}

	var didRegistryModels []mongo.WriteModel
	var didDataModels []mongo.WriteModel
	var didDocumentModels []mongo.WriteModel
	for i := range bs.sts {
		st := bs.sts[i]
		switch {
		case dstate.IsDesignStateKey(st.Key()):
			j, err := bs.handleDIDRegistryDesignState(st)
			if err != nil {
				return err
			}
			didRegistryModels = append(didRegistryModels, j...)
		case dstate.IsDataStateKey(st.Key()):
			j, err := bs.handleDIDDataState(st)
			if err != nil {
				return err
			}
			didDataModels = append(didDataModels, j...)
		case dstate.IsDocumentStateKey(st.Key()):
			j, err := bs.handleDIDDocumentState(st)
			if err != nil {
				return err
			}
			didDocumentModels = append(didDocumentModels, j...)
		default:
			continue
		}
	}

	bs.didRegistryModels = didRegistryModels
	bs.didDataModels = didDataModels
	bs.didDocumentModels = didDocumentModels

	return nil
}

func (bs *BlockSession) handleDIDRegistryDesignState(st base.State) ([]mongo.WriteModel, error) {
	if DIDDesignDoc, err := NewDIDRegistryDesignDoc(st, bs.st.Encoder()); err != nil {
		return nil, err
	} else {
		return []mongo.WriteModel{
			mongo.NewInsertOneModel().SetDocument(DIDDesignDoc),
		}, nil
	}
}

func (bs *BlockSession) handleDIDDataState(st base.State) ([]mongo.WriteModel, error) {
	if DIDDataDoc, err := NewDIDDataDoc(st, bs.st.Encoder()); err != nil {
		return nil, err
	} else {
		return []mongo.WriteModel{
			mongo.NewInsertOneModel().SetDocument(DIDDataDoc),
		}, nil
	}
}

func (bs *BlockSession) handleDIDDocumentState(st base.State) ([]mongo.WriteModel, error) {
	if DIDDocumentDoc, err := NewDIDDocumentDoc(st, bs.st.Encoder()); err != nil {
		return nil, err
	} else {
		return []mongo.WriteModel{
			mongo.NewInsertOneModel().SetDocument(DIDDocumentDoc),
		}, nil
	}
}
