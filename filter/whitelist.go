package filter

import (
	"github.com/ethereum/go-ethereum/common"
	db "github.com/upper/db/v4"
)

func OpenDatabase(adapterName string, settings db.ConnectionURL) (db.Session, error) {
	sDB, err := db.Open(adapterName, settings)
	if err != nil {
		return nil, err
	}

	sdb := sDB.SQL()

	_, err = sdb.Exec("set time_zone='+00:00'")
	if err != nil {
		return nil, err
	}
	_, err = sdb.Exec("set tx_isolation='READ-COMMITTED'")
	if err != nil {
		return nil, err
	}

	return sDB, nil
}

type Account struct {
	ID          uint64 `db:"id"`
	Address     string `db:"address"`
	Escrow      string `db:"escrow"`
	IsSuperNode bool   `db:"supernode"`
}

func (f *Filter) isWhitelistTx(from, to common.Address) bool {
	sdb, err := OpenDatabase(f.config.WhitelistDBAdapterName, f.config.WhitelistDBSettings)
	if err != nil {
		return false
	}
	defer sdb.Close()

	var account Account
	err = sdb.SQL().Select("escrow", "supernode").From("accounts").Where(
		"address", from.Hex()[2:]).One(&account)
	if err != nil {
		return false
	}

	if account.IsSuperNode {
		return true
	}

	if !common.IsHexAddress(account.Escrow) {
		return false
	}
	if common.HexToAddress(account.Escrow) == to {
		return true
	}

	return false
}
