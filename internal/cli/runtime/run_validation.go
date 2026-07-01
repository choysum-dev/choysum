// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
)

const runConfigFixValuesNext = "fix config values and rerun 'choysum run'"

type RunValidationError struct {
	ExitCode int
	ErrMsg   string
	Reason   string
	Next     string
}

func newRunValidationError(errMsg, reason, next string) *RunValidationError {
	return &RunValidationError{ExitCode: 3, ErrMsg: errMsg, Reason: reason, Next: next}
}

func ValidateRunModulesPath(path string) (string, *RunValidationError) {
	if strings.TrimSpace(path) == "" {
		return "", newRunValidationError("invalid config", "missing required fields", runConfigFixValuesNext)
	}
	if strings.TrimSpace(path) != path {
		return "", newRunValidationError("invalid config", "invalid value", runConfigFixValuesNext)
	}
	if ContainsControl(path) {
		return "", newRunValidationError("invalid config", "invalid value", runConfigFixValuesNext)
	}
	if HasPathListSeparator(path) {
		return "", newRunValidationError("invalid config", "invalid value", runConfigFixValuesNext)
	}
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if mkdirErr := os.MkdirAll(path, 0o755); mkdirErr != nil {
				return "", newRunValidationError("invalid config", fmt.Sprintf("path does not exist and cannot be created: %v", mkdirErr), runConfigFixValuesNext)
			}
			info, err = os.Lstat(path)
			if err != nil {
				return "", newRunValidationError("invalid config", "permission denied or not accessible", runConfigFixValuesNext)
			}
		} else {
			return "", newRunValidationError("invalid config", "permission denied or not accessible", runConfigFixValuesNext)
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", newRunValidationError("invalid config", "invalid value", runConfigFixValuesNext)
	}
	if !info.IsDir() {
		return "", newRunValidationError("invalid config", "invalid value", runConfigFixValuesNext)
	}
	if _, err := os.ReadDir(path); err != nil {
		return "", newRunValidationError("invalid config", "permission denied or not accessible", runConfigFixValuesNext)
	}
	return path, nil
}

func ValidateRunDBWithWarning(dbOptions RunDBOptions, warn func(string)) *RunValidationError {
	dialect := strings.ToLower(strings.TrimSpace(dbOptions.Dialect))
	if dialect == "" {
		return newRunValidationError("invalid config", "missing required fields", runConfigFixValuesNext)
	}
	if dialect != "sqlite" && dialect != "postgres" && dialect != "mysql" {
		return newRunValidationError("invalid config", "invalid value", runConfigFixValuesNext)
	}
	if dbOptions.DSN == "" || strings.TrimSpace(dbOptions.DSN) == "" {
		return newRunValidationError("invalid config", "missing required fields", runConfigFixValuesNext)
	}

	if dialect == "sqlite" {
		return ValidateRunSQLite(dbOptions.DSN, dbOptions.AllowCreate, warn)
	}
	return ValidateRunDatabaseDSN(dialect, dbOptions.DSN)
}

func ValidateRunDB(dbOptions RunDBOptions) *RunValidationError {
	return ValidateRunDBWithWarning(dbOptions, nil)
}

func PrepareRunDatabase(dbOptions RunDBOptions) *RunValidationError {
	if strings.ToLower(strings.TrimSpace(dbOptions.Dialect)) != "sqlite" || !dbOptions.AllowCreate {
		return nil
	}
	return PrepareRunSQLite(dbOptions.DSN)
}

func PrepareRunSQLite(dsn string) *RunValidationError {
	path, err := SQLitePathFromDSN(dsn)
	if err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return newRunValidationError("invalid sqlite path", "permission denied or not accessible", RunSQLitePathNext(true))
	}
	return nil
}

