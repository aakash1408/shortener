package shortcode

import(
	"fmt"
	"strings"
	"crypto/rand"
)

const (
    alphabet  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	genLength = 6
	minLength = 4
	maxLength = 20
)

func Generate() string{
	bytes := make([]byte, genLength)
	rand.Read(bytes)
	for i, b := range bytes{
		bytes[i] = alphabet[b%byte(len(alphabet))]
	}
	return string(bytes)
}

func Validate(code string) error {
	if len(code) < minLength || len(code) > maxLength {
		return fmt.Errorf("short code %q must be between %d and %d characters", code, minLength, maxLength)
	}
	for _, ch := range code {
		if !strings.ContainsRune(alphabet, ch) {
			return fmt.Errorf("short code %q contains invalid character %q", code, ch)
		}
	}
	return nil
}
