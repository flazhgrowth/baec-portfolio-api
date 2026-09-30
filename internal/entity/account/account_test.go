package account

import (
	"strings"
	"testing"
)

func TestRegisterRequestValidate(t *testing.T) {
	const okPassword = "hunter2222"
	for name, tc := range map[string]struct {
		username, password string
		wantInvalid        string // "" = valid, else which field is reported
	}{
		"typical":                 {"alex", okPassword, ""},
		"digits and underscore":   {"a_1_b", okPassword, ""},
		"exactly 3":               {"abc", okPassword, ""},
		"exactly 20":              {strings.Repeat("a", 20), okPassword, ""},
		"all digits":              {"12345", okPassword, ""},
		"leading underscore":      {"_bob", okPassword, ""},
		"2 characters":            {"ab", okPassword, "username"},
		"21 characters":           {strings.Repeat("a", 21), okPassword, "username"},
		"empty":                   {"", okPassword, "username"},
		"space inside":            {"a b", okPassword, "username"},
		"trailing space":          {"alex ", okPassword, "username"},
		"punctuation":             {"no spaces!", okPassword, "username"},
		"hyphen":                  {"a-b", okPassword, "username"},
		"dot":                     {"a.b", okPassword, "username"},
		"at sign (email-like)":    {"a@b.com", okPassword, "username"},
		"non-ascii letters":       {"émile", okPassword, "username"},
		"emoji":                   {"alex😀", okPassword, "username"},
		"newline":                 {"alex\n", okPassword, "username"},
		"password exactly 6":      {"alex", "abcdef", ""},
		"password exactly 72":     {"alex", strings.Repeat("a", 72), ""},
		"72 multibyte chars":      {"alex", strings.Repeat("€", 72), ""},
		"password 5":              {"alex", "abcde", "password"},
		"password empty":          {"alex", "", "password"},
		"password 73":             {"alex", strings.Repeat("a", 73), "password"},
		"password 73 multibyte":   {"alex", strings.Repeat("€", 73), "password"},
		"spaces count as length":  {"alex", "      ", ""},
		"both bad: username wins": {"a", "x", "username"},
	} {
		req := RegisterRequest{LoginRequest: LoginRequest{Username: tc.username, Password: tc.password}}
		reason := req.Validate()
		switch {
		case tc.wantInvalid == "" && reason != "":
			t.Errorf("%s: want valid, got %q", name, reason)
		case tc.wantInvalid != "" && !strings.HasPrefix(reason, tc.wantInvalid):
			t.Errorf("%s: want a %s error, got %q", name, tc.wantInvalid, reason)
		}
	}
}
