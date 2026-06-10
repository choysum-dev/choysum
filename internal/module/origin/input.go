package origin

import (
	"regexp"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

type InputKind string

const (
	InputKindLocal    InputKind = "local"
	InputKindRegistry InputKind = "registry"
)

var localNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
var versionPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-]*$`)

type ParsedInput struct {
	Raw        string
	Kind       InputKind
	LocalName  string
	ModuleName string
	Version    string
}

func (p ParsedInput) CanonicalRef() string {
	if p.Kind != InputKindRegistry {
		return strings.TrimSpace(p.LocalName)
	}
	version := strings.TrimSpace(p.Version)
	if version == "" {
		version = "latest"
	}
	return strings.TrimSpace(p.ModuleName) + "@" + version
}

func ParseInput(input string) (ParsedInput, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return ParsedInput{}, xfmt.Errorf("empty module input")
	}

	if strings.Contains(raw, "/") {
		return ParsedInput{}, xfmt.Errorf("registry alias syntax is no longer supported; use <module>@<version>")
	}

	if strings.Contains(raw, "@") {
		parts := strings.SplitN(raw, "@", 2)
		moduleName := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		if !localNamePattern.MatchString(moduleName) {
			return ParsedInput{}, xfmt.Errorf("invalid registry module name: %s", raw)
		}
		if version == "" {
			return ParsedInput{}, xfmt.Errorf("invalid registry reference: %s (expected <module>@<version>)", raw)
		}
		if !versionPattern.MatchString(version) {
			return ParsedInput{}, xfmt.Errorf("invalid registry version: %s", version)
		}
		return ParsedInput{
			Raw:        raw,
			Kind:       InputKindRegistry,
			ModuleName: moduleName,
			Version:    version,
		}, nil
	}

	if !localNamePattern.MatchString(raw) {
		return ParsedInput{}, xfmt.Errorf("invalid local module name: %s", raw)
	}

	return ParsedInput{
		Raw:        raw,
		Kind:       InputKindLocal,
		LocalName:  raw,
		ModuleName: raw,
	}, nil
}
