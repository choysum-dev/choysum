// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
)

// KeyProvider defines the interface for JWT key providers.
type KeyProvider interface {
	// GetPrivateKey returns the private key.
	GetPrivateKey() (*rsa.PrivateKey, error)

	// GetPublicKey returns the public key.
	GetPublicKey() (*rsa.PublicKey, error)

	// Close releases resources and persists keys when needed.
	Close() error
}

// FileKeyProvider loads JWT keys from files.
type FileKeyProvider struct {
	config      *config.JWTConfig
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	mu          sync.RWMutex
	initialized bool
}

// NewFileKeyProvider creates a file-backed key provider.
func NewFileKeyProvider(cfg *config.JWTConfig) (*FileKeyProvider, error) {
	provider := &FileKeyProvider{
		config: cfg,
	}

	// Initialize keys.
	if err := provider.initialize(); err != nil {
		return nil, err
	}

	return provider, nil
}

// initialize loads or generates keys as needed.
func (p *FileKeyProvider) initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	privateKeyPath, publicKeyPath, err := resolveJWTKeyFilePaths(p.config)
	if err != nil {
		return autherrors.WrapAuthError(err, autherrors.ErrKeyLoadingFailed, "failed to resolve JWT key paths")
	}

	if err := ensureDirectory(filepath.Dir(privateKeyPath), 0700); err != nil {
		return autherrors.WrapAuthError(err, autherrors.ErrKeyDirectoryCreateFailed, "failed to create private key directory")
	}
	if err := ensureDirectory(filepath.Dir(publicKeyPath), 0700); err != nil {
		return autherrors.WrapAuthError(err, autherrors.ErrKeyDirectoryCreateFailed, "failed to create public key directory")
	}

	privateExists, err := fileExists(privateKeyPath)
	if err != nil {
		return autherrors.WrapAuthError(err, autherrors.ErrKeyLoadingFailed, "failed to stat private key file")
	}
	publicExists, err := fileExists(publicKeyPath)
	if err != nil {
		return autherrors.WrapAuthError(err, autherrors.ErrKeyLoadingFailed, "failed to stat public key file")
	}

	if privateExists && publicExists {
		privateKey, publicKey, err := p.loadKeys(privateKeyPath, publicKeyPath)
		if err != nil {
			return err
		}
		p.privateKey = privateKey
		p.publicKey = publicKey
		p.initialized = true
		return nil
	}

	if !privateExists && !publicExists {
		if p.config.AutoGenerateKeys {
			privateKey, publicKey, err := p.generateKeys(privateKeyPath, publicKeyPath)
			if err != nil {
				return autherrors.WrapAuthError(err, autherrors.ErrKeyGenerationFailed, "failed to generate key pair")
			}
			p.privateKey = privateKey
			p.publicKey = publicKey
			p.initialized = true
			return nil
		}
		return autherrors.NewAuthError(autherrors.ErrKeyLoadingFailed, "key files do not exist and auto-generation is disabled")
	}

	return autherrors.NewAuthError(autherrors.ErrKeyLoadingFailed, "key files are incomplete: private and public keys must both exist")
}

func resolveJWTKeyFilePaths(cfg *config.JWTConfig) (string, string, error) {
	if cfg == nil {
		return "", "", fmt.Errorf("jwt config is required")
	}

	privateKeyPath := strings.TrimSpace(cfg.PrivateKeyFile)
	publicKeyPath := strings.TrimSpace(cfg.PublicKeyFile)

	if privateKeyPath == "" {
		return "", "", fmt.Errorf("jwt.privateKeyFile is required")
	}

	if publicKeyPath == "" {
		return "", "", fmt.Errorf("jwt.publicKeyFile is required")
	}

	privateKeyPath = filepath.Clean(privateKeyPath)
	publicKeyPath = filepath.Clean(publicKeyPath)
	if privateKeyPath == "." || privateKeyPath == string(filepath.Separator) {
		return "", "", fmt.Errorf("jwt.privateKeyFile must be a non-root file path")
	}
	if publicKeyPath == "." || publicKeyPath == string(filepath.Separator) {
		return "", "", fmt.Errorf("jwt.publicKeyFile must be a non-root file path")
	}

	return privateKeyPath, publicKeyPath, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ensureDirectory(path string, perm os.FileMode) error {
	path = filepath.Clean(path)
	if path == "." {
		return nil
	}

	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			targetInfo, statErr := os.Stat(path)
			if statErr != nil {
				return fmt.Errorf("symlink %s does not resolve to a directory: %w", path, statErr)
			}
			if !targetInfo.IsDir() {
				return fmt.Errorf("symlink %s does not point to a directory", path)
			}
			return nil
		}
		if !info.IsDir() {
			return fmt.Errorf("path %s exists and is not a directory", path)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	parent := filepath.Dir(path)
	if parent != path {
		if err := ensureDirectory(parent, perm); err != nil {
			return err
		}
	}

	if err := os.Mkdir(path, perm); err != nil {
		if os.IsExist(err) {
			return ensureDirectory(path, perm)
		}
		return err
	}

	return nil
}

// loadKeys loads keys from files.
func (p *FileKeyProvider) loadKeys(privateKeyPath, publicKeyPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Read the private key.
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrKeyLoadingFailed, "failed to read private key file")
	}

	privateKeyBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyBlock == nil || privateKeyBlock.Type != "RSA PRIVATE KEY" {
		return nil, nil, autherrors.NewAuthError(autherrors.ErrInvalidKeyFormat, "invalid private key PEM data")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrInvalidKeyFormat, "failed to parse private key")
	}

	// Read the public key.
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrKeyLoadingFailed, "failed to read public key file")
	}

	publicKeyBlock, _ := pem.Decode(publicKeyBytes)
	if publicKeyBlock == nil || publicKeyBlock.Type != "RSA PUBLIC KEY" {
		return nil, nil, autherrors.NewAuthError(autherrors.ErrInvalidKeyFormat, "invalid public key PEM data")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrInvalidKeyFormat, "failed to parse public key")
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, nil, autherrors.NewAuthError(autherrors.ErrInvalidKeyFormat, "public key is not an RSA public key")
	}

	return privateKey, publicKey, nil
}

// generateKeys generates a key pair.
func (p *FileKeyProvider) generateKeys(privateKeyPath, publicKeyPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Generate an RSA key pair.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrKeyGenerationFailed, "failed to generate RSA key pair")
	}

	publicKey := &privateKey.PublicKey

	// Encode the private key as PEM.
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode the public key as PEM.
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrKeyGenerationFailed, "failed to marshal public key")
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Write the private key file.
	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrKeyFileWriteFailed, "failed to write private key file")
	}

	// Write the public key file.
	if err := os.WriteFile(publicKeyPath, publicKeyPEM, 0644); err != nil {
		return nil, nil, autherrors.WrapAuthError(err, autherrors.ErrKeyFileWriteFailed, "failed to write public key file")
	}

	return privateKey, publicKey, nil
}

// GetPrivateKey implements KeyProvider.
func (p *FileKeyProvider) GetPrivateKey() (*rsa.PrivateKey, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, autherrors.NewAuthError(autherrors.ErrKeyProviderNotInitialized, "key provider is not initialized")
	}

	return p.privateKey, nil
}

// GetPublicKey implements KeyProvider.
func (p *FileKeyProvider) GetPublicKey() (*rsa.PublicKey, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, autherrors.NewAuthError(autherrors.ErrKeyProviderNotInitialized, "key provider is not initialized")
	}

	return p.publicKey, nil
}

// Close implements KeyProvider.
func (p *FileKeyProvider) Close() error {
	return nil
}
