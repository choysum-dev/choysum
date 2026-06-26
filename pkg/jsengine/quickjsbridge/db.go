// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/shopspring/decimal"
)

func normalizeDBParams(params []interface{}) ([]interface{}, error) {
	for i, value := range params {
		objectValue, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if stringValue, ok := objectValue["$bigint"].(string); ok {
			bigIntValue, err := strconv.ParseInt(stringValue, 10, 64)
			if err != nil {
				return nil, err
			}
			params[i] = bigIntValue
			continue
		}
		if stringValue, ok := objectValue["$bigdecimal"].(string); ok {
			decimalValue, err := decimal.NewFromString(stringValue)
			if err != nil {
				return nil, err
			}
			params[i] = decimalValue
			continue
		}

		if stringValue, ok := objectValue["$bytesBase64"].(string); ok {
			decoded, err := decodeBase64DBParam(stringValue)
			if err != nil {
				return nil, err
			}
			params[i] = decoded
			continue
		}

		serialized, err := json.Marshal(objectValue)
		if err != nil {
			return nil, err
		}
		params[i] = string(serialized)
	}

	return params, nil
}

func decodeBase64DBParam(input string) ([]byte, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return []byte{}, nil
	}

	compact := strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t', ' ':
			return -1
		default:
			return r
		}
	}, trimmed)
	if compact == "" {
		return []byte{}, nil
	}

	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(compact)
		if err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("invalid $bytesBase64 parameter")
}

func getExecSession(ctx *quickjs.Context, engine *quickjsengine.QuickjsEngine, logger *slog.Logger) (*scope.Session, *quickjs.Value) {
	execCtx := engine.ExecContext()
	session, ok := scope.SessionFromContext(execCtx)
	if !ok {
		logger.Error("missing db session in exec context")
		return nil, ctx.ThrowError(fmt.Errorf("missing db session in exec context"))
	}
	return session, nil
}

func performQuery(ctx *quickjs.Context, engine *quickjsengine.QuickjsEngine, args []*quickjs.Value, logger *slog.Logger) *quickjs.Value {
	if len(args) < 2 {
		return ctx.ThrowError(fmt.Errorf("need 2 args: sql, parametersJSON"))
	}
	session, errValue := getExecSession(ctx, engine, logger)
	if errValue != nil {
		return errValue
	}
	dialect := strings.ToLower(strings.TrimSpace(session.Dialector.Name()))

	sql := args[0].String()
	paramsJSON := args[1].String()

	var params []interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		logger.Error("db params unmarshal failed", "error", err)
		return ctx.ThrowError(err)
	}
	params, err := normalizeDBParams(params)
	if err != nil {
		logger.Error("db params normalize failed", "error", err)
		return ctx.ThrowError(err)
	}

	var results []map[string]interface{}
	maxDeadlockRetries := maxDeadlockRetriesForDialect(dialect)
	for attempt := 0; attempt < maxDeadlockRetries; attempt++ {
		err := session.Raw(sql, params...).Scan(&results).Error
		if err == nil {
			break
		}
		if !isDeadlockErr(err, dialect) || attempt == maxDeadlockRetries-1 {
			logger.Error("db query failed", "error", err)
			return ctx.ThrowError(err)
		}
		sleep := deadlockRetrySleep(dialect, attempt)
		logger.Warn("db query deadlock retry", "error", err, "attempt", attempt+1, "sleep", sleep)
		if waitErr := waitForDeadlockRetry(session, sleep); waitErr != nil {
			logger.Warn("db query deadlock retry canceled", "error", waitErr, "attempt", attempt+1)
			return ctx.ThrowError(waitErr)
		}
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		logger.Error("db query result marshal failed", "error", err)
		return ctx.ThrowError(err)
	}

	return ctx.String(string(jsonData))
}

func queryAsyncFactory(engine *quickjsengine.QuickjsEngine, logger *slog.Logger) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performQuery(ctx, engine, args, logger)
			if ret.IsError() {
				reject(ret)
			} else {
				resolve(ret)
			}
		})
	}
}

