package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/simple-faucet/log"
	"github.com/spf13/viper"
)

var logger = log.NewDefaultLogger()

type DripConfig struct {
	Amount   int64         `json:"amount"`
	Interval time.Duration `json:"interval"`
	Fee      sdk.Coins     `json:"fee"`
	Count    int           `json:"count"`
}

type Config struct {
	Home           string
	Mnemonic       string
	Port           string
	RestURL        string
	ChainID        string
	DripConfig     DripConfig
	AllowedOrigins []string
}

var singleton Config
var once sync.Once

const (
	mnemonicVar       = "MNEMONIC"
	portVar           = "PORT"
	restUrlVar        = "REST_URL"
	chainIDVar        = "CHAIN_ID"
	dripConfigVar     = "DRIP_CONFIG"
	homePathVar       = "HOME"
	allowedOriginsVar = "ALLOWED_ORIGINS"
)

func GetConfig() *Config {
	once.Do(func() {
		singleton = newConfig()
	})
	return &singleton
}

func newConfig() Config {
	c := Config{
		Mnemonic: func() string {
			mnemonic := viper.GetString(mnemonicVar)
			if mnemonic == "" {
				logger.Panicf("MNEMONIC variable is required")
			}
			return mnemonic
		}(),
		Port: func() string {
			port := viper.GetString(portVar)
			if port == "" {
				port = "4000"
			}
			return port
		}(),
		RestURL: func() string {
			restURL := viper.GetString(restUrlVar)
			if restURL == "" {
				logger.Panicf("REST_URL variable is required")
			}

			// NOTE: remove trailing slash from the rest url as a workaround for grpc-gateway route mismatch
			return strings.TrimRight(restURL, "/")
		}(),
		ChainID: func() string {
			chainID := viper.GetString(chainIDVar)
			if chainID == "" {
				logger.Panicf("CHAIN_ID variable is required")
			}
			return chainID
		}(),
		DripConfig: func() DripConfig {
			if !viper.IsSet(dripConfigVar) {
				logger.Panicf("DRIP_CONFIG is required")
			}

			conf := viper.GetStringMapString(dripConfigVar)
			amountStr, ok := conf["amount"]
			if !ok {
				logger.Panicf("DRIP_AMOUNT is not set")
			}
			amount, err := strconv.ParseInt(amountStr, 10, 64)
			if err != nil || amount < 1 {
				logger.Panicf("invalid DRIP_AMOUNT")
			}

			intervalStr, ok := conf["interval"]
			if !ok {
				logger.Panicf("DRIP_INTERVAL is not set")
			}
			interval, err := time.ParseDuration(intervalStr)
			if err != nil {
				logger.Panicf("invalid DRIP_INTERVAL")
			}

			feeStr, ok := conf["fee"]
			if !ok {
				logger.Panicf("FEE is not set. Please set FEE in DRIP_CONFIG even if the fee is zero")
			}
			fee, err := sdk.ParseCoinsNormalized(feeStr)
			if err != nil {
				logger.Panicf("invalid FEE")
			}

			countStr, ok := conf["count"]
			if !ok {
				logger.Panicf("DRIP_COUNT is not set")
			}
			count, err := strconv.Atoi(countStr)
			if err != nil || count < 1 {
				logger.Panicf("invalid DRIP_COUNT")
			}

			return DripConfig{
				Amount:   amount,
				Interval: interval,
				Fee:      fee,
				Count:    count,
			}
		}(),
		Home: func() string {
			home := viper.GetString(homePathVar)
			if home == "" {
				logger.Panicf("[monitor] HOME variable is required")
			}
			hs, err := os.Stat(home)
			if err != nil || !hs.IsDir() {
				logger.Panicf(fmt.Sprintf("HOME is invalid: %v", err))
			}
			return home
		}(),
		AllowedOrigins: func() []string {
			origins := viper.GetStringSlice(allowedOriginsVar)
			if len(origins) == 0 {
				logger.Panicf("ALLOWED_ORIGINS variable is required")
			}
			return origins
		}(),
	}
	loggingC := c
	loggingC.Mnemonic = "REDACTED"
	logger.Infof("[config] %+v\n", loggingC)
	return c
}
