// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"reflect"
	"strings"
	"testing"
)

func testOwner(domain string) ConfigRootOwner {
	return ConfigRootOwner{
		Domain:      domain,
		PackagePath: "internal/test",
		OptionsType: "TestOptions",
	}
}

func TestValidateRootOwnerMapWithConfig(t *testing.T) {
	if err := validateRootOwnerMap(reflect.TypeOf(Config{}), configRootOwnerMap); err != nil {
		t.Fatalf("validateRootOwnerMap(Config) returned error: %v", err)
	}
}

func TestValidateRootOwnerMapRejectsMissingOwner(t *testing.T) {
	type testConfig struct {
		First  string `mapstructure:"first"`
		Second string `mapstructure:"second"`
	}

	err := validateRootOwnerMap(reflect.TypeOf(testConfig{}), map[string]ConfigRootOwner{
		"first": testOwner("test"),
	})
	if err == nil {
		t.Fatal("expected missing owner map entry error")
	}
	if !strings.Contains(err.Error(), `missing from owner map`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootOwnerMapRejectsDuplicateMapstructureTag(t *testing.T) {
	type testConfig struct {
		First  string `mapstructure:"dup"`
		Second string `mapstructure:"dup"`
	}

	err := validateRootOwnerMap(reflect.TypeOf(testConfig{}), map[string]ConfigRootOwner{
		"dup": testOwner("test"),
	})
	if err == nil {
		t.Fatal("expected duplicate mapstructure key error")
	}
	if !strings.Contains(err.Error(), `duplicated`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootOwnerMapRejectsInvalidMapstructureTag(t *testing.T) {
	type testConfig struct {
		First string `mapstructure:""`
	}

	err := validateRootOwnerMap(reflect.TypeOf(testConfig{}), map[string]ConfigRootOwner{
		"first": testOwner("test"),
	})
	if err == nil {
		t.Fatal("expected invalid mapstructure tag error")
	}
	if !strings.Contains(err.Error(), `invalid mapstructure tag`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootOwnerMapRejectsUnknownOwnerKey(t *testing.T) {
	type testConfig struct {
		First string `mapstructure:"first"`
	}

	err := validateRootOwnerMap(reflect.TypeOf(testConfig{}), map[string]ConfigRootOwner{
		"first":  testOwner("test"),
		"unused": testOwner("test"),
	})
	if err == nil {
		t.Fatal("expected unknown owner map key error")
	}
	if !strings.Contains(err.Error(), `does not match any Config field`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRootOwnerMapRejectsEmptyOwnerMetadata(t *testing.T) {
	type testConfig struct {
		First string `mapstructure:"first"`
	}

	err := validateRootOwnerMap(reflect.TypeOf(testConfig{}), map[string]ConfigRootOwner{
		"first": {},
	})
	if err == nil {
		t.Fatal("expected empty owner metadata error")
	}
	if !strings.Contains(err.Error(), `invalid owner metadata`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
