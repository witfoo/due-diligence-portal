package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePassword(t *testing.T) {
	const (
		twoByte  = "é"          // é: 1 character, 2 bytes in UTF-8
		fourByte = "\U0001F600" // 😀: 1 character, 4 bytes in UTF-8
	)

	tests := []struct {
		name      string
		password  string
		wantBytes int // expected len(password), guards the fixture itself
		wantErr   error
	}{
		{name: "empty", password: "", wantBytes: 0, wantErr: ErrPasswordRequired},
		{name: "7 characters", password: "abcdefg", wantBytes: 7, wantErr: ErrPasswordTooShort},
		{name: "8 characters", password: "abcdefgh", wantBytes: 8},
		{name: "72 bytes", password: strings.Repeat("a", 72), wantBytes: 72},
		{name: "73 bytes", password: strings.Repeat("a", 73), wantBytes: 73, wantErr: ErrPasswordTooLong},
		{name: "7 two-byte characters is too short despite 14 bytes", password: strings.Repeat(twoByte, 7), wantBytes: 14, wantErr: ErrPasswordTooShort},
		{name: "8 two-byte characters", password: strings.Repeat(twoByte, 8), wantBytes: 16},
		{name: "36 two-byte characters is exactly 72 bytes", password: strings.Repeat(twoByte, 36), wantBytes: 72},
		{name: "two-byte characters plus one byte crosses 72 bytes", password: strings.Repeat(twoByte, 36) + "a", wantBytes: 73, wantErr: ErrPasswordTooLong},
		{name: "18 four-byte characters is exactly 72 bytes", password: strings.Repeat(fourByte, 18), wantBytes: 72},
		{name: "four-byte character straddling the 72 byte limit", password: strings.Repeat("a", 69) + fourByte, wantBytes: 73, wantErr: ErrPasswordTooLong},
		{name: "19 four-byte characters is far under 72 characters but over 72 bytes", password: strings.Repeat(fourByte, 19), wantBytes: 76, wantErr: ErrPasswordTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Len(t, tt.password, tt.wantBytes)

			err := ValidatePassword(tt.password)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			if tt.password != "" {
				assert.NotContains(t, err.Error(), tt.password, "error must not echo the password")
			}
		})
	}
}
