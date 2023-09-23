package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const domain string = "https://api.earthmc.net/v2/aurora"

func SendRequest(endpoint string, skipCache bool) ([]byte, error) {
	if skipCache == true {
		randStr := RandomString(12)
		endpoint += randStr
	}

	url := fmt.Sprintf("%s%s", domain, endpoint)
	client := http.Client{ Timeout: 6 * time.Second }

	fmt.Println("Request sent to: " + endpoint)

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(response.Body)
	defer response.Body.Close()

	return body, nil
}

func JsonRequest[T any](endpoint string, skipCache bool) (T, error) {
	var data T
	res, err := SendRequest(endpoint, skipCache)

	if err != nil { 
		return data, err
	}

	json.Unmarshal([]byte(res), &data)

	return data, nil
}