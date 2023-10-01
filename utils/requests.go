package utils

import (
	"net/http"
	"time"
	"fmt"
	"io"

	lop "github.com/samber/lo/parallel"
)

const OfficialApiDomain = "https://api.earthmc.net/v2/aurora"
const ToolkitApiDomain = "https://emctoolkit.vercel.app/api/aurora"

func TKAPIRequest[T interface{}](endpoint string) (T, error) {
	return JsonRequest[T](ToolkitApiDomain + endpoint)
}

func OAPIRequest[T interface{}](endpoint string, skipCache bool) (T, error) {
	if skipCache == true {
		endpoint += RandomString(10)
	}

	url := OfficialApiDomain + endpoint
	res, err := JsonRequest[T](url)

	return res, err
}

var client = http.Client{ Timeout: 8 * time.Second }
func Request(url string) ([]byte, error) {
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

func JsonRequest[T interface{}](endpoint string) (T, error) {
	var data T
	res, err := Request(endpoint)

	if err != nil { 
		return data, err
	}

	parsed, err := ParseJSON[T](res, data)

	if err != nil {
		fmt.Println(string(res))
	}

	return parsed, err
}

func OAPIConcurrentRequest[T any](endpoints []string, skipCache bool) ([]T, []error) {
	var (
        results	[]T
		errors	[]error
    )

	lop.ForEach(endpoints, func(ep string, _ int) {
		res, err := OAPIRequest[T](ep, skipCache)
	
		// Use `JsonRequest` here
		if err != nil {
			errors = append(errors, err)
		} else {
			results = append(results, res)
		}
	})

	return results, errors
}