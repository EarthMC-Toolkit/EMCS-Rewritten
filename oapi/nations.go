package oapi

import (
	"emcsrw/oapi/objs"
	"emcsrw/utils"
)

func QueryNations(identifiers ...string) ([]objs.NationInfo, error) {
	return utils.OAPIPostRequest[[]objs.NationInfo]("/nations", NewNamesQuery(identifiers...))
}

// func ConcurrentNations(identifiers []string) ([]objs.NationInfo, []error) {
// 	var endpoints []string
// 	for _, identifier := range identifiers {
// 		endpoints = append(endpoints, fmt.Sprintf("/nations", identifier))
// 	}

// 	return utils.OAPIConcurrentRequest[objs.NationInfo](endpoints, true)
// }
