package filter

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/params"
)

func TestCheckTransaction(t *testing.T) {

	func() {
		tx := types.NewTransaction(0, common.HexToAddress("0"), nil, 0, nil, nil)
		config := &params.DefaultConfig
		f, err := NewFilter(config, nil)
		if err != nil {
			log.Panic(err)
		}
		if status := f.checkTransaction(tx); status != params.StatusOK {
			log.Info(status)
		}
	}()

	func() {
		to := common.HexToAddress("0x1024")
		tx := types.NewTransaction(0, to, nil, 0, nil, nil)
		config := &params.DefaultConfig
		config.ToBlackList = map[common.Address]struct{}{to: {}}
		f, err := NewFilter(config, nil)
		if err != nil {
			log.Panic(err)
		}
		if status := f.checkTransaction(tx); status != params.StatusToAddressBlackList {
			log.Info(status)
		}
	}()
}
