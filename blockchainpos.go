package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"learn/goBlockchain/utils"
	"math/rand"
	//"runtime"
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

//var difficulty = 1
var Blockchain []utils.Block
var announcements = make(chan string)
var tempBlocks []utils.Block
var candidateBlocks = make(chan utils.Block)
var validators = make(map[string]int)

func run() error {
	muxR := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", httpAddr)
	s := &http.Server{
		Addr: ":8090",
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

	newBlock, err := utils.GenerateBlock(Blockchain[len(Blockchain) - 1], m.BPM, "")
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

func calhash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	go func(){
		for {
			msg := <- announcements
			io.WriteString(conn, msg)
		}
	}()
	var address string
	io.WriteString(conn, "Enter token balance:")
	scannerBalance := bufio.NewScanner(conn)
	for scannerBalance.Scan() {
		balance, err := strconv.Atoi(scannerBalance.Text())
		if err != nil {
			log.Printf("%v not a number")
		}
		t := time.Now()
		address = calhash(t.String())
		validators[address] = balance
		fmt.Println(validators)
		break
	}

	io.WriteString(conn, "Enter a new BPM:")

	scannerBPM :=bufio.NewScanner(conn)

	go func (){
		for {
			for scannerBPM.Scan() {
				bpm, err := strconv.Atoi(scannerBPM.Text())
				if(err != nil) {
					log.Printf("%v not a number %v", scannerBPM.Text(), err)
					delete(validators, address)
					conn.Close()
				}

				mutex.Lock()
				oldLastIndex := Blockchain[len(Blockchain) - 1]
				mutex.Unlock()

				newBlock, err := utils.GenerateBlock(oldLastIndex, bpm, address)
				if err != nil {
					log.Println(err)
					continue
				}
				if utils.IsBlockValid(newBlock, oldLastIndex) {
					candidateBlocks <- newBlock
				}
				io.WriteString(conn, "\nEnter a new BPM:")
			}
		}
	}()

	for {
		time.Sleep(35 * time.Second)
		mutex.Lock()
		output, err := json.Marshal(Blockchain)
		mutex.Unlock()
		if err != nil {
			log.Fatal(err)
		}
		io.WriteString(conn, string(output) + "\n")
	}
	//io.WriteString(conn, "Enter a new BPM:")
	//scanner := bufio.NewScanner(conn)
	//
	//go func(){
	//	for scanner.Scan() {
	//		bpm, err := strconv.Atoi(scanner.Text())
	//		if err != nil {
	//			log.Printf("%v not a number: %v", scanner.Text(), err)
	//			continue
	//		}
	//		newBlock, err := utils.GenerateBlock(Blockchain[len(Blockchain) - 1], bpm)
	//		if err != nil {
	//			log.Println(err)
	//			continue
	//		}
	//		if utils.IsBlockValid(newBlock, Blockchain[len(Blockchain) - 1]) {
	//			newBlockchain := append(Blockchain, newBlock)
	//			utils.ReplaceChain(newBlockchain, &Blockchain)
	//		}
	//		spew.Dump(Blockchain)
	//		bcServer <- Blockchain
	//		io.WriteString(conn, "\nEnter a new BPM:")
	//	}
	//}()
	//
	//go func(){
	//	for {
	//		time.Sleep(30 * time.Second)
	//		mutex.Lock()
	//		output, err := json.Marshal(Blockchain)
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//		mutex.Unlock()
	//		io.WriteString(conn, string(output))
	//	}
	//}()
	//
	//for _ = range bcServer {
	//	//spew.Dump(Blockchain)
	//}
}

func pickWinner() {
	time.Sleep(30 * time.Second)
	mutex.Lock()
	temp := tempBlocks
	mutex.Unlock()

	lotteryPool := []string{}

	if len(temp) > 0 {
		OUTER:
			for _, block := range temp {
				for _, node := range lotteryPool {
					if block.Validator == node {
						continue OUTER
					}
				}
				mutex.Lock()
				setValidators := validators
				mutex.Unlock()
				k, ok := setValidators[block.Validator]
				if ok {
					for i := 0; i < k; i ++ {
						lotteryPool = append(lotteryPool, block.Validator)
					}
				}
			}
			s := rand.NewSource(time.Now().Unix())
			r := rand.New(s)

			lotteryWinner := lotteryPool[r.Intn(len(lotteryPool))]

			for _, block := range temp {
				if block.Validator == lotteryWinner {
					mutex.Lock()
					Blockchain = append(Blockchain, block)
					mutex.Unlock()
					for range validators {
						announcements <- "\nwinning validator: " + lotteryWinner + "\n"
					}
					break
				}
			}
	}
	mutex.Lock()
	tempBlocks = []utils.Block{}
	mutex.Unlock()
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	bcServer = make(chan []utils.Block)

	go func() {
		t := time.Now()
		genesisBlock := utils.Block{0, t.String(), 0, "", "", "pos", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()

	server, err := net.Listen("tcp", ":" + os.Getenv("ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	go func() {
		for candidate := range candidateBlocks {
			mutex.Lock()
			tempBlocks = append(tempBlocks, candidate)
			mutex.Unlock()
		}
	}()

	go func() {
		for {
			pickWinner()
		}
	}()
	log.Fatal(run())
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}

}