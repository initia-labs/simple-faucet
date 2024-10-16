package limit

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/initia-labs/simple-faucet/config"
	"github.com/syndtr/goleveldb/leveldb"
)

const requestLimitSecs = 30

// RequestLog stores the Log of a Request
type RequestLog struct {
	CoinLog   CoinLog   `json:"coin_log"`
	Requested time.Time `json:"requested"`
}

type CoinLog struct {
	Requested []time.Time `json:"requested"`
}

func CheckAndUpdateLimit(
	db *leveldb.DB,
	account []byte,
	amount int64,
	address string,
	dripConfig config.DripConfig,
) error {
	now := time.Now()
	var requestLog RequestLog

	logBytes, _ := db.Get(account, nil)
	if logBytes != nil {
		if err := json.Unmarshal(logBytes, &requestLog); err != nil {
			return err
		}

		// check interval limit
		intervalSecs := now.Sub(requestLog.Requested).Seconds()
		if intervalSecs < requestLimitSecs {
			return errMsg(address)
		}

		// check amount limit
		if err := requestLog.checkDurationLimit(dripConfig, address, now); err != nil {
			return err
		}
	} else {
		requestLog.CoinLog = CoinLog{
			Requested: []time.Time{now},
		}
	}

	requestLog.Requested = now
	logBytes, _ = json.Marshal(requestLog)
	if err := db.Put(account, logBytes, nil); err != nil {
		return err
	}

	return nil
}

func (requestLog *RequestLog) checkDurationLimit(
	dripConfig config.DripConfig,
	address string,
	now time.Time,
) error {
	interval := dripConfig.Interval
	count := dripConfig.Count

	// filter requested times that are within the interval
	var filtered []time.Time

	coinLog := requestLog.CoinLog
	for _, t := range coinLog.Requested {
		if t.Add(interval).After(now) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= count {
		return errMsg(address)
	}

	coinLog.Requested = append(filtered, now)
	requestLog.CoinLog = coinLog

	return nil
}

func errMsg(address string) error {
	msg := fmt.Sprintf(`exceed request limit
Account Addr: %s
	
The account has recently received funds from the faucet!
Please try again in a bit later`, address)

	return errors.New(msg)
}
