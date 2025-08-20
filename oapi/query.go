package oapi

type NamesQuery struct {
	Query []string `json:"query"`
}

func NewNamesQuery(names ...string) NamesQuery {
	return NamesQuery{Query: names}
}
