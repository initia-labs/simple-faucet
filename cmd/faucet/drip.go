package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/initia-labs/initia/app"
	"github.com/initia-labs/simple-faucet/config"
)

const Denom = "uinit"

// BroadcastReq defines a tx broadcasting request.
type BroadcastReq struct {
	Tx   string `json:"tx_bytes"`
	Mode string `json:"mode"`
}

func drip(recipient sdk.AccAddress, amount int64, fee sdk.Coins) (string, error) {
	// set tx builder
	txBuilder := app.MakeEncodingConfig().TxConfig.NewTxBuilder()
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(1_500_000)
	txBuilder.SetMemo("faucet")
	txBuilder.SetTimeoutHeight(0)

	coins := sdk.NewCoins(sdk.NewCoin(Denom, math.NewInt(amount)))
	sendMsg := banktypes.NewMsgSend(faucetAddress, recipient, coins)
	txBuilder.SetMsgs(sendMsg)

	// create signature v2
	txConfig := cdc.TxConfig
	signMode, err := authsigning.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	if err != nil {
		return "", err
	}

	sigData := txsigning.SingleSignatureData{
		SignMode:  signMode,
		Signature: nil,
	}
	sigv2 := txsigning.SignatureV2{
		PubKey:   privKey.PubKey(),
		Data:     &sigData,
		Sequence: sequence,
	}
	txBuilder.SetSignatures(sigv2)

	sigV2, err := clienttx.SignWithPrivKey(
		context.Background(),
		signMode,
		authsigning.SignerData{
			ChainID:       config.GetConfig().ChainID,
			Address:       address,
			AccountNumber: accountNumber,
			Sequence:      sequence,
			PubKey:        nil,
		},
		txBuilder,
		privKey,
		txConfig,
		sequence,
	)
	if err != nil {
		return "", err
	}

	// set signature
	if err = txBuilder.SetSignatures(sigV2); err != nil {
		return "", err
	}

	// encode signed tx
	bz, err := cdc.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", err
	}

	// prepare to broadcast
	broadcastReq := BroadcastReq{
		Tx:   base64.StdEncoding.EncodeToString(bz),
		Mode: "BROADCAST_MODE_SYNC",
	}
	txBz, err := json.Marshal(broadcastReq)
	if err != nil {
		return "", err
	}

	// broadcast
	url := fmt.Sprintf("%v/cosmos/tx/v1beta1/txs", config.GetConfig().RestURL)
	response, err := http.Post(url, "application/json", bytes.NewReader(txBz))
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	stringBody := string(body)

	if response.StatusCode != 200 {
		err := fmt.Errorf("status: %v, message: %v", response.Status, stringBody)
		return "", err
	}

	parsedCode, err := parseRegexp(`"code": ?(\d+)?`, stringBody)
	if err != nil {
		return "", err
	}
	code, err := strconv.ParseUint(parsedCode, 10, 64)
	if err != nil {
		return "", errors.New("failed to parse code from tx response")
	}

	if strings.Contains(stringBody, "sequence mismatch") {
		return "", nil
	}

	if code != 0 {
		parsedRawLog, err := parseRegexp(`"raw_log":"((?:\\"|[^"])*)"`, stringBody)
		if err != nil {
			return "", err
		} else {
			return "", errors.New(parsedRawLog)
		}
	}

	return stringBody, nil
}
