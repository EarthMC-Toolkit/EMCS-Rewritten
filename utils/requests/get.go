package requests

import (
	"encoding/json"
	"fmt"
)

func Get(url string) ([]byte, error) {
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
