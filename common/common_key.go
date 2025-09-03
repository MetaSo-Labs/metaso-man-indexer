package common

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func GenerateKeyAndLegacyAddress(netParams *chaincfg.Params) (string, string, error) {
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return "", "", err
	}
	privateKeyHex := hex.EncodeToString(privateKey.Serialize())
	legacyAddress, err := btcutil.NewAddressPubKey(privateKey.PubKey().SerializeCompressed(), netParams)
	if err != nil {
		return "", "", err
	}
	return privateKeyHex, legacyAddress.EncodeAddress(), nil
}