func performExecute(ctx *quickjs.Context, engine *quickjsengine.QuickjsEngine, args []*quickjs.Value, logger *slog.Logger) *quickjs.Value {
	if len(args) < 2 {
		return ctx.ThrowError(fmt.Errorf("need 2 args: sql, parametersJSON"))
	}
	session, errValue := getExecSession(ctx, engine, logger)
	if errValue != nil {
		return errValue
	}
	dialect := strings.ToLower(strings.TrimSpace(session.Dialector.Name()))

	sql := args[0].String()
	paramsJSON := args[1].String()
	var params []interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		logger.Error("db params unmarshal failed", "error", err)
		return ctx.ThrowError(err)
	}
	params, err := normalizeDBParams(params)
	if err != nil {
		logger.Error("db params normalize failed", "error", err)
		return ctx.ThrowError(err)
	}

	var rowsAffected int64
	maxDeadlockRetries := maxDeadlockRetriesForDialect(dialect)
	for attempt := 0; attempt < maxDeadlockRetries; attempt++ {
		tx := session.Exec(sql, params...)
		if tx.Error == nil {
			rowsAffected = tx.RowsAffected
			break
		}
		if !isDeadlockErr(tx.Error, dialect) || attempt == maxDeadlockRetries-1 {
			logger.Error("db execute failed", "error", tx.Error)
			return ctx.ThrowError(tx.Error)
		}
		sleep := deadlockRetrySleep(dialect, attempt)
		logger.Warn("db execute deadlock retry", "error", tx.Error, "attempt", attempt+1, "sleep", sleep)
		if waitErr := waitForDeadlockRetry(session, sleep); waitErr != nil {
			logger.Warn("db execute deadlock retry canceled", "error", waitErr, "attempt", attempt+1)
			return ctx.ThrowError(waitErr)
		}
	}

	jsonData, err := json.Marshal(map[string]interface{}{"LastInsertId": nil, "RowsAffected": rowsAffected})
	if err != nil {
		logger.Error("db execute result marshal failed", "error", err)
		return ctx.ThrowError(err)
	}

	return ctx.String(string(jsonData))
}

func executeAsyncFactory(engine *quickjsengine.QuickjsEngine, logger *slog.Logger) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performExecute(ctx, engine, args, logger)
			if ret.IsError() {
				reject(ret)
			} else {
				resolve(ret)
			}
		})
	}
}

func isDeadlockErr(err error, dialect string) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "deadlock detected") {
		return true
	}
	if strings.Contains(message, "sqlstate 40p01") {
		return true
	}
	switch dialect {
	case "postgres", "postgresql":
		return strings.Contains(message, "sqlstate 40p01")
	case "mysql", "mariadb":
		return strings.Contains(message, "errno 1213") ||
			strings.Contains(message, "error 1213") ||
			strings.Contains(message, "1213") && strings.Contains(message, "deadlock") ||
			strings.Contains(message, "sqlstate 40001")
	case "sqlite", "sqlite3":
		return strings.Contains(message, "database is locked") ||
			strings.Contains(message, "database is busy") ||
			strings.Contains(message, "locking protocol")
	default:
		return false
	}
}

func maxDeadlockRetriesForDialect(dialect string) int {
	if dialect == "sqlite" || dialect == "sqlite3" {
		return 8
	}
	return 3
}

func deadlockRetrySleep(dialect string, attempt int) time.Duration {
	base := 80 * time.Millisecond
	if dialect == "sqlite" || dialect == "sqlite3" {
		base = 150 * time.Millisecond
	}
	return time.Duration(attempt+1) * base
}

