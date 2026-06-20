package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"go-ddd-template/internal/domain/shared"
	"go-ddd-template/internal/domain/transaction"
)

type Block struct {
	Hash         []byte
	Transactions []*transaction.Transaction
	PrevHash     []byte
	Nonce        int
}

func (block *Block) HasTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

func CreateBlock(txs []*transaction.Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0}
	proof := NewProofOfWork(block)
	nonce, hash := proof.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func CreateGenesisBlock(coinbase *transaction.Transaction) *Block {
	return CreateBlock([]*transaction.Transaction{coinbase}, []byte{})
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)

	shared.HandleError(err)

	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)

	shared.HandleError(err)

	return &block
}
