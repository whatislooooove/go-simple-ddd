package wallet

import (
	"encoding/json"
	"fmt"
	"go-ddd-template/internal/domain/shared"
	"os"
)

// todo: сделать рефакторинг и поместить файл в другом месте. Сделать прослойку в виде репозитория

const walletFile = "./tmp/wallet.json"

type Wallets struct {
	Wallets map[string]*Wallet
}

func (ws *Wallets) SaveFile() error {
	data, err := json.Marshal(ws.Wallets)
	shared.HandleError(err)

	return os.WriteFile(walletFile, data, 0644)
}

func (ws *Wallets) LoadFile() error {
	data, err := os.ReadFile(walletFile)
	shared.HandleError(err)

	return json.Unmarshal(data, &ws.Wallets)
}

func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile()

	return &wallets, err
}

func (ws *Wallets) GetWallet(address string) *Wallet {
	return ws.Wallets[address]
}

func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.MakeAddress())

	ws.Wallets[address] = wallet

	return address
}
