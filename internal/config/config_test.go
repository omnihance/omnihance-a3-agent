package config

import "testing"

func TestMaxFileUploadSizeBytesUsesDefaultForUnsetOrInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  *EnvVars
	}{
		{
			name: "nil config",
		},
		{
			name: "zero value",
			cfg:  &EnvVars{},
		},
		{
			name: "negative value",
			cfg:  &EnvVars{MaxFileUploadSizeMb: -1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.cfg.MaxFileUploadSizeBytes(); got != int64(DefaultMaxFileUploadSizeMb)*1024*1024 {
				t.Fatalf("expected default max upload size, got %d", got)
			}
		})
	}
}

func TestMaxFileUploadSizeBytesUsesConfiguredValue(t *testing.T) {
	cfg := &EnvVars{MaxFileUploadSizeMb: 7}

	if got := cfg.MaxFileUploadSizeBytes(); got != 7*1024*1024 {
		t.Fatalf("expected configured max upload size, got %d", got)
	}
}

func TestParseMaxFileUploadSizeMb(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{
			name:  "valid value",
			value: "512",
			want:  512,
		},
		{
			name:    "invalid value",
			value:   "large",
			want:    DefaultMaxFileUploadSizeMb,
			wantErr: true,
		},
		{
			name:    "zero value",
			value:   "0",
			want:    DefaultMaxFileUploadSizeMb,
			wantErr: true,
		},
		{
			name:    "negative value",
			value:   "-5",
			want:    DefaultMaxFileUploadSizeMb,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseMaxFileUploadSizeMb(test.value)
			if got != test.want {
				t.Fatalf("expected %d, got %d", test.want, got)
			}

			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}

			if !test.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
