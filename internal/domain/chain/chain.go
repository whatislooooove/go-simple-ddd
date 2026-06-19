package chain

import (
	"encoding/hex"
	"fmt"
	"go-ddd-template/internal/domain/block"
	"go-ddd-template/internal/domain/transaction"
	"os"
	runtime2 "runtime"
)
import "github.com/dgraph-io/badger"

const (
	dbPath      = "./tmp/chain.db" // поменять потом
	dbFile      = "./tmp/chain.db/MANIFEST"
	genesisData = "First transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	DataBase *badger.DB
}

func (chain *BlockChain) AddBlock(transactions []*transaction.Transaction) {
	var lastHash []byte

	err := chain.DataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last-hash"))
		block.HandleError(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	block.HandleError(err)

	newBlock := block.CreateBlock(transactions, lastHash)

	err = chain.DataBase.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		block.HandleError(err)

		err = txn.Set([]byte("last-hash"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	block.HandleError(err)
}

func DBExists() bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func InitBlockChain(address string) *BlockChain {
	if DBExists() {
		fmt.Println("Blockchain already exists")
		runtime2.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	block.HandleError(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := transaction.CoinbaseTx(address, genesisData)
		genesis := block.CreateGenesisBlock(cbtx)
		fmt.Println("Genesis block created")

		err := txn.Set(genesis.Hash, genesis.Serialize())
		block.HandleError(err)
		err = txn.Set([]byte("last-hash"), genesis.Hash)
		block.HandleError(err)

		lastHash = genesis.Hash

		return err
	})

	block.HandleError(err)

	return &BlockChain{lastHash, db}
}

func ContinueBlockChain(address string) *BlockChain {
	if DBExists() {
		fmt.Println("Blockchain already exists")
		runtime2.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	block.HandleError(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last-hash"))
		block.HandleError(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	block.HandleError(err)

	return &BlockChain{lastHash, db}
}

func (chain *BlockChain) FindUnspentTransactions(address string) []transaction.Transaction {
	var unspentTxs []transaction.Transaction

	spentTXOs := make(map[string][]int)
	iter := chain.Iterator()

	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxId := hex.EncodeToString(in.ID)
						spentTXOs[inTxId] = append(spentTXOs[inTxId], in.Out)
					}
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

func (chain *BlockChain) FindUTXO(address string) []transaction.TxOutput {
	var UTXOs []transaction.TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}
