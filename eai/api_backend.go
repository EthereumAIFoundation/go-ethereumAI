// Copyright 2018 The go-ethereumai Authors
// This file is part of the go-ethereumai library.
//
// The go-ethereumai library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereumai library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereumai library. If not, see <http://www.gnu.org/licenses/>.

package eai

import (
	"context"
	"math/big"

	"github.com/ethereumai/go-ethereumai/accounts"
	"github.com/ethereumai/go-ethereumai/common"
	"github.com/ethereumai/go-ethereumai/common/math"
	"github.com/ethereumai/go-ethereumai/core"
	"github.com/ethereumai/go-ethereumai/core/bloombits"
	"github.com/ethereumai/go-ethereumai/core/rawdb"
	"github.com/ethereumai/go-ethereumai/core/state"
	"github.com/ethereumai/go-ethereumai/core/types"
	"github.com/ethereumai/go-ethereumai/core/vm"
	"github.com/ethereumai/go-ethereumai/eai/downloader"
	"github.com/ethereumai/go-ethereumai/eai/gasprice"
	"github.com/ethereumai/go-ethereumai/eaidb"
	"github.com/ethereumai/go-ethereumai/event"
	"github.com/ethereumai/go-ethereumai/params"
	"github.com/ethereumai/go-ethereumai/rpc"
)

// EaiAPIBackend implements eaiapi.Backend for full nodes
type EaiAPIBackend struct {
	eai *EthereumAI
	gpo *gasprice.Oracle
}

func (b *EaiAPIBackend) ChainConfig() *params.ChainConfig {
	return b.eai.chainConfig
}

func (b *EaiAPIBackend) CurrentBlock() *types.Block {
	return b.eai.blockchain.CurrentBlock()
}

func (b *EaiAPIBackend) SetHead(number uint64) {
	b.eai.protocolManager.downloader.Cancel()
	b.eai.blockchain.SetHead(number)
}

func (b *EaiAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.eai.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.eai.blockchain.CurrentBlock().Header(), nil
	}
	return b.eai.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EaiAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.eai.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.eai.blockchain.CurrentBlock(), nil
	}
	return b.eai.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EaiAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.eai.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.eai.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EaiAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eai.blockchain.GetBlockByHash(hash), nil
}

func (b *EaiAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.eai.chainDb, hash); number != nil {
		return rawdb.ReadReceipts(b.eai.chainDb, hash, *number), nil
	}
	return nil, nil
}

func (b *EaiAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	number := rawdb.ReadHeaderNumber(b.eai.chainDb, hash)
	if number == nil {
		return nil, nil
	}
	receipts := rawdb.ReadReceipts(b.eai.chainDb, hash, *number)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *EaiAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.eai.blockchain.GetTdByHash(blockHash)
}

func (b *EaiAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.eai.BlockChain(), nil)
	return vm.NewEVM(context, state, b.eai.chainConfig, vmCfg), vmError, nil
}

func (b *EaiAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eai.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EaiAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eai.BlockChain().SubscribeChainEvent(ch)
}

func (b *EaiAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eai.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EaiAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eai.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EaiAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eai.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EaiAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.eai.txPool.AddLocal(signedTx)
}

func (b *EaiAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.eai.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EaiAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.eai.txPool.Get(hash)
}

func (b *EaiAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.eai.txPool.State().GetNonce(addr), nil
}

func (b *EaiAPIBackend) Stats() (pending int, queued int) {
	return b.eai.txPool.Stats()
}

func (b *EaiAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eai.TxPool().Content()
}

func (b *EaiAPIBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.eai.TxPool().SubscribeTxPreEvent(ch)
}

func (b *EaiAPIBackend) Downloader() *downloader.Downloader {
	return b.eai.Downloader()
}

func (b *EaiAPIBackend) ProtocolVersion() int {
	return b.eai.EaiVersion()
}

func (b *EaiAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EaiAPIBackend) ChainDb() eaidb.Database {
	return b.eai.ChainDb()
}

func (b *EaiAPIBackend) EventMux() *event.TypeMux {
	return b.eai.EventMux()
}

func (b *EaiAPIBackend) AccountManager() *accounts.Manager {
	return b.eai.AccountManager()
}

func (b *EaiAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.eai.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EaiAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eai.bloomRequests)
	}
}
