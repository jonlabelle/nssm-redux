package support

import "testing"

func TestQuoteArg(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":           `""`,
		"plain":      "plain",
		"two words":  `"two words"`,
		`a"b`:        `"a\"b"`,
		`trailing\`:  `trailing\`,
		`path with\`: `"path with\\"`,
	}

	for input, want := range cases {
		if got := QuoteArg(input); got != want {
			t.Fatalf("QuoteArg(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestJoinCommandLine(t *testing.T) {
	t.Parallel()

	got := JoinCommandLine([]string{"nssmr", "set", "svc", "DisplayName", "My Service"})
	want := `nssmr set svc DisplayName "My Service"`
	if got != want {
		t.Fatalf("JoinCommandLine() = %q, want %q", got, want)
	}
}
