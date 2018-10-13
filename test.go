package main

import "fmt"
import "reflect"
import "encoding/hex"
import "encoding/json"

type jt struct {
	A int `json:"re"`
  B string
}

func main() {
  x := []byte{12,44,6}
  y := hex.EncodeToString(x)
  fmt.Println(y)
  z := []jt{{
    A: 100,
    B: "what",
  }, {200, "hehe"}}
  w, err := json.MarshalIndent(z, "a", "b")
  if err != nil {
    fmt.Println(err)
  }
  fmt.Println(reflect.Typeof(w))
}
