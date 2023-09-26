package utils

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var Domain = "https://api.earthmc.net/v2/aurora"

func SendRequest(endpoint string, skipCache bool) ([]byte, error) {
	if skipCache == true {
		randStr := RandomString(12)
		endpoint += randStr
	}

	url := fmt.Sprintf("%s%s", Domain, endpoint)
	client := http.Client{ Timeout: 6 * time.Second }

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404 not found: %s", url)
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
		wg		sync.WaitGroup
    )

	for _, ep := range endpoints {
		wg.Add(1)

		go func(ep string) {
			res, err := JsonRequest[T](ep, skipCache)
	
			// Use `JsonRequest` here
			if err != nil {
				errors = append(errors, err)
			} else {
				results = append(results, res)
			}
			
			defer wg.Done()
		}(ep)
	}

	wg.Wait()

	return results, errors
}