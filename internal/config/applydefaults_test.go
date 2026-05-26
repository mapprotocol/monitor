package config

import (
	"reflect"
	"testing"
)

// TestApplyDefaults_Idempotent locks in that applyDefaults can be safely
// run multiple times without accumulating duplicate users / contract
// tokens / opts entries. Hot-reload calls applyDefaults on each new
// config; idempotency means we cannot poison state by reloading.
func TestApplyDefaults_Idempotent(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "no defaults, plain chains",
			cfg: Config{
				Chains: []RawChainConfig{
					{Name: "map", Id: "22776", Endpoint: "u"},
					{Name: "bsc", Endpoint: "u2"},
				},
			},
		},
		{
			name: "defaults users merged into empty chain.Users",
			cfg: Config{
				Defaults: Defaults{
					Users: []From{{Group: "g1", From: "0xa", WaterLine: "100"}},
				},
				Chains: []RawChainConfig{
					{Name: "map", Id: "22776", Endpoint: "u"},
					{Name: "bsc", Endpoint: "u2"},
				},
			},
		},
		{
			name: "defaults users merged into chain with own users",
			cfg: Config{
				Defaults: Defaults{
					Users: []From{{Group: "g1", From: "0xdefault", WaterLine: "100"}},
				},
				Chains: []RawChainConfig{
					{Name: "map", Id: "22776", Endpoint: "u"},
					{Name: "bsc", Endpoint: "u2", Users: []From{{Group: "g1", WaterLine: "200"}}},
				},
			},
		},
		{
			name: "defaults opts merged",
			cfg: Config{
				Defaults: Defaults{
					Opts: map[string]string{"waterLine": "1"},
				},
				Chains: []RawChainConfig{
					{Name: "map", Id: "22776", Endpoint: "u"},
					{Name: "bsc", Endpoint: "u2", Opts: map[string]string{"lightnode": "0xL"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.cfg
			b := tc.cfg
			a.applyDefaults()
			b.applyDefaults()
			b.applyDefaults() // double apply

			if !reflect.DeepEqual(a, b) {
				t.Fatalf("applyDefaults is not idempotent\nonce:   %#v\ntwice:  %#v", a, b)
			}
		})
	}
}
