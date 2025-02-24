package filter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/errors"
	lua "github.com/yuin/gopher-lua"
	mysql "gitlab.newtonproject.org/yangchenzhong/NewChainGuard/gluasql_mysql"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/notify"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/params"
)

type Filter struct {
	config   *params.Config
	ErrorLog *log.Logger
}

func NewFilter(config *params.Config, log *log.Logger) (*Filter, error) {
	if config == nil {
		return nil, params.GetStatusError(params.StatusFilterNoConfig)
	}

	return &Filter{config: config, ErrorLog: log}, nil
}

func (f *Filter) inWhitelistMethod(method string) bool {
	_, ok := f.config.MethodWhiteList[method]
	return ok
}

func (f *Filter) CheckJSONRequest(w http.ResponseWriter, r *http.Request, bodyBytes []byte) error {
	var in params.JSONRequest
	if err := json.Unmarshal(bodyBytes, &in); err != nil {
		params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusInvalidJSONRequest, in.ID, string(bodyBytes))
		return err
	}

	if len(strings.Split(in.Method, serviceMethodSeparator)) != 2 {
		params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusMethodNotAllowed, in.ID, string(bodyBytes))
		return params.GetStatusError(params.StatusMethodNotAllowed)
	}

	if !f.inWhitelistMethod(in.Method) {
		params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusMethodNotWhitelist, in.ID, string(bodyBytes))
		return params.GetStatusError(params.StatusMethodNotWhitelist)
	}

	switch in.Method {
	case "eth_sendRawTransaction":
		hexParam, err := decodeRawTransaction(in.Payload)
		if err != nil {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusDecodeTransactionError, in.ID, string(bodyBytes))
			return params.GetStatusError(params.StatusDecodeTransactionError)
		}
		tx, err := decodeTransaction(hexParam)
		if err != nil {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusDecodeTransactionError, in.ID, string(bodyBytes))
			return params.GetStatusError(params.StatusDecodeTransactionError)
		}
		status := f.checkTransaction(tx)
		if status == params.StatusOK {
			params.LogRequestInfo(r, f.ErrorLog, params.StatusOK, string(bodyBytes))
		} else if status == params.StatusValueTooLarge {
			params.LogRequestWarn(r, f.ErrorLog, params.StatusValueTooLarge, string(bodyBytes))
		} else {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, status, in.ID, string(bodyBytes))
			return params.ErrorInternalError
		}

		// transaction pass
		if f.config.EnableActiveMQ {
			notify.AddTx(hexParam)
		}

		return nil
	case "eth_gasPrice":
		gasPrice, err := getGasPrice(r.Context(), f.config.RawURL)
		if err != nil {
			params.LogRequestWarn(r, f.ErrorLog, params.StatusInternalError, err.Error())
			gasPrice = big.NewInt(0).Set(f.config.MinGasPriceInWEI)
		} else {
			if gasPrice.Cmp(f.config.MinGasPriceInWEI) < 0 {
				gasPrice.Set(f.config.MinGasPriceInWEI)
			}
			if f.config.MaxGasPriceInWEI != nil && gasPrice.Cmp(f.config.MaxGasPriceInWEI) > 0 {
				gasPrice.Set(f.config.MaxGasPriceInWEI)
			}
		}

		response := params.JSONSuccessResponse{
			Version: params.JSONRPCVersion,
			ID:      in.ID,
			Result:  hexutil.EncodeBig(gasPrice),
		}
		ret, err := json.Marshal(response)
		if err != nil {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusGasPriceError, in.ID, string(bodyBytes))
			return params.ErrorInternalError
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		buf := bytes.NewBuffer(ret).Bytes()
		buf = append(buf, '\n')
		params.LogAndResponseOK(w, r, f.ErrorLog, params.StatusOK, buf, string(bodyBytes))
		return params.ErrorGuard
	case "eth_maxPriorityFeePerGas":
		gasTipCap, err := getGasTipCap(r.Context(), f.config.RawURL)
		if err != nil {
			params.LogRequestWarn(r, f.ErrorLog, params.StatusInternalError, err.Error())
			gasTipCap = big.NewInt(0).Set(f.config.MinGasTipCapInWEI)
		} else {
			if gasTipCap.Cmp(f.config.MinGasTipCapInWEI) < 0 {
				gasTipCap.Set(f.config.MinGasTipCapInWEI)
			}
			if f.config.MaxGasTipCapInWEI != nil && gasTipCap.Cmp(f.config.MaxGasTipCapInWEI) > 0 {
				gasTipCap.Set(f.config.MaxGasTipCapInWEI)
			}
		}

		response := params.JSONSuccessResponse{
			Version: params.JSONRPCVersion,
			ID:      in.ID,
			Result:  hexutil.EncodeBig(gasTipCap),
		}
		ret, err := json.Marshal(response)
		if err != nil {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusGasPriceError, in.ID, string(bodyBytes))
			return params.ErrorInternalError
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		buf := bytes.NewBuffer(ret).Bytes()
		buf = append(buf, '\n')
		params.LogAndResponseOK(w, r, f.ErrorLog, params.StatusOK, buf, string(bodyBytes))
		return params.ErrorGuard
	}

	params.LogRequestInfo(r, f.ErrorLog, params.StatusOK, string(bodyBytes))
	return nil
}

