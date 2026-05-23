// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package storage_test

import (
	"context"
	"reflect"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/document/storage/db"
	_ "github.com/choysum-dev/choysum/internal/document/storage/s3"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/storage"
)

type stubDriver struct {
	provider string
}

func (d stubDriver) Provider() string { return d.provider }

func (d stubDriver) Put(ctx context.Context, input storage.PutPayloadInput) (storage.PayloadMutation, error) {
	_ = ctx
	_ = input
	return storage.PayloadMutation{}, nil
}

func (d stubDriver) Open(ctx context.Context, record storage.StoredContentRecord) ([]byte, error) {
	_ = ctx
	_ = record
	return nil, nil
}

func (d stubDriver) Delete(ctx context.Context, record storage.StoredContentRecord) error {
	_ = ctx
	_ = record
	return nil
}

func TestNewFactoryReturnsRegisteredDBAndS3Drivers(t *testing.T) {
	factory := storage.NewFactory()

	dbDriver, err := factory.NewDriver("db", nil)
	if err != nil {
		t.Fatalf("NewDriver(db) error = %v", err)
	}
	if dbDriver.Provider() != "db" {
		t.Fatalf("db driver provider = %q, want db", dbDriver.Provider())
	}

	att := &config.AttachmentConfig{S3: &config.AttachmentS3Config{
		Bucket:    "choysum-attachments-test",
		Endpoint:  "127.0.0.1:9000",
		AccessKey: "ak",
		SecretKey: "sk",
	}}
	s3Driver, err := factory.NewDriver("s3", att)
	if err != nil {
		t.Fatalf("NewDriver(s3) error = %v", err)
	}
	if s3Driver.Provider() != "s3" {
		t.Fatalf("s3 driver provider = %q, want s3", s3Driver.Provider())
	}
}

func TestRegisterAddsProviderAndFactoryResolvesIt(t *testing.T) {
	const provider = "phase1-test-provider"
	storage.Register(provider, func(att *config.AttachmentConfig) (storage.StoredContentDriver, error) {
		_ = att
		return stubDriver{provider: provider}, nil
	})

	if !storage.Exists(provider) {
		t.Fatalf("Exists(%q) = false, want true", provider)
	}

	providers := storage.Providers()
	if !contains(providers, provider) {
		t.Fatalf("Providers() = %v, want to contain %q", providers, provider)
	}

	driver, err := storage.NewFactory().NewDriver(provider, nil)
	if err != nil {
		t.Fatalf("NewDriver(%q) error = %v", provider, err)
	}
	if driver.Provider() != provider {
		t.Fatalf("driver.Provider() = %q, want %q", driver.Provider(), provider)
	}
}

func TestRegisterPanicsOnDuplicateProvider(t *testing.T) {
	const provider = "phase1-test-duplicate"
	storage.Register(provider, func(att *config.AttachmentConfig) (storage.StoredContentDriver, error) {
		_ = att
		return stubDriver{provider: provider}, nil
	})

	defer func() {
		if recover() == nil {
			t.Fatal("Register() should panic on duplicate provider")
		}
	}()

	storage.Register(provider, func(att *config.AttachmentConfig) (storage.StoredContentDriver, error) {
		_ = att
		return stubDriver{provider: provider}, nil
	})
}

func TestProvidersReturnsSortedCopy(t *testing.T) {
	providers := storage.Providers()
	if !reflect.DeepEqual(providers, append([]string(nil), providers...)) {
		t.Fatal("Providers() should return a copy")
	}
	if !sortCopyEqual(providers) {
		t.Fatalf("Providers() = %v, want sorted order", providers)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortCopyEqual(values []string) bool {
	clone := append([]string(nil), values...)
	for index := 1; index < len(clone); index++ {
		if clone[index-1] > clone[index] {
			return false
		}
	}
	return true
}
