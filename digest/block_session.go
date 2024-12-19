package digest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ProtoconNet/mitum-currency/v3/digest/isaac"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	cestate "github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/fixedtree"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var bulkWriteLimit = 500

type WriteModelPrepareFunc func(BlockSession, base.State) ([]mongo.WriteModel, error)

type BlockSessioner interface {
	Prepare() error
	Commit(context.Context) error
	Close() error
}

type BlockSession struct {
	sync.RWMutex
	block                 base.BlockMap
	ops                   []base.Operation
	opsTree               fixedtree.Tree
	sts                   []base.State
	st                    *Database
	proposal              base.ProposalSignFact
	opsTreeNodes          map[string]base.OperationFixedtreeNode
	WriteModels           map[string][]mongo.WriteModel
	WriteModelsFunc       map[string][]mongo.WriteModel
	blockModels           []mongo.WriteModel
	operationModels       []mongo.WriteModel
	accountModels         []mongo.WriteModel
	contractAccountModels []mongo.WriteModel
	balanceModels         []mongo.WriteModel
	currencyModels        []mongo.WriteModel
	didRegistryModels     []mongo.WriteModel
	didDataModels         []mongo.WriteModel
	didDocumentModels     []mongo.WriteModel
	statesValue           *sync.Map
	balanceAddressList    []string
	buildinfo             string
}

func NewBlockSession(st *Database, blk base.BlockMap, ops []base.Operation, opsTree fixedtree.Tree, sts []base.State, proposal base.ProposalSignFact, vs string) (*BlockSession, error) {
	if st.Readonly() {
		return nil, errors.Errorf("Readonly mode")
	}

	nst, err := st.New()
	if err != nil {
		return nil, err
	}

	return &BlockSession{
		st:          nst,
		block:       blk,
		ops:         ops,
		opsTree:     opsTree,
		sts:         sts,
		proposal:    proposal,
		statesValue: &sync.Map{},
		buildinfo:   vs,
	}, nil
}

func (bs *BlockSession) Prepare() error {
	bs.Lock()
	defer bs.Unlock()

	if err := bs.prepareOperationsTree(); err != nil {
		return err
	}

	if err := bs.prepareBlock(); err != nil {
		return err
	}

	if err := bs.prepareOperations(); err != nil {
		return err
	}

	if err := bs.prepareCurrencies(); err != nil {
		return err
	}

	if err := bs.prepareDIDRegistry(); err != nil {
		return err
	}

	return bs.prepareAccounts()
}

