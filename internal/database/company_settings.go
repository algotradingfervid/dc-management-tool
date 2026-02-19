package database

import (
	"context"
	"database/sql"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetCompanySettings fetches the single company settings row.
func GetCompanySettings() (*models.CompanySettings, error) {
	q := db.New(DB)
	row, err := q.GetCompanySettings(context.Background())
	if err != nil {
		return nil, err
	}
	return mapCompanySettings(row), nil
}

// UpdateCompanySettings persists all editable fields for the single company settings row.
func UpdateCompanySettings(cs *models.CompanySettings) error {
	q := db.New(DB)
	return q.UpdateCompanySettings(context.Background(), db.UpdateCompanySettingsParams{
		Name:           cs.Name,
		Address:        sql.NullString{String: cs.Address, Valid: cs.Address != ""},
		City:           sql.NullString{String: cs.City, Valid: cs.City != ""},
		State:          sql.NullString{String: cs.State, Valid: cs.State != ""},
		StateCode:      sql.NullString{String: cs.StateCode, Valid: cs.StateCode != ""},
		Pincode:        sql.NullString{String: cs.Pincode, Valid: cs.Pincode != ""},
		Gstin:          sql.NullString{String: cs.GSTIN, Valid: cs.GSTIN != ""},
		SignatureImage: sql.NullString{String: cs.SignatureImage, Valid: cs.SignatureImage != ""},
		Email:          sql.NullString{String: cs.Email, Valid: cs.Email != ""},
		Cin:            sql.NullString{String: cs.CIN, Valid: cs.CIN != ""},
	})
}

// UpdateCompanySignature updates only the signature image path.
func UpdateCompanySignature(signatureImage string) error {
	q := db.New(DB)
	return q.UpdateCompanySignature(context.Background(),
		sql.NullString{String: signatureImage, Valid: signatureImage != ""},
	)
}

// InitCompanySettings inserts the default company settings row if it does not exist.
func InitCompanySettings(cs *models.CompanySettings) error {
	q := db.New(DB)
	return q.InitCompanySettings(context.Background(), db.InitCompanySettingsParams{
		Name:      cs.Name,
		Address:   sql.NullString{String: cs.Address, Valid: cs.Address != ""},
		City:      sql.NullString{String: cs.City, Valid: cs.City != ""},
		State:     sql.NullString{String: cs.State, Valid: cs.State != ""},
		StateCode: sql.NullString{String: cs.StateCode, Valid: cs.StateCode != ""},
		Pincode:   sql.NullString{String: cs.Pincode, Valid: cs.Pincode != ""},
		Gstin:     sql.NullString{String: cs.GSTIN, Valid: cs.GSTIN != ""},
	})
}

// mapCompanySettings converts a sqlc GetCompanySettingsRow to a models.CompanySettings.
func mapCompanySettings(row db.GetCompanySettingsRow) *models.CompanySettings {
	return &models.CompanySettings{
		ID:             int(row.ID),
		Name:           row.Name,
		Address:        row.Address.String,
		City:           row.City.String,
		State:          row.State.String,
		StateCode:      row.StateCode.String,
		Pincode:        row.Pincode.String,
		GSTIN:          row.Gstin.String,
		SignatureImage: row.SignatureImage.String,
		Email:          row.Email,
		CIN:            row.Cin,
	}
}
