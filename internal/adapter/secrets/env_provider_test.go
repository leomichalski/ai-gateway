package secrets

import (
	"context"
	"testing"
)

func TestEnvProvider_GetProviderKey(t *testing.T) {
	const expectedKey = "test-google-api-key"

	tests := []struct {
		name         string
		providerName string
		envKey       string
		setEnv       bool
		want         string
		wantErr      bool
	}{
		{
			name:         "returns Gemini API key",
			providerName: "gemini",
			envKey:       expectedKey,
			setEnv:       true,
			want:         expectedKey,
			wantErr:      false,
		},
		{
			name:         "returns empty key when environment variable is unset",
			providerName: "gemini",
			setEnv:       false,
			want:         "",
			wantErr:      true,
		},
		{
			name:         "returns error for unknown provider",
			providerName: "unknown",
			setEnv:       false,
			want:         "",
			wantErr:      true,
		},
		{
			name:         "returns error for empty provider name",
			providerName: "",
			setEnv:       false,
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("GOOGLE_API_KEY", tt.envKey)
			} else {
				t.Setenv("GOOGLE_API_KEY", "")
			}

			provider := &EnvProvider{}

			got, err := provider.GetProviderKey(
				context.Background(),
				tt.providerName,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"GetProviderKey() error = %v, wantErr = %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"GetProviderKey() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
