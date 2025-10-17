package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var client = http.Client{Timeout: 10 * time.Second}

//const TOOLKIT_API_URL = "https://emctoolkit.vercel.app/api/aurora"

// func TKAPIRequest[T any](endpoint string) (T, error) {
// 	return JsonGetRequest[T](TOOLKIT_API_URL + endpoint)
// }

// Sends a POST request with a JSON body and since JSON is expected to be returned, the response is unmarshalled into the provided type.
func JsonPostRequest[T any](url string, body any) (T, error) {
	var data T

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("\nfailed to marshal query body into byte arr:\n%v\n", err)
	}

	res, err := PostRequest(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Printf("\n[POST] failed to unmarshal response body into struct:\n%v\n", err)
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
		fmt.Printf("\n[GET] failed to unmarshal response body into struct:\n%v\n", err)
	}

	return data, err
}

func PostRequest(url string, contentType string, reqBody io.Reader) ([]byte, error) {
	response, err := client.Post(url, contentType, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error during POST request to %s:\n  %v", url, err)
	}

	resBody, err := ReadResponseBody(response, url)
	if err != nil {
		err = fmt.Errorf("error during POST request to %s:\n  %v", url, err)
	}

	return resBody, err
}

func GetRequest(url string) ([]byte, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error during GET request to %s:\n  %v", url, err)
	}

	resBody, err := ReadResponseBody(response, url)
	if err != nil {
		err = fmt.Errorf("error during GET request to %s:\n  %v", url, err)
	}

	return resBody, err
}

// Reads the response body all at once with [io.ReadAll], but with an additional check for client/server error codes so that we know the body
// is safe to read. If the status code is <400 (successful, informational or redirectional). If the caller is not expecting an empty body,
// they should handle it appropriately with a length check as no error will be output in such a case.
func ReadResponseBody(response *http.Response, url string) ([]byte, error) {
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to read response body. %s", response.Status)
	}

	defer response.Body.Close()
	return io.ReadAll(response.Body)
}
