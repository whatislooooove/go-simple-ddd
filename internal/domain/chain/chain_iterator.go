package chain

import (
	bitem "go-ddd-template/internal/domain/block"

	"github.com/dgraph-io/badger"
)

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.DataBase}
}

func (iter *BlockChainIterator) Next() *bitem.Block {
	var block *bitem.Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		bitem.HandleError(err)
		encodedBlock, err := item.ValueCopy(nil)
		block = bitem.Deserialize(encodedBlock)

		return err
	})
	bitem.HandleError(err)

	iter.CurrentHash = block.PrevHash

	return block
}
