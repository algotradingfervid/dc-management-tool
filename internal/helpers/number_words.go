package helpers

import (
	"math"
	"strings"
)

// NumberToIndianWords converts a number to words in Indian numbering system.
// Example: 54162.50 -> "Fifty Four Thousand One Hundred Sixty Two Rupees and Fifty Paise Only"
func NumberToIndianWords(num float64) string {
	if num == 0 {
		return "Zero Rupees Only"
	}

	rupees := int64(num)
	paise := int64(math.Round((num - float64(rupees)) * 100))

	var words string

	if rupees > 0 {
		words = convertToIndianWords(rupees) + " Rupees"
	}

	if paise > 0 {
		if words != "" {
			words += " and "
		}
		words += convertToIndianWords(paise) + " Paise"
	}

	return words + " Only"
}

func convertToIndianWords(num int64) string {
	if num == 0 {
		return ""
	}

	ones := []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine"}
	tens := []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}
	teens := []string{"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}

	convertTwoDigits := func(n int64) string {
		if n < 10 {
			return ones[n]
		}
		if n < 20 {
			return teens[n-10]
		}
		return strings.TrimSpace(tens[n/10] + " " + ones[n%10])
	}

	convertThreeDigits := func(n int64) string {
		if n < 100 {
			return convertTwoDigits(n)
		}
		hundred := ones[n/100] + " Hundred"
		remainder := n % 100
		if remainder > 0 {
			return hundred + " " + convertTwoDigits(remainder)
		}
		return hundred
	}

	// Indian numbering: Crores, Lakhs, Thousands, Hundreds
	crore := num / 10000000
	num = num % 10000000

	lakh := num / 100000
	num = num % 100000

	thousand := num / 1000
	num = num % 1000

	var parts []string

	if crore > 0 {
		parts = append(parts, convertTwoDigits(crore)+" Crore")
	}
	if lakh > 0 {
		parts = append(parts, convertTwoDigits(lakh)+" Lakh")
	}
	if thousand > 0 {
		parts = append(parts, convertTwoDigits(thousand)+" Thousand")
	}
	if num > 0 {
		parts = append(parts, convertThreeDigits(num))
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}
