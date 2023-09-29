package utils

import (
	"net/http"
	"time"
	"fmt"
	"io"

	lop "github.com/samber/lo/parallel"
)

var Domain = "https://api.earthmc.net/v2/aurora"

func SendRequest(endpoint string, skipCache bool) ([]byte, error) {
	if skipCache == true {
		randStr := RandomString(12)
		endpoint += randStr
	}

	url := fmt.Sprintf("%s%s", Domain, endpoint)
	client := http.Client{ Timeout: 10 * time.Second }

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
	defer response.Body.Close()

	return body, nil
}

func JsonRequest[T any](endpoint string, skipCache bool) (T, error) {
	var data T
	res, err := SendRequest(endpoint, skipCache)

	if err != nil { 
		return data, err
	}

	return ParseJSON[T](res, data)
}

func ConcurrentJsonRequests[T any](endpoints []string, skipCache bool) ([]T, []error) {
	var (
        results	[]T
		errors	[]error
    )

	lop.ForEach(endpoints, func(ep string, _ int) {
		res, err := JsonRequest[T](ep, skipCache)
	
		// Use `JsonRequest` here
		if err != nil {
			errors = append(errors, err)
		} else {
			results = append(results, res)
		}
	})

	return results, errors
}