func ValidateRunDatabaseDSN(dialect, dsn string) *RunValidationError {
	if ContainsControl(dsn) {
		return newRunValidationError("invalid database dsn", "dsn contains NUL (\\x00) or newline (\\n/\\r)", "remove control characters from dsn and retry; if it comes from config, fix the config and rerun")
	}
	if strings.TrimSpace(dsn) == "" {
		return newRunValidationError("invalid config", "missing required fields", runConfigFixValuesNext)
	}
	if strings.TrimSpace(dsn) != dsn {
		return newRunValidationError("invalid database dsn", "dsn has leading or trailing whitespace", "remove leading/trailing whitespace from dsn and retry; if it comes from config, fix the config and rerun")
	}

	if scheme := URLScheme(dsn); scheme != "" {
		scheme = strings.ToLower(scheme)
		expected := dialect
		if scheme == "postgresql" {
			scheme = "postgres"
		}
		if scheme != expected {
			return newRunValidationError("invalid config", "db.dialect conflicts with dsn scheme", "make db.dialect and dsn scheme consistent and retry")
		}
	}

	return nil
}

func ValidateRunSQLite(dsn string, allowCreate bool, warn func(string)) *RunValidationError {
	if ContainsControl(dsn) {
		return newRunValidationError("invalid sqlite path", "path contains NUL (\\x00) or newline (\\n/\\r)", RunSQLitePathNext(allowCreate))
	}
	if strings.TrimSpace(dsn) == "" {
		return newRunValidationError("invalid sqlite dsn", "sqlite dsn must not be empty or whitespace", "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute")
	}
	if strings.EqualFold(strings.TrimSpace(dsn), ":memory:") {
		return newRunValidationError("invalid sqlite dsn", "sqlite dsn must not be :memory:", "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute")
	}
	if strings.TrimSpace(dsn) != dsn {
		return newRunValidationError("invalid sqlite path", "path has leading or trailing whitespace", "choose a valid sqlite file path and retry; for run, ensure the path is absolute and the DB file already exists and is a regular file")
	}
	path, parseErr := SQLitePathFromDSN(dsn)
	if parseErr != nil {
		return newRunValidationError("invalid sqlite dsn", "sqlite dsn must be a file: URI or plain file path (other URI schemes are not allowed)", "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute")
	}
	if strings.TrimSpace(path) == "" {
		return newRunValidationError("invalid sqlite dsn", "sqlite dsn must not be empty or whitespace", "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute")
	}
	if strings.EqualFold(strings.TrimSpace(path), ":memory:") {
		return newRunValidationError("invalid sqlite dsn", "sqlite dsn must not be :memory:", "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute")
	}
	if strings.TrimSpace(path) != path {
		return newRunValidationError("invalid sqlite path", "path has leading or trailing whitespace", RunSQLitePathNext(allowCreate))
	}
	if !filepath.IsAbs(path) {
		return newRunValidationError("invalid sqlite path", "path is not absolute", RunSQLitePathNext(allowCreate))
	}
	if HasParentSymlink(path) && warn != nil {
		warn("sqlite parent directory is a symlink")
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if allowCreate {
				if pragmaErr := ValidateRunSQLitePragmas(dsn); pragmaErr != nil {
					return pragmaErr
				}
				return nil
			}
			return newRunValidationError("invalid sqlite path", "path does not exist", RunSQLitePathNext(false))
		}
		return newRunValidationError("invalid sqlite path", "permission denied or not accessible", RunSQLitePathNext(allowCreate))
	}
	if info.IsDir() {
		return newRunValidationError("invalid sqlite path", "path is a directory", RunSQLitePathNext(allowCreate))
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return newRunValidationError("invalid sqlite path", "path is a symlink", RunSQLitePathNext(allowCreate))
	}
	if !info.Mode().IsRegular() {
		return newRunValidationError("invalid sqlite path", "path is not a regular file", RunSQLitePathNext(allowCreate))
	}
	if pragmaErr := ValidateRunSQLitePragmas(dsn); pragmaErr != nil {
		return pragmaErr
	}
	return nil
}

