// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
)

func newKeysTestConfig(t *testing.T) *config.JWTConfig {
	t.Helper()
	keysDir := t.TempDir()
	return &config.JWTConfig{
		PrivateKeyFile:   filepath.Join(keysDir, "private.pem"),
		PublicKeyFile:    filepath.Join(keysDir, "public.pem"),
		AutoGenerateKeys: true,
	}
}

func writeRSAKeyPair(t *testing.T, privatePath, publicPath string) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicBytes,
	})

	if err := os.MkdirAll(filepath.Dir(privatePath), 0o700); err != nil {
		t.Fatalf("create private key dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(publicPath), 0o700); err != nil {
		t.Fatalf("create public key dir: %v", err)
	}
	if err := os.WriteFile(privatePath, privatePEM, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(publicPath, publicPEM, 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	return privateKey, publicKey
}

func writeRSAPublicKey(t *testing.T, path string, publicKey *rsa.PublicKey) {
	t.Helper()
	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: publicBytes}), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
}

func TestFileKeyProviderInitializeAndAccessors(t *testing.T) {
	t.Run("generates keys when files are missing", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider, err := NewFileKeyProvider(cfg)
		if err != nil {
			t.Fatalf("NewFileKeyProvider() error = %v", err)
		}
		if _, err := os.Stat(cfg.PrivateKeyFile); err != nil {
			t.Fatalf("private key file not created: %v", err)
		}
		if _, err := os.Stat(cfg.PublicKeyFile); err != nil {
			t.Fatalf("public key file not created: %v", err)
		}
		privateKey, err := provider.GetPrivateKey()
		if err != nil || privateKey == nil {
			t.Fatalf("GetPrivateKey() = %#v, %v", privateKey, err)
		}
		publicKey, err := provider.GetPublicKey()
		if err != nil || publicKey == nil {
			t.Fatalf("GetPublicKey() = %#v, %v", publicKey, err)
		}
		if err := provider.initialize(); err != nil {
			t.Fatalf("initialize() should be idempotent, got %v", err)
		}
	})

	t.Run("returns loading error when files are missing and autogen disabled", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		cfg.AutoGenerateKeys = false
		_, err := NewFileKeyProvider(cfg)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyLoadingFailed) {
			t.Fatalf("expected key loading error, got %v", err)
		}
	})

	t.Run("loads existing key pair from disk", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		wantPrivate, wantPublic := writeRSAKeyPair(t, cfg.PrivateKeyFile, cfg.PublicKeyFile)
		provider, err := NewFileKeyProvider(cfg)
		if err != nil {
			t.Fatalf("NewFileKeyProvider() error = %v", err)
		}
		gotPrivate, err := provider.GetPrivateKey()
		if err != nil {
			t.Fatalf("GetPrivateKey() error = %v", err)
		}
		gotPublic, err := provider.GetPublicKey()
		if err != nil {
			t.Fatalf("GetPublicKey() error = %v", err)
		}
		if gotPrivate.D.Cmp(wantPrivate.D) != 0 {
			t.Fatalf("loaded private key does not match expected key")
		}
		if gotPublic.E != wantPublic.E || gotPublic.N.Cmp(wantPublic.N) != 0 {
			t.Fatalf("loaded public key does not match expected key")
		}
	})

	t.Run("accessors reject uninitialized provider", func(t *testing.T) {
		provider := &FileKeyProvider{config: newKeysTestConfig(t)}
		if _, err := provider.GetPrivateKey(); err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyProviderNotInitialized) {
			t.Fatalf("expected uninitialized private key error, got %v", err)
		}
		if _, err := provider.GetPublicKey(); err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyProviderNotInitialized) {
			t.Fatalf("expected uninitialized public key error, got %v", err)
		}
		if err := provider.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
}

