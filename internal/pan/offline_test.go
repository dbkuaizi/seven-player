package pan

import "testing"

func TestNormalizeOfflineQuota(t *testing.T) {
	tests := []struct {
		name           string
		quota          int64
		total          int64
		wantQuota      int64
		wantTotal      int64
	}{
		{
			name:      "normal remaining and total",
			quota:     3000,
			total:     3000,
			wantQuota: 3000,
			wantTotal: 3000,
		},
		{
			name:      "missing total falls back to quota",
			quota:     3000,
			total:     0,
			wantQuota: 3000,
			wantTotal: 3000,
		},
		{
			name:      "total never stays below quota",
			quota:     3000,
			total:     1000,
			wantQuota: 3000,
			wantTotal: 3000,
		},
		{
			name:      "negative values are clamped",
			quota:     -2,
			total:     -5,
			wantQuota: 0,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuota, gotTotal := normalizeOfflineQuota(tt.quota, tt.total)
			if gotQuota != tt.wantQuota || gotTotal != tt.wantTotal {
				t.Fatalf("normalizeOfflineQuota(%d, %d) = (%d, %d), want (%d, %d)", tt.quota, tt.total, gotQuota, gotTotal, tt.wantQuota, tt.wantTotal)
			}
		})
	}
}
