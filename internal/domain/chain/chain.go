package chain

import (
	"fmt"
	"go-ddd-template/internal/domain/block"
)
import "github.com/dgraph-io/badger"

const (
	dbPath = "./tmp/chain.db" // поменять потом
)

type BlockChain struct {
	LastHash []byte
	DataBase *badger.DB
}

func (chain *BlockChain) AddBlock(data string) {
	var lastHash []byte

	err := chain.DataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last-hash"))
		block.HandleError(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	block.HandleError(err)

	newBlock := block.CreateBlock(data, lastHash)

	err = chain.DataBase.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		block.HandleError(err)

		err = txn.Set([]byte("last-hash"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	block.HandleError(err)
}

func InitBlockChain() *BlockChain {
	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	block.HandleError(err)

	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("last-hash")); err == badger.ErrKeyNotFound {
			fmt.Println("Existing blockchain not found in database. Creating...")
			genesis := block.CreateGenesisBlock()

			fmt.Println("Genesis block created")
			err = txn.Set(genesis.Hash, genesis.Serialize())
			block.HandleError(err)

			err = txn.Set([]byte("last-hash"), genesis.Hash)
			lastHash = genesis.Hash

			return err
		} else {
			item, err := txn.Get([]byte("last-hash"))
			block.HandleError(err)

			lastHash, err = item.ValueCopy(nil)

			return err
		}
	})

	block.HandleError(err)

	return &BlockChain{lastHash, db}
}
