package digest

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var IndexPrefix = "mitum_digest_"

var BlockIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_block_height"),
	},
}

var AccountIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "address", Value: 1}, bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_account"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_account_height"),
	},
	{
		Keys: bson.D{bson.E{Key: "pubs", Value: 1}, bson.E{Key: "height", Value: 1}, bson.E{Key: "address", Value: 1}},
		Options: options.Index().
			SetName("mitum_digest_account_publiskeys"),
	},
}

var BalanceIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "address", Value: 1}, bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_balance"),
	},
	{
		Keys: bson.D{
			bson.E{Key: "address", Value: 1},
			bson.E{Key: "currency", Value: 1},
			bson.E{Key: "height", Value: -1},
		},
		Options: options.Index().
			SetName("mitum_digest_balance_currency"),
	},
	//{
	//	Keys: bson.D{bson.E{Key: "height", Value: -1}},
	//	Options: options.Index().
	//		SetName("mitum_digest_balance_height"),
	//},
}

var OperationIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "addresses", Value: 1}, bson.E{Key: "height", Value: 1}, bson.E{Key: "index", Value: 1}},
		Options: options.Index().
			SetName("mitum_digest_account_operation"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}, bson.E{Key: "index", Value: 1}},
		Options: options.Index().
			SetName("mitum_digest_operation"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("mitum_digest_operation_height"),
	},
}

var DidRegistryIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{
			bson.E{Key: "contract", Value: 1},
			bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName(IndexPrefix + "did_registry_contract_height"),
	},
}

var DidRegistryDataIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{
			bson.E{Key: "contract", Value: 1},
			bson.E{Key: "method_specific_id", Value: 1},
			bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName(IndexPrefix + "did_registry_data_contract_publicKey_height"),
	},
}

var DidRegistryDocumentIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{
			bson.E{Key: "contract", Value: 1},
			bson.E{Key: "did", Value: 1},
			bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName(IndexPrefix + "did_registry_document_contract_did_height"),
	},
}

var DefaultIndexes = map[string] /* collection */ []mongo.IndexModel{
	DefaultColNameBlock:       BlockIndexModels,
	DefaultColNameAccount:     AccountIndexModels,
	DefaultColNameBalance:     BalanceIndexModels,
	DefaultColNameOperation:   OperationIndexModels,
	DefaultColNameDIDRegistry: DidRegistryIndexModels,
	DefaultColNameDIDData:     DidRegistryDataIndexModels,
	DefaultColNameDIDDocument: DidRegistryDocumentIndexModels,
}
