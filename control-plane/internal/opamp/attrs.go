package opamp

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/open-telemetry/opamp-go/protobufs"
)

// opentelemetry semantic-convention keys we map from.
const (
	attrHostName       = "host.name"
	attrHostNameLegacy = "host.hostname"
	attrServiceName    = "service.name"
	attrServiceVersion = "service.version"
)

// unknown is the fallback used when an identifying attribute is absent.
const unknown = "unknown"

// Identify maps an agent description's identifying attributes to the
// hostname, agent type and version columns, applying fallbacks.
func Identify(desc *protobufs.AgentDescription) (hostname, agentType, version string) {
	var kvs []*protobufs.KeyValue
	if desc != nil {
		kvs = desc.IdentifyingAttributes
	}
	attrs := flattenAttrs(kvs)
	return Hostname(attrs), AgentType(attrs), Version(attrs)
}

// Hostname resolves the agent hostname, falling back to unknown.
func Hostname(attrs map[string]string) string {
	return firstNonEmpty(attrs, unknown, attrHostName, attrHostNameLegacy)
}

// AgentType resolves the agent type from service.name.
func AgentType(attrs map[string]string) string {
	return firstNonEmpty(attrs, unknown, attrServiceName)
}

// Version resolves the agent build version from service.version.
func Version(attrs map[string]string) string {
	return firstNonEmpty(attrs, unknown, attrServiceVersion)
}

// firstNonEmpty returns the first non-empty value among keys, else fallback.
func firstNonEmpty(attrs map[string]string, fallback string, keys ...string) string {
	for _, k := range keys {
		if v := attrs[k]; v != "" {
			return v
		}
	}
	return fallback
}

// flattenAttrs converts opamp key-values into a plain string map,
// stringifying primitive value kinds and skipping nil or unkeyed entries.
func flattenAttrs(kvs []*protobufs.KeyValue) map[string]string {
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		if kv == nil || kv.Key == "" {
			continue
		}
		out[kv.Key] = anyValueString(kv.Value)
	}
	return out
}

// anyValueString renders a primitive AnyValue as a string; empty for
// nil or non-primitive kinds.
func anyValueString(v *protobufs.AnyValue) string {
	if v == nil {
		return ""
	}
	switch x := v.GetValue().(type) {
	case *protobufs.AnyValue_StringValue:
		return x.StringValue
	case *protobufs.AnyValue_BoolValue:
		return strconv.FormatBool(x.BoolValue)
	case *protobufs.AnyValue_IntValue:
		return strconv.FormatInt(x.IntValue, 10)
	case *protobufs.AnyValue_DoubleValue:
		return strconv.FormatFloat(x.DoubleValue, 'g', -1, 64)
	default:
		return ""
	}
}

// InstanceUID renders the opamp instance uid bytes as a stable string key.
// 16-byte uids format as a canonical uuid; other lengths hex-encode.
func InstanceUID(raw []byte) string {
	switch len(raw) {
	case 0:
		return ""
	case 16:
		return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
	default:
		return hex.EncodeToString(raw)
	}
}
