package app

import (
	"github.com/jaevor/go-nanoid"
)

type Session struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendly_name"`
	WalletAddr   string `json:"wallet_addr"`
}

func GenerateToken() (string, error) {
	tokenGen, err := nanoid.Standard(64)
	if err != nil {
		return "", err
	}

	return tokenGen(), nil
}
