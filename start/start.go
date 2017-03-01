package main

import (
	"strconv"
	"time"

	"encoding/json"
	"fmt"

	"errors"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var cpPrefix = "cp:"
var accountPrefix = "acct:"

type SimpleChaincode struct {
}

const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond, (msInt%millisPerSecond)*nanosPerMillisecond), nil
}

func generateCUSIPSuffix(issueDate string, days int) (string, error) {
	t, err := msToTime(issueDate)
	if err != nil {
		return "", err
	}

	maturityDate := t.AddDate(0, 0, days)
	month := int(maturityDate.Month())
	day := maturityDate.Day()

	suffix := seventhDigit[month] + eigthDigit[day]
	return suffix, nil
}

func (t *SimpleChaincode) createAccount(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	fmt.Println("creating account")

	if len(args) != 1 {
		fmt.Println("Error obtaining username")
		return nil, errors.New("createAccount accepts a singe username argument")
	}
	username := args[0]

	var assetIds []string
	suffix := "000A"
	prefix := username + suffix
	var account = Account{ID: username, Prefix: prefix, CashBalance: 1000000.0, AssetsIds: assetIds}
	accountBytes, err := json.Marshal(&account)
	if err != nil {
		fmt.Println("error creating account" + account.ID)
		return nil, errors.New("Error creating account " + account.ID)
	}
	fmt.Println("Attempting to get state of any existing account for " + account.ID)
	existingBytes, err := stub.GetState(accountPrefix + account.ID)
	if err == nil {
		var company Account
		err = json.Unmarshal(existingBytes, &company)
		if err != nil {
			fmt.Println("Error unmarshalling account " + account.ID + "\n-->" + err.Error())

			if strings.Contains(err.Error(), "unexpected end") {
				fmt.Println("No data means existing account found for " + account.ID + ", initializing account.")
				err = stub.PutState(accountPrefix+account.ID, accountBytes)

				if err == nil {
					fmt.Println("created account" + accountPrefix + account.ID)
					return nil, nil
				} else {
					fmt.Println("failed to create initialize account for " + account.ID)
					return nil, errors.New("failed to initialize an account for " + account.ID + " => " + err.Error())
				}

			} else {
				return nil, errors.New("Error unmarshalling existing account" + account.ID)
			}
		} else {
			fmt.Println("Account already exists for " + account.ID + " " + company.ID)
			return nil, errors.New("Can't reinitialize existing user " + account.ID)

		}
	} else {
		fmt.Println("No existing account found for " + account.ID + ", initializing account.")
		err = stub.PutState(accountPrefix+account.ID, accountBytes)
		if err == nil {
			fmt.Println("created account" + accountPrefix + account.ID)
			return nil, nil

		} else {
			fmt.Println("failed to create initialize account for " + account.ID)
			return nil, errors.New("failed to initialize an account for " + account.ID + " => " + err.Error())

		}

	}
}

func GetCompany(companyID string, stub shim.ChaincodeStubInterface) (Account, error) {
	var company Account
	companyBytes, err := stub.GetState(accountPrefix + companyID)
	if err != nil {
		fmt.Println("Account not found" + companyID)
		return company, errors.New("Account not found" + companyID)
	}

	err = json.Unmarshal(companyBytes, &company)
	if err != nil {
		fmt.Println("Error unmarshalling account" + companyID + "\n err:" + err.Error())
		return company, errors.New("Error unmarshalling account" + companyID)
	}
	return company, nil
}

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("Init firing. Function will be ignored: " + function)
	fmt.Println("Initializing paper keys collection")
	var blank []string
	blankBytes, _ := json.Marshal(&blank)
	err := stub.PutState("PaperKeys", blankBytes)
	if err != nil {
		fmt.Println("Failed to initialize paper key collection")
	}
	fmt.Println("Initialization Complete")
	return nil, nil
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("Invoke running. function: " + function)
	if function == "createAccount" {
		return t.createAccount(stub, args)
	}

	return nil, errors.New("Received unknown function invocation: " + function)

}

func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if function == "GetCompany" {
		fmt.Println("Getting the company")
		company, err := GetCompany(args[0], stub)
		if err != nil {
			fmt.Println("Erros from GetCompany")
			return nil, err
		} else {
			companyBytes, err1 := json.Marshal(&company)
			if err1 != nil {
				fmt.Println("Error marshalling the company")
				return nil, err1
			}
			fmt.Println("All success, returning the company")
			return companyBytes, nil
		}
	} else {
		fmt.Println("Generic Query call")
		bytes, err := stub.GetState(args[0])

		if err != nil {
			fmt.Println("Some error happenend: " + err.Error())
			return nil, err
		}

		fmt.Println("All success, returning from generic")
		return bytes, nil
	}
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Println("Error starting Simple chaincode: %s", err)
	}
}

type Owner struct {
	Company  string `json:"company"`
	Quantity int    `json:"quantity"`
}

type CP struct {
	CUSIP     string  `json:"cusip"`
	Ticker    string  `json:"ticker"`
	Par       float64 `json:"par"`
	Qty       int     `json:"qty"`
	Discount  float64 `json:"discount"`
	Maturity  int     `json:"maturity"`
	Owners    []Owner `json:"owner"`
	Issuer    string  `json:"issuer"`
	IssueDate string  `json:"issueDate"`
}

type Account struct {
	ID          string   `json:"id"`
	Prefix      string   `json:"prefix"`
	CashBalance float64  `json:"cashBalance"`
	AssetsIds   []string `json:"assetIds"`
}

type Transaction struct {
	CUSIP       string  `json:"cusip"`
	FromCompany string  `json:"fromCompany"`
	ToCompany   string  `json:"toCompany"`
	Quantity    int     `json:"quantity"`
	Discount    float64 `json:"discount"`
}

var seventhDigit = map[int]string{
	1:  "A",
	2:  "B",
	3:  "C",
	4:  "D",
	5:  "E",
	6:  "F",
	7:  "G",
	8:  "H",
	9:  "J",
	10: "K",
	11: "L",
	12: "M",
	13: "N",
	14: "P",
	15: "Q",
	16: "R",
	17: "S",
	18: "T",
	19: "U",
	20: "V",
	21: "W",
	22: "X",
	23: "Y",
	24: "Z",
}

var eigthDigit = map[int]string{
	1:  "1",
	2:  "2",
	3:  "3",
	4:  "4",
	5:  "5",
	6:  "6",
	7:  "7",
	8:  "8",
	9:  "9",
	10: "A",
	11: "B",
	12: "C",
	13: "D",
	14: "E",
	15: "F",
	16: "G",
	17: "H",
	18: "J",
	19: "K",
	20: "L",
	21: "M",
	22: "N",
	23: "P",
	24: "Q",
	25: "R",
	26: "S",
	27: "T",
	28: "U",
	29: "V",
	30: "W",
	31: "X",
}
