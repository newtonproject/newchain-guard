package params

import (
	"errors"
)

const (
	ContentType             = "application/json"
	MaxRequestContentLength = 1024 * 128
)

// NewChainGuard status codes
const (
	StatusOK            = 200
	StatusValueTooLarge = 211

	StatusBodyNilOrEmpty           = 410
	StatusWhiteListNotSet          = 411
	StatusInvalidJSONRequest       = 412
	StatusNoMethodParams           = 413
	StatusJSONRPCVersion           = 414
	StatusMethodNotWhitelist       = 415
	StatusMethodParamsNotMatch     = 416
	StatusMethodParamsTypeError    = 417
	StatusGasLimitError            = 418
	StatusCreateContractNotAllowed = 419
	StatusFromAddressBlackList     = 420
	StatusEmptyFromAddress         = 421
	StatusIllegalChainID           = 422
	StatusSignatureVerifyFailed    = 423
	StatusGasPriceError            = 424
	StatusSecp256r1HalfN           = 425
	StatusTransactionHashNil       = 426
	StatusTransactionHashBlackList = 427
	StatusToAddressBlackList       = 428
	StatusReadBodyError            = 429
	StatusFilterNoConfig           = 430
	StatusMethodNotAllowed         = 431
	StatusDecodeTransactionError   = 432
	StatusValueTooLargeError       = 433
	StatusIPCDialError             = 434
	StatusIPCWriteError            = 435
	StatusIPCReadError             = 436

	StatusLuaInitError      = 440
	StatusLuaCallError      = 441
	StatusLuaReturnError    = 442
	StatusLuaFileNotDoError = 443

	StatusInternalError = 500
)

var statusText = map[int]string{
	StatusOK:            "OK",
	StatusValueTooLarge: "value too large",

	StatusBodyNilOrEmpty:           "body nil or empty",
	StatusWhiteListNotSet:          "whitelist not set",
	StatusInvalidJSONRequest:       "invalid JSON request",
	StatusNoMethodParams:           "no method and/or jsonrpc attribute",
	StatusJSONRPCVersion:           "jsonrpc version not supported",
	StatusMethodNotWhitelist:       "jsonrpc method is not Whitelist",
	StatusMethodParamsNotMatch:     "method params not match",
	StatusMethodParamsTypeError:    "method params type not string",
	StatusGasLimitError:            "gasLimit error",
	StatusCreateContractNotAllowed: "create contract not allowed",
	StatusFromAddressBlackList:     "black list from address",
	StatusEmptyFromAddress:         "empty from address",
	StatusIllegalChainID:           "illegal chainID",
	StatusSignatureVerifyFailed:    "signature verification failed",
	StatusGasPriceError:            "gas price is less than the minimum value",
	StatusSecp256r1HalfN:           "signature s is big than secp256r1HalfN",
	StatusTransactionHashNil:       "empty transaction hash",
	StatusTransactionHashBlackList: "black list transaction hash",
	StatusToAddressBlackList:       "black list to address",
	StatusReadBodyError:            "get body error",
	StatusFilterNoConfig:           "filter no config",
	StatusMethodNotAllowed:         "the method is not available",
	StatusDecodeTransactionError:   "decode transaction from hex string error",
	StatusValueTooLargeError:       "value too large error",
	StatusIPCDialError:             "IPC dial error",
	StatusIPCWriteError:            "IPC write error",
	StatusIPCReadError:             "IPC read error",

	StatusLuaInitError:      "lua init error",
	StatusLuaCallError:      "lua call error",
	StatusLuaReturnError:    "lua return error",
	StatusLuaFileNotDoError: "lua do file error",

	StatusInternalError: "Internal Error",
}

var statusError = map[int]error{
	StatusValueTooLarge: errors.New(statusText[StatusValueTooLarge]),

	StatusBodyNilOrEmpty:           errors.New(statusText[StatusValueTooLarge]),
	StatusWhiteListNotSet:          errors.New(statusText[StatusValueTooLarge]),
	StatusInvalidJSONRequest:       errors.New(statusText[StatusValueTooLarge]),
	StatusNoMethodParams:           errors.New(statusText[StatusValueTooLarge]),
	StatusJSONRPCVersion:           errors.New(statusText[StatusValueTooLarge]),
	StatusMethodNotWhitelist:       errors.New(statusText[StatusValueTooLarge]),
	StatusMethodParamsNotMatch:     errors.New(statusText[StatusValueTooLarge]),
	StatusMethodParamsTypeError:    errors.New(statusText[StatusValueTooLarge]),
	StatusGasLimitError:            errors.New(statusText[StatusValueTooLarge]),
	StatusCreateContractNotAllowed: errors.New(statusText[StatusValueTooLarge]),
	StatusFromAddressBlackList:     errors.New(statusText[StatusValueTooLarge]),
	StatusEmptyFromAddress:         errors.New(statusText[StatusValueTooLarge]),
	StatusIllegalChainID:           errors.New(statusText[StatusValueTooLarge]),
	StatusSignatureVerifyFailed:    errors.New(statusText[StatusValueTooLarge]),
	StatusGasPriceError:            errors.New(statusText[StatusValueTooLarge]),
	StatusSecp256r1HalfN:           errors.New(statusText[StatusValueTooLarge]),
	StatusReadBodyError:            errors.New(statusText[StatusValueTooLarge]),
	StatusFilterNoConfig:           errors.New(statusText[StatusValueTooLarge]),
	StatusMethodNotAllowed:         errors.New(statusText[StatusMethodNotAllowed]),
	StatusDecodeTransactionError:   errors.New(statusText[StatusDecodeTransactionError]),
	StatusValueTooLargeError:       errors.New(statusText[StatusValueTooLargeError]),
	StatusIPCDialError:             errors.New(statusText[StatusIPCDialError]),
	StatusIPCWriteError:            errors.New(statusText[StatusIPCWriteError]),
	StatusIPCReadError:             errors.New(statusText[StatusIPCReadError]),

	StatusLuaInitError:   errors.New(statusText[StatusLuaInitError]),
	StatusLuaCallError:   errors.New(statusText[StatusLuaCallError]),
	StatusLuaReturnError: errors.New(statusText[StatusLuaReturnError]),

	StatusInternalError: errors.New(statusText[StatusInternalError]),
}

var (
	ErrorGuard         = errors.New("ErrorGuard")
	ErrorInternalError = statusError[StatusInternalError]
)

// GetStatusText is thread-safe for read-only map
func GetStatusText(status int) string {
	return statusText[status]
}

// GetStatusError is thread-safe for read-only map
func GetStatusError(status int) error {
	return statusError[status]
}
