package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/notify"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/params"
)

func (f *Filter) HandleJSONRequest(w http.ResponseWriter, r *http.Request, bodyBytes []byte) (*params.LogRequest, error) {
	var rawmsg json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawmsg); err != nil {
		return nil, err
	}

	msg, batch := parseMessage(rawmsg)
	if batch {
		resList, statusList, err := f.checkJSONBatchRequest(r, msg)
		if err == params.ErrorGuard {
			responseBytes, fErr := json.Marshal(resList)
			if fErr != nil {
				return nil, fErr
			}
			params.LogBatchRequestAndResponse(w, r, f.ErrorLog, statusList, string(bodyBytes), responseBytes)
			return &params.LogRequest{
				R:          r,
				Logger:     f.ErrorLog,
				StatusList: statusList,
				ReqBody:    bodyBytes,
				IsBatch:    false,
			}, params.ErrorGuard
		} else if err != nil {
			return &params.LogRequest{
				R:          r,
				Logger:     f.ErrorLog,
				StatusList: statusList,
				ReqBody:    bodyBytes,
				IsBatch:    false,
			}, params.ErrorGuard
		}

		// params.LogBatchRequestInfo(r, f.ErrorLog, statusList, string(bodyBytes), "")

		return &params.LogRequest{
			R:          r,
			Logger:     f.ErrorLog,
			StatusList: statusList,
			ReqBody:    bodyBytes,
			IsBatch:    true,
		}, nil

	} else {
		if err := f.checkJSONRequest(w, r, msg[0]); err != nil {
			return &params.LogRequest{
				R:          r,
				Logger:     f.ErrorLog,
				StatusList: []int{params.StatusOK},
				ReqBody:    bodyBytes,
				IsBatch:    false,
			}, params.ErrorGuard
		}
		// params.LogRequestInfo(r, f.ErrorLog, params.StatusOK, in.String())
	}

	return &params.LogRequest{
		R:          r,
		Logger:     f.ErrorLog,
		StatusList: []int{params.StatusOK},
		ReqBody:    bodyBytes,
		IsBatch:    false,
	}, nil
}

func (f *Filter) checkJSONBatchRequest(r *http.Request, ins []*jsonrpcMessage) ([]interface{}, []int, error) {
	var (
		gpList       []int
		resList      []interface{}
		statusList   []int
		requestError bool
	)

	for i, in := range ins {
		if len(strings.Split(in.Method, serviceMethodSeparator)) != 2 {
			resList = append(resList, params.JSONErrResponse{
				Version: params.JSONRPCVersion,
				ID:      in.ID,
				Error: params.JSONError{
					Code:    params.StatusMethodNotAllowed,
					Message: fmt.Sprintf("%s - %d", params.ErrorInternalError.Error(), params.StatusMethodNotAllowed),
				},
			})
			statusList = append(statusList, params.StatusMethodNotAllowed)
			requestError = true
			continue
		}

		if !f.inWhitelistMethod(in.Method) {
			resList = append(resList, params.JSONErrResponse{
				Version: params.JSONRPCVersion,
				ID:      in.ID,
				Error: params.JSONError{
					Code:    params.StatusMethodNotWhitelist,
					Message: fmt.Sprintf("%s - %d", params.ErrorInternalError.Error(), params.StatusMethodNotWhitelist),
				},
			})
			statusList = append(statusList, params.StatusMethodNotWhitelist)
			requestError = true
			continue
		}

		switch in.Method {
		case "eth_sendRawTransaction":
			hexParam, err := decodeRawTransaction(in.Params)
			if err != nil {
				resList = append(resList, params.JSONErrResponse{
					Version: params.JSONRPCVersion,
					ID:      in.ID,
					Error: params.JSONError{
						Code: params.StatusDecodeTransactionError,
						Message: fmt.Sprintf("%s - %d",
							params.ErrorInternalError.Error(), params.StatusDecodeTransactionError),
					},
				})
				statusList = append(statusList, params.StatusDecodeTransactionError)
				requestError = true
				continue
			}
			tx, err := decodeTransaction(hexParam)
			if err != nil {
				resList = append(resList, params.JSONErrResponse{
					Version: params.JSONRPCVersion,
					ID:      in.ID,
					Error: params.JSONError{
						Code: params.StatusDecodeTransactionError,
						Message: fmt.Sprintf("%s - %d",
							params.ErrorInternalError.Error(), params.StatusDecodeTransactionError),
					},
				})
				statusList = append(statusList, params.StatusDecodeTransactionError)
				requestError = true
				continue
			}
			status := f.checkTransaction(tx)
			if status == params.StatusOK {
				// params.LogRequestInfo(r, f.ErrorLog, params.StatusOK, in.String())
				resList = append(resList, params.JSONSuccessResponse{
					Version: params.JSONRPCVersion,
					ID:      in.ID,
				})
				statusList = append(statusList, status)
				continue
			} else if status == params.StatusValueTooLarge {
				params.LogRequestWarn(r, f.ErrorLog, params.StatusValueTooLarge, in.String())
			} else {
				resList = append(resList, params.JSONErrResponse{
					Version: params.JSONRPCVersion,
					ID:      in.ID,
					Error: params.JSONError{
						Code: status,
						Message: fmt.Sprintf("%s - %d",
							params.ErrorInternalError.Error(), status),
					},
				})
				statusList = append(statusList, status)
				requestError = true
				continue
			}

			// transaction pass
			if f.config.EnableActiveMQ {
				notify.AddTx(hexParam)
			}

			continue
		case "eth_gasPrice", "eth_maxPriorityFeePerGas":
			resList = append(resList, params.JSONSuccessResponse{
				Version: params.JSONRPCVersion,
				ID:      in.ID,
			})
			statusList = append(statusList, params.StatusOK)
			gpList = append(gpList, i)
		// TODO: add gas price check
		// gasPrice, err := getGasPrice(r.Context(), f.config.RawURL)
		// if err != nil {
		// 	params.LogRequestWarn(r, f.ErrorLog, params.StatusInternalError, err.Error())
		// 	gasPrice = big.NewInt(0).Set(f.config.MinGasPriceInWEI)
		// } else {
		// 	if gasPrice.Cmp(f.config.MinGasPriceInWEI) < 0 {
		// 		gasPrice.Set(f.config.MinGasPriceInWEI)
		// 	}
		// 	if f.config.MaxGasPriceInWEI != nil && gasPrice.Cmp(f.config.MaxGasPriceInWEI) > 0 {
		// 		gasPrice.Set(f.config.MaxGasPriceInWEI)
		// 	}
		// }
		//
		// response := params.JSONSuccessResponse{
		// 	Version: params.JSONRPCVersion,
		// 	ID:      in.ID,
		// 	Result:  hexutil.EncodeBig(gasPrice),
		// }
		// ret, err := json.Marshal(response)
		// if err != nil {
		// 	params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusGasPriceError, in.ID, in.String())
		// 	return params.ErrorInternalError
		// }
		// w.Header().Set("Content-Type", "application/json")
		// buf := bytes.NewBuffer(ret).Bytes()
		// buf = append(buf, '\n')
		// params.LogAndResponseOK(w, r, f.ErrorLog, params.StatusOK, buf, in.String())
		// return params.ErrorGuard
		default:
			resList = append(resList, params.JSONSuccessResponse{
				Version: params.JSONRPCVersion,
				ID:      in.ID,
			})
			statusList = append(statusList, params.StatusOK)
		}
	}

	if requestError {
		return resList, statusList, params.ErrorGuard
	}

	return resList, statusList, nil
}

