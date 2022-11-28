package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	client, err := ethclient.Dial("https://eth-mainnet.g.alchemy.com/v2/VjJQTgpAV6pC3vTOGCQG_dJ8DhIB4Eqb")
	if err != nil {
		log.Fatal(err)
	}

	// Get the balance of an account
	account := common.HexToAddress("0x71c7656ec7ab88b098defb751b7401b5f6d8976f")
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		log.Fatalln("Error = main - client.BalanceAt: ", err.Error())
	}
	fmt.Println("Account balance:", balance) // 25893180161173005034

	// Get the latest known block
	block, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Fatalln("Error = main - client.BlockByNumber: ", err.Error())
	}
	fmt.Println("Latest block:", block.Number().Uint64())

	txHash := common.HexToHash("0x3e4acda755e036f62c8893ea15d7c587a272252944b0433db4ea5464f235f379")

	var tx *types.Transaction
	tx, _, err = client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		log.Fatalln("Error = main - client.TransactionByHash: ", err.Error())
	}
	ParseAndPrintTransactionBase(tx)

	contractABI, err := abi.JSON(strings.NewReader(GetLocalABI("./boredApeABI.json")))
	if err != nil {
		log.Fatalln("Error = main - abi.JSON: ", err.Error())
	}
	DecodeTransactionInputData(contractABI, tx.Data())
	DecodeTransactionLogs(GetTransactionReceipt(client, txHash), &contractABI)
	fmt.Println("===========================================================")
}

func DecodeTransactionLogs(receipt *types.Receipt, contractABI *abi.ABI) {
	for _, vLog := range receipt.Logs {
		// topic[0] is the event name
		event, err := contractABI.EventByID(vLog.Topics[0])
		if err != nil {
			log.Fatalln("Error = DecodeTransactionLogs - contractABI.EventByID: ", err.Error())
		}
		fmt.Println("Event name: ", event.Name)

		// topic[1:] are other indexed params in the event
		if len(vLog.Topics) > 1 {
			for i, param := range vLog.Topics[1:] {
				contractABI.EventByID(param)
				input := event.Inputs[i]
				// fmt.Printf("Indexed params %d name %s in hex: %s\n", i, input.Name, param)
				if strings.HasPrefix(input.Type.String(), "uint") {
					inputInt, err := strconv.ParseInt(removeZerosAndX(param.Hex()), 16, 64)
					if err != nil {
						log.Fatalln("Error = DecodeTransactionLogs - strconv.ParseInt: ", err.Error())
					}
					fmt.Printf("Indexed params %d name %s value decoded %d\n", i, input.Name, inputInt)
				} else {
					fmt.Printf("Indexed params %d name %s value decoded %s\n", i, input.Name, common.HexToAddress(param.Hex()))
				}
			}
		}

		if len(vLog.Data) > 1 {
			fmt.Printf("Log Data in Hex: %s\n", hex.EncodeToString(vLog.Data))
			outputDataMap := make(map[string]any)
			err = contractABI.UnpackIntoMap(outputDataMap, event.Name, vLog.Data)
			if err != nil {
				log.Fatalln("Error = DecodeTransactionLogs - contractABI.UnpackIntoMap: ", err.Error())
			}
			fmt.Printf("Event outputs: %+v\n", outputDataMap)
		}
	}
}

func DecodeTransactionInputData(contractABI abi.ABI, data []byte) {
	var err error
	methodSigData := data[:4]
	method, err := contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatalln("Error = DecodeTransactionInputData - contractABI.MethodById :", err.Error())
	}

	inputsData := data[4:]
	inputsMap := make(map[string]interface{})
	err = method.Inputs.UnpackIntoMap(inputsMap, inputsData)
	if err != nil {
		log.Fatalln("Error = DecodeTransactionInputData - method.Inputs.UnpackIntoMap :", err.Error())
	}

	fmt.Printf("Method Name: %s\n", method.Name)
	fmt.Printf("Method Inputs: %+v\n", inputsMap)
}

func ParseAndPrintTransactionBase(tx *types.Transaction) {
	fmt.Printf("Hash: %s\n", tx.Hash().String())
	fmt.Printf("ChainID: %d\n", tx.ChainId())
	fmt.Printf("Value: %s\n", tx.Value().String())
	fmt.Printf("From: %s\n", GetTransactionAsMessage(tx).From().Hex())
	fmt.Printf("To: %s\n", tx.To().Hex())
	fmt.Printf("Gas: %d\n", tx.Gas())
	fmt.Printf("GasPrice: %s\n", tx.GasPrice().String())
	fmt.Printf("Nonce: %d\n", tx.Nonce())
}

func GetTransactionReceipt(client *ethclient.Client, txHash common.Hash) *types.Receipt {
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		log.Fatalln("Error = GetTransactionReceipt - client.TransactionReceipt :", err.Error())
	}
	return receipt
}

func GetTransactionAsMessage(tx *types.Transaction) types.Message {
	msg, err := tx.AsMessage(types.LatestSignerForChainID(tx.ChainId()), nil)
	if err != nil {
		log.Fatalln("Error = GetTransactionMessage - tx.AsMessage :", err.Error())
	}
	fmt.Printf("tx.AsMessage: %+v\n", msg)
	return msg
}

func GetLocalABI(path string) string {
	abiFile, err := os.Open(path)
	if err != nil {
		log.Fatalln("Error = GetLocalABI - os.Open: ", err.Error())
	}
	defer abiFile.Close()

	stream, err := io.ReadAll(abiFile)
	if err != nil {
		log.Fatalln("Error = GetLocalABI - io.ReadAll: ", err.Error())
	}

	return string(stream)
}

func removeZerosAndX(text string) string {
	text = strings.ReplaceAll(text, "0", "")
	text = strings.ReplaceAll(text, "x", "")
	return text
}
