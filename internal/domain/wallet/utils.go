package wallet

import (
	"go-ddd-template/internal/domain/shared"

	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode) // todo тоже выяснить как работает
}

func Base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	shared.HandleError(err)

	return decode
}
