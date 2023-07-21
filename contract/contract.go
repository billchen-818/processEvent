package contract

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"processEvent/monitor"
)

type contract struct {
	http string
	ws   string

	idGen func() uint32

	address common.Address
	abi     abi.ABI

	subs       map[uint32]subscription
	subIds     map[string][]uint32
	eventNames map[string]string

	mutex sync.Mutex
}

type Event struct {
	Hash        common.Hash
	BlockNumber uint64
	Data        interface{}
}

type subscription struct {
	subId uint32
	c     chan Event
	eType reflect.Type
}

func genId() func() uint32 {
	var count uint32

	return func() uint32 {
		atomic.AddUint32(&count, 1)
		return count
	}
}

func NewContract(address common.Address, contractABI string, http, ws string) (*contract, error) {
	ABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		log.Printf("Generate contract abi error:%v\n", err.Error())
		return nil, err
	}

	return &contract{
		http:       http,
		ws:         ws,
		idGen:      genId(),
		address:    address,
		abi:        ABI,
		subs:       make(map[uint32]subscription),
		subIds:     make(map[string][]uint32),
		eventNames: make(map[string]string),
	}, nil
}

func (c *contract) StartListen(fromBlock int64) {
	events := make(chan types.Log, 10)

	s := make(chan struct{})

	go func() {
		defer close(s)

		lgs, err := filterLogs(c.address, fromBlock, c.http)
		if err != nil {
			log.Printf("filter logs err, contract: %v, fromBlock:%v, err: %v\n", c.address, fromBlock, err)
			return
		}

		for _, l := range lgs {
			events <- l
		}
	}()

	go func() {
		<-s

		monitor.AddMonitor(c.ws, c.address, fromBlock, events)
	}()

	go c.emitLog(events)
}

func (c *contract) SubEvent(name string, eventChan chan Event, typ reflect.Type) (uint32, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	event, ok := c.abi.Events[name]
	if !ok {
		return 0, fmt.Errorf("unknown event name: %v\n", name)
	}

	s := subscription{
		subId: c.idGen(),
		c:     eventChan,
		eType: typ,
	}

	eventSigHash := event.ID.Hex()

	c.subIds[eventSigHash] = append(c.subIds[eventSigHash], s.subId)
	c.subs[s.subId] = s
	c.eventNames[eventSigHash] = name

	return s.subId, nil
}

func (c *contract) emitLog(logs chan types.Log) {
	for l := range logs {
		eventSigHash := l.Topics[0].Hex()

		var out interface{}
		if len(c.subIds[eventSigHash]) > 0 {
			out = reflect.New(c.subs[c.subIds[eventSigHash][0]].eType).Interface()

			err := unpackLog(c.abi, out, c.eventNames[eventSigHash], l)
			if err != nil {
				log.Printf("unpack log err: %v\n", err.Error())
				continue
			}
		}

		outEvent := Event{
			Hash:        l.TxHash,
			BlockNumber: l.BlockNumber,
			Data:        out,
		}

		for _, subId := range c.subIds[eventSigHash] {
			subs, ok := c.subs[subId]
			if ok {
				go func() {
					subs.c <- outEvent
				}()
			}
		}
	}
}

func filterLogs(contractAddr common.Address, fromBlock int64, http string) ([]types.Log, error) {
	client, err := ethclient.Dial(http)
	if err != nil {
		return nil, fmt.Errorf("connect network url: %v, error: %v\n", http, err)
	}

	return client.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		Addresses: []common.Address{contractAddr},
	})
}

func unpackLog(contractABI abi.ABI, out interface{}, event string, log types.Log) error {
	if log.Topics[0] != contractABI.Events[event].ID {
		return fmt.Errorf("event signature mismatch, require %v, get %v\n", contractABI.Events[event].ID, log.Topics[0])
	}

	if len(log.Data) > 0 {
		if err := contractABI.UnpackIntoInterface(out, event, log.Data); err != nil {

			fmt.Printf("UnpackIntoInterface err:%v\n", err.Error())
			return err
		}
	}

	var indexed abi.Arguments
	for _, arg := range contractABI.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}

	return abi.ParseTopics(out, indexed, log.Topics[1:])
}
