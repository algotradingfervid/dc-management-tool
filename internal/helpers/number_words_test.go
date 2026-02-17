package helpers

import "testing"

func TestNumberToIndianWords(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"zero", 0, "Zero Rupees Only"},
		{"one rupee", 1, "One Rupees Only"},
		{"ten", 10, "Ten Rupees Only"},
		{"eleven", 11, "Eleven Rupees Only"},
		{"twenty", 20, "Twenty Rupees Only"},
		{"hundred", 100, "One Hundred Rupees Only"},
		{"thousands", 1234, "One Thousand Two Hundred Thirty Four Rupees Only"},
		{"lakhs", 154000, "One Lakh Fifty Four Thousand Rupees Only"},
		{"crores", 10000000, "One Crore Rupees Only"},
		{"mixed crore lakh thousand", 12345678, "One Crore Twenty Three Lakh Forty Five Thousand Six Hundred Seventy Eight Rupees Only"},
		{"with paise", 54162.50, "Fifty Four Thousand One Hundred Sixty Two Rupees and Fifty Paise Only"},
		{"paise only rounding", 0.99, "Ninety Nine Paise Only"},
		{"large amount", 9999999, "Ninety Nine Lakh Ninety Nine Thousand Nine Hundred Ninety Nine Rupees Only"},
		{"teens", 15, "Fifteen Rupees Only"},
		{"two crore", 20000000, "Two Crore Rupees Only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NumberToIndianWords(tt.input)
			if result != tt.expected {
				t.Errorf("NumberToIndianWords(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