func TestFileKeyProviderLoadKeysErrors(t *testing.T) {
	t.Run("rejects invalid private pem block", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider := &FileKeyProvider{config: cfg}
		if err := os.WriteFile(cfg.PrivateKeyFile, []byte("not a pem"), 0o600); err != nil {
			t.Fatalf("write invalid private pem: %v", err)
		}
		publicKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			t.Fatalf("generate rsa key: %v", err)
		}
		writeRSAPublicKey(t, cfg.PublicKeyFile, &publicKey.PublicKey)

		_, _, err = provider.loadKeys(cfg.PrivateKeyFile, cfg.PublicKeyFile)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidKeyFormat) {
			t.Fatalf("expected invalid private key format error, got %v", err)
		}
	})

	t.Run("rejects unparsable private key", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider := &FileKeyProvider{config: cfg}
		if err := os.WriteFile(cfg.PrivateKeyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("bad")}), 0o600); err != nil {
			t.Fatalf("write invalid private key: %v", err)
		}
		publicKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			t.Fatalf("generate rsa key: %v", err)
		}
		writeRSAPublicKey(t, cfg.PublicKeyFile, &publicKey.PublicKey)

		_, _, err = provider.loadKeys(cfg.PrivateKeyFile, cfg.PublicKeyFile)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidKeyFormat) {
			t.Fatalf("expected private key parse error, got %v", err)
		}
	})

	t.Run("rejects invalid public pem block", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider := &FileKeyProvider{config: cfg}
		writeRSAKeyPair(t, cfg.PrivateKeyFile, cfg.PublicKeyFile)
		if err := os.WriteFile(cfg.PublicKeyFile, []byte("not a pem"), 0o644); err != nil {
			t.Fatalf("write invalid public pem: %v", err)
		}

		_, _, err := provider.loadKeys(cfg.PrivateKeyFile, cfg.PublicKeyFile)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidKeyFormat) {
			t.Fatalf("expected invalid public key format error, got %v", err)
		}
	})

	t.Run("rejects non-rsa public key", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider := &FileKeyProvider{config: cfg}
		writeRSAKeyPair(t, cfg.PrivateKeyFile, cfg.PublicKeyFile)
		ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generate ecdsa key: %v", err)
		}
		publicBytes, err := x509.MarshalPKIXPublicKey(&ecdsaKey.PublicKey)
		if err != nil {
			t.Fatalf("marshal ecdsa public key: %v", err)
		}
		if err := os.WriteFile(cfg.PublicKeyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: publicBytes}), 0o644); err != nil {
			t.Fatalf("write ecdsa public key: %v", err)
		}

		_, _, err = provider.loadKeys(cfg.PrivateKeyFile, cfg.PublicKeyFile)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidKeyFormat) {
			t.Fatalf("expected non-rsa public key error, got %v", err)
		}
	})
}

