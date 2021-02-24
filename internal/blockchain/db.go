package blockchain

import (
	"github.com/jiuzhou-zhao/bolt-client/pkg/boltc"
	"github.com/jiuzhou-zhao/bolt-client/pkg/db"
)

func NewDB() (db.DB, error) {
	return boltc.NewDBClient("http://127.0.0.1:12311", "test.db")
	// return boltw.NewDB(blockChainDataFile)
}

func DBRebuild4Debug() error {
	return boltc.DBRebuild4Debug("http://127.0.0.1:12311", "test.db")
	// return os.Remove(blockChainDataFile)
}
