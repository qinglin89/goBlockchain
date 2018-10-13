package main

import (
	"bufio"
	"learn/goBlockchain/utils"
	"strconv"
	"sync"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var difficulty int = 1
var Blockchain []utils.Block

func run() error {
  muxR := makeMuxRouter()
  httpAddr := os.Getenv("ADDR")
  log.Println("Listening on ", httpAddr)
  s := &http.Server{
    Addr: ":" + httpAddr,
    Handler: muxR,
    ReadTimeout: 10 * time.Second,
    WriteTimeout: 10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }

  if err := s.ListenAndServe(); err != nil {
    return err
  }

  return nil
}

func makeMuxRouter() http.Handler {
    muxRouter := mux.NewRouter()
    muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
    muxRouter.HandleFunc("/", handleWriteBlockchain).Methods("POST")
    return muxRouter
}

type Message struct {
  BPM int
}

func handleGetBlockchain (w http.ResponseWriter, r *http.Request) {
  bytes, err := json.MarshalIndent(Blockchain, "", "  ")
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  //io.WriteString(w, string(bytes))
  //w.WriteString(string(bytes))
  //w.Write(bytes)
  fmt.Fprint(w, string(bytes))
}


func handleWriteBlockchain(w http.ResponseWriter, r *http.Request) {
  var m Message
  decoder := json.NewDecoder(r.Body)
  log.Println(r)
  log.Println(r.Body)
  if err := decoder.Decode(&m); err != nil {
    log.Println(err) 
    respondWithJSON(w, r, http.StatusBadRequest, r.Body)
    return
  }
  defer r.Body.Close()

  newBlock, err := utils.GenerateBlock(Blockchain[len(Blockchain) - 1], m.BPM, difficulty)
  if err != nil {
    respondWithJSON(w, r, http.StatusInternalServerError, m)
    return
  }
  if utils.IsBlockValid(newBlock, Blockchain[len(Blockchain) - 1]) {
    newBlockchain := append(Blockchain, newBlock)
    utils.ReplaceChain(newBlockchain, &Blockchain)
    spew.Dump(Blockchain)
  }
  respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
  response, err := json.MarshalIndent(payload, "", " ")
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte("HTTP 500:Internal Server Error"))
    return
  }
  w.WriteHeader(code)
  w.Write(response)
}

var bcServer chan []utils.Block
var mutex = &sync.Mutex{}

func handleConn(conn net.Conn) {
	io.WriteString(conn, "Enter a new BPM:")
	scanner := bufio.NewScanner(conn)

	go func(){
		for scanner.Scan() {
			bpm, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Printf("%v not a number: %v", scanner.Text(), err)
				continue
			}
			newBlock, err := utils.GenerateBlock(Blockchain[len(Blockchain) - 1], bpm, difficulty)
			if err != nil {
				log.Println(err)
				continue
			}
			if utils.IsBlockValid(newBlock, Blockchain[len(Blockchain) - 1]) {
				newBlockchain := append(Blockchain, newBlock)
				utils.ReplaceChain(newBlockchain, &Blockchain)
			}
			spew.Dump(Blockchain)
			bcServer <- Blockchain
			io.WriteString(conn, "\nEnter a new BPM:")
		}
	}()

	go func(){
		for {
			time.Sleep(30 * time.Second)
			mutex.Lock()
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			mutex.Unlock()
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		//spew.Dump(Blockchain)
	}
}


func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal(err)
  }

  bcServer = make(chan []utils.Block)

  go func() {
    t := time.Now()
    genesisBlock := utils.Block{0, t.String(), 0, "", "", "", 1}
    spew.Dump(genesisBlock)
    Blockchain = append(Blockchain, genesisBlock)
  }()

  server, err := net.Listen("tcp", ":" + os.Getenv("ADDR"))
  if err != nil {
  	log.Fatal(err)
  }
  defer server.Close()

  for {
  	conn, err := server.Accept()
  	if err != nil {
  		log.Fatal(err)
	}
  	go handleConn(conn)
  }

  log.Fatal(run())

}