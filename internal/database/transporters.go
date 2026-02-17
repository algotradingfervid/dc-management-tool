package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetTransportersByProjectID returns all transporters for a project, optionally filtered to active only.
func GetTransportersByProjectID(projectID int, activeOnly bool) ([]*models.Transporter, error) {
	q := db.New(DB)
	ctx := context.Background()

	var rows []db.Transporter
	var err error
	if activeOnly {
		rows, err = q.GetActiveTransportersByProjectID(ctx, int64(projectID))
	} else {
		rows, err = q.GetTransportersByProjectID(ctx, int64(projectID))
	}
	if err != nil {
		return nil, err
	}

	transporters := make([]*models.Transporter, 0, len(rows))
	for _, row := range rows {
		t := mapTransporter(row)
		vehicles, err := GetVehiclesByTransporterID(t.ID)
		if err != nil {
			return nil, err
		}
		t.Vehicles = vehicles
		transporters = append(transporters, t)
	}
	return transporters, nil
}

// GetTransporterByID fetches a single transporter plus its vehicles.
func GetTransporterByID(id int) (*models.Transporter, error) {
	q := db.New(DB)
	row, err := q.GetTransporterByID(context.Background(), int64(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transporter not found")
	}
	if err != nil {
		return nil, err
	}

	t := mapTransporter(row)
	vehicles, err := GetVehiclesByTransporterID(t.ID)
	if err != nil {
		return nil, err
	}
	t.Vehicles = vehicles
	return t, nil
}

// CreateTransporter inserts a new transporter and sets t.ID.
func CreateTransporter(t *models.Transporter) error {
	q := db.New(DB)
	result, err := q.CreateTransporter(context.Background(), db.CreateTransporterParams{
		ProjectID:     int64(t.ProjectID),
		CompanyName:   t.CompanyName,
		ContactPerson: sql.NullString{String: t.ContactPerson, Valid: t.ContactPerson != ""},
		Phone:         sql.NullString{String: t.Phone, Valid: t.Phone != ""},
		GstNumber:     sql.NullString{String: t.GSTNumber, Valid: t.GSTNumber != ""},
		IsActive:      sql.NullBool{Bool: t.IsActive, Valid: true},
	})
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

// UpdateTransporter updates an existing transporter record.
func UpdateTransporter(t *models.Transporter) error {
	q := db.New(DB)
	return q.UpdateTransporter(context.Background(), db.UpdateTransporterParams{
		CompanyName:   t.CompanyName,
		ContactPerson: sql.NullString{String: t.ContactPerson, Valid: t.ContactPerson != ""},
		Phone:         sql.NullString{String: t.Phone, Valid: t.Phone != ""},
		GstNumber:     sql.NullString{String: t.GSTNumber, Valid: t.GSTNumber != ""},
		IsActive:      sql.NullBool{Bool: t.IsActive, Valid: true},
		ID:            int64(t.ID),
		ProjectID:     int64(t.ProjectID),
	})
}

// DeactivateTransporter sets is_active = 0 for the given transporter.
func DeactivateTransporter(id, projectID int) error {
	q := db.New(DB)
	return q.DeactivateTransporter(context.Background(), db.DeactivateTransporterParams{
		ID:        int64(id),
		ProjectID: int64(projectID),
	})
}

// ActivateTransporter sets is_active = 1 for the given transporter.
func ActivateTransporter(id, projectID int) error {
	q := db.New(DB)
	return q.ActivateTransporter(context.Background(), db.ActivateTransporterParams{
		ID:        int64(id),
		ProjectID: int64(projectID),
	})
}

// SearchTransporters searches/paginates transporters for a project.
func SearchTransporters(projectID int, search string, page int, perPage int) (*models.TransporterPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := db.New(DB)
	ctx := context.Background()

	var total int64
	var rows []db.Transporter
	var err error

	if search == "" {
		total, err = q.SearchTransportersCountNoFilter(ctx, int64(projectID))
		if err != nil {
			return nil, err
		}

		totalPages := (int(total) + perPage - 1) / perPage
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
		offset := (page - 1) * perPage

		rows, err = q.SearchTransportersNoFilter(ctx, db.SearchTransportersNoFilterParams{
			ProjectID: int64(projectID),
			Limit:     int64(perPage),
			Offset:    int64(offset),
		})
	} else {
		like := "%" + search + "%"
		countParams := db.SearchTransportersCountParams{
			ProjectID:     int64(projectID),
			CompanyName:   like,
			ContactPerson: sql.NullString{String: like, Valid: true},
			Phone:         sql.NullString{String: like, Valid: true},
			GstNumber:     sql.NullString{String: like, Valid: true},
		}
		total, err = q.SearchTransportersCount(ctx, countParams)
		if err != nil {
			return nil, err
		}

		totalPages := (int(total) + perPage - 1) / perPage
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
		offset := (page - 1) * perPage

		rows, err = q.SearchTransporters(ctx, db.SearchTransportersParams{
			ProjectID:     int64(projectID),
			CompanyName:   like,
			ContactPerson: sql.NullString{String: like, Valid: true},
			Phone:         sql.NullString{String: like, Valid: true},
			GstNumber:     sql.NullString{String: like, Valid: true},
			Limit:         int64(perPage),
			Offset:        int64(offset),
		})
	}
	if err != nil {
		return nil, err
	}

	totalInt := int(total)
	totalPages := (totalInt + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	transporters := make([]*models.Transporter, 0, len(rows))
	for _, row := range rows {
		t := mapTransporter(row)
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
		TotalCount:   totalInt,
		TotalPages:   totalPages,
		Search:       search,
	}, nil
}

// GetTransporterCount returns the number of active transporters for a project.
func GetTransporterCount(projectID int) (int, error) {
	q := db.New(DB)
	count, err := q.GetTransporterCount(context.Background(), int64(projectID))
	return int(count), err
}

// ---------------------------------------------------------------------------
// Vehicle operations
// ---------------------------------------------------------------------------

// GetVehiclesByTransporterID fetches all vehicles for a transporter.
func GetVehiclesByTransporterID(transporterID int) ([]*models.TransporterVehicle, error) {
	q := db.New(DB)
	rows, err := q.GetVehiclesByTransporterID(context.Background(), int64(transporterID))
	if err != nil {
		return nil, err
	}

	vehicles := make([]*models.TransporterVehicle, 0, len(rows))
	for _, row := range rows {
		vehicles = append(vehicles, mapVehicleRow(row))
	}
	return vehicles, nil
}

// GetVehicleByID fetches a single vehicle by ID.
func GetVehicleByID(id int) (*models.TransporterVehicle, error) {
	q := db.New(DB)
	row, err := q.GetVehicleByID(context.Background(), int64(id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vehicle not found")
	}
	if err != nil {
		return nil, err
	}
	return mapGetVehicleByIDRow(row), nil
}

// CreateVehicle inserts a new vehicle and sets v.ID.
func CreateVehicle(v *models.TransporterVehicle) error {
	q := db.New(DB)
	result, err := q.CreateVehicle(context.Background(), db.CreateVehicleParams{
		TransporterID: int64(v.TransporterID),
		VehicleNumber: v.VehicleNumber,
		VehicleType:   sql.NullString{String: v.VehicleType, Valid: v.VehicleType != ""},
		DriverName:    sql.NullString{String: v.DriverName, Valid: v.DriverName != ""},
		DriverPhone1:  sql.NullString{String: v.DriverPhone1, Valid: v.DriverPhone1 != ""},
		DriverPhone2:  sql.NullString{String: v.DriverPhone2, Valid: v.DriverPhone2 != ""},
	})
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
	q := db.New(DB)
	count, err := q.IsVehicleUsedInDC(context.Background(),
		sql.NullString{String: vehicleNumber, Valid: vehicleNumber != ""},
	)
	return count > 0, err
}

// DeleteVehicle removes a vehicle by id and transporter ownership check.
func DeleteVehicle(id, transporterID int) error {
	q := db.New(DB)
	return q.DeleteVehicle(context.Background(), db.DeleteVehicleParams{
		ID:            int64(id),
		TransporterID: int64(transporterID),
	})
}

// ---------------------------------------------------------------------------
// Type mappers
// ---------------------------------------------------------------------------

func mapTransporter(row db.Transporter) *models.Transporter {
	t := &models.Transporter{
		ID:            int(row.ID),
		ProjectID:     int(row.ProjectID),
		CompanyName:   row.CompanyName,
		ContactPerson: row.ContactPerson.String,
		Phone:         row.Phone.String,
		GSTNumber:     row.GstNumber.String,
		IsActive:      row.IsActive.Bool,
	}
	if row.CreatedAt.Valid {
		t.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		t.UpdatedAt = row.UpdatedAt.Time
	}
	return t
}

func mapVehicleRow(row db.GetVehiclesByTransporterIDRow) *models.TransporterVehicle {
	v := &models.TransporterVehicle{
		ID:            int(row.ID),
		TransporterID: int(row.TransporterID),
		VehicleNumber: row.VehicleNumber,
		VehicleType:   row.VehicleType.String,
		DriverName:    row.DriverName.String,
		DriverPhone1:  row.DriverPhone1.String,
		DriverPhone2:  row.DriverPhone2.String,
	}
	if row.CreatedAt.Valid {
		v.CreatedAt = row.CreatedAt.Time
	}
	return v
}

func mapGetVehicleByIDRow(row db.GetVehicleByIDRow) *models.TransporterVehicle {
	v := &models.TransporterVehicle{
		ID:            int(row.ID),
		TransporterID: int(row.TransporterID),
		VehicleNumber: row.VehicleNumber,
		VehicleType:   row.VehicleType.String,
		DriverName:    row.DriverName.String,
		DriverPhone1:  row.DriverPhone1.String,
		DriverPhone2:  row.DriverPhone2.String,
	}
	if row.CreatedAt.Valid {
		v.CreatedAt = row.CreatedAt.Time
	}
	return v
}

// ensure time import is used (time.Time fields on models)
var _ = time.Time{}
