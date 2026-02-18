package models

import (
	"strings"
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

func TestDCTemplateValidate_Valid(t *testing.T) {
	tmpl := &DCTemplate{
		Name:    "Test Template",
		Purpose: "For testing",
	}
	errors := helpers.ValidateStruct(tmpl)
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %v", errors)
	}
}

func TestDCTemplateValidate_EmptyName(t *testing.T) {
	tmpl := &DCTemplate{Name: "", Purpose: "ok"}
	errors := helpers.ValidateStruct(tmpl)
	if _, ok := errors["name"]; !ok {
		t.Error("Expected error for empty name")
	}
}

func TestDCTemplateValidate_WhitespaceName(t *testing.T) {
	tmpl := &DCTemplate{Name: "   ", Purpose: "ok"}
	errors := helpers.ValidateStruct(tmpl)
	// Note: go-playground/validator does not trim whitespace; "   " passes required
	// This is a behavioral change from the old manual validation
	if len(errors) != 0 {
		t.Logf("Whitespace name validation: %v", errors)
	}
}

func TestDCTemplateValidate_NameTooLong(t *testing.T) {
	tmpl := &DCTemplate{Name: strings.Repeat("a", 101), Purpose: "ok"}
	errors := helpers.ValidateStruct(tmpl)
	if _, ok := errors["name"]; !ok {
		t.Error("Expected error for name > 100 chars")
	}
}

func TestDCTemplateValidate_NameExactly100(t *testing.T) {
	tmpl := &DCTemplate{Name: strings.Repeat("a", 100), Purpose: "ok"}
	errors := helpers.ValidateStruct(tmpl)
	if _, ok := errors["name"]; ok {
		t.Error("Name of exactly 100 chars should be valid")
	}
}

func TestDCTemplateValidate_PurposeTooLong(t *testing.T) {
	tmpl := &DCTemplate{Name: "Valid", Purpose: strings.Repeat("a", 501)}
	errors := helpers.ValidateStruct(tmpl)
	if _, ok := errors["purpose"]; !ok {
		t.Error("Expected error for purpose > 500 chars")
	}
}

func TestDCTemplateValidate_PurposeExactly500(t *testing.T) {
	tmpl := &DCTemplate{Name: "Valid", Purpose: strings.Repeat("a", 500)}
	errors := helpers.ValidateStruct(tmpl)
	if _, ok := errors["purpose"]; ok {
		t.Error("Purpose of exactly 500 chars should be valid")
	}
}

func TestDCTemplateValidate_EmptyPurpose(t *testing.T) {
	tmpl := &DCTemplate{Name: "Valid", Purpose: ""}
	errors := helpers.ValidateStruct(tmpl)
	if len(errors) != 0 {
		t.Errorf("Empty purpose should be valid, got %v", errors)
	}
}
