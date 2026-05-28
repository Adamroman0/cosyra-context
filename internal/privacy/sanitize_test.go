package privacy

import (
	"strings"
	"testing"
)

func TestSanitizeRedactsCommonSecrets(t *testing.T) {
	input := "Authorization: Bearer abcdef1234567890 token=secretvalue123 sk-abcdefghijklmnopqrstuvwxyz ghp_abcdefghijklmnopqrstuvwxyz prtbl_abcdefghijklmnopqrstuvwxyz"
	got := Sanitize(input)
	for _, forbidden := range []string{"abcdef1234567890", "secretvalue123", "sk-abcdefghijklmnopqrstuvwxyz", "ghp_abcdefghijklmnopqrstuvwxyz", "prtbl_abcdefghijklmnopqrstuvwxyz"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("expected %q to be redacted from %q", forbidden, got)
		}
	}
}
