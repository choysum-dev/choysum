// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	leasemodel "github.com/choysum-dev/choysum/internal/state/lease/model"

	"github.com/choysum-dev/choysum/internal/logger"
	modulestaging "github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	origincontract "github.com/choysum-dev/choysum/internal/module/origin/contract"
	"github.com/choysum-dev/choysum/internal/state/lease"
	configpkg "github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
)

const (
	bootstrapInitLeaseResource     = "system:init"
	bootstrapInitLeaseTTL          = 60 * time.Second
	bootstrapInitLeaseRenew        = 20 * time.Second
	bootstrapDefaultPollAfterMs    = int64(1000)
	bootstrapDefaultRedirectTo     = "/web/login"
	bootstrapClientHashingPrefix   = "$CH$"
	bootstrapAdminUsernameMaxBytes = 256
	bootstrapPasswordMaxBytes      = 4096
	bootstrapModuleInstallTimeout  = time.Duration(configpkg.DefaultBootstrapModuleInstallTimeoutSeconds) * time.Second
)

var (
	errBootstrapAdminExternalIDNotFound = errors.New("bootstrap admin external id not found")
	errBootstrapAdminModelNotFound      = errors.New("bootstrap admin model not found")
	errBootstrapAdminModelTableMissing  = errors.New("bootstrap admin model table missing")
	errBootstrapAdminRecordNotFound     = errors.New("bootstrap admin record not found")
)

type initializeInput struct {
	AdminUsername  string
	Password       string
	IdempotencyKey string
}

type leaseHandle struct {
	ownerID  string
	stopCh   chan struct{}
	renewErr chan error
}

type coordinator struct {
	runtimeScope scope.Scope

	store bootstrapStatusStore
	now   func() time.Time

	newOperationID  func() string
	runAsync        func(fn func())
	nextPollAfterMs int64
	lockerFactory   statepkg.LockerFactory

	acquireInitLease        func(ctx context.Context) (*leaseHandle, error)
	releaseInitLease        func(handle *leaseHandle)
	checkWorkspaceFreshness func(ctx context.Context) error
	installMinimalModules   func(ctx context.Context, operationID string) error
	updateAdminAndMarker    func(ctx context.Context, input initializeInput) error
	validateRuntimeReady    func(ctx context.Context) error
	switchMode              func(ctx context.Context) error
}

func newCoordinator(runtimeScope scope.Scope) *coordinator {
	c := &coordinator{
		runtimeScope:   runtimeScope,
		store:          newInMemoryStatusStore(time.Now),
		now:            time.Now,
		newOperationID: func() string { return xid.New().String() },
		runAsync: func(fn func()) {
			go fn()
		},
		nextPollAfterMs: bootstrapDefaultPollAfterMs,
		lockerFactory: func(runtimeScope scope.Scope) statepkg.Locker {
			return lease.New(runtimeScope)
		},
	}

	c.acquireInitLease = c.defaultAcquireInitLease
	c.releaseInitLease = c.defaultReleaseInitLease
	c.checkWorkspaceFreshness = c.defaultCheckWorkspaceFreshness
	c.installMinimalModules = c.defaultInstallMinimalModules
	c.updateAdminAndMarker = c.defaultUpdateAdminAndMarker
	c.validateRuntimeReady = c.defaultValidateRuntimeReady
	c.switchMode = c.defaultSwitchMode

	return c
}

func (c *coordinator) startInitialization(ctx context.Context, input initializeInput) (operationSnapshot, bool, error) {
	if err := validateInitializeInput(input); err != nil {
		return operationSnapshot{}, false, err
	}

	opID := c.newOperationID()
	op, outcome, conflictingID := c.store.beginOperation(opID, input.IdempotencyKey, c.nextPollAfterMs)
	switch outcome {
	case beginReused:
		return op, true, nil
	case beginConflict:
		msg := "another bootstrap initialization is already running"
		if conflictingID != "" {
			msg = "another bootstrap initialization is already running: " + conflictingID
		}
		return op, false, newBootstrapError(bootstrapErrCodeConflict, msg, nil)
	case beginCreated:
		if c.runtimeScope != nil && c.runtimeScope.Logger() != nil {
			c.runtimeScope.Logger().Info("bootstrap initialization started")
		}
		if ctx == nil {
			ctx = context.Background()
		}
		runCtx := context.WithoutCancel(ctx)
		c.runAsync(func() {
			c.executeInitialization(runCtx, op.OperationID, input)
		})
		return op, false, nil
	default:
		return operationSnapshot{}, false, newBootstrapError(bootstrapErrCodeGateError, "failed to start initial setup", nil)
	}
}