func decodeRawTransaction(raw json.RawMessage) (string, error) {
	var hexParams [1]string
	err := json.Unmarshal(raw, &hexParams)
	if err != nil {
		return "", err
	}
	return hexParams[0], nil
}

func decodeTransaction(hexParam string) (*types.Transaction, error) {
	encodedTx, err := hexutil.Decode(hexParam)
	if err != nil {
		return nil, err
	}
	if len(encodedTx) <= 0 {
		return nil, fmt.Errorf("decode transaction error")
	}
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encodedTx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (f *Filter) checkTransaction(tx *types.Transaction) int {
	status := params.StatusOK

	if tx == nil {
		return params.StatusInternalError
	}

	// ChainID
	// if tx.ChainId().Uint64() != f.config.ChainID {
	if tx.ChainId().Cmp(f.config.ChainID) != 0 {
		return params.StatusIllegalChainID
	}

	// gas price
	if tx.GasPrice().Cmp(f.config.MinGasPriceInWEI) < 0 {
		return params.StatusGasPriceError
	}
	if f.config.MaxGasPriceInWEI != nil && tx.GasPrice().Cmp(f.config.MaxGasPriceInWEI) > 0 {
		return params.StatusGasPriceError
	}

	if tx.GasTipCap() != nil {
		if tx.GasTipCap().Cmp(f.config.MinGasTipCapInWEI) < 0 {
			return params.StatusGasPriceError
		}
		if f.config.MaxGasTipCapInWEI != nil && tx.GasTipCap().Cmp(f.config.MaxGasTipCapInWEI) > 0 {
			return params.StatusGasPriceError
		}
	}

	// gas limit
	if tx.Gas() < f.config.MinGasLimit {
		return params.StatusGasLimitError
	}
	if f.config.EnableMaxGasLimitVerify {
		if tx.Gas() > f.config.MaxGasLimit {
			return params.StatusGasLimitError
		}
	}

	// call lua
	if f.config.EnableLuaFilter {
		hash := tx.Hash()
		to := tx.To()
		if f.config.EnableFromCheck {
			signer := types.NewLondonSigner(f.config.ChainID)
			from, err := signer.Sender(tx)
			if err != nil {
				log.Errorln("Sender error:", err)
				return params.StatusSignatureVerifyFailed
			}
			if from == (common.Address{}) {
				return params.StatusEmptyFromAddress
			}
			code, err := f.CheckTx(&hash, &from, to)
			if err != nil {
				log.Errorln("CheckTx error:", err)
			}
			return code
		} else {
			code, err := f.CheckTx(&hash, nil, to)
			if err != nil {
				log.Errorln("CheckTx error:", err)
			}
			return code
		}
	}

	// to
	if tx.To() == nil {
		if f.config.DisableContractCreate {
			return params.StatusCreateContractNotAllowed
		}
	} else {
		if len(f.config.ToBlackList) > 0 {
			to := *tx.To()
			if _, ok := f.config.ToBlackList[to]; ok {
				return params.StatusToAddressBlackList
			}
		}
	}

	// value
	if tx.Value().Cmp(f.config.MaxValueInWEI) > 0 {
		status = params.StatusValueTooLarge
		if f.config.EnableMaxValueVerify {
			return params.StatusValueTooLargeError
		}
	}

	// tx hash
	if len(f.config.TxHashBlackList) > 0 {
		hash := tx.Hash()
		if hash == (common.Hash{}) {
			return params.StatusTransactionHashNil
		}
		if _, ok := f.config.TxHashBlackList[hash]; ok {
			return params.StatusTransactionHashBlackList
		}
	}

	// from
	if f.config.EnableWhitelistDB || len(f.config.FromBlackList) > 0 {
		signer := NewEIP155Signer(f.config.ChainID)
		from, err := signer.Sender(tx)
		if err != nil {
			return params.StatusSignatureVerifyFailed
		}
		if from == (common.Address{}) {
			return params.StatusEmptyFromAddress
		}

		// check from black
		if len(f.config.FromBlackList) > 0 {
			if _, ok := f.config.FromBlackList[from]; ok {
				return params.StatusFromAddressBlackList
			}
		}

		if f.config.EnableWhitelistDB {
			// from is whitelist or from/to match
			if tx.To() == nil {
				return params.StatusFromAddressBlackList
			}
			if ok := f.isWhitelistTx(from, *tx.To()); !ok {
				return params.StatusFromAddressBlackList
			}
		}
	}

	return status
}

func (f *Filter) CheckTx(hash *common.Hash, from, to *common.Address) (int, error) {
	if !f.config.EnableLuaFilter {
		return params.StatusInternalError, nil // StatusInternalError
	}

	ls := lua.NewState()
	defer ls.Close()
	ls.PreloadModule("mysql", mysql.Loader)
	if err := ls.DoFile(f.config.LuaFile); err != nil {
		return params.StatusLuaFileNotDoError, fmt.Errorf("DoFile error: %v", err)
	}

	if f.config.LuaCallFunctionName == "" {
		return params.StatusLuaCallError, errors.New("function name is empty")
	}

	var (
		err   error
		lHash lua.LValue
		lFrom lua.LValue
		lTo   lua.LValue
	)
	if hash == nil {
		lHash = lua.LNil
	} else {
		lHash = lua.LString(strings.ToLower(hash.Hex()[2:]))
	}
	if from == nil {
		lFrom = lua.LNil
	} else {
		lFrom = lua.LString(strings.ToLower(from.Hex()[2:]))
	}
	if to == nil {
		lTo = lua.LNil
	} else {
		lTo = lua.LString(strings.ToLower(to.Hex()[2:]))
	}

	err = ls.CallByParam(lua.P{
		Fn:      ls.GetGlobal(f.config.LuaCallFunctionName),
		NRet:    1,
		Protect: true,
	}, lHash, lFrom, lTo)
	if err != nil {
		return params.StatusLuaCallError, fmt.Errorf("call error: %v", err)
	}
	ret := ls.Get(-1) // returned value
	ls.Pop(1)         // remove received value

	if ret.Type() != lua.LTNumber {
		return params.StatusLuaReturnError, errors.New("filter return error")
	}

	return strconv.Atoi(ret.String())
}

func getGasPrice(ctx context.Context, rpcurl string) (*big.Int, error) {
	client, err := ethclient.Dial(rpcurl)
	if err != nil {
		return nil, err
	}
	return client.SuggestGasPrice(ctx)
}

func getGasTipCap(ctx context.Context, rpcurl string) (*big.Int, error) {
	client, err := ethclient.Dial(rpcurl)
	if err != nil {
		return nil, err
	}
	return client.SuggestGasTipCap(ctx)
}
