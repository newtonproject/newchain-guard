package cli

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/newtonproject/newchain-guard/params"
	"github.com/spf13/viper"
	"github.com/upper/db/v4/adapter/mysql"
)

const defaultConfigFile = "./config.toml"
const defaultRPCURL = "https://rpc1.newchain.newtonproject.org"

func (cli *CLI) defaultConfig() {
	viper.BindPFlag("rpcURL", cli.rootCmd.PersistentFlags().Lookup("rpcURL"))

	viper.SetDefault("rpcURL", defaultRPCURL)
}

func (cli *CLI) setupConfig() error {

	// var ret bool
	var err error

	cli.defaultConfig()

	viper.SetConfigName(defaultConfigFile)
	viper.AddConfigPath(".")
	cfgFile := cli.configpath
	if cfgFile != "" {
		if _, err = os.Stat(cfgFile); err == nil {
			viper.SetConfigFile(cfgFile)
			err = viper.ReadInConfig()
			if err != nil {
				fmt.Println(err)
			}
		} else {
			// The default configuration is enabled.
			// fmt.Println(err)
			err = nil
		}
	} else {
		// The default configuration is enabled.
		err = nil
	}

	if rpcURL := viper.GetString("rpcURL"); rpcURL != "" {
		cli.rpcURL = viper.GetString("rpcURL")
	}

	if log := viper.GetString("log"); log != "" {
		cli.logfile = viper.GetString("log")
	}

	return nil
}