func RunSQLitePathNext(allowCreate bool) string {
	if allowCreate {
		return "choose a valid sqlite file path and retry; for run, ensure the path is absolute and accessible"
	}
	return "choose a valid sqlite file path and retry; for run, ensure the path is absolute and the DB file already exists and is a regular file"
}

func ValidateRunSQLitePragmas(dsn string) *RunValidationError {
	params, err := SQLiteDSNQueryParams(dsn)
	if err != nil {
		return newRunValidationError("invalid sqlite dsn", "sqlite dsn query params are invalid", "set sqlite dsn like file:/absolute/path/choysum.sqlite?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL and retry")
	}

	missing := make([]string, 0, 3)
	if strings.TrimSpace(params.Get("_fk")) != "1" {
		missing = append(missing, "_fk=1")
	}

	busyTimeoutRaw := strings.TrimSpace(params.Get("_busy_timeout"))
	busyTimeoutMs, parseErr := strconv.Atoi(busyTimeoutRaw)
	if busyTimeoutRaw == "" || parseErr != nil || busyTimeoutMs <= 0 {
		missing = append(missing, "_busy_timeout>0")
	}

	if !strings.EqualFold(strings.TrimSpace(params.Get("_journal_mode")), "WAL") {
		missing = append(missing, "_journal_mode=WAL")
	}

	if len(missing) == 0 {
		return nil
	}

	return newRunValidationError("invalid sqlite dsn", fmt.Sprintf("sqlite dsn missing required params: %s", strings.Join(missing, ", ")), "set sqlite dsn like file:/absolute/path/choysum.sqlite?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL and retry")
}

func SQLiteDSNQueryParams(dsn string) (url.Values, error) {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(strings.ToLower(trimmed), "file:") || strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return nil, err
		}
		return parsed.Query(), nil
	}
	queryIndex := strings.Index(trimmed, "?")
	if queryIndex < 0 {
		return url.Values{}, nil
	}
	return url.ParseQuery(trimmed[queryIndex+1:])
}

func IsDefaultRunSQLitePath(dsn string, defaultPath string) bool {
	if strings.TrimSpace(dsn) == "" || strings.TrimSpace(defaultPath) == "" {
		return false
	}
	path, err := SQLitePathFromDSN(dsn)
	if err != nil {
		return false
	}
	return filepath.Clean(path) == filepath.Clean(defaultPath)
}

func SQLitePathFromDSN(dsn string) (string, error) {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(strings.ToLower(trimmed), "file:") || strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", err
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "" && scheme != "file" {
			return "", errors.New("unsupported sqlite dsn scheme")
		}
		if parsed.Path != "" {
			return parsed.Path, nil
		}
		return parsed.Opaque, nil
	}
	if queryIndex := strings.Index(trimmed, "?"); queryIndex >= 0 {
		return strings.TrimSpace(trimmed[:queryIndex]), nil
	}
	return trimmed, nil
}

func HasPathListSeparator(path string) bool {
	if goruntime.GOOS == "windows" {
		if strings.Contains(path, ";") {
			return true
		}
		if strings.Contains(path, ":") {
			if len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
				return false
			}
			return true
		}
		return false
	}
	return strings.Contains(path, ":")
}

func ContainsControl(value string) bool {
	return strings.ContainsRune(value, '\x00') || strings.ContainsRune(value, '\n') || strings.ContainsRune(value, '\r')
}

func URLScheme(dsn string) string {
	value := strings.ToLower(dsn)
	if strings.HasPrefix(value, "postgres://") || strings.HasPrefix(value, "postgresql://") || strings.HasPrefix(value, "mysql://") {
		idx := strings.Index(value, "://")
		if idx > 0 {
			return value[:idx]
		}
	}
	return ""
}

func HasParentSymlink(path string) bool {
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}

	current := filepath.Dir(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}
