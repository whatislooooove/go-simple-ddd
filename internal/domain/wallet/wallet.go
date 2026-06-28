package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-ddd-template/internal/domain/shared"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLen = 4
	version     = byte(0x00) // todo выяснить почему так
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// WalletJSON TODO: вникнуть в работу по сериализации для json - в методы тоже
type WalletJSON struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func (w *Wallet) MarshalJSON() ([]byte, error) {
	return json.Marshal(WalletJSON{
		PrivateKey: hex.EncodeToString(w.PrivateKey.D.Bytes()),
		PublicKey:  hex.EncodeToString(w.PublicKey),
	})
}

func (w *Wallet) UnmarshalJSON(data []byte) error {
	var aux WalletJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	privateKeyBytes, err := hex.DecodeString(aux.PrivateKey)
	if err != nil {
		return err
	}

	publicKeyBytes, err := hex.DecodeString(aux.PublicKey)
	if err != nil {
		return err
	}

	curve := elliptic.P256()
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = curve
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(privateKeyBytes)

	w.PrivateKey = *privateKey
	w.PublicKey = publicKeyBytes

	return nil
}

func (w *Wallet) MakeAddress() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := GenerateChecksum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash) // todo:почему то адрес длиннее чем pubHash

	fmt.Printf("public key: %x\n", w.PublicKey)
	fmt.Printf("public hash: %x\n", pubHash)
	fmt.Printf("address: %x\n", address)

	return address
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLen]
	targetChecksum := GenerateChecksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	shared.HandleError(err)

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...) // todo обновить

	return *private, pub
}

func MakeWallet() *Wallet {
	private, public := NewKeyPair()

	return &Wallet{private, public}
}

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	shared.HandleError(err)

	return hasher.Sum(nil)
}

func GenerateChecksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLen]
}
