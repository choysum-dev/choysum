// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	xfmt "golang.org/x/exp/errors/fmt"
)

const (
	LockTTLDefault                    = 30 * time.Second
	LockHeartbeatIntervalDefault      = 10 * time.Second
	LockConflictErrorCodeDefault      = "LOCK_CONFLICT"
	LockRetryBackoffInitialDefault    = 200 * time.Millisecond
	LockRetryBackoffMultiplierDefault = 2
	LockRetryBackoffMaxDefault        = 3 * time.Second
)

type ModulesLockLease struct {
	Owner         string `json:"owner"`
	PID           int    `json:"pid"`
	StartedAt     string `json:"startedAt"`
	TTL           string `json:"ttl"`
	Operation     string `json:"operation"`
	LastHeartbeat string `json:"lastHeartbeat"`
	ErrorCode     string `json:"errorCode"`
	RetryBackoff  string `json:"retryBackoff"`
}

type LockConflictError struct {
	Path     string
	Metadata ModulesLockLease
}

func (e *LockConflictError) Error() string {
	retry := strings.TrimSpace(e.Metadata.RetryBackoff)
	if retry == "" {
		retry = LockRetryBackoffInitialDefault.String()
	}
	return xfmt.Sprintf("workspace modules lock is busy (path=%s, retry_after=%s)", e.Path, retry)
}

func AcquireModulesLockLease(workspaceRoot string, operation string, defaultChoysumPath string) (func() error, error) {
	lockPath, err := modulesLockLeasePath(workspaceRoot, defaultChoysumPath)
	if err != nil {
		return nil, xfmt.Errorf("resolve workspace modules lock lease path failed: %w", err)
	}
	choysumDir, err := workspaceChoysumDir(workspaceRoot, defaultChoysumPath)
	if err != nil {
		return nil, xfmt.Errorf("resolve workspace .choysum dir failed: %w", err)
	}
	if err := os.MkdirAll(choysumDir, 0o755); err != nil {
		return nil, xfmt.Errorf("create workspace .choysum dir failed: %w", err)
	}

	now := time.Now().UTC()
	hostname, _ := os.Hostname()
	lease := ModulesLockLease{
		Owner:         strings.TrimSpace(hostname),
		PID:           os.Getpid(),
		StartedAt:     now.Format(time.RFC3339Nano),
		TTL:           LockTTLDefault.String(),
		Operation:     strings.TrimSpace(operation),
		LastHeartbeat: now.Format(time.RFC3339Nano),
		ErrorCode:     LockConflictErrorCodeDefault,
		RetryBackoff:  LockRetryBackoffInitialDefault.String(),
	}
	if lease.Owner == "" {
		lease.Owner = "unknown"
	}
	if lease.Operation == "" {
		lease.Operation = "modules-lock-write"
	}

	for attempt := 0; attempt < 2; attempt++ {
		err := writeLeaseFileExclusive(lockPath, lease)
		if err == nil {
			released := false
			return func() error {
				if released {
					return nil
				}
				released = true
				if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
					return xfmt.Errorf("release workspace modules lock failed: %w", err)
				}
				return nil
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, xfmt.Errorf("acquire workspace modules lock failed: %w", err)
		}

		existing, readErr := readLeaseFile(lockPath)
		if readErr == nil && leaseExpired(existing, now) {
			if rmErr := os.Remove(lockPath); rmErr == nil || os.IsNotExist(rmErr) {
				continue
			}
		}

		if readErr != nil {
			existing = ModulesLockLease{
				Owner:        "unknown",
				PID:          0,
				ErrorCode:    LockConflictErrorCodeDefault,
				RetryBackoff: LockRetryBackoffInitialDefault.String(),
			}
		}
		return nil, &LockConflictError{Path: lockPath, Metadata: existing}
	}

	return nil, xfmt.Errorf("acquire workspace modules lock failed")
}

func writeLeaseFileExclusive(path string, lease ModulesLockLease) error {
	payload, err := json.Marshal(lease)
	if err != nil {
		return xfmt.Errorf("encode lease metadata failed: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return os.ErrExist
		}
		return err
	}
	defer f.Close()
	if _, err := f.Write(payload); err != nil {
		return err
	}
	return nil
}

func readLeaseFile(path string) (ModulesLockLease, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ModulesLockLease{}, err
	}
	lease := ModulesLockLease{}
	if err := json.Unmarshal(data, &lease); err != nil {
		return ModulesLockLease{}, err
	}
	return lease, nil
}

func leaseExpired(lease ModulesLockLease, now time.Time) bool {
	ttl := LockTTLDefault
	if parsedTTL, err := time.ParseDuration(strings.TrimSpace(lease.TTL)); err == nil && parsedTTL > 0 {
		ttl = parsedTTL
	}

	heartbeatAt := now
	if ts := strings.TrimSpace(lease.LastHeartbeat); ts != "" {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			heartbeatAt = t
		}
	} else if ts := strings.TrimSpace(lease.StartedAt); ts != "" {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			heartbeatAt = t
		}
	}
	return now.Sub(heartbeatAt) > ttl
}

func (l ModulesLockLease) Label() string {
	parts := []string{l.Owner}
	if l.PID > 0 {
		parts = append(parts, strconv.Itoa(l.PID))
	}
	return strings.Join(parts, ":")
}
