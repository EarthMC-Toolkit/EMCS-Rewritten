package main

import (
	"emcsrw/oapi"
	"emcsrw/utils"
	"testing"
)

func TestServerInfo(t *testing.T) {
	t.SkipNow()

	info, err := oapi.ServerInfo()
	logVal(t, info, err)
}

func TestTown(t *testing.T) {
	t.SkipNow()

	town, err := oapi.Town("Venice")
	logVal(t, town, err)
}

func TestNation(t *testing.T) {
	t.SkipNow()

	nation, err := oapi.Nation("Venice")
	logVal(t, nation, err)
}

func TestResident(t *testing.T) {
	t.SkipNow()

	res, err := oapi.Resident("Fruitloopins")
	logVal(t, res, err)
}

func TestConcurrentResidents(t *testing.T) {
	t.SkipNow()

	residents, _ := oapi.ConcurrentResidents([]string{"Owen3H", "Fruitloopins"})
	logVal(t, residents, nil)
}

func TestAlphanumeric(t *testing.T) {
	var actual bool

	actual = utils.ContainsNonAlphanumeric("<blue> owen")
	if actual != true {
		t.Errorf("Expected '%v' but got '%v'", true, actual)
	}

	actual = utils.ContainsNonAlphanumeric("owen3h")
	if actual != false {
		t.Errorf("Expected '%v' but got '%v'", false, actual)
	}
}

func logVal(t *testing.T, v any, e error) {
	if e != nil {
		t.Log(e)
	} else {
		t.Log(utils.Prettify(v))
	}
}