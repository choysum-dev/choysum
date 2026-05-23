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
var registryRefPattern = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9_-]*)/([a-zA-Z0-9][a-zA-Z0-9_-]*)(?:@([a-zA-Z0-9][a-zA-Z0-9._+\-]*))?$`)

type ParsedInput struct {
	Raw           string
	Kind          InputKind
	LocalName     string
	RegistryAlias string
	ModuleName    string
	Version       string
}

func (p ParsedInput) CanonicalRef() string {
	if p.Kind != InputKindRegistry {
		return strings.TrimSpace(p.LocalName)
	}
	version := strings.TrimSpace(p.Version)
	if version == "" {
		version = "latest"
	}
	return strings.TrimSpace(p.RegistryAlias) + "/" + strings.TrimSpace(p.ModuleName) + "@" + version
}

func ParseInput(input string) (ParsedInput, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return ParsedInput{}, xfmt.Errorf("empty module input")
	}

	if strings.Contains(raw, "/") {
		m := registryRefPattern.FindStringSubmatch(raw)
		if len(m) == 0 {
			return ParsedInput{}, xfmt.Errorf("invalid registry reference: %s", raw)
		}
		version := strings.TrimSpace(m[3])
		if version == "" {
			version = "latest"
		}
		return ParsedInput{
			Raw:           raw,
			Kind:          InputKindRegistry,
			RegistryAlias: strings.TrimSpace(m[1]),
			ModuleName:    strings.TrimSpace(m[2]),
			Version:       version,
		}, nil
	}

	if !localNamePattern.MatchString(raw) {
		name := raw
		version := ""
		if strings.Contains(raw, "@") {
			parts := strings.SplitN(raw, "@", 2)
			name = strings.TrimSpace(parts[0])
			version = strings.TrimSpace(parts[1])
			if version == "" {
				version = "latest"
			}
		}
		if !localNamePattern.MatchString(name) {
			return ParsedInput{}, xfmt.Errorf("invalid local module name: %s", raw)
		}
		return ParsedInput{
			Raw:        raw,
			Kind:       InputKindLocal,
			LocalName:  name,
			ModuleName: name,
			Version:    version,
		}, nil
	}

	return ParsedInput{
		Raw:        raw,
		Kind:       InputKindLocal,
		LocalName:  raw,
		ModuleName: raw,
	}, nil
}
