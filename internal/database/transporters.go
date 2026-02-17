package database

import (
	"database/sql"
	"fmt"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

func GetTransportersByProjectID(projectID int, activeOnly bool) ([]*models.Transporter, error) {
	query := `
		SELECT id, project_id, company_name, contact_person, phone, gst_number,
		       is_active, created_at, updated_at
		FROM transporters
		WHERE project_id = ?
	`
	args := []interface{}{projectID}
	if activeOnly {
		query += " AND is_active = 1"
	}
	query += " ORDER BY company_name ASC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transporters []*models.Transporter
	for rows.Next() {
		t := &models.Transporter{}
		err := rows.Scan(
			&t.ID, &t.ProjectID, &t.CompanyName, &t.ContactPerson,
			&t.Phone, &t.GSTNumber, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transporters = append(transporters, t)
	}

	// Load vehicles for each transporter
	for _, t := range transporters {
		vehicles, err := GetVehiclesByTransporterID(t.ID)
		if err != nil {
			return nil, err
		}
		t.Vehicles = vehicles
	}

	return transporters, nil
}

func GetTransporterByID(id int) (*models.Transporter, error) {
	query := `
		SELECT id, project_id, company_name, contact_person, phone, gst_number,
		       is_active, created_at, updated_at
		FROM transporters
		WHERE id = ?
	`

	t := &models.Transporter{}
	err := DB.QueryRow(query, id).Scan(
		&t.ID, &t.ProjectID, &t.CompanyName, &t.ContactPerson,
		&t.Phone, &t.GSTNumber, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transporter not found")
	}
	if err != nil {
		return nil, err
	}

	// Load vehicles
	vehicles, err := GetVehiclesByTransporterID(t.ID)
	if err != nil {
		return nil, err
	}
	t.Vehicles = vehicles

	return t, nil
}

func CreateTransporter(t *models.Transporter) error {
	query := `
		INSERT INTO transporters (project_id, company_name, contact_person, phone, gst_number, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := DB.Exec(query,
		t.ProjectID, t.CompanyName, t.ContactPerson, t.Phone, t.GSTNumber, t.IsActive,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = int(id)
	return nil
}

func UpdateTransporter(t *models.Transporter) error {
	query := `
		UPDATE transporters SET
			company_name = ?, contact_person = ?, phone = ?, gst_number = ?,
			is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND project_id = ?
	`

	_, err := DB.Exec(query,
		t.CompanyName, t.ContactPerson, t.Phone, t.GSTNumber, t.IsActive,
		t.ID, t.ProjectID,
	)
	return err
}

func DeactivateTransporter(id, projectID int) error {
	_, err := DB.Exec(
		"UPDATE transporters SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND project_id = ?",
		id, projectID,
	)
	return err
}

func ActivateTransporter(id, projectID int) error {
	_, err := DB.Exec(
		"UPDATE transporters SET is_active = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND project_id = ?",
		id, projectID,
	)
	return err
}

func SearchTransporters(projectID int, search string, page int, perPage int) (*models.TransporterPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	where := "WHERE project_id = ?"
	args := []interface{}{projectID}
	if search != "" {
		where += " AND (company_name LIKE ? OR contact_person LIKE ? OR phone LIKE ? OR gst_number LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM transporters " + where
	if err := DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT id, project_id, company_name, contact_person, phone, gst_number,
		       is_active, created_at, updated_at
		FROM transporters %s
		ORDER BY company_name ASC
		LIMIT ? OFFSET ?
	`, where)

	queryArgs := append(args, perPage, offset)
	rows, err := DB.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transporters []*models.Transporter
	for rows.Next() {
		t := &models.Transporter{}
		err := rows.Scan(
			&t.ID, &t.ProjectID, &t.CompanyName, &t.ContactPerson,
			&t.Phone, &t.GSTNumber, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Load vehicles for each transporter
		vehicles, err := GetVehiclesByTransporterID(t.ID)
		if err != nil {
			return nil, err
		}
		t.Vehicles = vehicles
		transporters = append(transporters, t)
	}

	return &models.TransporterPage{
		Transporters: transporters,
		CurrentPage:  page,
		PerPage:      perPage,
		TotalCount:   total,
		TotalPages:   totalPages,
		Search:       search,
	}, nil
}

func GetTransporterCount(projectID int) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM transporters WHERE project_id = ? AND is_active = 1", projectID).Scan(&count)
	return count, err
}

// Vehicle operations

func GetVehiclesByTransporterID(transporterID int) ([]*models.TransporterVehicle, error) {
	query := `
		SELECT id, transporter_id, vehicle_number, vehicle_type,
		       driver_name, driver_phone1, driver_phone2, created_at
		FROM transporter_vehicles
		WHERE transporter_id = ?
		ORDER BY vehicle_number ASC
	`

	rows, err := DB.Query(query, transporterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []*models.TransporterVehicle
	for rows.Next() {
		v := &models.TransporterVehicle{}
		err := rows.Scan(&v.ID, &v.TransporterID, &v.VehicleNumber, &v.VehicleType,
			&v.DriverName, &v.DriverPhone1, &v.DriverPhone2, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}

	return vehicles, nil
}

func GetVehicleByID(id int) (*models.TransporterVehicle, error) {
	query := `
		SELECT id, transporter_id, vehicle_number, vehicle_type,
		       driver_name, driver_phone1, driver_phone2, created_at
		FROM transporter_vehicles
		WHERE id = ?
	`

	v := &models.TransporterVehicle{}
	err := DB.QueryRow(query, id).Scan(&v.ID, &v.TransporterID, &v.VehicleNumber, &v.VehicleType,
		&v.DriverName, &v.DriverPhone1, &v.DriverPhone2, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vehicle not found")
	}
	if err != nil {
		return nil, err
	}

	return v, nil
}

func CreateVehicle(v *models.TransporterVehicle) error {
	query := `INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2) VALUES (?, ?, ?, ?, ?, ?)`

	result, err := DB.Exec(query, v.TransporterID, v.VehicleNumber, v.VehicleType, v.DriverName, v.DriverPhone1, v.DriverPhone2)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	v.ID = int(id)
	return nil
}

// IsVehicleUsedInDC checks if a vehicle number is referenced in any DC transit details.
func IsVehicleUsedInDC(vehicleNumber string) (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM dc_transit_details WHERE vehicle_number = ?", vehicleNumber).Scan(&count)
	return count > 0, err
}

func DeleteVehicle(id, transporterID int) error {
	_, err := DB.Exec("DELETE FROM transporter_vehicles WHERE id = ? AND transporter_id = ?", id, transporterID)
	return err
}
