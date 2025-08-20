package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const OFFICIAL_API_URL = "https://api.earthmc.net/v3/aurora"
const TOOLKIT_API_URL = "https://emctoolkit.vercel.app/api/aurora"

var client = http.Client{Timeout: 8 * time.Second}

func TKAPIRequest[T any](endpoint string) (T, error) {
	return JsonGetRequest[T](TOOLKIT_API_URL + endpoint)
}

func OAPIGetRequest[T any](endpoint string) (T, error) {
	url := OFFICIAL_API_URL + endpoint
	res, err := JsonGetRequest[T](url)

	return res, err
}

func OAPIPostRequest[T any](endpoint string, body any) (T, error) {
	url := OFFICIAL_API_URL + endpoint
	res, err := JsonPostRequest[T](url, body)

	return res, err
}

// func OAPIConcurrentRequest[T any](endpoints []string, skipCache bool) ([]T, []error) {
// 	var results []T
// 	var errors []error

// 	lop.ForEach(endpoints, func(ep string, _ int) {
// 		res, err := OAPIGetRequest[T](ep)

// 		// Use JsonRequest here
// 		if err != nil {
// 			errors = append(errors, err)
// 		} else {
// 			results = append(results, res)
// 		}
// 	})

// 	return results, errors
// }

// Sends a POST request with a JSON body and since JSON is expected to be returned, the response is unmarshalled into the provided type.
func JsonPostRequest[T any](url string, body any) (T, error) {
	var data T

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Failed to Serialize to JSON from native Go struct type: %v", err)
	}

	res, err := PostRequest(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Println(string(res))
	}

	return data, err
}

func JsonGetRequest[T any](url string) (T, error) {
	var data T

	res, err := GetRequest(url)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Println(string(res))
	}

	return data, err
}

func PostRequest(url string, contentType string, body io.Reader) ([]byte, error) {
	response, err := client.Post(url, contentType, body)
	if err != nil {
		return nil, err
	}

	return ReadResponseBody(response, url)
}

func GetRequest(url string) ([]byte, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	return ReadResponseBody(response, url)
}

func ReadResponseBody(response *http.Response, url string) ([]byte, error) {
	if response.StatusCode == http.StatusNotFound {
		errStr := fmt.Errorf("404 Not Found: %s", url)
		fmt.Println(errStr)

		return nil, errStr
	}

	if response.StatusCode == http.StatusGatewayTimeout {
		errStr := fmt.Errorf("504 Gateway Timeout: %s", url)
		fmt.Println(errStr)

		return nil, errStr
	}

	defer response.Body.Close()
	return io.ReadAll(response.Body)
}
