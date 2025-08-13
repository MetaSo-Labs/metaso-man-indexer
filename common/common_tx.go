package common

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	bsvec2 "github.com/bitcoinsv/bsvd/bsvec"
	chaincfg2 "github.com/bitcoinsv/bsvd/chaincfg"
	chainhash2 "github.com/bitcoinsv/bsvd/chaincfg/chainhash"
	txscript2 "github.com/bitcoinsv/bsvd/txscript"
	wire2 "github.com/bitcoinsv/bsvd/wire"
	bsvutil2 "github.com/bitcoinsv/bsvutil"
	"github.com/btcsuite/btcd/wire"
)

type SignMode string

const (
	SignModeSegwit  SignMode = "segwit"
	SignModeTaproot SignMode = "taproot"
	SignModeLegacy  SignMode = "legacy"
)

type TxInputUtxo struct {
	TxId     string
	TxIndex  int64
	PkScript string
	Amount   uint64
	PriHex   string
	SignMode SignMode
}

type TxOutput struct {
	Address string
	Amount  int64
}

func BuildMvcCommonTx(netParam *chaincfg2.Params, ins []*TxInputUtxo, outs []*TxOutput, changeAddress string, feeRate int64, isUnSign bool) (*wire2.MsgTx, error) {
	tx := wire2.NewMsgTx(2)
	totalAmount := int64(0)
	outAmount := int64(0)
	for _, out := range outs {
		addr, err := bsvutil2.DecodeAddress(out.Address, netParam)
		if err != nil {
			return nil, err
		}
		pkScript, err := txscript2.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire2.NewTxOut(out.Amount, pkScript))
		outAmount = outAmount + out.Amount
	}
	if changeAddress != "" {
		addr, err := bsvutil2.DecodeAddress(changeAddress, netParam)
		if err != nil {
			return nil, err
		}
		pkScriptByte, err := txscript2.PayToAddrScript(addr)
		tx.AddTxOut(wire2.NewTxOut(0, pkScriptByte))
	}

	for _, in := range ins {
		hash, err := chainhash2.NewHashFromStr(in.TxId)
		if err != nil {
			return nil, err
		}
		prevOut := wire2.NewOutPoint(hash, uint32(in.TxIndex))
		txIn := wire2.NewTxIn(prevOut, nil)
		tx.AddTxIn(txIn)
		totalAmount = totalAmount + int64(in.Amount)

	}
	txTotalSize := tx.SerializeSize()

	//txSize := tx.SerializeSize() + SpendSize*len(tx.TxIn)
	txFee := int64(txTotalSize) * feeRate

	fmt.Printf("txTotalSize:%d, txFee:%d, feeRate:%d, totalAmount:%d, outAmount:%d\n", txTotalSize, txFee, feeRate, totalAmount, outAmount)
	if totalAmount-outAmount < int64(txFee) {
		return nil, errors.New("insufficient fee")
	}

	changeVal := totalAmount - outAmount - int64(txFee)
	if changeVal >= 600 && changeAddress != "" {
		tx.TxOut[len(tx.TxOut)-1].Value = changeVal
	} else {
		tx.TxOut = tx.TxOut[:len(tx.TxOut)-1]
	}

	if !isUnSign {
		for i, in := range ins {
			privateKeyBytes, err := hex.DecodeString(in.PriHex)
			if err != nil {
				return nil, err
			}
			privateKey, _ := bsvec2.PrivKeyFromBytes(bsvec2.S256(), privateKeyBytes)

			pkScriptByte, err := hex.DecodeString(in.PkScript)
			if err != nil {
				return nil, err
			}

			var sigScript []byte
			sigScript, err = txscript2.SignatureScript(tx, i, int64(in.Amount), pkScriptByte, txscript2.SigHashAll, privateKey, true)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}

			tx.TxIn[i].SignatureScript = sigScript
		}
	}

	return tx, nil
}

func BuildMvcTransferAllTx(netParam *chaincfg2.Params, ins []*TxInputUtxo, out *TxOutput, feeRate int64, isUnSign bool) (*wire2.MsgTx, error) {
	tx := wire2.NewMsgTx(2)
	totalAmount := int64(0)
	outAmount := int64(0)

	addr, err := bsvutil2.DecodeAddress(out.Address, netParam)
	if err != nil {
		return nil, err
	}
	pkScript, err := txscript2.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}
	tx.AddTxOut(wire2.NewTxOut(out.Amount, pkScript))

	emptylegacySignature := make([]byte, 107)
	txSignSize := 0
	for _, in := range ins {
		hash, err := chainhash2.NewHashFromStr(in.TxId)
		if err != nil {
			return nil, err
		}
		prevOut := wire2.NewOutPoint(hash, uint32(in.TxIndex))
		txIn := wire2.NewTxIn(prevOut, nil)
		tx.AddTxIn(txIn)
		totalAmount = totalAmount + int64(in.Amount)
		txSignSize += 40 + wire.VarIntSerializeSize(uint64(len(emptylegacySignature))) + len(emptylegacySignature)
	}
	txTotalSize := tx.SerializeSize() + txSignSize

	txFee := int64(txTotalSize) * feeRate
	outAmount = totalAmount - int64(txFee)

	tx.TxOut[0].Value = outAmount

	if !isUnSign {
		for i, in := range ins {
			privateKeyBytes, err := hex.DecodeString(in.PriHex)
			if err != nil {
				return nil, err
			}
			privateKey, _ := bsvec2.PrivKeyFromBytes(bsvec2.S256(), privateKeyBytes)

			pkScriptByte, err := hex.DecodeString(in.PkScript)
			if err != nil {
				return nil, err
			}

			var sigScript []byte
			sigScript, err = txscript2.SignatureScript(tx, i, int64(in.Amount), pkScriptByte, txscript2.SigHashAll, privateKey, true)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}

			tx.TxIn[i].SignatureScript = sigScript
		}
	}

	return tx, nil
}

func MvcToRaw(tx *wire2.MsgTx) (string, error) {
	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return "", err
	}
	txHex := hex.EncodeToString(buf.Bytes())
	tx.TxHash()
	return txHex, nil
}