func (c *coordinator) getInitializationStatus(operationID string) (operationSnapshot, bool) {
	return c.store.getOperation(operationID)
}

func (c *coordinator) executeInitialization(ctx context.Context, operationID string, input initializeInput) {
	c.store.markRunning(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_ACQUIRE_LOCK, 10)

	handle, err := c.acquireInitLease(ctx)
	if err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_ACQUIRE_LOCK, err, bootstrapErrCodeConflict, "failed to start setup safely")
		return
	}
	defer c.releaseInitLease(handle)

	if err := c.checkRenewErr(handle); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_ACQUIRE_LOCK, err, bootstrapErrCodeGateError, "setup could not keep its safety lock")
		return
	}

	c.store.markStage(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_CHECK_WORKSPACE_FRESHNESS, 25)
	checkWorkspaceFreshness := c.checkWorkspaceFreshness
	if checkWorkspaceFreshness == nil {
		checkWorkspaceFreshness = func(context.Context) error { return nil }
	}
	if err := checkWorkspaceFreshness(ctx); err != nil {
		if bootstrapErrorCode(err) == bootstrapErrCodeWorkspaceNotFresh && c.runtimeScope != nil && c.runtimeScope.Logger() != nil {
			c.runtimeScope.Logger().Warn(
				"bootstrap initialization rejected",
				"operation_id", operationID,
				"error_code", bootstrapErrCodeWorkspaceNotFresh,
				"error", err.Error(),
			)
		}
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_CHECK_WORKSPACE_FRESHNESS, err, bootstrapErrCodeGateError, "failed to verify whether setup can continue")
		return
	}

	if err := c.checkRenewErr(handle); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_CHECK_WORKSPACE_FRESHNESS, err, bootstrapErrCodeGateError, "setup could not keep its safety lock")
		return
	}

	c.store.markStage(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_ENSURE_MINIMAL_RUNTIME, 35)
	installCtx := modulestaging.WithOpID(ctx, operationID)
	if err := c.installMinimalModules(installCtx, operationID); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_ENSURE_MINIMAL_RUNTIME, err, bootstrapErrCodeRuntimePrepare, "failed to prepare required system components")
		return
	}

	if err := c.checkRenewErr(handle); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_ENSURE_MINIMAL_RUNTIME, err, bootstrapErrCodeGateError, "setup could not keep its safety lock")
		return
	}

	c.store.markStage(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_VALIDATE_RUNTIME_READY, 65)
	if err := c.validateRuntimeReady(ctx); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_VALIDATE_RUNTIME_READY, err, bootstrapErrCodeRuntimeNotReady, "required system files are not ready")
		return
	}

	if err := c.checkRenewErr(handle); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_VALIDATE_RUNTIME_READY, err, bootstrapErrCodeGateError, "setup could not keep its safety lock")
		return
	}

	c.store.markStage(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_UPDATE_ADMIN, 80)
	if err := c.updateAdminAndMarker(ctx, input); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_UPDATE_ADMIN, err, bootstrapErrCodeAdminUpdateFailed, "failed to save administrator setup")
		return
	}

	if err := c.checkRenewErr(handle); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_UPDATE_ADMIN, err, bootstrapErrCodeGateError, "setup could not keep its safety lock")
		return
	}

	c.store.markStage(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_SWITCH_MODE, 90)
	if err := c.switchMode(ctx); err != nil {
		c.failOperation(operationID, bootstrappb.InitializationStage_INITIALIZATION_STAGE_SWITCH_MODE, err, bootstrapErrCodeSwitchFailed, "failed to finish setup")
		return
	}

	c.store.markSucceeded(operationID, bootstrapDefaultRedirectTo)
	if c.runtimeScope != nil && c.runtimeScope.Logger() != nil {
		op, ok := c.store.getOperation(operationID)
		if ok {
			c.runtimeScope.Logger().Info(
				"bootstrap initialization completed",
				"duration_ms", c.operationDurationMs(op.CreatedAt),
			)
		}
	}
}

