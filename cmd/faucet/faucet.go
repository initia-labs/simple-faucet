package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	"github.com/initia-labs/simple-faucet/config"
	"github.com/initia-labs/simple-faucet/log"
	"github.com/rs/cors"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tendermint/tmlibs/bech32"
)

var privKey cryptotypes.PrivKey
var address string
var faucetAddress sdk.AccAddress
var sequence uint64
var accountNumber uint64
var cdc *params.EncodingConfig
var logger = log.NewDefaultLogger()

const ( // new core hasn't these yet.
	MicroUnit              = int64(1e6)
	accountAddresPrefix    = "init"
	accountPubKeyPrefix    = "initpub"
	validatorAddressPrefix = "initvaloper"
	validatorPubKeyPrefix  = "initvaloperpub"
	consNodeAddressPrefix  = "initvalcons"
	consNodePubKeyPrefix   = "initvalconspub"
)

func newCodec() *params.EncodingConfig {
	ec := app.MakeEncodingConfig()

	config := sdk.GetConfig()
	config.SetCoinType(app.CoinType)
	config.SetPurpose(44)
	config.SetBech32PrefixForAccount(accountAddresPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	config.SetAddressVerifier(app.VerifyAddressLen())
	config.Seal()

	return &ec
}

func loadAccountInfo() {
	// Query current faucet sequence
	url := fmt.Sprintf("%v/cosmos/auth/v1beta1/accounts/%v", config.GetConfig().RestURL, address)
	response, err := http.Get(url)
	if err != nil {
		logger.Errorf("failed to get account info: %v", err)
		return
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("failed to read response body: %v", err)
		return
	}

	bodyStr := string(body)
	var seq uint64

	if strings.Contains(bodyStr, `"sequence"`) {
		parsedSeq, err := parseRegexp(`"sequence":"?(\d+)"?`, bodyStr)
		if err != nil {
			logger.Errorf("failed to parse sequence: %+v", err)
			return
		}
		seq, _ = strconv.ParseUint(parsedSeq, 10, 64)
	} else {
		seq = 0
	}

	sequence = atomic.LoadUint64(&seq)

	if strings.Contains(bodyStr, `"account_number"`) {
		parsedAccountNumber, err := parseRegexp(`"account_number":"?(\d+)"?`, bodyStr)
		if err != nil {
			logger.Errorf("failed to parse account number: %+v", err)
			return
		}
		accountNumber, _ = strconv.ParseUint(parsedAccountNumber, 10, 64)
	} else {
		accountNumber = 0
	}

	logger.Infof("loadAccountInfo: address %v account# %v sequence %v\n", address, accountNumber, sequence)
}

func init() {
	if len(os.Args) < 2 {
		panic("config file path is not set")
	}

	viper.SetConfigFile(os.Args[1])
	if err := viper.ReadInConfig(); err != nil {
		panic("failed to read config file")
	}
}

func main() {
	var err error

	cfg := config.GetConfig()

	db, err := leveldb.OpenFile(path.Join(cfg.Home, "db/faucetdb"), nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	cdc = newCodec()

	fullFundraiserPath := "m/44'/" + strconv.Itoa(app.CoinType) + "'/0'/0/0"
	derivedPriv, err := hd.Secp256k1.Derive()(cfg.Mnemonic, "", fullFundraiserPath)
	if err != nil {
		panic(err)
	}

	privKey = hd.Secp256k1.Generate()(derivedPriv)
	pubk := privKey.PubKey() 

	address, err = bech32.ConvertAndEncode("init", pubk.Address())
	if err != nil {
		panic(err)
	}

	faucetAddress, err = sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}

	// Load account number and sequence
	loadAccountInfo()

	dripMutex := &sync.Mutex{}

	// Application server.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	// normal claim endpoint
	mux.HandleFunc("/claim", claimHandler(dripMutex, db))

	config := config.GetConfig()
	c := cors.New(cors.Options{
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		AllowOriginRequestFunc: func(r *http.Request, origin string) bool {
			for _, allowedOrigin := range config.AllowedOrigins {
				if origin == allowedOrigin {
					return true
				}
			}

			return false
		},
	})

	handler := c.Handler(mux)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), handler); err != nil {
		logger.Fatalf("failed to start server: %+v", err)
	}
}
