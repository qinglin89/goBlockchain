package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type Block struct {
	Index int
	Timestamp string
	BPM int
	Hash string
	PrevHash string
	Nonce string
	Validator string
	//Difficulty int
}

func IsHashValid(hash string, difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

func CalculateHash(block Block) string{
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash + block.Nonce
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func GenerateBlock(oldBlock Block, BPM int, address string) (Block, error) {
	var newBlock Block
	t := time.Now()
	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = CalculateHash(newBlock)
	newBlock.Validator = address
	//newBlock.Difficulty = difficulty

	//for i := 0; ; i ++ {
	//	hex := fmt.Sprintf("%x", i)
		newBlock.Nonce = "pos"
		//if !IsHashValid(CalculateHash(newBlock), newBlock.Difficulty) {
		//	fmt.Println(CalculateHash(newBlock), "do more work!")
			//time.Sleep(time.Second)
			//continue
		//} else {
		//	fmt.Println(CalculateHash(newBlock), "work done!")
			newBlock.Hash = CalculateHash(newBlock)
			//log.Println(CalculateHash(newBlock))
			//break
		//}
	//}
	return newBlock, nil
}

func IsBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index + 1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if CalculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

func ReplaceChain(newBlocks []Block, currentBlockchain *[]Block) {
	if len(newBlocks) > len(*currentBlockchain) {
		//Blockchain = newBlocks
		currentBlockchain = &newBlocks
	}
}