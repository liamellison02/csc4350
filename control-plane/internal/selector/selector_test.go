package selector

import (
	"maps"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool
	}{
		{"empty matches all", "", map[string]string{}, false},
		{"whitespace only", "   ", map[string]string{}, false},
		{"single pair", "env=prod", map[string]string{"env": "prod"}, false},
		{"multi pair trimmed", " env=prod , region=us-east ", map[string]string{"env": "prod", "region": "us-east"}, false},
		{"missing value", "env=", nil, true},
		{"missing key", "=prod", nil, true},
		{"no equals", "envprod", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if err == nil && !maps.Equal(got, tc.want) {
				t.Fatalf("Parse(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	labels := map[string]string{"env": "prod", "region": "us-east"}
	cases := []struct {
		name string
		sel  map[string]string
		want bool
	}{
		{"empty matches all", map[string]string{}, true},
		{"exact pair", map[string]string{"env": "prod"}, true},
		{"all pairs must match", map[string]string{"env": "prod", "region": "eu"}, false},
		{"absent key", map[string]string{"team": "obs"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Matches(tc.sel, labels); got != tc.want {
				t.Fatalf("Matches(%v) = %v, want %v", tc.sel, got, tc.want)
			}
		})
	}
}
