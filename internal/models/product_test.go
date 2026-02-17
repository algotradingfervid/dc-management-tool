package models

import (
	"testing"
)

func TestProductValidate_ValidProduct(t *testing.T) {
	p := &Product{
		ItemName:        "Test Product",
		ItemDescription: "A test product",
		HSNCode:         "94054090",
		UoM:             "Nos",
		BrandModel:      "TestBrand",
		PerUnitPrice:    100.0,
		GSTPercentage:   18,
	}
	errors := p.Validate()
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %v", errors)
	}
}

func TestProductValidate_RequiredFields(t *testing.T) {
	p := &Product{}
	errors := p.Validate()

	required := []string{"item_name", "item_description", "uom", "brand_model", "per_unit_price"}
	for _, field := range required {
		if _, ok := errors[field]; !ok {
			t.Errorf("Expected error for field %s", field)
		}
	}
}

func TestProductValidate_HSNCode(t *testing.T) {
	tests := []struct {
		name    string
		hsn     string
		wantErr bool
	}{
		{"empty (optional)", "", false},
		{"6 digits valid", "123456", false},
		{"7 digits valid", "1234567", false},
		{"8 digits valid", "12345678", false},
		{"5 digits invalid", "12345", true},
		{"9 digits invalid", "123456789", true},
		{"letters invalid", "abcdef", true},
		{"mixed invalid", "1234ab", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProduct()
			p.HSNCode = tt.hsn
			errors := p.Validate()
			_, hasErr := errors["hsn_code"]
			if hasErr != tt.wantErr {
				t.Errorf("HSN %q: got error=%v, want error=%v", tt.hsn, hasErr, tt.wantErr)
			}
		})
	}
}

func TestProductValidate_Price(t *testing.T) {
	tests := []struct {
		name    string
		price   float64
		wantErr bool
	}{
		{"positive", 100.0, false},
		{"small positive", 0.01, false},
		{"zero", 0, true},
		{"negative", -10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProduct()
			p.PerUnitPrice = tt.price
			errors := p.Validate()
			_, hasErr := errors["per_unit_price"]
			if hasErr != tt.wantErr {
				t.Errorf("Price %.2f: got error=%v, want error=%v", tt.price, hasErr, tt.wantErr)
			}
		})
	}
}

func TestProductValidate_GST(t *testing.T) {
	tests := []struct {
		name    string
		gst     float64
		wantErr bool
	}{
		{"0%", 0, false},
		{"18%", 18, false},
		{"100%", 100, false},
		{"negative", -1, true},
		{"over 100", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProduct()
			p.GSTPercentage = tt.gst
			errors := p.Validate()
			_, hasErr := errors["gst_percentage"]
			if hasErr != tt.wantErr {
				t.Errorf("GST %.0f: got error=%v, want error=%v", tt.gst, hasErr, tt.wantErr)
			}
		})
	}
}

func TestProductPriceWithGST(t *testing.T) {
	tests := []struct {
		name     string
		price    float64
		gst      float64
		expected float64
	}{
		{"no GST", 100, 0, 100},
		{"5% GST", 100, 5, 105},
		{"12% GST", 100, 12, 112},
		{"18% GST", 100, 18, 118},
		{"28% GST", 100, 28, 128},
		{"18% on 250", 250, 18, 295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Product{PerUnitPrice: tt.price, GSTPercentage: tt.gst}
			got := p.PriceWithGST()
			diff := got - tt.expected
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("PriceWithGST() = %.4f, want %.2f", got, tt.expected)
			}
		})
	}
}

func validProduct() *Product {
	return &Product{
		ItemName:        "Test Product",
		ItemDescription: "Description",
		HSNCode:         "12345678",
		UoM:             "Nos",
		BrandModel:      "Brand",
		PerUnitPrice:    100,
		GSTPercentage:   18,
	}
}
