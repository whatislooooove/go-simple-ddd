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

func (chain *BlockChain) AddBlock(transactions []*transaction.Transaction) {
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
	if DBExists() != false {
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

func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []transaction.Transaction {
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
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) {
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

func (chain *BlockChain) FindUTXO(pubKeyHash []byte) []transaction.TxOutput {
	var UTXOs []transaction.TxOutput
	unspentTransactions := chain.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
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

func NewTransaction(from, to string, amount int, chain *BlockChain) *transaction.Transaction {
	var inputs []transaction.TxInput
	var outputs []transaction.TxOutput

	wallets, err := wallet.CreateWallets()
	shared.HandleError(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := chain.FindSpendableOutputs(pubKeyHash, amount)

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
	chain.SignTransaction(&tx, w.PrivateKey)

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
	prevTXs := make(map[string]transaction.Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		shared.HandleError(err)

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}
