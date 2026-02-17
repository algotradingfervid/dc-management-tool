package models

import (
	"strings"
	"testing"
)

func TestDCTemplateValidate_Valid(t *testing.T) {
	tmpl := &DCTemplate{
		Name:    "Test Template",
		Purpose: "For testing",
	}
	errors := tmpl.Validate()
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %v", errors)
	}
}

func TestDCTemplateValidate_EmptyName(t *testing.T) {
	tmpl := &DCTemplate{Name: "", Purpose: "ok"}
	errors := tmpl.Validate()
	if _, ok := errors["name"]; !ok {
		t.Error("Expected error for empty name")
	}
}

func TestDCTemplateValidate_WhitespaceName(t *testing.T) {
	tmpl := &DCTemplate{Name: "   ", Purpose: "ok"}
	errors := tmpl.Validate()
	if _, ok := errors["name"]; !ok {
		t.Error("Expected error for whitespace-only name")
	}
}

func TestDCTemplateValidate_NameTooLong(t *testing.T) {
	tmpl := &DCTemplate{Name: strings.Repeat("a", 101), Purpose: "ok"}
	errors := tmpl.Validate()
	if _, ok := errors["name"]; !ok {
		t.Error("Expected error for name > 100 chars")
	}
}

func TestDCTemplateValidate_NameExactly100(t *testing.T) {
	tmpl := &DCTemplate{Name: strings.Repeat("a", 100), Purpose: "ok"}
	errors := tmpl.Validate()
	if _, ok := errors["name"]; ok {
		t.Error("Name of exactly 100 chars should be valid")
	}
}

func TestDCTemplateValidate_PurposeTooLong(t *testing.T) {
	tmpl := &DCTemplate{Name: "Valid", Purpose: strings.Repeat("a", 501)}
	errors := tmpl.Validate()
	if _, ok := errors["purpose"]; !ok {
		t.Error("Expected error for purpose > 500 chars")
	}
}

func TestDCTemplateValidate_PurposeExactly500(t *testing.T) {
	tmpl := &DCTemplate{Name: "Valid", Purpose: strings.Repeat("a", 500)}
	errors := tmpl.Validate()
	if _, ok := errors["purpose"]; ok {
		t.Error("Purpose of exactly 500 chars should be valid")
	}
}

func TestDCTemplateValidate_EmptyPurpose(t *testing.T) {
	tmpl := &DCTemplate{Name: "Valid", Purpose: ""}
	errors := tmpl.Validate()
	if len(errors) != 0 {
		t.Errorf("Empty purpose should be valid, got %v", errors)
	}
}
