package pipeline

import "testing"

func TestOriginType_String(t *testing.T) {
	tests := []struct {
		origin OriginType
		want   string
	}{
		{OriginLocal, "local"},
		{OriginRemote, "remote"},
		{OriginInstalledUntracked, "untracked"},
		{OriginInstalledModified, "modified"},
		{OriginType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.origin.String(); got != tt.want {
			t.Errorf("OriginType(%d).String() = %q, want %q", tt.origin, got, tt.want)
		}
	}
}

func TestCheckStatus_String(t *testing.T) {
	tests := []struct {
		status CheckStatus
		want   string
	}{
		{CheckPass, "PASS"},
		{CheckWarn, "WARN"},
		{CheckFail, "FAIL"},
		{CheckStatus(99), "—"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("CheckStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
