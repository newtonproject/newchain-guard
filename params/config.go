package params

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	db "github.com/upper/db/v4"
)

var (
	Big1NEW = big.NewInt(1000000000000000000)
)

type Config struct {
	RawURL string

	// method
	MethodWhiteList map[string]struct{}

	// from address
	// EnableFromVerify bool
	FromBlackList map[common.Address]struct{}

	// to address
	DisableContractCreate bool
	ToBlackList           map[common.Address]struct{}

	// hash
	TxHashBlackList map[common.Hash]struct{}

	// value
	EnableMaxValueVerify bool
	MaxValueInWEI        *big.Int // value in WEI

	// gas limit
	EnableMaxGasLimitVerify bool
	// EnableMinGasLimitVerify bool // force gas limit big than 21000
	MaxGasLimit uint64
	MinGasLimit uint64

	// gas price
	// EnableMinGasPriceVerify bool // force gas price big than 1 WEI
	MinGasPriceInWEI *big.Int // uint64 for no more than 1NEW
	MaxGasPriceInWEI *big.Int

	// maxPriorityFeePerGas
	MinGasTipCapInWEI *big.Int
	MaxGasTipCapInWEI *big.Int

	// ChainID
	ChainID *big.Int

	// Apache ActiveMQ
	EnableActiveMQ bool

	// Whitelist from DB
	EnableWhitelistDB      bool
	WhitelistDBAdapterName string
	WhitelistDBSettings    db.ConnectionURL

	// lua
	EnableLuaFilter     bool
	LuaFile             string // path to lua file when is EnableLuaFilter
	EnableFromCheck     bool
	LuaCallFunctionName string // the function name to call
}

var (
	// DefaultConfig is the default config.
	DefaultConfig = Config{
		RawURL: "https://rpc1.newchain.newtonproject.org",

		// method
		MethodWhiteList: make(map[string]struct{}),

		// from address
		// EnableFromVerify bool
		FromBlackList: make(map[common.Address]struct{}),

		// to address
		ToBlackList: make(map[common.Address]struct{}),

		// hash
		TxHashBlackList: make(map[common.Hash]struct{}),

		// value
		EnableMaxValueVerify: false,
		MaxValueInWEI:        big.NewInt(0).Mul(big.NewInt(1000000), Big1NEW), // 1000000 NEW

		// gas limit
		EnableMaxGasLimitVerify: false,
		// EnableMinGasLimitVerify: true,
		MaxGasLimit: 0,
		MinGasLimit: 21000,

		// gas price
		// EnableMinGasPriceVerify: true,
		MinGasPriceInWEI:  big.NewInt(1), // uint64 for no more than 1NEW
		MinGasTipCapInWEI: big.NewInt(0),

		ChainID: big.NewInt(16888),
	}
)

func Copy(c *Config) *Config {
	config := &Config{
		RawURL:                  c.RawURL,
		DisableContractCreate:   c.DisableContractCreate,
		EnableMaxValueVerify:    c.EnableMaxValueVerify,
		MaxValueInWEI:           big.NewInt(0).Set(c.MaxValueInWEI),
		EnableMaxGasLimitVerify: c.EnableMaxGasLimitVerify,
		MaxGasLimit:             c.MaxGasLimit,
		MinGasLimit:             c.MinGasLimit,
		MinGasPriceInWEI:        big.NewInt(0).Set(c.MinGasPriceInWEI),
		MinGasTipCapInWEI:       big.NewInt(0).Set(c.MinGasTipCapInWEI),
		ChainID:                 c.ChainID,
		EnableActiveMQ:          c.EnableActiveMQ,
	}

	if methodWhitelist := c.MethodWhiteList; len(methodWhitelist) > 0 {
		config.MethodWhiteList = make(map[string]struct{}, len(methodWhitelist))
		for method := range methodWhitelist {
			config.MethodWhiteList[method] = struct{}{}
		}
	}

	if fromBlackList := c.FromBlackList; len(fromBlackList) > 0 {
		config.FromBlackList = make(map[common.Address]struct{}, len(fromBlackList))
		for addr := range fromBlackList {
			config.FromBlackList[addr] = struct{}{}
		}
	}

	if toBlackList := c.ToBlackList; len(toBlackList) > 0 {
		config.ToBlackList = make(map[common.Address]struct{}, len(toBlackList))
		for addr := range toBlackList {
			config.ToBlackList[addr] = struct{}{}
		}
	}

	if txHashBlacklist := c.TxHashBlackList; len(txHashBlacklist) > 0 {
		config.TxHashBlackList = make(map[common.Hash]struct{}, len(txHashBlacklist))
		for hash := range txHashBlacklist {
			config.TxHashBlackList[hash] = struct{}{}
		}
	}

	return config
}

// MarshalJSON encodes to json format.
func (c *Config) MarshalJSON() ([]byte, error) {
	type config struct {
		RawURL                  string
		MethodWhiteList         []string
		FromBlackList           int
		DisableContractCreate   bool
		ToBlackList             int
		TxHashBlackList         int
		EnableMaxValueVerify    bool
		MaxValueInWEI           *big.Int
		EnableMaxGasLimitVerify bool
		MaxGasLimit             uint64
		MinGasLimit             uint64
		MinGasPriceInWEI        *big.Int
		MinGasTipCapInWEI       *big.Int
		ChainID                 *big.Int
		EnableActiveMQ          bool
	}
	enc := &config{
		RawURL:                  c.RawURL,
		FromBlackList:           len(c.FromBlackList),
		DisableContractCreate:   c.DisableContractCreate,
		ToBlackList:             len(c.ToBlackList),
		TxHashBlackList:         len(c.TxHashBlackList),
		EnableMaxValueVerify:    c.EnableMaxValueVerify,
		MaxValueInWEI:           big.NewInt(0).Set(c.MaxValueInWEI),
		EnableMaxGasLimitVerify: c.EnableMaxGasLimitVerify,
		MaxGasLimit:             c.MaxGasLimit,
		MinGasLimit:             c.MinGasLimit,
		MinGasPriceInWEI:        big.NewInt(0).Set(c.MinGasPriceInWEI),
		MinGasTipCapInWEI:       big.NewInt(0).Set(c.MinGasTipCapInWEI),
		ChainID:                 c.ChainID,
		EnableActiveMQ:          c.EnableActiveMQ,
	}

	if methodWhitelist := c.MethodWhiteList; len(methodWhitelist) > 0 {
		enc.MethodWhiteList = make([]string, 0)
		for method := range methodWhitelist {
			enc.MethodWhiteList = append(enc.MethodWhiteList, method)
		}
	}

	return json.Marshal(&enc)
}
