package main

import (
	"fmt"
	"os"
	"strconv"
	"net/http"
	"strings"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"github.com/ericlagergren/decimal"
)

var (
	precision  int
	iterations uint64
	hard       bool
	cache      []*decimal.Big
	hostname   string
	threadsDone chan bool
	iterationsComplete chan bool
)
//Config is for unmarshalling config.json
type Config struct {
	ServerHostname   string `json:"serverHostname"`
	ThreadCount   int `json:"threadCount"`
}

func main() {
	//Get settings from config file
	var config Config
	// Open our jsonFile
	jsonFile, err := os.Open("./config.json")
	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened users.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	if len(byteValue) == 0 {
		panic("No JSON could be loaded")
	}
	json.Unmarshal(byteValue, &config)
	hostname = config.ServerHostname
	//Get settings from the server
	response, err := http.Get("http://"+hostname+"/work/settings")
	if err != nil {
		panic(err)
	}
	body, _ := ioutil.ReadAll(response.Body)
	bodyString := fmt.Sprintf("%s",body)
	settings := strings.Split(bodyString, ";")
	precision, _ = strconv.Atoi(settings[0])
	limit, _ := strconv.Atoi(settings[1])
	cache = make([]*decimal.Big, (limit*2)+4)


	//precompute some factorials
	cache[0], cache[1] = decimal.WithPrecision(precision).SetUint64(1), decimal.WithPrecision(precision).SetUint64(1)


	//Run n parallel while loops, n = threadcount
	threadsDone = make(chan bool, config.ThreadCount)
	for i :=0; i < config.ThreadCount; i++ {
		go coordinator() //Coordinator actually handles the work
	}
	for i :=0; i < config.ThreadCount; i++ {
		<- threadsDone //Wait for the threads to be done
	}
}
func coordinator() {
	notDone := true
	for notDone {
		//Get work
		response, err := http.Get("http://"+hostname+"/work/get")
		if err != nil {
			//fmt.Println(err)
			fmt.Println("Compute (probably) done, breaking")
			break
		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
			break
		}
		work, err := strconv.Atoi(fmt.Sprintf("%s",body))
		if err != nil {
			fmt.Println(err)
			break
		}
		if work == -1 {
			notDone = false
			fmt.Println("Compute done, breaking")
		} else {
			http.Post("http://"+hostname+"/results/submit",
			"text/plain", 
			bytes.NewBuffer([]byte(fmt.Sprintf("%v",iteration(uint64(work))))))
			// fmt.Println(iteration(uint64(work)))
		}

	}
	threadsDone <- true
}

func iteration(n uint64) *decimal.Big {
	add := decimal.WithPrecision(precision).SetUint64(((2 * n) + 2))
	add.Quo(add, factorial((2*n)+1))
	return add
}

func factorial(n uint64) *decimal.Big {
	//If it's in the map then return from map
	if cache[n] != nil {
		//fmt.Print("cache hit")
		return cache[n]
	}
	//Otherwise, you actually gotta work it out
	temp := decimal.WithPrecision(precision).SetUint64(n)
	if n == 0 || n == 1 { //Base case
		return (decimal.WithPrecision(precision).SetUint64(1))
	}
	cache[n] = temp.Mul(temp, factorial(n-1))
	return cache[n]
}
