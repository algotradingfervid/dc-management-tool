package models

import (
	"testing"

	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

func validateProject(p *Project) map[string]string {
	errors := helpers.ValidateStruct(p)
	// SeqPadding: 0 is valid (default), but if set must be 2-6
	if p.SeqPadding != 0 && (p.SeqPadding < 2 || p.SeqPadding > 6) {
		errors["seq_padding"] = "Sequence padding must be between 2 and 6"
	}
	return errors
}

func TestProjectValidate(t *testing.T) {
	tests := []struct {
		name       string
		project    Project
		wantErrors []string
	}{
		{
			name:       "valid project",
			project:    Project{Name: "Test", DCPrefix: "SCP"},
			wantErrors: nil,
		},
		{
			name:       "missing name",
			project:    Project{DCPrefix: "SCP"},
			wantErrors: []string{"name"},
		},
		{
			name:       "missing prefix",
			project:    Project{Name: "Test"},
			wantErrors: []string{"dc_prefix"},
		},
		{
			name:       "prefix too long",
			project:    Project{Name: "Test", DCPrefix: "TOOLONGPREFIX"},
			wantErrors: []string{"dc_prefix"},
		},
		{
			name:       "invalid GSTIN",
			project:    Project{Name: "Test", DCPrefix: "SCP", CompanyGSTIN: "short"},
			wantErrors: []string{"company_gstin"},
		},
		{
			name:       "valid GSTIN",
			project:    Project{Name: "Test", DCPrefix: "SCP", CompanyGSTIN: "36AACCF9742K1Z8"},
			wantErrors: nil,
		},
		{
			name:       "invalid email",
			project:    Project{Name: "Test", DCPrefix: "SCP", CompanyEmail: "notanemail"},
			wantErrors: []string{"company_email"},
		},
		{
			name:       "valid email",
			project:    Project{Name: "Test", DCPrefix: "SCP", CompanyEmail: "test@example.com"},
			wantErrors: nil,
		},
		{
			name:       "seq padding too small",
			project:    Project{Name: "Test", DCPrefix: "SCP", SeqPadding: 1},
			wantErrors: []string{"seq_padding"},
		},
		{
			name:       "seq padding too large",
			project:    Project{Name: "Test", DCPrefix: "SCP", SeqPadding: 7},
			wantErrors: []string{"seq_padding"},
		},
		{
			name:       "valid seq padding",
			project:    Project{Name: "Test", DCPrefix: "SCP", SeqPadding: 4},
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateProject(&tt.project)
			if tt.wantErrors == nil {
				if len(errors) != 0 {
					t.Errorf("expected no errors, got %v", errors)
				}
				return
			}
			for _, key := range tt.wantErrors {
				if _, ok := errors[key]; !ok {
					t.Errorf("expected error for %q, got errors: %v", key, errors)
				}
			}
		})
	}
}
