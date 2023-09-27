package main

import (
	"emcsrw/oapi"
	"emcsrw/utils"
	"testing"
)

func TestServerInfo(t *testing.T) {
	info, err := oapi.ServerInfo()
	logVal(t, info, err)
}

func TestTown(t *testing.T) {
	town, err := oapi.Town("Venice")
	logVal(t, town, err)
}

func TestNation(t *testing.T) {
	nation, err := oapi.Nation("Venice")
	logVal(t, nation, err)
}

func TestResident(t *testing.T) {
	res, err := oapi.Resident("Fruitloopins")
	logVal(t, res, err)
}

func logVal(t *testing.T, v any, e error) {
	if e != nil {
		t.Log(e)
	} else {
		t.Log(utils.Prettify(v))
	}
}