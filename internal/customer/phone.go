package customer

import (
	"strings"
	"unicode"
)

func NormalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	var digits strings.Builder
	for _, item := range value {
		if unicode.IsDigit(item) {
			digits.WriteRune(item)
		}
	}
	phone := digits.String()
	if phone == "" {
		return "", ErrInvalidPhone
	}
	if strings.HasPrefix(phone, "0") {
		phone = "62" + strings.TrimLeft(phone, "0")
	}
	if strings.HasPrefix(phone, "8") {
		phone = "62" + phone
	}
	if !strings.HasPrefix(phone, "62") || len(phone) < 10 || len(phone) > 16 {
		return "", ErrInvalidPhone
	}
	return phone, nil
}
