package opamp

import (
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
)

func TestHostname(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{"present", map[string]string{attrHostName: "collector-01"}, "collector-01"},
		{"legacy fallback", map[string]string{attrHostNameLegacy: "legacy-host"}, "legacy-host"},
		{"prefers canonical", map[string]string{attrHostName: "canon", attrHostNameLegacy: "legacy"}, "canon"},
		{"missing", map[string]string{attrServiceName: "x"}, unknown},
		{"empty value", map[string]string{attrHostName: ""}, unknown},
		{"nil map", nil, unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Hostname(tt.attrs); got != tt.want {
				t.Errorf("Hostname(%v) = %q, want %q", tt.attrs, got, tt.want)
			}
		})
	}
}

func TestAgentType(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{"present", map[string]string{attrServiceName: "io.otel.collector"}, "io.otel.collector"},
		{"missing", map[string]string{attrHostName: "h"}, unknown},
		{"empty value", map[string]string{attrServiceName: ""}, unknown},
		{"nil map", nil, unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AgentType(tt.attrs); got != tt.want {
				t.Errorf("AgentType(%v) = %q, want %q", tt.attrs, got, tt.want)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{"present", map[string]string{attrServiceVersion: "0.147.0"}, "0.147.0"},
		{"missing", map[string]string{attrServiceName: "x"}, unknown},
		{"empty value", map[string]string{attrServiceVersion: ""}, unknown},
		{"nil map", nil, unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Version(tt.attrs); got != tt.want {
				t.Errorf("Version(%v) = %q, want %q", tt.attrs, got, tt.want)
			}
		})
	}
}

func TestIdentify(t *testing.T) {
	desc := &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			strKV(attrHostName, "collector-prod-01"),
			strKV(attrServiceName, "io.opentelemetry.collector"),
			strKV(attrServiceVersion, "0.147.0"),
		},
	}
	host, typ, ver := Identify(desc)
	if host != "collector-prod-01" || typ != "io.opentelemetry.collector" || ver != "0.147.0" {
		t.Errorf("Identify(full) = (%q, %q, %q)", host, typ, ver)
	}

	// missing attributes fall back to unknown across the board.
	host, typ, ver = Identify(&protobufs.AgentDescription{})
	if host != unknown || typ != unknown || ver != unknown {
		t.Errorf("Identify(empty) = (%q, %q, %q), want all %q", host, typ, ver, unknown)
	}

	// nil description must not panic and must fall back.
	host, typ, ver = Identify(nil)
	if host != unknown || typ != unknown || ver != unknown {
		t.Errorf("Identify(nil) = (%q, %q, %q), want all %q", host, typ, ver, unknown)
	}
}

func TestAnyValueString(t *testing.T) {
	tests := []struct {
		name string
		in   *protobufs.AnyValue
		want string
	}{
		{"nil", nil, ""},
		{"string", &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: "s"}}, "s"},
		{"bool", &protobufs.AnyValue{Value: &protobufs.AnyValue_BoolValue{BoolValue: true}}, "true"},
		{"int", &protobufs.AnyValue{Value: &protobufs.AnyValue_IntValue{IntValue: 42}}, "42"},
		{"double", &protobufs.AnyValue{Value: &protobufs.AnyValue_DoubleValue{DoubleValue: 1.5}}, "1.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := anyValueString(tt.in); got != tt.want {
				t.Errorf("anyValueString = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstanceUID(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"empty", nil, ""},
		{
			"canonical uuid",
			[]byte{0x01, 0x93, 0xd2, 0x4a, 0x2b, 0xc4, 0x7e, 0x0a, 0x9f, 0x11, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			"0193d24a-2bc4-7e0a-9f11-aabbccddeeff",
		},
		{"short hex", []byte{0xde, 0xad, 0xbe, 0xef}, "deadbeef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InstanceUID(tt.in); got != tt.want {
				t.Errorf("InstanceUID(%x) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// strKV builds a string-valued identifying attribute.
func strKV(key, val string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key:   key,
		Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: val}},
	}
}
