package main

import (
	"flag"
	"fmt"
	bentry "go-ddd-template/internal/domain/block"
	bchain "go-ddd-template/internal/domain/chain"
	"os"
	runtime2 "runtime"
	"strconv"
)

type CommandLine struct {
	blockchain *bchain.BlockChain
}

func (cli *CommandLine) Help() {
	fmt.Println("Usage:")
	fmt.Println(" add -block {block_data} - add block to the chain")
	fmt.Println(" print - print the blocks in the chain")
}

func (cli *CommandLine) ValidateArgs() {
	if len(os.Args) < 2 {
		cli.Help()
		runtime2.Goexit()
	}
}

func (cli *CommandLine) addBlock(data string) {
	cli.blockchain.AddBlock(data)
	fmt.Println("Added new block")
}

func (cli *CommandLine) run() {
	cli.ValidateArgs()

	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "Block data")

	switch os.Args[1] {
	case "add":
		err := addBlockCmd.Parse(os.Args[2:])
		bentry.HandleError(err)

	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		bentry.HandleError(err)

	default:
		cli.Help()
		runtime2.Goexit()
	}

	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime2.Goexit()
		}
		cli.addBlock(*addBlockData)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func (cli *CommandLine) printChain() {
	iter := cli.blockchain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previous hash: %x\n", block.PrevHash)
		fmt.Printf("Current Hash: %x\n", block.Hash)
		fmt.Printf("Data: %s\n", block.Data)

		pow := bentry.NewProofOfWork(block)
		fmt.Printf("PoW hash: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func main() {
	defer os.Exit(0)
	chain := bchain.InitBlockChain()
	defer chain.DataBase.Close()

	cli := CommandLine{blockchain: chain}
	cli.run()
}
