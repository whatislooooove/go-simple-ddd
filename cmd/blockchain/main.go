package main

import (
	"flag"
	"fmt"
	bentry "go-ddd-template/internal/domain/block"
	bchain "go-ddd-template/internal/domain/chain"
	"go-ddd-template/internal/domain/transaction"
	"os"
	runtime2 "runtime"
	"strconv"
)

type CommandLine struct{}

func (cli *CommandLine) Help() {
	fmt.Println("Usage:")
	fmt.Println(" getbalance -address {address} - Get the balance for address")
	fmt.Println(" createblockchain -address {address} - Creates a new blockchain}")
	fmt.Println(" printchain - Prints the blocks in the blockchain")
	fmt.Println(" send -from {from} -to {to} -amount {amount} - Send amount from")
}

func (cli *CommandLine) ValidateArgs() {
	if len(os.Args) < 2 {
		cli.Help()
		runtime2.Goexit()
	}
}

func (cli *CommandLine) run() {
	cli.ValidateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to createblockchain for")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		bentry.HandleError(err)

	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		bentry.HandleError(err)

	case "send":
		err := sendCmd.Parse(os.Args[2:])
		bentry.HandleError(err)

	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		bentry.HandleError(err)

	default:
		cli.Help()
		runtime2.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime2.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime2.Goexit()
		}

		cli.createBlockChain(*createBlockchainAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount == 0 {
			sendCmd.Usage()
			runtime2.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func (cli *CommandLine) printChain() {
	chain := bchain.ContinueBlockChain("")
	defer chain.DataBase.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previous hash: %x\n", block.PrevHash)
		fmt.Printf("Current Hash: %x\n", block.Hash)

		pow := bentry.NewProofOfWork(block)
		fmt.Printf("PoW hash: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address string) {
	chain := bchain.InitBlockChain(address)
	chain.DataBase.Close()
	fmt.Println("Finished: created new blockchain")
}

func (cli *CommandLine) getBalance(address string) {
	chain := bchain.ContinueBlockChain(address)
	defer chain.DataBase.Close()

	balance := 0
	UTXOs := chain.FindUTXO(address)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	chain := bchain.ContinueBlockChain(from)
	defer chain.DataBase.Close()

	tx := transaction.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*transaction.Transaction{tx})
	fmt.Println("Success! Transaction executed")
}

func main() {
	defer os.Exit(0)
	cli := CommandLine{}
	cli.run()
}
