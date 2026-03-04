package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func TestParseQuantityForm(t *testing.T) {
	products := []*models.TemplateProductRow{
		{Product: models.Product{ID: 10, ItemName: "Widget A"}, DefaultQuantity: 5},
		{Product: models.Product{ID: 20, ItemName: "Widget B"}, DefaultQuantity: 3},
	}
	shipToAddressIDs := []int{100, 200}

	t.Run("parses qty fields correctly", func(t *testing.T) {
		form := url.Values{}
		form.Set("qty_10_100", "7")
		form.Set("qty_10_200", "3")
		form.Set("qty_20_100", "2")
		form.Set("qty_20_200", "4")

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		quantities := parseQuantityForm(c, products, shipToAddressIDs)

		if quantities[10][100] != 7 {
			t.Errorf("expected qty_10_100=7, got %d", quantities[10][100])
		}
		if quantities[10][200] != 3 {
			t.Errorf("expected qty_10_200=3, got %d", quantities[10][200])
		}
		if quantities[20][100] != 2 {
			t.Errorf("expected qty_20_100=2, got %d", quantities[20][100])
		}
		if quantities[20][200] != 4 {
			t.Errorf("expected qty_20_200=4, got %d", quantities[20][200])
		}
	})

	t.Run("missing fields default to zero", func(t *testing.T) {
		form := url.Values{}
		form.Set("qty_10_100", "5")
		// qty_10_200, qty_20_100, qty_20_200 are missing

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		quantities := parseQuantityForm(c, products, shipToAddressIDs)

		if quantities[10][100] != 5 {
			t.Errorf("expected qty_10_100=5, got %d", quantities[10][100])
		}
		if quantities[10][200] != 0 {
			t.Errorf("expected qty_10_200=0, got %d", quantities[10][200])
		}
		if quantities[20][100] != 0 {
			t.Errorf("expected qty_20_100=0, got %d", quantities[20][100])
		}
		if quantities[20][200] != 0 {
			t.Errorf("expected qty_20_200=0, got %d", quantities[20][200])
		}
	})

	t.Run("invalid values default to zero", func(t *testing.T) {
		form := url.Values{}
		form.Set("qty_10_100", "abc")
		form.Set("qty_20_100", "")

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		quantities := parseQuantityForm(c, products, shipToAddressIDs)

		if quantities[10][100] != 0 {
			t.Errorf("expected qty_10_100=0 for invalid input, got %d", quantities[10][100])
		}
		if quantities[20][100] != 0 {
			t.Errorf("expected qty_20_100=0 for empty input, got %d", quantities[20][100])
		}
	})

	t.Run("empty products and addresses", func(t *testing.T) {
		form := url.Values{}
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		quantities := parseQuantityForm(c, []*models.TemplateProductRow{}, []int{})

		if len(quantities) != 0 {
			t.Errorf("expected empty map, got %d entries", len(quantities))
		}
	})
}

func TestValidateQuantities(t *testing.T) {
	products := []*models.TemplateProductRow{
		{Product: models.Product{ID: 10, ItemName: "Widget A"}, DefaultQuantity: 5},
		{Product: models.Product{ID: 20, ItemName: "Widget B"}, DefaultQuantity: 3},
	}
	shipToAddressIDs := []int{100, 200}

	t.Run("valid quantities pass", func(t *testing.T) {
		quantities := map[int]map[int]int{
			10: {100: 5, 200: 3},
			20: {100: 2, 200: 4},
		}
		errs := validateQuantities(quantities, products, shipToAddressIDs)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("all zero product rejected", func(t *testing.T) {
		quantities := map[int]map[int]int{
			10: {100: 5, 200: 3}, // product 10 OK
			20: {100: 0, 200: 0}, // product 20 all zero
		}
		errs := validateQuantities(quantities, products, shipToAddressIDs)
		if errs["product_20"] == "" {
			t.Error("expected error for product 20 with all-zero quantities")
		}
		if errs["product_10"] != "" {
			t.Error("product 10 should not have an error")
		}
	})

	t.Run("negative values rejected", func(t *testing.T) {
		quantities := map[int]map[int]int{
			10: {100: 5, 200: -3},
			20: {100: 2, 200: 4},
		}
		errs := validateQuantities(quantities, products, shipToAddressIDs)
		if errs["qty_10_200"] == "" {
			t.Error("expected error for negative quantity qty_10_200")
		}
	})

	t.Run("all locations zero rejected", func(t *testing.T) {
		quantities := map[int]map[int]int{
			10: {100: 0, 200: 0},
			20: {100: 0, 200: 0},
		}
		errs := validateQuantities(quantities, products, shipToAddressIDs)
		if errs["global"] == "" {
			t.Error("expected global error when all quantities are zero")
		}
		// Should also have per-product errors
		if errs["product_10"] == "" {
			t.Error("expected error for product 10 with all-zero quantities")
		}
		if errs["product_20"] == "" {
			t.Error("expected error for product 20 with all-zero quantities")
		}
	})

	t.Run("mixed zero and nonzero is fine", func(t *testing.T) {
		// Product 10 has qty at addr 100, product 20 has qty at addr 200
		// Addr 100 has product 10 only, addr 200 has product 20 only
		quantities := map[int]map[int]int{
			10: {100: 3, 200: 0},
			20: {100: 0, 200: 2},
		}
		errs := validateQuantities(quantities, products, shipToAddressIDs)
		if len(errs) != 0 {
			t.Errorf("expected no errors for mixed quantities, got %v", errs)
		}
	})

	t.Run("single product single location", func(t *testing.T) {
		singleProduct := []*models.TemplateProductRow{
			{Product: models.Product{ID: 10, ItemName: "Widget A"}, DefaultQuantity: 5},
		}
		singleAddr := []int{100}
		quantities := map[int]map[int]int{
			10: {100: 1},
		}
		errs := validateQuantities(quantities, singleProduct, singleAddr)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})
}
