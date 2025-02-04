// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package g

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/g/gasprice"
	"github.com/ethereum/go-ethereum/g/tracers"
	"github.com/ethereum/go-ethereum/gdb"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// GAPIBackend implements ethapi.Backend for full nodes
type GAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	g                   *Ethereum
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *GAPIBackend) ChainConfig() *params.ChainConfig {
	return b.g.blockchain.Config()
}

func (b *GAPIBackend) CurrentBlock() *types.Block {
	return b.g.blockchain.CurrentBlock()
}

func (b *GAPIBackend) SetHead(number uint64) {
	b.g.handler.downloader.Cancel()
	b.g.blockchain.SetHead(number)
}

func (b *GAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.g.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.g.blockchain.CurrentBlock().Header(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		block := b.g.blockchain.CurrentFinalizedBlock()
		if block != nil {
			return block.Header(), nil
		}
		return nil, errors.New("finalized block not found")
	}
	if number == rpc.SafeBlockNumber {
		block := b.g.blockchain.CurrentSafeBlock()
		if block != nil {
			return block.Header(), nil
		}
		return nil, errors.New("safe block not found")
	}
	return b.g.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *GAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.g.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.g.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *GAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.g.blockchain.GetHeaderByHash(hash), nil
}

func (b *GAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.g.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.g.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		return b.g.blockchain.CurrentFinalizedBlock(), nil
	}
	if number == rpc.SafeBlockNumber {
		return b.g.blockchain.CurrentSafeBlock(), nil
	}
	return b.g.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *GAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.g.blockchain.GetBlockByHash(hash), nil
}

func (b *GAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.g.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.g.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.g.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *GAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.g.miner.PendingBlockAndReceipts()
}

func (b *GAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.g.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.g.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *GAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.g.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.g.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *GAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.g.blockchain.GetReceiptsByHash(hash), nil
}

func (b *GAPIBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return rawdb.ReadLogs(b.g.chainDb, hash, number, b.ChainConfig()), nil
}

func (b *GAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if header := b.g.blockchain.GetHeaderByHash(hash); header != nil {
		return b.g.blockchain.GetTd(hash, header.Number.Uint64())
	}
	return nil
}

func (b *GAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	if vmConfig == nil {
		vmConfig = b.g.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.g.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.g.blockchain.Config(), *vmConfig), state.Error, nil
}

func (b *GAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.g.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *GAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.g.miner.SubscribePendingLogs(ch)
}

func (b *GAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.g.BlockChain().SubscribeChainEvent(ch)
}

func (b *GAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.g.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *GAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.g.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *GAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.g.BlockChain().SubscribeLogsEvent(ch)
}

func (b *GAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.g.txPool.AddLocal(signedTx)
}

func (b *GAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending := b.g.txPool.Pending(false)
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *GAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.g.txPool.Get(hash)
}

func (b *GAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.g.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *GAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.g.txPool.Nonce(addr), nil
}

func (b *GAPIBackend) Stats() (pending int, queued int) {
	return b.g.txPool.Stats()
}

func (b *GAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.g.TxPool().Content()
}

func (b *GAPIBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return b.g.TxPool().ContentFrom(addr)
}

func (b *GAPIBackend) TxPool() *core.TxPool {
	return b.g.TxPool()
}

func (b *GAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.g.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *GAPIBackend) SyncProgress() ethereum.SyncProgress {
	return b.g.Downloader().Progress()
}

func (b *GAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *GAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *GAPIBackend) ChainDb() gdb.Database {
	return b.g.ChainDb()
}

func (b *GAPIBackend) EventMux() *event.TypeMux {
	return b.g.EventMux()
}

func (b *GAPIBackend) AccountManager() *accounts.Manager {
	return b.g.AccountManager()
}

func (b *GAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *GAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *GAPIBackend) RPCGasCap() uint64 {
	return b.g.config.RPCGasCap
}

func (b *GAPIBackend) RPCEVMTimeout() time.Duration {
	return b.g.config.RPCEVMTimeout
}

func (b *GAPIBackend) RPCTxFeeCap() float64 {
	return b.g.config.RPCTxFeeCap
}

func (b *GAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.g.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *GAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.g.bloomRequests)
	}
}

func (b *GAPIBackend) Engine() consensus.Engine {
	return b.g.engine
}

func (b *GAPIBackend) CurrentHeader() *types.Header {
	return b.g.blockchain.CurrentHeader()
}

func (b *GAPIBackend) Miner() *miner.Miner {
	return b.g.Miner()
}

func (b *GAPIBackend) StartMining(threads int) error {
	return b.g.StartMining(threads)
}

func (b *GAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.g.StateAtBlock(block, reexec, base, readOnly, preferDisk)
}

func (b *GAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.g.stateAtTransaction(block, txIndex, reexec)
}
