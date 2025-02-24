package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

var (
	big10        = big.NewInt(10)
	big1NEWInWEI = new(big.Int).Exp(big10, big.NewInt(18), nil)

	errClientNil           = errors.New("Failed to connect to the NewChain client")
	errCliNil              = errors.New("Cli error")
	errCliTranNil          = errors.New("Cli tran error")
	errBigSetString        = errors.New("conver string to big error")
	errLessThan0Wei        = errors.New("The transaction amount is less than 0 WEI")
	errIllegalAmount       = errors.New("Illegal Amount")
	errIllegalUnit         = errors.New("Illegal Unit")
	errRequiredFromAddress = errors.New(`required flag(s) "from" not set`)
)

var IsDecimalString = regexp.MustCompile(`^[1-9]\d*$|^0$|^0\.\d*$|^[1-9](\d)*\.(\d)*$`).MatchString

func showSuccess(msg string, args ...interface{}) {
	fmt.Printf(msg+"\n", args...)
}

func showError(fields logrus.Fields, msg string, args ...interface{}) {
	logrus.WithFields(fields).Errorf(msg, args...)
}

// getPassPhrase retrieves the password associated with an account,
// requested interactively from the user.
func getPassPhrase(promptStr string, confirmation bool) (string, error) {
	// prompt the user for the password
	if promptStr != "" {
		fmt.Println(promptStr)
	}
	password, err := prompt.Stdin.PromptPassword("Enter passphrase (empty for no passphrase): ")
	if err != nil {
		return "", err
	}
	if confirmation {
		confirm, err := prompt.Stdin.PromptPassword("Enter same passphrase again: ")
		if err != nil {
			return "", err
		}
		if password != confirm {
			return "", fmt.Errorf("Passphrases do not match")
		}
	}
	return password, nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// DenominationString is for denomination string
//const DenominationString = "Available unit: Wei, Ada, Babbage, Shannon, Szabo, Finney, Ether, Einstein, Douglas, Gwei"
const DenominationString = "Available unit: NEW, WEI"

// DenominationList is array for denomination string
// var DenominationList = []string{"Wei", "Ada", "Babbage", "Shannon", "Szabo", "Finney", "Ether", "Einstein", "Douglas", "Gwei"}
var DenominationList = []string{"NEW", "WEI"}

func getAmountWei(amountStr, unit string) (*big.Int, error) {
	switch unit {
	case "NEW":
		index := strings.IndexByte(amountStr, '.')
		if index <= 0 {
			amountWei, ok := new(big.Int).SetString(amountStr, 10)
			if !ok {
				return nil, errBigSetString
			}
			return new(big.Int).Mul(amountWei, big1NEWInWEI), nil
		}
		amountStrInt := amountStr[:index]
		amountStrDec := amountStr[index+1:]
		amountStrDecLen := len(amountStrDec)
		if amountStrDecLen > 18 {
			return nil, errIllegalAmount
		}
		amountStrInt = amountStrInt + strings.Repeat("0", 18)
		amountStrDec = amountStrDec + strings.Repeat("0", 18-amountStrDecLen)

		amountStrIntBig, ok := new(big.Int).SetString(amountStrInt, 10)
		if !ok {
			return nil, errBigSetString
		}
		amountStrDecBig, ok := new(big.Int).SetString(amountStrDec, 10)
		if !ok {
			return nil, errBigSetString
		}

		return new(big.Int).Add(amountStrIntBig, amountStrDecBig), nil
	case "WEI":
		amountWei, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return nil, errBigSetString
		}
		return amountWei, nil
	}

	return nil, errIllegalUnit
}

func getWeiAmountTextUnitByUnit(amount *big.Int, unit string) string {
	if amount == nil {
		return "0 WEI"
	}
	amountStr := amount.String()
	amountStrLen := len(amountStr)
	if unit == "" {
		if amountStrLen <= 18 {
			// show in WEI
			unit = "WEI"
		} else {
			unit = "NEW"
		}
	}

	return fmt.Sprintf("%s %s", getWeiAmountTextByUnit(amount, unit), unit)
}

func getWeiAmountTextByUnit(amount *big.Int, unit string) string {
	if amount == nil {
		return "0"
	}
	amountStr := amount.String()
	amountStrLen := len(amountStr)

	switch unit {
	case "NEW":
		var amountStrDec, amountStrInt string
		if amountStrLen <= 18 {
			amountStrDec = strings.Repeat("0", 18-amountStrLen) + amountStr
			amountStrInt = "0"
		} else {
			amountStrDec = amountStr[amountStrLen-18:]
			amountStrInt = amountStr[:amountStrLen-18]
		}
		amountStrDec = strings.TrimRight(amountStrDec, "0")
		if len(amountStrDec) <= 0 {
			return amountStrInt
		}
		return amountStrInt + "." + amountStrDec

	case "WEI":
		return amountStr
	}

	return errIllegalUnit.Error()
}

func showTransactionReceipt(url, txStr string) {
	sendJSONPostAndShow(url, "eth_getTransactionReceipt", txStr)
}

func sendJSONPostAndShow(url, method string, args ...interface{}) {
	ctx := context.Background()
	client, err := rpc.DialContext(ctx, url)
	if err != nil {
		fmt.Println("DialContext: ", err)
		return
	}
	var raw json.RawMessage
	if err := client.CallContext(ctx, &raw, method, args...); err != nil {
		fmt.Println("CallContext Error: ", err)
		return
	}
	rawStr, err := json.MarshalIndent(raw, "", "\t")
	if err != nil {
		fmt.Println("JSON marshaling failed: ", err)
		return
	}
	fmt.Printf("%s\n", rawStr)

	return
}

func getFaucet(rpcURL, address string) {
	url := fmt.Sprintf("%s/faucet?address=%s", rpcURL, address)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get error: %v\n", err)
		return
	}
	if resp.StatusCode == 200 {
		fmt.Printf("Get faucet for %s\n", address)
	}
}
