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

func (chain *BlockChain) AddBlock(transactions []*transaction.Transaction) *block.Block {
	var lastHash []byte

	err := chain.DataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last-hash"))
		shared.HandleError(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	shared.HandleError(err)

	newBlock := block.CreateBlock(transactions, lastHash)

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

func ContinueBlockChain(address string) *BlockChain {
	if !DBExists() {
		fmt.Println("Blockchain already exists")
		runtime2.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
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

func NewTransaction(from, to string, amount int, UTXO *UTXOSet) *transaction.Transaction {
	var inputs []transaction.TxInput
	var outputs []transaction.TxOutput

	wallets, err := wallet.CreateWallets()
	shared.HandleError(err)
	w := wallets.GetWallet(from)
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