func (bs *BlockSession) Commit(ctx context.Context) error {
	bs.Lock()
	defer bs.Unlock()

	started := time.Now()
	defer func() {
		bs.statesValue.Store("commit", time.Since(started))
		_ = bs.close()
	}()

	_, err := bs.st.digestDB.Client().WithSession(func(txnCtx mongo.SessionContext, collection func(string) *mongo.Collection) (interface{}, error) {
		if err := bs.writeModels(txnCtx, defaultColNameBlock, bs.blockModels); err != nil {
			return nil, err
		}

		if len(bs.operationModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameOperation, bs.operationModels); err != nil {
				return nil, err
			}
		}

		if len(bs.currencyModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameCurrency, bs.currencyModels); err != nil {
				return nil, err
			}
		}

		if len(bs.accountModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameAccount, bs.accountModels); err != nil {
				return nil, err
			}
		}

		if len(bs.contractAccountModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameContractAccount, bs.contractAccountModels); err != nil {
				return nil, err
			}
		}

		if len(bs.balanceModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameBalance, bs.balanceModels); err != nil {
				return nil, err
			}
		}

		if len(bs.didRegistryModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameDIDRegistry, bs.didRegistryModels); err != nil {
				return nil, err
			}
		}

		if len(bs.didDataModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameDIDData, bs.didDataModels); err != nil {
				return nil, err
			}
		}

		if len(bs.didDocumentModels) > 0 {
			if err := bs.writeModels(txnCtx, defaultColNameDIDDocument, bs.didDocumentModels); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

func (bs *BlockSession) Close() error {
	bs.Lock()
	defer bs.Unlock()

	return bs.close()
}

func (bs *BlockSession) prepareOperationsTree() error {
	nodes := map[string]base.OperationFixedtreeNode{}

	if err := bs.opsTree.Traverse(func(_ uint64, no fixedtree.Node) (bool, error) {
		nno := no.(base.OperationFixedtreeNode)
		if nno.Reason() == nil {
			nodes[nno.Key()] = nno
		} else {
			nodes[nno.Key()[:len(nno.Key())-1]] = nno
		}

		return true, nil
	}); err != nil {
		return err
	}

	bs.opsTreeNodes = nodes

	return nil
}

func (bs *BlockSession) prepareBlock() error {
	if bs.block == nil {
		return nil
	}

	bs.blockModels = make([]mongo.WriteModel, 1)

	manifest := isaac.NewManifest(
		bs.block.Manifest().Height(),
		bs.block.Manifest().Previous(),
		bs.block.Manifest().Proposal(),
		bs.block.Manifest().OperationsTree(),
		bs.block.Manifest().StatesTree(),
		bs.block.Manifest().Suffrage(),
		bs.block.Manifest().ProposedAt(),
	)

	doc, err := NewManifestDoc(manifest, bs.st.digestDB.Encoder(), bs.block.Manifest().Height(), bs.ops, bs.block.SignedAt(), bs.proposal.ProposalFact().Proposer(), bs.proposal.ProposalFact().Point().Round(), bs.buildinfo)
	if err != nil {
		return err
	}
	bs.blockModels[0] = mongo.NewInsertOneModel().SetDocument(doc)

	return nil
}

func (bs *BlockSession) prepareOperations() error {
	if len(bs.ops) < 1 {
		return nil
	}

	node := func(h util.Hash) (bool, bool, base.OperationProcessReasonError) {
		no, found := bs.opsTreeNodes[h.String()]
		if !found {
			return false, false, nil
		}

		return true, no.InState(), no.Reason()
	}

	bs.operationModels = make([]mongo.WriteModel, len(bs.ops))

	for i := range bs.ops {
		op := bs.ops[i]

		var doc OperationDoc
		switch found, inState, reason := node(op.Fact().Hash()); {
		case !found:
			return util.ErrNotFound.Errorf("Operation, %v in operations tree", op.Fact().Hash().String())
		default:
			var reasonMsg string
			switch {
			case reason == nil:
				reasonMsg = ""
			default:
				reasonMsg = reason.Msg()
			}
			d, err := NewOperationDoc(
				op,
				bs.st.digestDB.Encoder(),
				bs.block.Manifest().Height(),
				bs.block.SignedAt(),
				inState,
				reasonMsg,
				uint64(i),
			)
			if err != nil {
				return err
			}
			doc = d
		}

		bs.operationModels[i] = mongo.NewInsertOneModel().SetDocument(doc)
	}

	return nil
}

func (bs *BlockSession) prepareAccounts() error {
	if len(bs.sts) < 1 {
		return nil
	}

	var accountModels []mongo.WriteModel
	var balanceModels []mongo.WriteModel
	var contractAccountModels []mongo.WriteModel
	for i := range bs.sts {
		st := bs.sts[i]

		switch {
		case ccstate.IsAccountStateKey(st.Key()):
			j, err := bs.handleAccountState(st)
			if err != nil {
				return err
			}
			accountModels = append(accountModels, j...)
		case ccstate.IsBalanceStateKey(st.Key()):
			j, address, err := bs.handleBalanceState(st)
			if err != nil {
				return err
			}
			balanceModels = append(balanceModels, j...)
			bs.balanceAddressList = append(bs.balanceAddressList, address)
		case cestate.IsStateContractAccountKey(st.Key()):
			j, err := bs.handleContractAccountState(st)
			if err != nil {
				return err
			}
			contractAccountModels = append(contractAccountModels, j...)
		default:
			continue
		}
	}

	bs.accountModels = accountModels
	bs.contractAccountModels = contractAccountModels
	bs.balanceModels = balanceModels
	return nil
}

func (bs *BlockSession) prepareCurrencies() error {
	if len(bs.sts) < 1 {
		return nil
	}

	var currencyModels []mongo.WriteModel
	for i := range bs.sts {
		st := bs.sts[i]
		switch {
		case ccstate.IsDesignStateKey(st.Key()):
			j, err := bs.handleCurrencyState(st)
			if err != nil {
				return err
			}
			currencyModels = append(currencyModels, j...)
		default:
			continue
		}
	}

	bs.currencyModels = currencyModels

	return nil
}

func (bs *BlockSession) handleAccountState(st base.State) ([]mongo.WriteModel, error) {
	if rs, err := NewAccountValue(st); err != nil {
		return nil, err
	} else if doc, err := NewAccountDoc(rs, bs.st.digestDB.Encoder()); err != nil {
		return nil, err
	} else {
		return []mongo.WriteModel{mongo.NewInsertOneModel().SetDocument(doc)}, nil
	}
}

func (bs *BlockSession) handleBalanceState(st base.State) ([]mongo.WriteModel, string, error) {
	doc, address, err := NewBalanceDoc(st, bs.st.digestDB.Encoder())
	if err != nil {
		return nil, "", err
	}
	return []mongo.WriteModel{mongo.NewInsertOneModel().SetDocument(doc)}, address, nil
}

func (bs *BlockSession) handleContractAccountState(st base.State) ([]mongo.WriteModel, error) {
	doc, err := NewContractAccountStatusDoc(st, bs.st.digestDB.Encoder())
	if err != nil {
		return nil, err
	}
	return []mongo.WriteModel{mongo.NewInsertOneModel().SetDocument(doc)}, nil
}

func (bs *BlockSession) handleCurrencyState(st base.State) ([]mongo.WriteModel, error) {
	doc, err := NewCurrencyDoc(st, bs.st.digestDB.Encoder())
	if err != nil {
		return nil, err
	}
	return []mongo.WriteModel{mongo.NewInsertOneModel().SetDocument(doc)}, nil
}

func (bs *BlockSession) writeModels(ctx context.Context, col string, models []mongo.WriteModel) error {
	started := time.Now()
	defer func() {
		bs.statesValue.Store(fmt.Sprintf("write-models-%s", col), time.Since(started))
	}()

	n := len(models)
	if n < 1 {
		return nil
	} else if n <= bulkWriteLimit {
		return bs.writeModelsChunk(ctx, col, models)
	}

	z := n / bulkWriteLimit
	if n%bulkWriteLimit != 0 {
		z++
	}

	for i := 0; i < z; i++ {
		s := i * bulkWriteLimit
		e := s + bulkWriteLimit
		if e > n {
			e = n
		}

		if err := bs.writeModelsChunk(ctx, col, models[s:e]); err != nil {
			return err
		}
	}

	return nil
}

func (bs *BlockSession) writeModelsChunk(ctx context.Context, col string, models []mongo.WriteModel) error {
	opts := options.BulkWrite().SetOrdered(false)
	if res, err := bs.st.digestDB.Client().Collection(col).BulkWrite(ctx, models, opts); err != nil {
		return err
	} else if res != nil && res.InsertedCount < 1 {
		return errors.Errorf("Not inserted to %s", col)
	}

	return nil
}

func (bs *BlockSession) close() error {
	bs.block = nil
	bs.ops = nil
	bs.opsTree = fixedtree.EmptyTree()
	bs.sts = nil
	bs.proposal = nil
	bs.opsTreeNodes = nil
	bs.blockModels = nil
	bs.operationModels = nil
	bs.currencyModels = nil
	bs.accountModels = nil
	bs.contractAccountModels = nil
	bs.didRegistryModels = nil
	bs.didDataModels = nil
	bs.didDocumentModels = nil
	bs.balanceModels = nil

	return bs.st.Close()
}
