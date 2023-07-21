package main

import (
	"fmt"
	"math/big"
	"os"
	"processEvent/monitor"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"

	"processEvent/contract"
)

var UniswapV2FactoryAbi = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeToSetter\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"len\",\"type\":\"uint256\"}],\"name\":\"PairCreated\",\"type\":\"event\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"allPairs\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"allPairsLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"createPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"feeTo\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"feeToSetter\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"getPair\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeTo\",\"type\":\"address\"}],\"name\":\"setFeeTo\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"_feeToSetter\",\"type\":\"address\"}],\"name\":\"setFeeToSetter\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
var Address = "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"

var ws = "wss://eth-mainnet.g.alchemy.com/v2/"
var http = "https://eth-mainnet.g.alchemy.com/v2/"

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}

func main() {
	key := os.Getenv("PRIKEY")
	ws = ws + key
	http = http + key

	factory, err := contract.NewContract(common.HexToAddress(Address), UniswapV2FactoryAbi, http, ws)
	if err != nil {
		fmt.Printf("New Factory Contract error: %v\n", err)
		panic(err)
	}

	go func() {
		c := make(chan contract.Event)

		factory.SubEvent("PairCreated", c, reflect.TypeOf(PairCreatedEvent{}))

		listenPairCreated(c)
	}()

	factory.StartListen(17630533)

	monitor.Serve()

	select {}
}

type PairCreatedEvent struct {
	Token0 common.Address
	Token1 common.Address
	Pair   common.Address
	Len    *big.Int
}

func listenPairCreated(c chan contract.Event) {
	for e := range c {
		pe := e.Data.(*PairCreatedEvent)

		fmt.Printf("token0:%v, token1:%v,pair:%v\n", pe.Token0, pe.Token1, pe.Pair)
	}
}
