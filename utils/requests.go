package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	lop "github.com/samber/lo/parallel"
)

const OFFICIAL_API_URL = "https://api.earthmc.net/v3/aurora"
const TOOLKIT_API_URL = "https://emctoolkit.vercel.app/api/aurora"

var client = http.Client{Timeout: 8 * time.Second}

func TKAPIRequest[T any](endpoint string) (T, error) {
	return JsonGetRequest[T](TOOLKIT_API_URL + endpoint)
}

func OAPIRequest[T any](endpoint string) (T, error) {
	url := OFFICIAL_API_URL + endpoint
	res, err := JsonGetRequest[T](url)

	return res, err
}

func OAPIConcurrentRequest[T any](endpoints []string, skipCache bool) ([]T, []error) {
	var results []T
	var errors []error

	lop.ForEach(endpoints, func(ep string, _ int) {
		res, err := OAPIRequest[T](ep)

		// Use `JsonRequest` here
		if err != nil {
			errors = append(errors, err)
		} else {
			results = append(results, res)
		}
	})

	return results, errors
}

func JsonGetRequest[T any](endpoint string) (T, error) {
	var data T

	res, err := GetRequest(endpoint)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(res, &data)
	if err != nil {
		fmt.Println(string(res))
	}

	return data, err
}

// TODO: Implement me
func PostRequest() ([]byte, error) {
	return nil, nil
}

func GetRequest(url string) ([]byte, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

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

	body, _ := io.ReadAll(response.Body)
	defer response.Body.Close() // TODO: Already at end of function, why defer??

	return body, nil
}
