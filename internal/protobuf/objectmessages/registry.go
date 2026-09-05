// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package objectmessages registers named TypeScript object types that emit
// dedicated protobuf messages instead of google.protobuf.Value.
package objectmessages

import (
	"sort"
	"strings"
)

// Def is a frozen object-message definition keyed by the TypeScript type name.
type Def struct {
	// TSName is the TypeScript type alias / interface name (e.g. FieldRuleSpec).
	TSName string
	// ProtoName is the emitted protobuf message name (usually equal to TSName).
	ProtoName string
	// Body is the message body (fields only), without the surrounding message { }.
	// Field numbers are frozen: never reuse; delete via reserved in a follow-up.
	Body string
}

// registry lists named object types that emit dedicated protobuf messages.
// New types require an explicit entry and golden coverage; do not auto-synthesize
// from arbitrary object shapes.
var registry = []Def{
	{
		TSName:    "FieldRuleSpec",
		ProtoName: "FieldRuleSpec",
		Body: strings.TrimSpace(`
  repeated string denyReadFields = 1;
  repeated string denyWriteFields = 2;
  string reason = 3;
  repeated string hitRuleIds = 4;
`),
	},
	{
		TSName:    "ConditionEnvelope",
		ProtoName: "ConditionEnvelope",
		// kind mirrors the TS discriminant; expr stays Value until the condition
		// tree is modeled as its own message.
		Body: strings.TrimSpace(`
  string kind = 1;
  google.protobuf.Value expr = 2;
  string reason = 3;
  repeated string hitRuleIds = 4;
`),
	},
}

var byTSName map[string]Def

func init() {
	byTSName = make(map[string]Def, len(registry))
	for _, def := range registry {
		byTSName[def.TSName] = def
	}
}

// Lookup returns the registered definition for a TypeScript type name.
func Lookup(tsName string) (Def, bool) {
	def, ok := byTSName[strings.TrimSpace(tsName)]
	return def, ok
}

// ProtoNameForTS returns the protobuf message name when tsName is registered.
func ProtoNameForTS(tsName string) (string, bool) {
	def, ok := Lookup(tsName)
	if !ok {
		return "", false
	}
	return def.ProtoName, true
}

// IsRegisteredProtoName reports whether name is a registered object message.
func IsRegisteredProtoName(name string) bool {
	name = strings.TrimSpace(name)
	for _, def := range registry {
		if def.ProtoName == name {
			return true
		}
	}
	return false
}

// CollectUsed returns registered defs referenced by the given protobuf type
// strings (parameters and returns), sorted by ProtoName for stable emission.
func CollectUsed(protoTypes ...string) []Def {
	seen := make(map[string]Def)
	for _, pt := range protoTypes {
		pt = strings.TrimSpace(pt)
		if pt == "" {
			continue
		}
		// Strip repeated prefix if present (e.g. repeated FieldRuleSpec).
		pt = strings.TrimPrefix(pt, "repeated ")
		pt = strings.TrimSpace(pt)
		def, ok := Lookup(pt)
		if !ok {
			// ProtoName may differ from TSName in future registry entries.
			def, ok = lookupByProtoName(pt)
		}
		if !ok {
			continue
		}
		seen[def.ProtoName] = def
	}
	out := make([]Def, 0, len(seen))
	for _, def := range seen {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ProtoName < out[j].ProtoName
	})
	return out
}

func lookupByProtoName(protoName string) (Def, bool) {
	protoName = strings.TrimSpace(protoName)
	for _, def := range registry {
		if def.ProtoName == protoName {
			return def, true
		}
	}
	return Def{}, false
}

// All returns a copy of the full registry (stable ProtoName order).
func All() []Def {
	out := make([]Def, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ProtoName < out[j].ProtoName
	})
	return out
}

// MessageSource returns the full `message Name { ... }` text for a def.
func MessageSource(def Def) string {
	return "message " + def.ProtoName + " {\n" + def.Body + "\n}"
}