func TestFileKeyProviderInitializeAndGenerateErrors(t *testing.T) {
	t.Run("initialize accepts symlink directory for key paths", func(t *testing.T) {
		root := t.TempDir()
		targetDir := filepath.Join(root, "target")
		if err := os.MkdirAll(targetDir, 0o700); err != nil {
			t.Fatalf("mkdir target dir: %v", err)
		}

		symlinkDir := filepath.Join(root, "workspace-link")
		if err := os.Symlink(targetDir, symlinkDir); err != nil {
			t.Skipf("symlink is not available in this environment: %v", err)
		}

		cfg := &config.JWTConfig{
			PrivateKeyFile:   filepath.Join(symlinkDir, "jwtkeys", "private.pem"),
			PublicKeyFile:    filepath.Join(symlinkDir, "jwtkeys", "public.pem"),
			AutoGenerateKeys: true,
		}

		provider, err := NewFileKeyProvider(cfg)
		if err != nil {
			t.Fatalf("expected symlink directory to be accepted, got %v", err)
		}

		if _, err := os.Stat(filepath.Join(targetDir, "jwtkeys", "private.pem")); err != nil {
			t.Fatalf("private key was not generated under symlink target: %v", err)
		}
		if _, err := os.Stat(filepath.Join(targetDir, "jwtkeys", "public.pem")); err != nil {
			t.Fatalf("public key was not generated under symlink target: %v", err)
		}

		if err := provider.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	t.Run("initialize returns directory creation errors", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		blockedPath := filepath.Join(t.TempDir(), "blocked")
		if err := os.WriteFile(blockedPath, []byte("x"), 0o600); err != nil {
			t.Fatalf("write blocker file: %v", err)
		}
		cfg.PrivateKeyFile = filepath.Join(blockedPath, "private.pem")
		cfg.PublicKeyFile = filepath.Join(blockedPath, "public.pem")

		_, err := NewFileKeyProvider(cfg)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyDirectoryCreateFailed) {
			t.Fatalf("expected key directory creation failure, got %v", err)
		}
	})

	t.Run("generateKeys returns private key write errors", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider := &FileKeyProvider{config: cfg}
		privatePath := filepath.Join(filepath.Dir(cfg.PrivateKeyFile), "private-dir")
		if err := os.Mkdir(privatePath, 0o700); err != nil {
			t.Fatalf("mkdir private path: %v", err)
		}

		_, _, err := provider.generateKeys(privatePath, cfg.PublicKeyFile)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyFileWriteFailed) {
			t.Fatalf("expected private key write failure, got %v", err)
		}
	})

	t.Run("generateKeys returns public key write errors", func(t *testing.T) {
		cfg := newKeysTestConfig(t)
		provider := &FileKeyProvider{config: cfg}
		publicPath := filepath.Join(filepath.Dir(cfg.PublicKeyFile), "public-dir")
		if err := os.Mkdir(publicPath, 0o700); err != nil {
			t.Fatalf("mkdir public path: %v", err)
		}

		_, _, err := provider.generateKeys(cfg.PrivateKeyFile, publicPath)
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyFileWriteFailed) {
			t.Fatalf("expected public key write failure, got %v", err)
		}
	})
}

func TestResolveJWTKeyFilePathsRequiresExplicitPaths(t *testing.T) {
	t.Run("rejects nil config", func(t *testing.T) {
		if _, _, err := resolveJWTKeyFilePaths(nil); err == nil || !strings.Contains(err.Error(), "jwt config is required") {
			t.Fatalf("expected nil jwt config error, got %v", err)
		}
	})

	t.Run("rejects missing private key path", func(t *testing.T) {
		cfg := &config.JWTConfig{PublicKeyFile: filepath.Join(t.TempDir(), "public.pem")}
		if _, _, err := resolveJWTKeyFilePaths(cfg); err == nil || !strings.Contains(err.Error(), "jwt.privateKeyFile is required") {
			t.Fatalf("expected missing private key path error, got %v", err)
		}
	})

	t.Run("rejects missing public key path", func(t *testing.T) {
		cfg := &config.JWTConfig{PrivateKeyFile: filepath.Join(t.TempDir(), "private.pem")}
		if _, _, err := resolveJWTKeyFilePaths(cfg); err == nil || !strings.Contains(err.Error(), "jwt.publicKeyFile is required") {
			t.Fatalf("expected missing public key path error, got %v", err)
		}
	})

	t.Run("rejects root path", func(t *testing.T) {
		cfg := &config.JWTConfig{PrivateKeyFile: "/", PublicKeyFile: filepath.Join(t.TempDir(), "public.pem")}
		if _, _, err := resolveJWTKeyFilePaths(cfg); err == nil || !strings.Contains(err.Error(), "jwt.privateKeyFile must be a non-root file path") {
			t.Fatalf("expected root private key path error, got %v", err)
		}
	})
}

func TestNewFileKeyProviderRejectsMissingJWTKeyPaths(t *testing.T) {
	_, err := NewFileKeyProvider(&config.JWTConfig{AutoGenerateKeys: true})
	if err == nil || !autherrors.IsAuthError(err, autherrors.ErrKeyLoadingFailed) {
		t.Fatalf("expected wrapped key loading error, got %v", err)
	}
	if !strings.Contains(err.Error(), "jwt.privateKeyFile is required") {
		t.Fatalf("expected missing private key path cause, got %v", err)
	}
}
