package webhooks

import (
	"testing"
)

func TestInputValidation(t *testing.T) {
	cases := []struct {
		name    string
		input   Input
		wantErr bool
	}{
		{
			name:    "empty url",
			input:   Input{URL: ""},
			wantErr: true,
		},
		{
			name:    "invalid url scheme",
			input:   Input{URL: "ftp://example.com/webhook"},
			wantErr: true,
		},
		{
			name:    "relative url",
			input:   Input{URL: "/webhook/sonde"},
			wantErr: true,
		},
		{
			name:    "valid https url",
			input:   Input{URL: "https://antenne.facile.studio/webhook/sonde"},
			wantErr: false,
		},
		{
			name:    "valid http url",
			input:   Input{URL: "http://localhost:9090/webhook/sonde"},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.normalize()
			err := tc.input.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestInputNormalizationDefaultsHeader(t *testing.T) {
	in := Input{
		Name: " Test ",
		URL:  "https://example.com ",
	}
	in.normalize()
	if in.Name != "Test" {
		t.Errorf("expected trimmed name, got %q", in.Name)
	}
	if in.URL != "https://example.com" {
		t.Errorf("expected trimmed url, got %q", in.URL)
	}
	if in.SecretHeader != DefaultSecretHeader {
		t.Errorf("expected default header %q, got %q", DefaultSecretHeader, in.SecretHeader)
	}
}