func loadParamsConfig() (*params.Config, error) {
	config := params.DefaultConfig

	if rpcURL := viper.GetString("rpcURL"); rpcURL != "" {
		config.RawURL = rpcURL
	}

	client, err := ethclient.Dial(config.RawURL)
	if err != nil {
		return nil, err
	}
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}
	config.ChainID = chainID

	if methodWhitelist := viper.GetStringSlice("MethodWhitelist"); len(methodWhitelist) > 0 {
		config.MethodWhiteList = make(map[string]struct{}, len(methodWhitelist))
		for _, method := range methodWhitelist {
			config.MethodWhiteList[method] = struct{}{}
		}
	}

	if fromBlacklistConfig := viper.GetString("FromBlacklistConfig"); len(fromBlacklistConfig) > 0 {
		v := viper.New()
		v.SetConfigFile(fromBlacklistConfig)
		err := v.ReadInConfig()
		if err != nil {
			return nil, err
		}
		if fromBlackList := v.GetStringSlice("FromBlackList"); len(fromBlackList) > 0 {
			config.FromBlackList = make(map[common.Address]struct{}, len(fromBlackList))
			for _, address := range fromBlackList {
				if !common.IsHexAddress(address) {
					return nil, fmt.Errorf("address %s not invalid hex-encode\n", address)
				}
				config.FromBlackList[common.HexToAddress(address)] = struct{}{}
			}
		}
	}

	if fromBlacklistConfig := viper.GetString("ToBlacklistConfig"); len(fromBlacklistConfig) > 0 {
		v := viper.New()
		v.SetConfigFile(fromBlacklistConfig)
		err := v.ReadInConfig()
		if err != nil {
			return nil, err
		}
		if toBlackList := v.GetStringSlice("ToBlackList"); len(toBlackList) > 0 {
			config.ToBlackList = make(map[common.Address]struct{}, len(toBlackList))
			for _, address := range toBlackList {
				if !common.IsHexAddress(address) {
					return nil, fmt.Errorf("address %s not invalid hex-encode\n", address)
				}
				config.ToBlackList[common.HexToAddress(address)] = struct{}{}
			}
		}
	}
	config.DisableContractCreate = viper.GetBool("DisableContractCreate")

	if fromBlacklistConfig := viper.GetString("TxHashBlackListConfig"); len(fromBlacklistConfig) > 0 {
		v := viper.New()
		v.SetConfigFile(fromBlacklistConfig)
		err := v.ReadInConfig()
		if err != nil {
			return nil, err
		}
		if txHashBlacklist := v.GetStringSlice("TxHashBlackList"); len(txHashBlacklist) > 0 {
			config.TxHashBlackList = make(map[common.Hash]struct{}, len(txHashBlacklist))
			for _, hashStr := range txHashBlacklist {
				if len(hashStr) > 1 {
					if hashStr[0:2] == "0x" || hashStr[0:2] == "0X" {
						hashStr = hashStr[2:]
					}
				}
				if len(hashStr)%2 == 1 {
					hashStr = "0" + hashStr
				}
				hashByte, err := hex.DecodeString(hashStr)
				if err != nil {
					return nil, err
				}
				config.TxHashBlackList[common.BytesToHash(hashByte)] = struct{}{}
			}
		}
	}

	config.EnableMaxValueVerify = viper.GetBool("EnableMaxValueVerify")
	maxValueInNEW := viper.GetInt64("MaxValueInNEW")
	if maxValueInNEW > 0 {
		config.MaxValueInWEI = big.NewInt(0).Mul(big.NewInt(maxValueInNEW), params.Big1NEW)
	}

	config.EnableMaxGasLimitVerify = viper.GetBool("EnableMaxGasLimitVerify")
	if config.EnableMaxGasLimitVerify {
		config.MaxGasLimit = uint64(viper.GetInt64("MaxGasLimit"))
		if config.MaxGasLimit < 21000 {
			config.MaxGasLimit = 21000
		}
	}

	config.MinGasLimit = uint64(viper.GetInt64("MinGasLimit"))
	if config.MinGasLimit < 21000 {
		config.MinGasLimit = 21000
	}

	if viper.IsSet("MinGasPriceInWEI") {
		minGasPriceInWEI := viper.GetInt64("MinGasPriceInWEI")
		if minGasPriceInWEI >= 0 {
			config.MinGasPriceInWEI = big.NewInt(minGasPriceInWEI)
		}
	} else {
		config.MinGasPriceInWEI = big.NewInt(1) // min
	}
	if viper.IsSet("MaxGasPriceInWEI") {
		maxGasPriceInWEI := viper.GetInt64("MaxGasPriceInWEI")
		if maxGasPriceInWEI >= 0 {
			config.MaxGasPriceInWEI = big.NewInt(maxGasPriceInWEI)
		}
	}
	if config.MaxGasPriceInWEI != nil && config.MinGasPriceInWEI != nil {
		if config.MaxGasPriceInWEI.Cmp(config.MinGasPriceInWEI) < 0 {
			return nil, errors.New("GasPrice max less then min")
		}
	}

	if viper.IsSet("MinGasTipCapInWEI") {
		minGasTipCapInWEI := viper.GetInt64("MinGasTipCapInWEI")
		if minGasTipCapInWEI >= 0 {
			config.MinGasTipCapInWEI = big.NewInt(minGasTipCapInWEI)
		}
	} else {
		config.MinGasTipCapInWEI = big.NewInt(0) // min
	}
	if viper.IsSet("MaxGasTipCapInWEI") {
		maxGasTipCapInWEI := viper.GetInt64("MaxGasTipCapInWEI")
		if maxGasTipCapInWEI >= 0 {
			config.MaxGasTipCapInWEI = big.NewInt(maxGasTipCapInWEI)
		}
	}
	if config.MaxGasTipCapInWEI != nil && config.MinGasTipCapInWEI != nil {
		if config.MaxGasTipCapInWEI.Cmp(config.MinGasTipCapInWEI) < 0 {
			return nil, errors.New("GasTipCap max less then min")
		}
	}

	config.EnableActiveMQ = viper.GetBool("EnableActiveMQ")

	config.EnableWhitelistDB = viper.GetBool("EnableWhitelistDB")
	if config.EnableWhitelistDB {
		// adapterName string, settings db.ConnectionURL
		adapterName := viper.GetString("database_whitelist.db")
		if adapterName == "" {
			return nil, errors.New("EnableWhitelistDB enable but adapter name is empty")
		}
		settings := mysql.ConnectionURL{
			User:     viper.GetString("database_whitelist.user"),
			Password: viper.GetString("database_whitelist.password"),
			Database: viper.GetString("database_whitelist.database"),
			Host:     viper.GetString("database_whitelist.host"),
		}
		config.WhitelistDBAdapterName = adapterName
		config.WhitelistDBSettings = settings
	}

	config.EnableLuaFilter = viper.GetBool("EnableLuaFilter")
	if config.EnableLuaFilter {
		config.LuaFile = viper.GetString("LuaFile")
		config.LuaCallFunctionName = viper.GetString("LuaCallFunctionName")
		config.EnableFromCheck = viper.GetBool("EnableFromCheck")
		if config.LuaFile == "" {
			return nil, errors.New("EnableLuaFilter is true but LuaFile not set")
		}
		if _, err := os.Stat(config.LuaFile); os.IsNotExist(err) {
			return nil, errors.New("lua file not exist")
		}
		if config.LuaCallFunctionName == "" {
			return nil, errors.New("EnableLuaFilter is true but LuaCallFunctionName not set")
		}
	}

	return &config, nil
}
