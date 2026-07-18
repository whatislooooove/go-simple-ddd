package main

import (
	"flag"
	"fmt"
	bentry "go-ddd-template/internal/domain/block"
	bchain "go-ddd-template/internal/domain/chain"
	"go-ddd-template/internal/domain/network"
	"go-ddd-template/internal/domain/shared"
	"go-ddd-template/internal/domain/transaction"
	wallet "go-ddd-template/internal/domain/wallet"
	"log"
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
	fmt.Println(" send -from FROM -to TO -amount AMOUNT -mine - Send amount of coins. Then -mine flag is set, mine off of this node")
	fmt.Println(" createwallet - Creates a new wallet")
	fmt.Println(" listaddresses - Lists the addresses in our wallet file")
	fmt.Println(" reindexutxo - Rebuilds the UTXO set")
	fmt.Println(" startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

func (cli *CommandLine) ValidateArgs() {
	if len(os.Args) < 2 {
		cli.Help()
		runtime2.Goexit()
	}
}

func (cli *CommandLine) StartNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	network.StartServer(nodeID, minerAddress)
}

func (cli *CommandLine) run() {
	// todo тоже стоит куда то вынести
	cli.ValidateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime2.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to createblockchain for")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")

	switch os.Args[1] {
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		shared.HandleError(err)
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		shared.HandleError(err)

	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		shared.HandleError(err)
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		shared.HandleError(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		shared.HandleError(err)

	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		shared.HandleError(err)

	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		shared.HandleError(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		shared.HandleError(err)

	default:
		cli.Help()
		runtime2.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime2.Goexit()
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime2.Goexit()
		}

		cli.createBlockChain(*createBlockchainAddress, nodeID)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount == 0 {
			sendCmd.Usage()
			runtime2.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			runtime2.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}
}

func (cli *CommandLine) printChain(nodeId string) {
	chain := bchain.ContinueBlockChain(nodeId)
	defer chain.DataBase.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previous hash: %x\n", block.PrevHash)
		fmt.Printf("Current Hash: %x\n", block.Hash)

		pow := bentry.NewProofOfWork(block)
		fmt.Printf("PoW hash: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address, nodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Invalid address to create blockchain")
	}
	chain := bchain.InitBlockChain(address, nodeId)
	defer chain.DataBase.Close()

	UTXOSet := bchain.UTXOSet{chain}
	UTXOSet.Reindex()

	fmt.Println("Finished: created new blockchain")
}

func (cli *CommandLine) getBalance(address, nodeId string) {
	// TODO:вынести проверку
	if !wallet.ValidateAddress(address) {
		log.Panic("Invalid address")
	}

	chain := bchain.ContinueBlockChain(nodeId)
	UTXOSet := bchain.UTXOSet{chain}
	defer chain.DataBase.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int, nodeId string, mineNow bool) {
	// TODO: исправить отправку токенов
	if !wallet.ValidateAddress(to) {
		log.Panic("Invalid address `to`")
	}
	if !wallet.ValidateAddress(from) {
		log.Panic("Invalid address `from`")
	}

	chain := bchain.ContinueBlockChain(nodeId)
	UTXOSet := bchain.UTXOSet{chain}
	defer chain.DataBase.Close()

	wallets, err := wallet.CreateWallets(nodeId)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx := bchain.NewTransaction(wallet, to, amount, &UTXOSet)
	if mineNow {
		cbTx := transaction.CoinbaseTx(from, "")
		txs := []*transaction.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("send tx")
	}

	fmt.Println("Success!")
}

func (cli *CommandLine) listAddresses(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeId)

	fmt.Printf("Created new wallet with address %s\n", address)
}

func (cli *CommandLine) reindexUTXO(nodeId string) {
	chain := bchain.ContinueBlockChain(nodeId)
	defer chain.DataBase.Close()

	UTXOSet := bchain.UTXOSet{chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done. There are %d transactions in the UTXO set\n", count)
}

func main() {
	defer os.Exit(0)
	cli := CommandLine{}
	cli.run()
}
