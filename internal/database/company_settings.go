package database

import (
	"database/sql"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetCompanySettings fetches the single company settings row.
func GetCompanySettings() (*models.CompanySettings, error) {
	cs := &models.CompanySettings{}
	var signatureImage sql.NullString

	err := DB.QueryRow(
		`SELECT id, name, address, city, state, state_code, pincode, gstin, signature_image
		 FROM company_settings WHERE id = 1`,
	).Scan(
		&cs.ID, &cs.Name, &cs.Address, &cs.City, &cs.State,
		&cs.StateCode, &cs.Pincode, &cs.GSTIN, &signatureImage,
	)
	if err != nil {
		return nil, err
	}

	if signatureImage.Valid {
		cs.SignatureImage = signatureImage.String
	}

	return cs, nil
}
