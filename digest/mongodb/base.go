package mongodbstorage

import (
	"context"
	"sync"
	"time"

	dutil "github.com/ProtoconNet/mitum-currency/v3/digest/util"
	bsonenc "github.com/ProtoconNet/mitum-currency/v3/digest/util/bson"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/bluele/gcache"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	ColNameInfo         = "info"
	ColNameManifest     = "manifest"
	ColNameBlockdataMap = "blockdata_map"
)

var allCollections = []string{
	ColNameInfo,
	ColNameManifest,
	ColNameBlockdataMap,
}

type Database struct {
	sync.RWMutex
	*logging.Logging
	client              *Client
	encs                *encoder.Encoders
	enc                 encoder.Encoder
	lastManifest        base.Manifest
	lastManifestHeight  base.Height
	readonly            bool
	cache               gcache.Cache
	lastINITVoteproof   base.Voteproof
	lastACCEPTVoteproof base.Voteproof
}

func NewDatabase(client *Client, encs *encoder.Encoders, enc encoder.Encoder) (*Database, error) {
	// NOTE call Initialize() later.
	if enc == nil {
		e, found := encs.Find(bsonenc.BSONEncoderHint)
		if !found {
			return nil, util.ErrNotFound.Errorf("Unknown encoder hint, %q", bsonenc.BSONEncoderHint)
		} else {
			enc = e
		}
	}

	return &Database{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "mongodb-database")
		}),
		client:             client,
		encs:               encs,
		enc:                enc,
		lastManifestHeight: base.NilHeight,
	}, nil
}

func NewDatabaseFromURI(uri string, encs *encoder.Encoders) (*Database, error) {
	parsed, err := dutil.ParseURL(uri, false)
	if err != nil {
		return nil, errors.Wrap(err, "invalid storge uri")
	}

	connectTimeout := time.Second * 7
	execTimeout := time.Second * 7
	{
		query := parsed.Query()
		if d, err := parseDurationFromQuery(query, "connectTimeout", connectTimeout); err != nil {
			return nil, err
		} else {
			connectTimeout = d
		}
		if d, err := parseDurationFromQuery(query, "execTimeout", execTimeout); err != nil {
			return nil, err
		} else {
			execTimeout = d
		}
	}

	var be encoder.Encoder
	if e, found := encs.Find(bsonenc.BSONEncoderHint); !found { // NOTE get latest bson encoder
		return nil, util.ErrNotFound.Errorf("Unknown encoder hint, %q", bsonenc.BSONEncoderHint)
	} else {
		be = e
	}

	if client, err := NewClient(uri, connectTimeout, execTimeout); err != nil {
		return nil, err
	} else if st, err := NewDatabase(client, encs, be); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}

func (st *Database) Initialize() error {
	if st.readonly {
		st.lastManifestHeight = base.Height(int(^uint(0) >> 1))

		return nil
	}

	// if err := st.loadLastBlock(); err != nil && !errors.Is(err, util.ErrNotFound) {
	// 	return err
	// }
	/*
		if err := st.cleanupIncompleteData(); err != nil {
			return err
		}
	*/

	return st.initialize()
}

func (st *Database) Client() *Client {
	return st.client
}

func (st *Database) Close() error {
	// FUTURE return st.client.Close()
	return nil
}

func (st *Database) Encoder() encoder.Encoder {
	return st.enc
}

func (st *Database) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *Database) initialize() error {
	if st.readonly {
		return errors.Errorf("Readonly mode")
	}

	for col, models := range defaultIndexes {
		if err := st.CreateIndex(col, models, IndexPrefix); err != nil {
			return err
		}
	}

	return nil
}

// Clean will drop the existing collections. To keep safe the another
// collections by user, drop collections instead of drop database.
func (st *Database) Clean() error {
	if st.readonly {
		return errors.Errorf("Readonly mode")
	}

	drop := func(c string) error {
		return st.client.Collection(c).Drop(context.Background())
	}

	for _, c := range allCollections {
		if err := drop(c); err != nil {
			return err
		}
	}

	if err := st.initialize(); err != nil {
		return err
	}

	st.Lock()
	defer st.Unlock()

	st.lastManifest = nil
	st.lastManifestHeight = base.NilHeight
	st.lastINITVoteproof = nil
	st.lastACCEPTVoteproof = nil

	return nil
}

func (st *Database) cleanByHeight(height base.Height) error {
	if st.readonly {
		return errors.Errorf("Readonly mode")
	}

	if height <= base.GenesisHeight {
		return st.Clean()
	}

	opts := options.BulkWrite().SetOrdered(true)
	removeByHeight := mongo.NewDeleteManyModel().SetFilter(bson.M{"height": bson.M{"$gte": height}})

	for _, col := range allCollections {
		res, err := st.client.Collection(col).BulkWrite(
			context.Background(),
			[]mongo.WriteModel{removeByHeight},
			opts,
		)
		if err != nil {
			return err
		}

		st.Log().Debug().Str("collection", col).Interface("result", res).Msg("clean collection by height")
	}

	return nil
}

func (st *Database) CreateIndex(col string, models []mongo.IndexModel, prefix string) error {
	if st.readonly {
		return errors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	iv := st.client.Collection(col).Indexes()

	cursor, err := iv.List(context.TODO())
	if err != nil {
		return err
	}

	var existings []string
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return err
	} else {
		for _, r := range results {
			name := r["name"].(string)
			if !isIndexName(name, prefix) {
				continue
			}

			existings = append(existings, name)
		}
	}

	if len(existings) > 0 {
		for _, name := range existings {
			if _, err := iv.DropOne(context.TODO(), name); err != nil {
				return err
			}
		}
	}

	if len(models) < 1 {
		return nil
	}

	if _, err := iv.CreateMany(context.TODO(), models); err != nil {
		return err
	}

	return nil
}

func (st *Database) New() (*Database, error) {
	var client *Client
	if cl, err := st.client.New(""); err != nil {
		return nil, err
	} else {
		client = cl
	}

	st.RLock()
	defer st.RUnlock()

	if nst, err := NewDatabase(client, st.encs, st.enc); err != nil {
		return nil, err
	} else {
		nst.lastManifest = st.lastManifest
		nst.lastManifestHeight = st.lastManifestHeight

		return nst, nil
	}
}

func (st *Database) SetInfo(key string, b []byte) error {
	if st.readonly {
		return errors.Errorf("readonly mode")
	}

	if doc, err := NewInfoDoc(key, b, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(ColNameInfo, doc); err != nil {
		return err
	} else {
		return nil
	}
}

func (st *Database) Info(key string) ([]byte, bool, error) {
	var b []byte
	if err := st.client.GetByID(ColNameInfo, infoDocKey(key),
		func(res *mongo.SingleResult) error {
			if i, err := loadInfo(res.Decode, st.encs); err != nil {
				return err
			} else {
				b = i
			}

			return nil
		},
	); err != nil {
		if errors.Is(err, util.ErrNotFound) || errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return b, b != nil, nil
}

func (st *Database) Readonly() (*Database, error) {
	if nst, err := st.New(); err != nil {
		return nil, err
	} else {
		nst.readonly = true

		return nst, nil
	}
}
