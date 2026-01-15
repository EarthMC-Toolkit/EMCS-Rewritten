package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func Post(url string, contentType string, reqBody io.Reader) ([]byte, error) {
	http.DefaultClient.Timeout = 10 * time.Second

	response, err := http.Post(url, contentType, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error during POST request to %s:\n  %v", url, err)
	}

	resBody, err := ReadResponseBody(response, url)
	if err != nil {
		err = fmt.Errorf("error during POST request to %s:\n  %v", url, err)
	}

	return resBody, err
}

// Sends a POST request with a JSON body and since JSON is expected to be returned, the response is unmarshalled into the provided type.
func JsonPost[T any](url string, body any) (T, error) {
	var data T

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("\nfailed to marshal query body into byte slice:\n%v\n", err)
	}

	res, err := Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Printf("\n[POST] failed to unmarshal response body into struct:\n%v\n", err)
	}

	return data, err
}
