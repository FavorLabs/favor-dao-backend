package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"sort"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/pkg/errcode"
	"favor-dao-backend/pkg/util"
	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
	unipass_sigverify "github.com/unipassid/unipass-sigverify-go"
)

type WalletType string

const (
	WalletConnect WalletType = "wallet_connect"
	MetaMask      WalletType = "meta_mask"
	OKX           WalletType = "okx"
	Unipass_Std   WalletType = "unipass_std"
	Unipass_eth   WalletType = "unipass_eth"
)

type AuthByWalletRequest struct {
	Timestamp  int64      `json:"timestamp"     binding:"required"`
	WalletAddr string     `json:"wallet_addr"   binding:"required"`
	Signature  string     `json:"signature"     binding:"required"`
	Type       WalletType `json:"type"          binding:"required"`
}

func VerifySignMessage(ctx context.Context, auth *AuthByWalletRequest, guessMessage string) (bool, error) {
	walletBytes, err := hexutil.Decode(auth.WalletAddr)
	if err != nil {
		return false, errcode.InvalidParams
	}
	signature, err := hexutil.Decode(auth.Signature)
	if err != nil {
		return false, errcode.InvalidParams
	}

	// check valid timestamp
	if time.Now().After(time.UnixMilli(auth.Timestamp).Add(time.Minute)) {
		return false, errcode.UnauthorizedTokenTimeout
	}

	var ok bool

	// parse message
	switch auth.Type {
	case WalletConnect, MetaMask, OKX:
		ethMessage := []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(guessMessage), guessMessage))
		// Convert to btcec input format with 'recovery id' v at the beginning.
		btcsig := make([]byte, 65)
		btcsig[0] = signature[64]
		copy(btcsig[1:], signature)
		rawKey, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, crypto.Keccak256(ethMessage))
		if err == nil {
			pubkey := (*ecdsa.PublicKey)(rawKey)
			pubBytes := elliptic.Marshal(btcec.S256(), pubkey.X, pubkey.Y)
			var signer common.Address
			copy(signer[:], crypto.Keccak256(pubBytes[1:])[12:])
			ok = bytes.Equal(walletBytes, signer.Bytes())
		}
	case Unipass_Std:
		ok, _ = unipass_sigverify.VerifyMessageSignature(ctx, common.BytesToAddress(walletBytes), []byte(guessMessage), signature, false, eth)
	case Unipass_eth:
		ok, _ = unipass_sigverify.VerifyMessageSignature(ctx, common.BytesToAddress(walletBytes), []byte(guessMessage), signature, true, eth)
	default:
		return false, errcode.InvalidParams
	}

	return ok, nil
}

func GetParamSign(param map[string]interface{}, secretKey string) string {
	signRaw := ""

	rawStrs := []string{}
	for k, v := range param {
		if k != "sign" {
			rawStrs = append(rawStrs, k+"="+fmt.Sprintf("%v", v))
		}
	}

	sort.Strings(rawStrs)
	for _, v := range rawStrs {
		signRaw += v
	}

	if conf.ServerSetting.RunMode == "debug" {
		logrus.Info(map[string]string{
			"signRaw": signRaw,
			"sysSign": util.EncodeMD5(signRaw + secretKey),
		})
	}

	return util.EncodeMD5(signRaw + secretKey)
}
