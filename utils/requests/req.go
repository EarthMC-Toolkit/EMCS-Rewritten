package requests

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var client = http.Client{Timeout: 10 * time.Second}

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
