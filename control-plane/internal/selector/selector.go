// Package selector parses and evaluates configuration label selectors.
package selector

import (
	"fmt"
	"strings"
)

// Parse turns "k=v[,k=v]" into a map. empty input means match-all.
func Parse(s string) (map[string]string, error) {
	out := map[string]string{}
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	for _, pair := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(pair, "=")
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if !ok || k == "" || v == "" {
			return nil, fmt.Errorf("malformed selector pair %q", strings.TrimSpace(pair))
		}
		out[k] = v
	}
	return out, nil
}

// Matches reports whether every selector pair equals the agent label.
func Matches(sel, labels map[string]string) bool {
	for k, v := range sel {
		if labels[k] != v {
			return false
		}
	}
	return true
}