func waitForDeadlockRetry(session *scope.Session, sleep time.Duration) error {
	if sleep <= 0 {
		return nil
	}
	if session == nil || session.Statement == nil || session.Statement.Context == nil {
		time.Sleep(sleep)
		return nil
	}

	timer := time.NewTimer(sleep)
	defer timer.Stop()

	select {
	case <-session.Statement.Context.Done():
		return session.Statement.Context.Err()
	case <-timer.C:
		return nil
	}
}

func performSavepoint(ctx *quickjs.Context, engine *quickjsengine.QuickjsEngine, args []*quickjs.Value, logger *slog.Logger) *quickjs.Value {
	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("savepoint requires a name"))
	}
	session, errValue := getExecSession(ctx, engine, logger)
	if errValue != nil {
		return errValue
	}

	savepointName := args[0].String()
	if err := session.Savepoint(savepointName); err != nil {
		logger.Error("savepoint creation failed", "name", savepointName, "error", err)
		return ctx.ThrowError(err)
	}

	return ctx.String(savepointName)
}

func savepointAsyncFactoryWithEngine(engine *quickjsengine.QuickjsEngine, logger *slog.Logger) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performSavepoint(ctx, engine, args, logger)
			if ret.IsError() {
				reject(ret)
			} else {
				resolve(ret)
			}
		})
	}
}

func performRollbackToSavepoint(ctx *quickjs.Context, engine *quickjsengine.QuickjsEngine, args []*quickjs.Value, logger *slog.Logger) *quickjs.Value {
	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("rollbackToSavepoint requires a name"))
	}
	session, errValue := getExecSession(ctx, engine, logger)
	if errValue != nil {
		return errValue
	}

	savepointName := args[0].String()
	if err := session.RollbackToSavepoint(savepointName); err != nil {
		logger.Error("savepoint rollback failed", "name", savepointName, "error", err)
		return ctx.ThrowError(err)
	}

	return ctx.Null()
}

func rollbackToSavepointAsyncFactoryWithEngine(engine *quickjsengine.QuickjsEngine, logger *slog.Logger) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performRollbackToSavepoint(ctx, engine, args, logger)
			if ret.IsError() {
				reject(ret)
			} else {
				resolve(ret)
			}
		})
	}
}

func performReleaseSavepoint(ctx *quickjs.Context, engine *quickjsengine.QuickjsEngine, args []*quickjs.Value, logger *slog.Logger) *quickjs.Value {
	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("releaseSavepoint requires a name"))
	}
	session, errValue := getExecSession(ctx, engine, logger)
	if errValue != nil {
		return errValue
	}

	savepointName := args[0].String()
	if err := session.ReleaseSavepoint(savepointName); err != nil {
		logger.Error("savepoint release failed", "name", savepointName, "dialect", session.Dialector.Name(), "error", err)
		return ctx.ThrowError(err)
	}

	return ctx.Null()
}

func releaseSavepointAsyncFactoryWithEngine(engine *quickjsengine.QuickjsEngine, logger *slog.Logger) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performReleaseSavepoint(ctx, engine, args, logger)
			if ret.IsError() {
				reject(ret)
			} else {
				resolve(ret)
			}
		})
	}
}

func WithDb(dialect string, logger *slog.Logger) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		engine := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := engine.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = engine.Ctx.Object()
		}

		dbObj := engine.Ctx.Object()
		dbObj.Set("query", engine.Ctx.NewFunction(queryAsyncFactory(engine, logger)))
		dbObj.Set("execute", engine.Ctx.NewFunction(executeAsyncFactory(engine, logger)))
		dbObj.Set("savepoint", engine.Ctx.NewFunction(savepointAsyncFactoryWithEngine(engine, logger)))
		dbObj.Set("rollbackToSavepoint", engine.Ctx.NewFunction(rollbackToSavepointAsyncFactoryWithEngine(engine, logger)))
		dbObj.Set("releaseSavepoint", engine.Ctx.NewFunction(releaseSavepointAsyncFactoryWithEngine(engine, logger)))

		dbObj.Set("dialectName", engine.Ctx.String(dialect))

		choysumObj.Set("db", dbObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}