func (f *Filter) checkJSONRequest(w http.ResponseWriter, r *http.Request, in *jsonrpcMessage) error {
	if len(strings.Split(in.Method, serviceMethodSeparator)) != 2 {
		params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusMethodNotAllowed, in.ID, in.String())
		return params.GetStatusError(params.StatusMethodNotAllowed)
	}

	if !f.inWhitelistMethod(in.Method) {
		params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusMethodNotWhitelist, in.ID, in.String())
		return params.GetStatusError(params.StatusMethodNotWhitelist)
	}

	switch in.Method {
	case "eth_sendRawTransaction":
		hexParam, err := decodeRawTransaction(in.Params)
		if err != nil {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusDecodeTransactionError, in.ID, in.String())
			return params.GetStatusError(params.StatusDecodeTransactionError)
		}
		tx, err := decodeTransaction(hexParam)
		if err != nil {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusDecodeTransactionError, in.ID, in.String())
			return params.GetStatusError(params.StatusDecodeTransactionError)
		}
		status := f.checkTransaction(tx)
		if status == params.StatusOK {
			params.LogRequestInfo(r, f.ErrorLog, params.StatusOK, in.String())
		} else if status == params.StatusValueTooLarge {
			params.LogRequestWarn(r, f.ErrorLog, params.StatusValueTooLarge, in.String())
		} else {
			params.LogAndResponseJSONError(w, r, f.ErrorLog, status, in.ID, in.String())
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
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusGasPriceError, in.ID, in.String())
			return params.ErrorInternalError
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		buf := bytes.NewBuffer(ret).Bytes()
		buf = append(buf, '\n')
		params.LogAndResponseOK(w, r, f.ErrorLog, params.StatusOK, buf, in.String())
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
			params.LogAndResponseJSONError(w, r, f.ErrorLog, params.StatusGasPriceError, in.ID, in.String())
			return params.ErrorInternalError
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		buf := bytes.NewBuffer(ret).Bytes()
		buf = append(buf, '\n')
		params.LogAndResponseOK(w, r, f.ErrorLog, params.StatusOK, buf, in.String())
		return params.ErrorGuard
	}

	// params.LogRequestInfo(r, f.ErrorLog, params.StatusOK, in.String())
	return nil
}

// parseMessage parses raw bytes as a (batch of) JSON-RPC message(s). There are no error
// checks in this function because the raw message has already been syntax-checked when it
// is called. Any non-JSON-RPC messages in the input return the zero value of
// jsonrpcMessage.
func parseMessage(raw json.RawMessage) ([]*jsonrpcMessage, bool) {
	if !isBatch(raw) {
		msgs := []*jsonrpcMessage{{}}
		json.Unmarshal(raw, &msgs[0])
		return msgs, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.Token() // skip '['
	var msgs []*jsonrpcMessage
	for dec.More() {
		msgs = append(msgs, new(jsonrpcMessage))
		dec.Decode(&msgs[len(msgs)-1])
	}
	return msgs, true
}

// isBatch returns true when the first non-whitespace characters is '['
func isBatch(raw json.RawMessage) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}