func (c *coordinator) failOperation(operationID string, stage bootstrappb.InitializationStage, err error, fallbackCode, fallbackMessage string) {
	be := wrapBootstrapError(err, fallbackCode, fallbackMessage)
	c.store.markFailed(operationID, stage, be.code, be.Error())
	if c.runtimeScope != nil && c.runtimeScope.Logger() != nil {
		state := bootstrappb.InitializationState_INITIALIZATION_STATE_FAILED.String()
		stageName := stage.String()
		durationMs := int64(0)
		if op, ok := c.store.getOperation(operationID); ok {
			state = op.State.String()
			stageName = op.Stage.String()
			durationMs = c.operationDurationMs(op.CreatedAt)
		}
		c.runtimeScope.Logger().Error(
			"bootstrap initialization failed",
			"operation_id", operationID,
			"state", state,
			"stage", stageName,
			"duration_ms", durationMs,
			"error_code", be.code,
			"error", be.Error(),
		)
	}
}

func (c *coordinator) operationDurationMs(createdAt time.Time) int64 {
	if createdAt.IsZero() {
		return 0
	}
	d := c.now().UTC().Sub(createdAt.UTC())
	if d < 0 {
		return 0
	}
	return d.Milliseconds()
}

func wrapBootstrapError(err error, fallbackCode, fallbackMessage string) *bootstrapError {
	if err == nil {
		return newBootstrapError(fallbackCode, fallbackMessage, nil)
	}
	var be *bootstrapError
	if errors.As(err, &be) {
		return be
	}
	return newBootstrapError(fallbackCode, fallbackMessage+": "+err.Error(), err)
}

func validateInitializeInput(input initializeInput) error {
	username := input.AdminUsername
	if strings.TrimSpace(username) == "" {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "admin_username is required", nil)
	}
	if strings.TrimSpace(username) != username {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "admin_username must not have leading/trailing whitespace", nil)
	}
	if containsControlChars(username) {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "admin_username contains control characters", nil)
	}
	if len([]byte(username)) > bootstrapAdminUsernameMaxBytes {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "admin_username is too long", nil)
	}

	password := input.Password
	if len([]byte(password)) == 0 {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "password is required", nil)
	}
	if strings.ContainsAny(password, "\n\r") {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "password must be a single line", nil)
	}
	if len([]byte(password)) > bootstrapPasswordMaxBytes {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "password is too long", nil)
	}

	if containsControlChars(input.IdempotencyKey) {
		return newBootstrapError(bootstrapErrCodeInputInvalid, "idempotency_key contains control characters", nil)
	}

	return nil
}

func containsControlChars(v string) bool {
	for _, r := range v {
		if r == 0 || r == '\n' || r == '\r' {
			return true
		}
	}
	return false
}

func (c *coordinator) defaultAcquireInitLease(ctx context.Context) (*leaseHandle, error) {
	if c.runtimeScope == nil {
		return nil, newBootstrapError(bootstrapErrCodeGateError, "scope is not available", nil)
	}

	session := c.runtimeScope.Session()
	if session == nil || session.DB == nil {
		return nil, newBootstrapError(bootstrapErrCodeGateError, "database session is not available", nil)
	}

	if !session.Migrator().HasTable((&leasemodel.IrLockLease{}).TableName()) {
		if err := session.AutoMigrate(&leasemodel.IrLockLease{}); err != nil {
			return nil, newBootstrapError(bootstrapErrCodeGateError, "failed to initialize the setup lock", err)
		}
	}

	locker := c.lockerFactory(c.runtimeScope)
	ownerID := xid.New().String()
	if err := locker.Acquire(ctx, bootstrapInitLeaseResource, ownerID, bootstrapInitLeaseTTL); err != nil {
		if errors.Is(err, lease.ErrLeaseBusy) {
			return nil, newBootstrapError(bootstrapErrCodeConflict, "another setup request is already running", err)
		}
		return nil, newBootstrapError(bootstrapErrCodeGateError, "failed to acquire the setup lock", err)
	}

	handle := &leaseHandle{
		ownerID:  ownerID,
		stopCh:   make(chan struct{}),
		renewErr: make(chan error, 1),
	}

	go c.renewLease(ctx, locker, handle)

	return handle, nil
}

