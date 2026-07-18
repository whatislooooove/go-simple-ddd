package wallet

import (
	"encoding/json"
	"fmt"
	"go-ddd-template/internal/domain/shared"
	"os"
)

// todo: сделать рефакторинг и поместить файл в другом месте. Сделать прослойку в виде репозитория

const walletFile = "./tmp/wallets_%s.json"

type Wallets struct {
	Wallets map[string]*Wallet
}

func (ws *Wallets) SaveFile(nodeId string) error {
	// todo: если файла нету - создать
	walletFile := fmt.Sprintf(walletFile, nodeId)
	data, err := json.Marshal(ws.Wallets)
	shared.HandleError(err)

	return os.WriteFile(walletFile, data, 0644)
}

func (ws *Wallets) LoadFile(nodeId string) error {
	walletFile := fmt.Sprintf(walletFile, nodeId)
	data, err := os.ReadFile(walletFile)
	if err != nil {
		if os.IsNotExist(err) {
			ws.Wallets = make(map[string]*Wallet)
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &ws.Wallets)
}

func CreateWallets(nodeId string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile(nodeId)
	if err != nil {
		return nil, err
	}

	return &wallets, nil
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
