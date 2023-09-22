package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const domain string = "https://api.earthmc.net/v2/aurora"

func SendRequest(endpoint string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", domain, endpoint)
	client := http.Client{ Timeout: 6 * time.Second }

	fmt.Println("Sending request to: " + url)

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(response.Body)
	defer response.Body.Close()

	return body, nil
}

func JsonRequest[T any](endpoint string) (T, error) {
	var data T
	res, err := SendRequest(endpoint)

	if err != nil { 
		return data, err
	}

	json.Unmarshal([]byte(res), &data)

	return data, nil
}