package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ericlagergren/decimal"
	"github.com/gorilla/mux"
)

var (
	work      chan int
	results   chan *decimal.Big
	total     *decimal.Big
	limit     = 1625
	precision = 10001
	done = false
	finished chan bool
)

func main() {
	//Parse flags for settings for the server
	portPtr := flag.Int("p", 8081, "Port number to run the server on")
	precPtr := flag.Int("a", 10001, "Accuracy/precision for calculations")
	iterPtr := flag.Int("l", 1625, "Limit; Value of infinity")
	direPtr := flag.Bool("reverse", false, "Create queue of work backwards")

	flag.Parse()

	port := *portPtr
	limit = *iterPtr
	direction := *direPtr
	precision = *precPtr

	//Create neccesary channels and vars
	total = decimal.WithPrecision(precision).SetUint64(0)
	work = make(chan int, limit)
	results = make(chan *decimal.Big, limit)

	switch direction {
	case false:
		for i := 0; i < limit; i++ { //Forwards
			work <- i
		}
	case true:
		for i := limit - 1; i >= 0; i-- { //Backwards
			work <- i
		}
	}

	go startServer(port)

	//Recover the results from queue
	for i := 0; i < limit; i++ {
		total = total.Add(total, <-results)
	}
	fmt.Println(total)
}

func startServer(port int) {
	apiRouter := mux.NewRouter()

	//API routes for work
	workRouter := apiRouter.PathPrefix("/work").Subrouter()
	/*Get some work	*/ workRouter.HandleFunc("/get", workHandler).Methods("GET")
	/*Get settings 	*/ workRouter.HandleFunc("/settings", settingSender).Methods("GET")

	//API routes for results
	resultRouter := apiRouter.PathPrefix("/results").Subrouter()
	/*Submit result	*/ resultRouter.HandleFunc("/submit", resultSubmit).Methods("POST")
	/*Show result  	*/ resultRouter.HandleFunc("/result", resultGet).Methods("GET")

	fmt.Printf("Listening for requests on 0.0.0.0:%v\r\n", port)
	http.ListenAndServe(fmt.Sprintf(":%v", port), apiRouter)
}
func settingSender(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%v;%v", precision, limit)
}

func resultSubmit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Result submitted by", r.RemoteAddr)
	//get body of request, add it to a queue of results to process because otherwise race conditions and multithread stuff will break things
	var buffer []byte
	buffer, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Printf("%s\n", buffer)
	res, _ := decimal.WithPrecision(precision).SetString(fmt.Sprintf("%s", buffer))
	results <- res
}

func resultGet(w http.ResponseWriter, r *http.Request) { //Gives result/current status to API
	//return the total, writing to the request
	fmt.Println("Result got by", r.RemoteAddr)
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, total)
}

func workHandler(w http.ResponseWriter, r *http.Request) { //Gives client some work
	//somehow reply to request with an int for some work to do
	fmt.Println("Work requested by",r.RemoteAddr)
	if len(work) != 0 {
		fmt.Fprint(w, <-work)
	} else {
		fmt.Fprint(w, "-1")
	}
}
