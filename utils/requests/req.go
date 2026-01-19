package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var pingClient = http.Client{Timeout: 2 * time.Second} // Use when performing HEAD requests.
var client = http.Client{Timeout: 8 * time.Second}     // Use when performing all other requests.

// Sends a HEAD request to url, returning the received response.
func Head(url string) (*http.Response, error) {
	r, err := pingClient.Head(url)
	if err != nil {
		return nil, err // network error or timeout
	}

	defer r.Body.Close()
	return r, nil
}

//#region GET

// Sends a request with the "GET" method without a body and reads the response body.
// It is up to the caller to know how to read the byte[].
// If using this func just to unmarshal to JSON, prefer JsonGet().
func Get(url string) ([]byte, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error during GET request to %s:\n\t%s", url, err)
	}

	if _, ok := GetResponseStatus(response.StatusCode); !ok {
		err := fmt.Errorf("%s. refused to read body of non-OK response", response.Status)
		return nil, fmt.Errorf("error during GET request to %s:\n\t%s", url, err)
	}

	resBody, err := ReadResponseBody(response, url)
	if err != nil {
		err = fmt.Errorf("error during GET request to %s:\n\t%s", url, err)
	}

	return resBody, err
}

// Sends a request without a body using the "GET" method.
//
// Since JSON is expected to be returned, the response is unmarshalled into T.
func JsonGet[T any](url string) (T, error) {
	var data T

	res, err := Get(url)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Printf("\n[GET] failed to unmarshal response body into struct:\n%v\n", err)
	}

	return data, err
}

//#endregion

//#region POST

// Sends a request with a body using the "POST" method and reads the response body.
// It is up to the caller to know how to read the byte[].
//
// If using this func only to unmarshal to JSON, prefer JsonPost().
func Post(url string, contentType string, reqBody io.Reader) ([]byte, error) {
	response, err := client.Post(url, contentType, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error during POST request to %s:\n\t%s", url, err)
	}

	if _, ok := GetResponseStatus(response.StatusCode); !ok {
		err := fmt.Errorf("%s. refused to read body of non-OK response", response.Status)
		return nil, fmt.Errorf("error during POST request to %s:\n\t%s", url, err)
	}

	resBody, err := ReadResponseBody(response, url)
	if err != nil {
		err = fmt.Errorf("error during POST request to %s:\n\t%s", url, err)
	}

	return resBody, err
}

// Sends a request with a JSON body using the "POST" method.
//
// Since JSON is expected to be returned, the response is unmarshalled into T.
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

//#endregion
