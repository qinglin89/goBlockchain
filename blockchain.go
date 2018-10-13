package main

import (
	"bufio"
	"strconv"
	"sync"

	//  "fmt"
	"crypto/sha256"
	"encoding/hex"
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

type Block struct {
  Index int
  Timestamp string
  BPM int
  Hash string
  PrevHash string
}

var Blockchain []Block

func calculateHash(block Block) string{
  record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
  h := sha256.New()
  h.Write([]byte(record))
  hashed := h.Sum(nil)
  return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, BPM int) (Block, error) {
  var newBlock Block
  t := time.Now()
  newBlock.Index = oldBlock.Index + 1
  newBlock.Timestamp = t.String()
  newBlock.BPM = BPM
  newBlock.PrevHash = oldBlock.Hash
  newBlock.Hash = calculateHash(newBlock)

  return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {
  if oldBlock.Index + 1 != newBlock.Index {
    return false
  }

  if oldBlock.Hash != newBlock.PrevHash {
    return false
  }

  if calculateHash(newBlock) != newBlock.Hash {
    return false
  }

  return true
}

func replaceChain(newBlocks []Block) {
  if len(newBlocks) > len(Blockchain) {
    Blockchain = newBlocks
  }
}

func run() error {
  mux := makeMuxRouter()
  httpAddr := os.Getenv("ADDR")
  log.Println("Listening on ", httpAddr)
  s := &http.Server{
    Addr: ":" + httpAddr,
    Handler: mux,
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

  newBlock, err := generateBlock(Blockchain[len(Blockchain) - 1], m.BPM)
  if err != nil {
    respondWithJSON(w, r, http.StatusInternalServerError, m)
    return
  }
  if isBlockValid(newBlock, Blockchain[len(Blockchain) - 1]) {
    newBlockchain := append(Blockchain, newBlock)
    replaceChain(newBlockchain)
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

var bcServer chan []Block
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
			newBlock, err := generateBlock(Blockchain[len(Blockchain) - 1], bpm)
			if err != nil {
				log.Println(err)
				continue
			}
			if isBlockValid(newBlock, Blockchain[len(Blockchain) - 1]) {
				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)
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

  bcServer = make(chan []Block)

  go func() {
    t := time.Now()
    genesisBlock := Block{0, t.String(), 0, "", ""}
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
