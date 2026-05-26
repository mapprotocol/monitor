package config

import "testing"

func TestParseNativeWaterLine_HumanReadableUnits(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		decimals int32
		want     string
	}{
		{name: "evm integer", value: "5", decimals: 18, want: "5000000000000000000"},
		{name: "evm decimal", value: "0.2", decimals: 18, want: "200000000000000000"},
		{name: "near integer", value: "5", decimals: 24, want: "5000000000000000000000000"},
		{name: "near decimal", value: "0.5", decimals: 24, want: "500000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseNativeWaterLine(tt.value, tt.decimals)
			if !ok {
				t.Fatalf("ParseNativeWaterLine(%q, %d) returned ok=false", tt.value, tt.decimals)
			}
			if got.String() != tt.want {
				t.Fatalf("ParseNativeWaterLine(%q, %d)=%s, want %s", tt.value, tt.decimals, got, tt.want)
			}
		})
	}
}

func TestParseNativeWaterLine_LegacySmallestUnit(t *testing.T) {
	tests := []string{
		"200000000000000000",
		"400000000000000",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			got, ok := ParseNativeWaterLine(tt, 18)
			if !ok {
				t.Fatal("expected legacy smallest-unit waterLine to parse")
			}
			if got.String() != tt {
				t.Fatalf("legacy waterLine changed: got %s, want %s", got, tt)
			}
		})
	}
}

func TestParseNativeWaterLine_Invalid(t *testing.T) {
	tests := []string{"", "abc", "-1", "0.0000000000000000001"}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if got, ok := ParseNativeWaterLine(tt, 18); ok {
				t.Fatalf("ParseNativeWaterLine(%q) ok=true got %s, want ok=false", tt, got)
			}
		})
	}
}
