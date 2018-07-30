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

// Package les implements the Light EthereumAI Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereumai/go-ethereumai/accounts"
	"github.com/ethereumai/go-ethereumai/common"
	"github.com/ethereumai/go-ethereumai/common/hexutil"
	"github.com/ethereumai/go-ethereumai/consensus"
	"github.com/ethereumai/go-ethereumai/core"
	"github.com/ethereumai/go-ethereumai/core/bloombits"
	"github.com/ethereumai/go-ethereumai/core/rawdb"
	"github.com/ethereumai/go-ethereumai/core/types"
	"github.com/ethereumai/go-ethereumai/eai"
	"github.com/ethereumai/go-ethereumai/eai/downloader"
	"github.com/ethereumai/go-ethereumai/eai/filters"
	"github.com/ethereumai/go-ethereumai/eai/gasprice"
	"github.com/ethereumai/go-ethereumai/eaidb"
	"github.com/ethereumai/go-ethereumai/event"
	"github.com/ethereumai/go-ethereumai/internal/eaiapi"
	"github.com/ethereumai/go-ethereumai/light"
	"github.com/ethereumai/go-ethereumai/log"
	"github.com/ethereumai/go-ethereumai/node"
	"github.com/ethereumai/go-ethereumai/p2p"
	"github.com/ethereumai/go-ethereumai/p2p/discv5"
	"github.com/ethereumai/go-ethereumai/params"
	rpc "github.com/ethereumai/go-ethereumai/rpc"
)

type LightEthereumAI struct {
	config *eai.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb eaidb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *eaiapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *eai.Config) (*LightEthereumAI, error) {
	chainDb, err := eai.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	leai := &LightEthereumAI{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           eai.CreateConsensusEngine(ctx, &config.Eaiash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     eai.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	leai.relay = NewLesTxRelay(peers, leai.reqDist)
	leai.serverPool = newServerPool(chainDb, quitSync, &leai.wg)
	leai.retriever = newRetrieveManager(peers, leai.reqDist, leai.serverPool)
	leai.odr = NewLesOdr(chainDb, leai.chtIndexer, leai.bloomTrieIndexer, leai.bloomIndexer, leai.retriever)
	if leai.blockchain, err = light.NewLightChain(leai.odr, leai.chainConfig, leai.engine); err != nil {
		return nil, err
	}
	leai.bloomIndexer.Start(leai.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		leai.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	leai.txPool = light.NewTxPool(leai.chainConfig, leai.blockchain, leai.relay)
	if leai.protocolManager, err = NewProtocolManager(leai.chainConfig, true, ClientProtocolVersions, config.NetworkId, leai.eventMux, leai.engine, leai.peers, leai.blockchain, nil, chainDb, leai.odr, leai.relay, quitSync, &leai.wg); err != nil {
		return nil, err
	}
	leai.ApiBackend = &LesApiBackend{leai, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	leai.ApiBackend.gpo = gasprice.NewOracle(leai.ApiBackend, gpoParams)
	return leai, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// EtherAIbase is the address that mining rewards will be send to
func (s *LightDummyAPI) EtherAIbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for EtherAIbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the ethereumai package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightEthereumAI) APIs() []rpc.API {
	return append(eaiapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "eai",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "eai",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "eai",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightEthereumAI) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightEthereumAI) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightEthereumAI) TxPool() *light.TxPool              { return s.txPool }
func (s *LightEthereumAI) Engine() consensus.Engine           { return s.engine }
func (s *LightEthereumAI) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightEthereumAI) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightEthereumAI) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightEthereumAI) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// EthereumAI protocol implementation.
func (s *LightEthereumAI) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = eaiapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// EthereumAI protocol.
func (s *LightEthereumAI) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
