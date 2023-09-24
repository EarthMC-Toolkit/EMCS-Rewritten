package utils

import (
	"encoding/json"
	"math/rand"
	"strconv"
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

func HexToInt(hex string) int {
	str := strings.Replace(hex, "0x", "", -1)
	output, _ := strconv.ParseUint(str, 16, 32)

	return int(output)
}

func FormatTimestamp(unixTs float64) string {
	return strconv.FormatFloat(unixTs / 1000, 'f', 0, 64)
}