package monitor

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var monitors = make(chan monitor)

type monitor struct {
	url           string
	contract      common.Address
	beginBlockNum int64
	logs          chan types.Log
}

func AddMonitor(url string, address common.Address, beginBlockNum int64, logs chan types.Log) {
	monitors <- monitor{
		url:           url,
		contract:      address,
		beginBlockNum: beginBlockNum,
		logs:          logs,
	}
}

func Serve() {
	for m := range monitors {
		_ = m.start()
	}
}

func (m *monitor) start() error {
	events := make(chan types.Log)
	sub, err := m.subscribeLog(events)
	if err != nil {
		log.Printf("Subscribe contract: %v error: %v\n", m.contract, err.Error())
		return err
	}

	go processLogs(events, sub.Err(), func(b bool, t types.Log) {
		if b {
			m.logs <- t
		} else {
			resubscribe(m)
		}
	})

	return nil
}

func resubscribe(m *monitor) {
	time.AfterFunc(2*time.Minute, func() {
		monitors <- *m
	})
}

func (m *monitor) subscribeLog(events chan types.Log) (ethereum.Subscription, error) {
	client, err := ethclient.Dial(m.url)
	if err != nil {
		log.Printf("Can not connect to %v, %v\n", m.url, err.Error())
		return nil, err
	}

	log.Printf("Success to Connect node: %v\n", m.url)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(m.beginBlockNum),
		Addresses: []common.Address{m.contract},
	}

	sub, err := client.SubscribeFilterLogs(context.Background(), query, events)
	if err != nil {
		log.Printf("Subscribe logs error: %v\n", err.Error())
		return nil, err
	}

	return sub, nil
}

func processLogs(events chan types.Log, errs <-chan error, f func(bool, types.Log)) {
	for {
		select {
		case err := <-errs:
			log.Printf("Received errors: %v\n", err.Error())
			f(false, types.Log{})

		case vLog := <-events:
			log.Printf("Received log %v\n", vLog)
			f(true, vLog)
		}
	}
}
