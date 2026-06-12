package main

import (
	"fmt"
	bentry "go-ddd-template/internal/domain/block"
	bchain "go-ddd-template/internal/domain/chain"
	"strconv"
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

		pow := bentry.NewProofOfWork(block)
		fmt.Printf("PoW hash: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}
