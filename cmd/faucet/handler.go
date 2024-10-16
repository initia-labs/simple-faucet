package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/initia-labs/simple-faucet/config"
	"github.com/initia-labs/simple-faucet/limit"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tendermint/tmlibs/bech32"
)

// Claim wraps a faucet claim
type Claim struct {
	Address string `json:"address"`
}

func claimHandler(dripMutex *sync.Mutex, db *leveldb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		var claim Claim

		// decode JSON response from front end
		defer request.Body.Close()
		requestBody, err := io.ReadAll(request.Body)
		if err != nil {
			writeErrorResponse(w, err)
			return
		}
		if err := json.Unmarshal(requestBody, &claim); err != nil {
			writeErrorResponse(w, err)
			return
		}

		dripConfig := config.GetConfig().DripConfig
		amount := dripConfig.Amount * MicroUnit

		// make sure address is bech32
		readableAddress, decodedAddress, decodeErr := bech32.DecodeAndConvert(claim.Address)
		if decodeErr != nil {
			writeErrorResponse(w, decodeErr)
			return
		}

		// re-encode the address in bech32
		encodedAddress, encodeErr := bech32.ConvertAndEncode(readableAddress, decodedAddress)
		if encodeErr != nil {
			writeErrorResponse(w, encodeErr)
			return
		}

		// sending the coins!

		// Limiting request speed
		limitErr := limit.CheckAndUpdateLimit(db, decodedAddress, amount, claim.Address, dripConfig)
		if limitErr != nil {
			writeErrorResponse(w, limitErr)
			return
		}

		logger.Infof("req encodedAddress:%+v amount:%+v", encodedAddress, amount)

		// lock the mutex to avoid sequence mismatch
		dripMutex.Lock()

		body, err := drip(decodedAddress, amount, dripConfig.Fee)
		if err != nil {
			// unlock the mutex on error
			dripMutex.Unlock()

			writeErrorResponse(w, err)
			return
		}

		retry := 3
		// Sequence mismatch if the body length is zero
		for body == "" {
			loadAccountInfo()
			body, err = drip(decodedAddress, amount, dripConfig.Fee)
			if err != nil {
				// unlock the mutex on error
				dripMutex.Unlock()

				writeErrorResponse(w, err)
				return
			}

			retry--
			if retry == 0 {
				break
			}
		}

		if len(body) != 0 {
			sequence++
		}

		// create local variable to prevent async sequence increase
		sequenceForReport := sequence

		// unlock the mutex
		dripMutex.Unlock()

		logger.Infof("seq %v %v\n", sequenceForReport, body)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"amount": %v, "response": %v}`, amount, body)
	}
}

func writeErrorResponse(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), 400)
}
