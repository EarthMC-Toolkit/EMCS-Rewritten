package utils

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"
)

var alphabet []rune = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
var alphabetLen = len(alphabet)
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomString(length int) string {
	var sb strings.Builder
  
	for i := 0; i < length; i++ {
		ch := alphabet[rand.Intn(alphabetLen)]
	  	sb.WriteRune(ch)
	}
  
	return sb.String()
}

func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}