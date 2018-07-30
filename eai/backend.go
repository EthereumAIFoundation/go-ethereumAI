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

// Package eai implements the EthereumAI protocol.
package eai

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ethereumai/go-ethereumai/accounts"
	"github.com/ethereumai/go-ethereumai/common"
	"github.com/ethereumai/go-ethereumai/common/hexutil"
	"github.com/ethereumai/go-ethereumai/consensus"
	"github.com/ethereumai/go-ethereumai/consensus/clique"
	"github.com/ethereumai/go-ethereumai/consensus/eaiash"
	"github.com/ethereumai/go-ethereumai/core"
	"github.com/ethereumai/go-ethereumai/core/bloombits"
	"github.com/ethereumai/go-ethereumai/core/rawdb"
	"github.com/ethereumai/go-ethereumai/core/types"
	"github.com/ethereumai/go-ethereumai/core/vm"
	"github.com/ethereumai/go-ethereumai/eai/downloader"
	"github.com/ethereumai/go-ethereumai/eai/filters"
	"github.com/ethereumai/go-ethereumai/eai/gasprice"
	"github.com/ethereumai/go-ethereumai/eaidb"
	"github.com/ethereumai/go-ethereumai/event"
	"github.com/ethereumai/go-ethereumai/internal/eaiapi"
	"github.com/ethereumai/go-ethereumai/log"
	"github.com/ethereumai/go-ethereumai/miner"
	"github.com/ethereumai/go-ethereumai/node"
	"github.com/ethereumai/go-ethereumai/p2p"
	"github.com/ethereumai/go-ethereumai/params"
	"github.com/ethereumai/go-ethereumai/rlp"
	"github.com/ethereumai/go-ethereumai/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// EthereumAI implements the EthereumAI full node service.
type EthereumAI struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the EthereumAI

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb eaidb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *EaiAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	etheraibase common.Address

	networkId     uint64
	netRPCService *eaiapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etheraibase)
}

func (s *EthereumAI) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new EthereumAI object (including the
// initialisation of the common EthereumAI object)
func New(ctx *node.ServiceContext, config *Config) (*EthereumAI, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run eai.EthereumAI in light sync mode, use les.LightEthereumAI")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	eai := &EthereumAI{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Eaiash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		etheraibase:      config.EtherAIbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising EthereumAI protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geai upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	eai.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, eai.chainConfig, eai.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eai.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	eai.bloomIndexer.Start(eai.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	eai.txPool = core.NewTxPool(config.TxPool, eai.chainConfig, eai.blockchain)

	if eai.protocolManager, err = NewProtocolManager(eai.chainConfig, config.SyncMode, config.NetworkId, eai.eventMux, eai.txPool, eai.engine, eai.blockchain, chainDb); err != nil {
		return nil, err
	}
	eai.miner = miner.New(eai, eai.chainConfig, eai.EventMux(), eai.engine)
	eai.miner.SetExtra(makeExtraData(config.ExtraData))

	eai.APIBackend = &EaiAPIBackend{eai, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eai.APIBackend.gpo = gasprice.NewOracle(eai.APIBackend, gpoParams)

	return eai, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geai",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (eaidb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*eaidb.LDBDatabase); ok {
		db.Meter("eai/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an EthereumAI service
func CreateConsensusEngine(ctx *node.ServiceContext, config *eaiash.Config, chainConfig *params.ChainConfig, db eaidb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == eaiash.ModeFake:
		log.Warn("Eaiash used in fake mode")
		return eaiash.NewFaker()
	case config.PowMode == eaiash.ModeTest:
		log.Warn("Eaiash used in test mode")
		return eaiash.NewTester()
	case config.PowMode == eaiash.ModeShared:
		log.Warn("Eaiash used in shared mode")
		return eaiash.NewShared()
	default:
		engine := eaiash.New(eaiash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the ethereumai package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *EthereumAI) APIs() []rpc.API {
	apis := eaiapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eai",
			Version:   "1.0",
			Service:   NewPublicEthereumAIAPI(s),
			Public:    true,
		}, {
			Namespace: "eai",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "eai",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "eai",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *EthereumAI) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *EthereumAI) EtherAIbase() (eb common.Address, err error) {
	s.lock.RLock()
	etheraibase := s.etheraibase
	s.lock.RUnlock()

	if etheraibase != (common.Address{}) {
		return etheraibase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etheraibase := accounts[0].Address

			s.lock.Lock()
			s.etheraibase = etheraibase
			s.lock.Unlock()

			log.Info("EtherAIbase automatically configured", "address", etheraibase)
			return etheraibase, nil
		}
	}
	return common.Address{}, fmt.Errorf("etheraibase must be explicitly specified")
}

// SetEtherAIbase sets the mining reward address.
func (s *EthereumAI) SetEtherAIbase(etheraibase common.Address) {
	s.lock.Lock()
	s.etheraibase = etheraibase
	s.lock.Unlock()

	s.miner.SetEtherAIbase(etheraibase)
}

func (s *EthereumAI) StartMining(local bool) error {
	eb, err := s.EtherAIbase()
	if err != nil {
		log.Error("Cannot start mining without etheraibase", "err", err)
		return fmt.Errorf("etheraibase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("EtherAIbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so none will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *EthereumAI) StopMining()         { s.miner.Stop() }
func (s *EthereumAI) IsMining() bool      { return s.miner.Mining() }
func (s *EthereumAI) Miner() *miner.Miner { return s.miner }

func (s *EthereumAI) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *EthereumAI) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *EthereumAI) TxPool() *core.TxPool               { return s.txPool }
func (s *EthereumAI) EventMux() *event.TypeMux           { return s.eventMux }
func (s *EthereumAI) Engine() consensus.Engine           { return s.engine }
func (s *EthereumAI) ChainDb() eaidb.Database            { return s.chainDb }
func (s *EthereumAI) IsListening() bool                  { return true } // Always listening
func (s *EthereumAI) EaiVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *EthereumAI) NetVersion() uint64                 { return s.networkId }
func (s *EthereumAI) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *EthereumAI) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// EthereumAI protocol implementation.
func (s *EthereumAI) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = eaiapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// EthereumAI protocol.
func (s *EthereumAI) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
