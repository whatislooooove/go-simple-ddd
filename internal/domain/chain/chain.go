package chain

import "go-ddd-template/internal/domain/block"

type BlockChain struct {
	Blocks []*block.Block
}

func (chain *BlockChain) AddBlock(data string) {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	newBlock := block.CreateBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, newBlock)
}

func InitBlockChain() *BlockChain {
	return &BlockChain{[]*block.Block{block.CreateGenesisBlock()}}
}
