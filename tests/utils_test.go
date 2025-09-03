package tests

import (
	"emcsrw/utils"
	"testing"
)

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
