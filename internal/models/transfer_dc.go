package models

import "time"

// TransferDC stores metadata specific to Transfer DCs.
type TransferDC struct {
	ID              int       `json:"id"`
	DCID            int       `json:"dc_id"`
	HubAddressID    int       `json:"hub_address_id"`
	TemplateID      *int      `json:"template_id"`
	TaxType         string    `json:"tax_type"`
	ReverseCharge   string    `json:"reverse_charge"`
	TransporterName string    `json:"transporter_name"`
	VehicleNumber   string    `json:"vehicle_number"`
	EwayBillNumber  string    `json:"eway_bill_number"`
	DocketNumber    string    `json:"docket_number"`
	Notes           string    `json:"notes"`
	NumDestinations int       `json:"num_destinations"`
	NumSplit        int       `json:"num_split"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Computed/joined fields
	HubAddressName string  `json:"hub_address_name"`
	TemplateName   string  `json:"template_name"`
	DCNumber       string  `json:"dc_number"`
	DCStatus       string  `json:"dc_status"`
	ChallanDate    *string `json:"challan_date"`
	ProjectID      int     `json:"project_id"`
}

// TransferDCDestination maps a delivery destination to a Transfer DC.
type TransferDCDestination struct {
	ID              int       `json:"id"`
	TransferDCID    int       `json:"transfer_dc_id"`
	ShipToAddressID int       `json:"ship_to_address_id"`
	SplitGroupID    *int      `json:"split_group_id"`
	IsSplit         bool      `json:"is_split"`
	CreatedAt       time.Time `json:"created_at"`

	// Computed/joined fields
	AddressName string                     `json:"address_name"`
	Address     *Address                   `json:"-"` // full address object (populated in handler)
	Quantities  []TransferDCDestinationQty `json:"quantities"`
}

// TransferDCDestinationQty stores per-product, per-destination planned quantities.
type TransferDCDestinationQty struct {
	ID            int `json:"id"`
	DestinationID int `json:"destination_id"`
	ProductID     int `json:"product_id"`
	Quantity      int `json:"quantity"`

	// Computed/joined
	ProductName string `json:"product_name"`
}

// TransferDCSplit tracks a split operation linking a Transfer DC to a child shipment group.
type TransferDCSplit struct {
	ID              int       `json:"id"`
	TransferDCID    int       `json:"transfer_dc_id"`
	ShipmentGroupID int       `json:"shipment_group_id"`
	SplitNumber     int       `json:"split_number"`
	CreatedBy       int       `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`

	// Computed/joined
	ShipmentGroup *ShipmentGroup           `json:"shipment_group,omitempty"`
	Destinations  []*TransferDCDestination `json:"destinations,omitempty"`
	CanDelete     bool                     `json:"can_delete"`
}

// TransferDCSummary holds aggregate stats for a Transfer DC.
type TransferDCSummary struct {
	TotalDestinations   int `json:"total_destinations"`
	SplitDestinations   int `json:"split_destinations"`
	PendingDestinations int `json:"pending_destinations"`
	TotalProducts       int `json:"total_products"`
	TotalQuantity       int `json:"total_quantity"`
	SplitCount          int `json:"split_count"`
}
