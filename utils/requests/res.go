package requests

import (
	"io"
	"net/http"
)

type ResponseStatus = int

const (
	RESPONSE_STATUS_DOWN    ResponseStatus = iota // Request succeeded but server error code.
	RESPONSE_STATUS_PARTIAL                       // Request went through but not success or down.
	RESPONSE_STATUS_FAILED                        // Request failed to go through due to network or dns error.
	RESPONSE_STATUS_OK                            // Everything succeeded as it should like a good little request.
)

func WithResponseStatus(r *http.Response, e error) (status ResponseStatus, ok bool) {
	if e != nil {
		return RESPONSE_STATUS_FAILED, false
	}

	return GetResponseStatus(r.StatusCode)
}

func GetResponseStatus(code int) (status ResponseStatus, ok bool) {
	// 501 Not Implemented not required here due to its very nature:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/501
	switch code {
	case 429, 304, 204, 200:
		return RESPONSE_STATUS_OK, true
	case 505, 503, 502, 500, 404:
		return RESPONSE_STATUS_DOWN, false
	}

	// Any non-error or non-success is a grey area.
	// In this case we mark the status as "not ok" as it is probably safer to just
	// not send a request if we aren't 100% sure we will get a body.
	return RESPONSE_STATUS_PARTIAL, false
}

// Reads the response body all at once with [io.ReadAll], but with an additional check for client/server error codes so that we know the body
// is safe to read. If the status code is <400 (successful, informational or redirectional). If the caller is not expecting an empty body,
// they should handle it appropriately with a length check as no error will be output in such a case.
func ReadResponseBody(r *http.Response, url string) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
