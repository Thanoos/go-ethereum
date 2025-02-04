// Copyright 2021 The go-ethereum Authors
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

// Package gconfig contains the configuration of the G and LES protocols.
package gconfig

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/gash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/g/downloader"
	"github.com/ethereum/go-ethereum/g/gasprice"
	"github.com/ethereum/go-ethereum/gdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

// FullNodeGPO contains default gasprice oracle settings for full node.
var FullNodeGPO = gasprice.Config{
	Blocks:           20,
	Percentile:       60,
	MaxHeaderHistory: 1024,
	MaxBlockHistory:  1024,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// LightClientGPO contains default gasprice oracle settings for light client.
var LightClientGPO = gasprice.Config{
	Blocks:           2,
	Percentile:       60,
	MaxHeaderHistory: 300,
	MaxBlockHistory:  5,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// Defaults contains default settings for use on the Ethereum main net.
var Defaults = Config{
	SyncMode: downloader.SnapSync,
	Gash: gash.Config{
		CacheDir:         "gash",
		CachesInMem:      2,
		CachesOnDisk:     3,
		CachesLockMmap:   false,
		DatasetsInMem:    1,
		DatasetsOnDisk:   2,
		DatasetsLockMmap: false,
	},
	NetworkId:               1,
	TxLookupLimit:           2350000,
	LightPeers:              100,
	UltraLightFraction:      75,
	DatabaseCache:           512,
	TrieCleanCache:          154,
	TrieCleanCacheJournal:   "triecache",
	TrieCleanCacheRejournal: 60 * time.Minute,
	TrieDirtyCache:          256,
	TrieTimeout:             60 * time.Minute,
	SnapshotCache:           102,
	FilterLogCacheSize:      32,
	Miner: miner.Config{
		GasCeil:  30000000,
		GasPrice: big.NewInt(params.GWei),
		Recommit: 3 * time.Second,
	},
	TxPool:        core.DefaultTxPoolConfig,
	RPCGasCap:     50000000,
	RPCEVMTimeout: 5 * time.Second,
	GPO:           FullNodeGPO,
	RPCTxFeeCap:   1, // 1 ac
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "darwin" {
		Defaults.Gash.DatasetDir = filepath.Join(home, "Library", "Gash")
	} else if runtime.GOOS == "windows" {
		localappdata := os.Getenv("LOCALAPPDATA")
		if localappdata != "" {
			Defaults.Gash.DatasetDir = filepath.Join(localappdata, "Gash")
		} else {
			Defaults.Gash.DatasetDir = filepath.Join(home, "AppData", "Local", "Gash")
		}
	} else {
		Defaults.Gash.DatasetDir = filepath.Join(home, ".gash")
	}
}

//go:generate go run github.com/fjl/gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for of the G and LES protocols.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Ethereum main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// for nodes to connect to.
	EthDiscoveryURLs  []string
	SnapDiscoveryURLs []string

	NoPruning  bool // Whether to disable pruning and flush everything to disk
	NoPrefetch bool // Whether to disable prefetching and only load state on demand

	TxLookupLimit uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.

	// RequiredBlocks is a set of block number -> hash mappings which must be in the
	// canonical chain of all remote peers. Setting the option makes geth verify the
	// presence of these blocks for every new peer connection.
	RequiredBlocks map[uint64]common.Hash `toml:"-"`

	// Light client options
	LightServ          int  `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightIngress       int  `toml:",omitempty"` // Incoming bandwidth limit for light servers
	LightEgress        int  `toml:",omitempty"` // Outgoing bandwidth limit for light servers
	LightPeers         int  `toml:",omitempty"` // Maximum number of LES client peers
	LightNoPrune       bool `toml:",omitempty"` // Whether to disable light chain pruning
	LightNoSyncServe   bool `toml:",omitempty"` // Whether to serve light clients before syncing
	SyncFromCheckpoint bool `toml:",omitempty"` // Whether to sync the header chain from the configured checkpoint

	// Ultra Light client options
	UltraLightServers      []string `toml:",omitempty"` // List of trusted ultra light servers
	UltraLightFraction     int      `toml:",omitempty"` // Percentage of trusted servers to accept an announcement
	UltraLightOnlyAnnounce bool     `toml:",omitempty"` // Whether to only announce headers, or also serve them

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string

	TrieCleanCache          int
	TrieCleanCacheJournal   string        `toml:",omitempty"` // Disk journal directory for trie cache to survive node restarts
	TrieCleanCacheRejournal time.Duration `toml:",omitempty"` // Time interval to regenerate the journal for clean cache
	TrieDirtyCache          int
	TrieTimeout             time.Duration
	SnapshotCache           int
	Preimages               bool

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int

	// Mining options
	Miner miner.Config

	// Gash options
	Gash gash.Config

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// RPCGasCap is the global gas cap for g-call variants.
	RPCGasCap uint64

	// RPCEVMTimeout is the global timeout for g-call.
	RPCEVMTimeout time.Duration

	// RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for
	// send-transaction variants. The unit is ac.
	RPCTxFeeCap float64

	// Checkpoint is a hardcoded checkpoint which can be nil.
	Checkpoint *params.TrustedCheckpoint `toml:",omitempty"`

	// CheckpointOracle is the configuration for checkpoint oracle.
	CheckpointOracle *params.CheckpointOracleConfig `toml:",omitempty"`

	// OverrideTerminalTotalDifficulty (TODO: remove after the fork)
	OverrideTerminalTotalDifficulty *big.Int `toml:",omitempty"`

	// OverrideTerminalTotalDifficultyPassed (TODO: remove after the fork)
	OverrideTerminalTotalDifficultyPassed *bool `toml:",omitempty"`
}

// CreateConsensusEngine creates a consensus engine for the given chain configuration.
func CreateConsensusEngine(stack *node.Node, gashConfig *gash.Config, cliqueConfig *params.CliqueConfig, notify []string, noverify bool, db gdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	var engine consensus.Engine
	if cliqueConfig != nil {
		engine = clique.New(cliqueConfig, db)
	} else {
		switch gashConfig.PowMode {
		case gash.ModeFake:
			log.Warn("Gash used in fake mode")
		case gash.ModeTest:
			log.Warn("Gash used in test mode")
		case gash.ModeShared:
			log.Warn("Gash used in shared mode")
		}
		engine = gash.New(gash.Config{
			PowMode:          gashConfig.PowMode,
			CacheDir:         stack.ResolvePath(gashConfig.CacheDir),
			CachesInMem:      gashConfig.CachesInMem,
			CachesOnDisk:     gashConfig.CachesOnDisk,
			CachesLockMmap:   gashConfig.CachesLockMmap,
			DatasetDir:       gashConfig.DatasetDir,
			DatasetsInMem:    gashConfig.DatasetsInMem,
			DatasetsOnDisk:   gashConfig.DatasetsOnDisk,
			DatasetsLockMmap: gashConfig.DatasetsLockMmap,
			NotifyFull:       gashConfig.NotifyFull,
		}, notify, noverify)
		engine.(*gash.Gash).SetThreads(-1) // Disable CPU mining
	}
	return beacon.New(engine)
}