func (c *coordinator) renewLease(ctx context.Context, locker statepkg.Locker, handle *leaseHandle) {
	ticker := time.NewTicker(bootstrapInitLeaseRenew)
	defer ticker.Stop()

	for {
		select {
		case <-handle.stopCh:
			return
		case <-ticker.C:
			if err := locker.Renew(ctx, bootstrapInitLeaseResource, handle.ownerID, bootstrapInitLeaseTTL); err != nil {
				if errors.Is(err, lease.ErrLeaseNotOwner) {
					select {
					case handle.renewErr <- err:
					default:
					}
					return
				}
				if isRecoverableSqliteLeaseErr(c.runtimeScope, err) {
					continue
				}
				select {
				case handle.renewErr <- err:
				default:
				}
				return
			}
		}
	}
}

func isRecoverableSqliteLeaseErr(runtimeScope scope.Scope, err error) bool {
	if err == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).dbDialect), "sqlite") {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database schema is locked")
}

func (c *coordinator) defaultReleaseInitLease(handle *leaseHandle) {
	if handle == nil {
		return
	}
	close(handle.stopCh)

	if c.runtimeScope == nil {
		return
	}
	locker := c.lockerFactory(c.runtimeScope)
	releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = locker.Release(releaseCtx, bootstrapInitLeaseResource, handle.ownerID)
}

func (c *coordinator) checkRenewErr(handle *leaseHandle) error {
	if handle == nil {
		return nil
	}
	select {
	case err := <-handle.renewErr:
		if err == nil {
			return nil
		}
		return newBootstrapError(bootstrapErrCodeGateError, "the setup lock could not be renewed", err)
	default:
		return nil
	}
}

// isNetworkError reports whether err is caused by a network-level failure
// (DNS, timeout, connection refused, TLS handshake) as opposed to a
// business-logic error such as an invalid module version.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, keyword := range []string{
		"connection refused",
		"no such host",
		"network is unreachable",
		"tls handshake",
		"context deadline exceeded",
		"timeout",
		"connection reset",
	} {
		if strings.Contains(msg, keyword) {
			return true
		}
	}
	return false
}

func (c *coordinator) moduleInstallTimeout() time.Duration {
	runtimeTimeoutSeconds := runtimeOptionsFromScope(c.runtimeScope).bootstrapModuleInstallTimeoutSeconds
	if runtimeTimeoutSeconds <= 0 {
		return bootstrapModuleInstallTimeout
	}
	return time.Duration(runtimeTimeoutSeconds) * time.Second
}

