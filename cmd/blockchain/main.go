package main

import (
	"fmt"
	bchain "go-ddd-template/internal/domain/chain"
)

func main() {
	chain := bchain.InitBlockChain()

	chain.AddBlock("First Block after Genesis")
	chain.AddBlock("Second Block after Genesis")
	chain.AddBlock("Third Block after Genesis")

	for _, block := range chain.Blocks {
		fmt.Printf("Previous hash: %x\n", block.PrevHash)
		fmt.Printf("Current Hash: %x\n", block.Hash)
		fmt.Printf("Data: %s\n\n", block.Data)
	}
}
