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

func TestIsHexColor(t *testing.T) {
	realCol := "#1A2B3C"
	ok := utils.ValidateHexColour(realCol)
	if !ok {
		t.Errorf("Expected a valid HEX colour, got '%v'", realCol)
	}

	bogusCol := "#bogus"
	ok = utils.ValidateHexColour(bogusCol)
	if ok {
		t.Errorf("Expected an invalid HEX colour. got valid one: %v", bogusCol)
	}

	longCol := "#1A2B3Cffff"
	ok = utils.ValidateHexColour(longCol)
	if ok {
		t.Errorf("Expected to fail when HEX color is too long. '%v' is not 6 chars long.", longCol)
	}
}
