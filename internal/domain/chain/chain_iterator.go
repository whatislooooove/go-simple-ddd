package chain

import (
	bitem "go-ddd-template/internal/domain/block"
	"go-ddd-template/internal/domain/shared"

	"github.com/dgraph-io/badger"
)

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.DataBase}

	return iter
}

func (iter *BlockChainIterator) Next() *bitem.Block {
	var block *bitem.Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		shared.HandleError(err)
		encodedBlock, err := item.ValueCopy(nil)
		block = bitem.Deserialize(encodedBlock)

		return err
	})
	shared.HandleError(err)

	iter.CurrentHash = block.PrevHash

	return block
}