func (c *coordinator) defaultInstallMinimalModules(ctx context.Context, operationID string) error {
	if c.runtimeScope == nil {
		return newBootstrapError(bootstrapErrCodeRuntimePrepare, "scope is not available", nil)
	}

	if err := c.rejectAlreadyInstalledAuthModule(ctx); err != nil {
		return err
	}

	// Add a timeout to prevent the bootstrap from hanging indefinitely during
	// module download and installation (especially over slow networks).
	installTimeout := c.moduleInstallTimeout()
	installCtx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	progress := logger.NewProgressLine(os.Stderr)
	installCtx = logger.WithProgressLine(installCtx, progress)
	spinnerTicker := logger.NewProgressTicker(progress, logger.ProgressTickerOptions{Interval: 120 * time.Millisecond})
	defer spinnerTicker.Clear()
	defer spinnerTicker.Stop()
	installCtx = logger.WithProgressTicker(installCtx, spinnerTicker)

	updateFetchProgressMessage := func(message string) {
		spinnerTicker.SetMessage(message)
	}

	installCtx = origincontract.WithFetchProgressReporter(installCtx, func(stage origincontract.FetchProgressStage, moduleName string) {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" {
			moduleName = "core module"
		}
		switch stage {
		case origincontract.FetchProgressStageDownload:
			c.store.markStageDetail(operationID, "downloading module package: "+moduleName+"...")
			updateFetchProgressMessage(fmt.Sprintf("%s: downloading from registry...", moduleName))
		case origincontract.FetchProgressStageVerify:
			c.store.markStageDetail(operationID, "verifying module package integrity: "+moduleName+"...")
			updateFetchProgressMessage(fmt.Sprintf("%s: verifying package...", moduleName))
		case origincontract.FetchProgressStageExtract:
			c.store.markStageDetail(operationID, "extracting module package: "+moduleName+"...")
			updateFetchProgressMessage(fmt.Sprintf("%s: extracting package...", moduleName))
		default:
			// Keep existing stage detail if unknown progress stage is received.
		}
	})

	installScope := c.runtimeScope.WithContext(installCtx)
	if installScope == nil {
		installScope = c.runtimeScope
	}
	executor, err := jsexecutor.NewCompilerExecutor(installScope)
	if err != nil {
		return newBootstrapError(bootstrapErrCodeRuntimePrepare, "failed to prepare the module installer", err)
	}
	if err := executor.Start(); err != nil {
		return newBootstrapError(bootstrapErrCodeRuntimePrepare, "failed to start the module installer", err)
	}
	defer executor.Stop()

	c.store.markStageDetail(operationID, "resolving core module installation plan...")
	spinnerTicker.SetMessage("document: preparing metadata tables")

	moduleLifecycle := lifecycle.NewService(installScope, executor)
	if err := moduleLifecycle.Install(installCtx, lifecycle.InstallRequest{Name: "document", WithDemo: false}); err != nil {
		if progress != nil {
			progress.Done("✗", "core module installation failed")
		}
		// Classify the error to produce an actionable message.
		if errors.Is(err, context.DeadlineExceeded) {
			return newBootstrapError(
				bootstrapErrCodeModuleInstallTimeout,
				"module installation timed out after "+installTimeout.String()+". "+
					"Check your network connection or place the required modules (document and its dependencies) in ModulesPath.",
				err,
			)
		}
		if isNetworkError(err) {
			return newBootstrapError(
				bootstrapErrCodeRuntimePrepare,
				"unable to download required modules. "+
					"Check your network connection or place the required module sources in ModulesPath.",
				err,
			)
		}
		return newBootstrapError(bootstrapErrCodeRuntimePrepare, "failed to install required system components", err)
	}

	c.store.markStageDetail(operationID, "core module installation completed")

	return nil
}

