package uicommon

import (
	"testing"
)

func TestParseEpicCodeFromWebViewPayload(t *testing.T) {
	tests := []struct {
		name     string
		payload  epicWebViewPayload
		wantCode string
		wantOK   bool
	}{
		{
			name: "direct code field",
			payload: epicWebViewPayload{
				Code: "1234567890abcdef1234567890abcdef",
			},
			wantCode: "1234567890abcdef1234567890abcdef",
			wantOK:   true,
		},
		{
			name: "code from url query parameter",
			payload: epicWebViewPayload{
				URL: "https://www.epicgames.com/id/api/redirect?code=abcdef1234567890abcdef1234567890",
			},
			wantCode: "abcdef1234567890abcdef1234567890",
			wantOK:   true,
		},
		{
			name: "code from raw json payload",
			payload: epicWebViewPayload{
				Raw: `{"authorizationCode":"0123456789abcdef0123456789abcdef"}`,
			},
			wantCode: "0123456789abcdef0123456789abcdef",
			wantOK:   true,
		},
		{
			name: "empty payload",
			payload: epicWebViewPayload{
				Code: "",
				Raw:  "",
				URL:  "https://www.epicgames.com/id/login",
			},
			wantCode: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotOK := parseEpicCodeFromWebViewPayload(tt.payload)
			if gotOK != tt.wantOK {
				t.Errorf("parseEpicCodeFromWebViewPayload() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
			if gotCode != tt.wantCode {
				t.Errorf("parseEpicCodeFromWebViewPayload() gotCode = %v, want %v", gotCode, tt.wantCode)
			}
		})
	}
}

func TestParseEpicCodeFromClipboard(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantCode string
		wantOK   bool
	}{
		{
			name:     "json payload",
			content:  `{"exchange_code": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"}`,
			wantCode: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
			wantOK:   true,
		},
		{
			name:     "url parameter",
			content:  "https://example.com/?authorization_code=ffeeddccbbaa99887766554433221100",
			wantCode: "ffeeddccbbaa99887766554433221100",
			wantOK:   true,
		},
		{
			name:     "raw code string",
			content:  "0123456789abcdef0123456789abcdef",
			wantCode: "0123456789abcdef0123456789abcdef",
			wantOK:   true,
		},
		{
			name:     "invalid string",
			content:  "invalid_code",
			wantCode: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotOK := parseEpicCodeFromClipboard(tt.content)
			if gotOK != tt.wantOK {
				t.Errorf("parseEpicCodeFromClipboard() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
			if gotCode != tt.wantCode {
				t.Errorf("parseEpicCodeFromClipboard() gotCode = %v, want %v", gotCode, tt.wantCode)
			}
		})
	}
}
