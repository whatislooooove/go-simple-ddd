package chain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"go-ddd-template/internal/domain/block"
	"go-ddd-template/internal/domain/shared"
	"go-ddd-template/internal/domain/transaction"
	"go-ddd-template/internal/domain/wallet"
	"log"
	"os"
	"path/filepath"
	runtime2 "runtime"
	"strings"
)
import "github.com/dgraph-io/badger"

const (
	dbPath      = "./tmp/chain_blocks_%s"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	DataBase *badger.DB
}

func (chain *BlockChain) MineBlock(transactions []*transaction.Transaction) *block.Block {
	var lastHash []byte
	var lastHeight int

	err := chain.DataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last-hash"))
		shared.HandleError(err)
		lastHash, err = item.ValueCopy(nil)

		item, err = txn.Get(lastHash)
		shared.HandleError(err)
		lastBlockData, _ := item.ValueCopy(nil)

		lastBlock := block.Deserialize(lastBlockData)

		lastHeight = lastBlock.Height

		return err
	})
	shared.HandleError(err)

	newBlock := block.CreateBlock(transactions, lastHash, lastHeight+1)

	err = chain.DataBase.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		shared.HandleError(err)

		err = txn.Set([]byte("last-hash"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	shared.HandleError(err)

	return newBlock
}

func (chain *BlockChain) AddBlock(blockItem *block.Block) {
	err := chain.DataBase.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(blockItem.Hash); err == nil {
			return nil
		}

		blockData := blockItem.Serialize()
		err := txn.Set(blockItem.Hash, blockData)
		shared.HandleError(err)

		item, err := txn.Get([]byte("lh"))
		shared.HandleError(err)
		lastHash, _ := item.ValueCopy(nil)

		item, err = txn.Get(lastHash)
		shared.HandleError(err)
		lastBlockData, _ := item.ValueCopy(nil)

		lastBlock := block.Deserialize(lastBlockData)

		if blockItem.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), blockItem.Hash)
			shared.HandleError(err)
			chain.LastHash = blockItem.Hash
		}

		return nil
	})
	shared.HandleError(err)
}

func (chain *BlockChain) GetBlock(blockHash []byte) (block.Block, error) {
	var blockItem block.Block

	err := chain.DataBase.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block is not found")
		} else {
			blockData, _ := item.ValueCopy(nil)

			blockItem = *block.Deserialize(blockData)
		}
		return nil
	})
	if err != nil {
		return blockItem, err
	}

	return blockItem, nil
}

func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := chain.Iterator()

	for {
		blockItem := iter.Next()

		blocks = append(blocks, blockItem.Hash)

		if len(blockItem.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (chain *BlockChain) GetBestHeight() int {
	var lastBlock block.Block

	err := chain.DataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		shared.HandleError(err)
		lastHash, _ := item.ValueCopy(nil)

		item, err = txn.Get(lastHash)
		shared.HandleError(err)
		lastBlockData, _ := item.ValueCopy(nil)

		lastBlock = *block.Deserialize(lastBlockData)

		return nil
	})
	shared.HandleError(err)

	return lastBlock.Height
}

func DBExists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}

	return true
}

func InitBlockChain(address, nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if DBExists(path) {
		fmt.Println("Blockchain already exists")
		runtime2.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path

	db, err := openDB(path, opts)
	shared.HandleError(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := transaction.CoinbaseTx(address, genesisData)
		genesis := block.CreateGenesisBlock(cbtx)
		fmt.Println("Genesis block created")

		err := txn.Set(genesis.Hash, genesis.Serialize())
		shared.HandleError(err)
		err = txn.Set([]byte("last-hash"), genesis.Hash)
		shared.HandleError(err)

		lastHash = genesis.Hash

		return err
	})

	shared.HandleError(err)

	return &BlockChain{lastHash, db}
}

func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if !DBExists(path) {
		fmt.Println("Blockchain already exists")
		runtime2.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path

	db, err := openDB(path, opts)
	shared.HandleError(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last-hash"))
		shared.HandleError(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	shared.HandleError(err)

	return &BlockChain{lastHash, db}
}

func (u UTXOSet) FindUnspentTransactions(pubKeyHash []byte) []transaction.TxOutput {
	var UTXOs []transaction.TxOutput
	db := u.BlockChain.DataBase

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			shared.HandleError(err)
			outs := transaction.DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})

	shared.HandleError(err)

	return UTXOs
}

func (chain *BlockChain) FindUTXO() map[string]transaction.TxOutputs {
	UTXO := make(map[string]transaction.TxOutputs)
	spentTXOs := make(map[string][]int)

	it := chain.Iterator()

	for {
		block := it.Next()

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
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return UTXO
}

func NewTransaction(w *wallet.Wallet, to string, amount int, UTXO *UTXOSet) *transaction.Transaction {
	var inputs []transaction.TxInput
	var outputs []transaction.TxOutput

	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("Not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		shared.HandleError(err)

		for _, out := range outs {
			input := transaction.TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	from := fmt.Sprintf("%s", w.MakeAddress())
	outputs = append(outputs, *transaction.NewTransactionOutput(amount, to))

	if acc > amount {
		outputs = append(outputs, *transaction.NewTransactionOutput(acc-amount, from))
	}

	tx := transaction.Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXO.BlockChain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func (bc *BlockChain) FindTransaction(ID []byte) (transaction.Transaction, error) {
	iter := bc.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return transaction.Transaction{}, errors.New("Transaction not found")
}

func (bc *BlockChain) SignTransaction(tx *transaction.Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]transaction.Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		shared.HandleError(err)

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (bc *BlockChain) VerifyTransaction(tx *transaction.Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]transaction.Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		shared.HandleError(err)

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}