func (c *coordinator) rejectAlreadyInstalledAuthModule(ctx context.Context) error {
	if c.runtimeScope == nil {
		return newBootstrapError(bootstrapErrCodeRuntimePrepare, "scope is not available", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	txRoot := c.runtimeScope.WithContext(ctx)
	return txRoot.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		session := txScope.Session()
		if session == nil || session.DB == nil {
			return newBootstrapError(bootstrapErrCodeRuntimePrepare, "database session is not available", nil)
		}

		if !session.Migrator().HasTable((&meta.IrModule{}).TableName()) {
			return nil
		}

		var module meta.IrModule
		err := session.Select("id").Where("name = ?", "auth").Take(&module).Error
		if err == nil {
			return newBootstrapError(bootstrapErrCodeWorkspaceNotFresh, "initial setup has already been completed: auth is already installed", nil)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return newBootstrapError(bootstrapErrCodeRuntimePrepare, "failed to verify required system components", err)
	})
}

func (c *coordinator) defaultUpdateAdminAndMarker(ctx context.Context, input initializeInput) error {
	if c.runtimeScope == nil {
		return newBootstrapError(bootstrapErrCodeAdminUpdateFailed, "scope is not available", nil)
	}

	passwordHash, err := hashAdminPasswordForBootstrap(input.Password)
	if err != nil {
		return err
	}

	now := c.now().UTC()
	txRoot := c.runtimeScope.WithContext(ctx)
	err = txRoot.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		var modelData metadata.IrModelData
		if err := txScope.Session().Where("module = ? AND external_id = ?", "auth", "user_admin").Take(&modelData).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errBootstrapAdminExternalIDNotFound
			}
			return err
		}

		var model meta.IrModel
		if err := txScope.Session().Where("application = ? AND name = ?", "auth", "User").Take(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errBootstrapAdminModelNotFound
			}
			return err
		}

		if strings.TrimSpace(model.ModelTable) == "" {
			return errBootstrapAdminModelTableMissing
		}

		result := txScope.Session().Table(model.ModelTable).Where("id = ?", modelData.ResID).Updates(map[string]any{
			"username":      input.AdminUsername,
			"password_hash": passwordHash,
			"updated_at":    now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errBootstrapAdminRecordNotFound
		}

		if err := upsertBootstrapSetting(txScope.Session(), "system.init.done", "true"); err != nil {
			return err
		}
		if err := upsertBootstrapSetting(txScope.Session(), "system.init.at", now.Format(time.RFC3339)); err != nil {
			return err
		}

		return nil
	})
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, errBootstrapAdminExternalIDNotFound):
		return newBootstrapError(bootstrapErrCodeAdminUpdateFailed, "administrator account reference was not found", err)
	case errors.Is(err, errBootstrapAdminModelNotFound), errors.Is(err, errBootstrapAdminModelTableMissing):
		return newBootstrapError(bootstrapErrCodeAdminUpdateFailed, "administrator account schema is not available", err)
	case errors.Is(err, errBootstrapAdminRecordNotFound):
		return newBootstrapError(bootstrapErrCodeAdminUpdateFailed, "admin record was not found", err)
	default:
		return newBootstrapError(bootstrapErrCodeAdminUpdateFailed, "failed to save administrator setup", err)
	}
}

func hashAdminPasswordForBootstrap(password string) (string, error) {
	clientHashHex, err := normalizeWirePassword(password)
	if err != nil {
		return "", err
	}

	bcrypted, err := bcrypt.GenerateFromPassword([]byte(clientHashHex), bcrypt.DefaultCost)
	if err != nil {
		return "", newBootstrapError(bootstrapErrCodeAdminUpdateFailed, "failed to hash admin password", err)
	}
	return string(bcrypted), nil
}

func normalizeWirePassword(password string) (string, error) {
	if !strings.HasPrefix(password, bootstrapClientHashingPrefix) {
		return "", newBootstrapError(bootstrapErrCodeInputInvalid, "password must be client-hashed", nil)
	}

	rawHex := strings.TrimPrefix(password, bootstrapClientHashingPrefix)
	if len(rawHex) != 64 {
		return "", newBootstrapError(bootstrapErrCodeInputInvalid, "invalid client hash length", nil)
	}

	for _, ch := range rawHex {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return "", newBootstrapError(bootstrapErrCodeInputInvalid, "invalid client hash encoding", nil)
		}
	}

	return strings.ToLower(rawHex), nil
}

func upsertBootstrapSetting(session *scope.Session, key, value string) error {
	if session == nil || session.DB == nil {
		return errors.New("database session is not available")
	}

	var setting metadata.IrSetting
	err := session.Unscoped().Where("key = ?", key).Take(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return session.Create(&metadata.IrSetting{Key: key, Value: value}).Error
		}
		return err
	}

	return session.Unscoped().Model(&setting).Updates(map[string]any{
		"value":      value,
		"deleted_at": nil,
	}).Error
}

func (c *coordinator) defaultSwitchMode(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return newBootstrapError(bootstrapErrCodeSwitchFailed, "switch mode context cancelled", err)
	}
	// Phase C keeps runtime switch as an async no-op marker.
	return nil
}
