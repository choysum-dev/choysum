// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"context"
	"encoding/json"
	"errors"
	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	"github.com/rs/xid"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

type ApplyOptions struct {
	WithDemo bool
}

type Loader struct {
	runtimeScope scope.Scope
}

func New(runtimeScope scope.Scope) *Loader {
	return &Loader{runtimeScope: runtimeScope}
}

type moduleInfo struct {
	ID          string
	Application string
}

type moduleRules struct {
	OwnerName  string
	OwnerApp   string
	ModuleInfo map[string]moduleInfo
	Allowed    map[string]struct{}
}

func buildModuleRules(tx *gorm.DB, owner *meta.IrModule) (*moduleRules, error) {
	if owner == nil {
		return nil, xfmt.Errorf("nil owner module")
	}
	ownerName := strings.TrimSpace(owner.Name)
	if ownerName == "" {
		return nil, xfmt.Errorf("owner module has empty name")
	}
	modules, idToName, err := loadModuleIndex(tx)
	if err != nil {
		return nil, err
	}
	info, ok := modules[ownerName]
	if !ok {
		return buildModuleRulesFromOwner(owner)
	}
	ownerApp := strings.TrimSpace(owner.ApplicationStr)
	if ownerApp == "" {
		ownerApp = strings.TrimSpace(info.Application)
	}
	if ownerApp == "" {
		return nil, xfmt.Errorf("owner module %s has empty application", ownerName)
	}
	allowed, err := dependencyClosure(tx, info.ID, idToName)
	if err != nil {
		return nil, err
	}
	if allowed == nil {
		allowed = map[string]struct{}{}
	}
	allowed[ownerName] = struct{}{}
	return &moduleRules{OwnerName: ownerName, OwnerApp: ownerApp, ModuleInfo: modules, Allowed: allowed}, nil
}

func buildModuleRulesFromOwner(owner *meta.IrModule) (*moduleRules, error) {
	if owner == nil {
		return nil, xfmt.Errorf("nil owner module")
	}
	ownerName := strings.TrimSpace(owner.Name)
	if ownerName == "" {
		return nil, xfmt.Errorf("owner module has empty name")
	}
	ownerApp := strings.TrimSpace(owner.ApplicationStr)
	if ownerApp == "" {
		return nil, xfmt.Errorf("owner module %s has empty application", ownerName)
	}
	modules := map[string]moduleInfo{
		ownerName: {Application: ownerApp},
	}
	allowed := map[string]struct{}{ownerName: {}}
	queue := make([]*meta.IrModule, 0, len(owner.Dependencies))
	queue = append(queue, owner.Dependencies...)
	seen := map[string]struct{}{ownerName: {}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == nil {
			continue
		}
		name := strings.TrimSpace(cur.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		app := strings.TrimSpace(cur.ApplicationStr)
		if app == "" {
			return nil, xfmt.Errorf("dependency module %s has empty application", name)
		}
		modules[name] = moduleInfo{Application: app}
		allowed[name] = struct{}{}
		seen[name] = struct{}{}
		if len(cur.Dependencies) > 0 {
			queue = append(queue, cur.Dependencies...)
		}
	}
	return &moduleRules{OwnerName: ownerName, OwnerApp: ownerApp, ModuleInfo: modules, Allowed: allowed}, nil
}

func loadModuleIndex(tx *gorm.DB) (map[string]moduleInfo, map[string]string, error) {
	if tx == nil {
		return nil, nil, xfmt.Errorf("nil db session")
	}
	var rows []struct {
		ID          string `gorm:"column:id"`
		Name        string `gorm:"column:name"`
		Application string `gorm:"column:application_str"`
	}
	if err := tx.Model(&meta.IrModule{}).Select("id", "name", "application_str").Find(&rows).Error; err != nil {
		return nil, nil, xfmt.Errorf("load module index: %w", err)
	}
	modules := make(map[string]moduleInfo, len(rows))
	idToName := make(map[string]string, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		modules[name] = moduleInfo{ID: strings.TrimSpace(r.ID), Application: strings.TrimSpace(r.Application)}
		idToName[strings.TrimSpace(r.ID)] = name
	}
	return modules, idToName, nil
}

func dependencyClosure(tx *gorm.DB, ownerID string, idToName map[string]string) (map[string]struct{}, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, nil
	}
	var rows []struct {
		ModuleID       string `gorm:"column:module_id"`
		DependModuleID string `gorm:"column:depend_module_id"`
	}
	if err := tx.Table("meta_ir_module_dependencies").Select("module_id", "depend_module_id").Find(&rows).Error; err != nil {
		return nil, xfmt.Errorf("load module dependencies: %w", err)
	}
	adj := map[string][]string{}
	for _, r := range rows {
		m := strings.TrimSpace(r.ModuleID)
		d := strings.TrimSpace(r.DependModuleID)
		if m == "" || d == "" {
			continue
		}
		adj[m] = append(adj[m], d)
	}
	seen := map[string]struct{}{}
	queue := []string{ownerID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if _, ok := seen[cur]; ok {
			continue
		}
		seen[cur] = struct{}{}
		for _, dep := range adj[cur] {
			if _, ok := seen[dep]; ok {
				continue
			}
			queue = append(queue, dep)
		}
	}
	out := map[string]struct{}{}
	for id := range seen {
		name := strings.TrimSpace(idToName[id])
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out, nil
}

func validateRecordModule(rules *moduleRules, filePath string, recordIndex int, rec record) error {
	moduleName := strings.TrimSpace(rec.Module)
	if moduleName == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModule, FilePath: filePath, RecordIndex: recordIndex, Message: "missing module"}
	}
	info, ok := rules.ModuleInfo[moduleName]
	if !ok {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeModuleNotFound, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), Message: "record.module not found in registry"}
	}
	if strings.TrimSpace(info.Application) != rules.OwnerApp {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeModuleCrossApplication, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), Message: "record.module belongs to a different application"}
	}
	if _, ok := rules.Allowed[moduleName]; !ok {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeModuleNotInDependencyChain, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), Message: "record.module is outside owner dependency closure"}
	}
	return nil
}

type dataFile struct {
	Records []record `json:"records"`
}

type batchRecord struct {
	FileRel     string
	FilePath    string
	RecordIndex int
	Rec         record
}

type record struct {
	Module     string         `json:"module"`
	ExternalID string         `json:"external_id"`
	Model      string         `json:"model"`
	NoUpdate   *bool          `json:"noupdate,omitempty"`
	Values     map[string]any `json:"values"`
}

type LoadErrorKind string

const (
	LoadErrorKindValidation LoadErrorKind = "validation"
	LoadErrorKindRef        LoadErrorKind = "ref"
	LoadErrorKindDB         LoadErrorKind = "db"
)

const (
	LoadErrorCodeMissingModule              = "missing_module"
	LoadErrorCodeModuleMismatch             = "module_mismatch"
	LoadErrorCodeModuleNotFound             = "module_not_found"
	LoadErrorCodeModuleCrossApplication     = "module_cross_application"
	LoadErrorCodeModuleNotInDependencyChain = "module_not_in_dependency_chain"
	LoadErrorCodeMissingExternalID          = "missing_external_id"
	LoadErrorCodeMissingModel               = "missing_model"
	LoadErrorCodeInvalidModel               = "invalid_model"
	LoadErrorCodeMissingValues              = "missing_values"
	LoadErrorCodeDuplicateExternalIDInInput = "duplicate_external_id_in_input"
	LoadErrorCodeInvalidRef                 = "invalid_ref"
	LoadErrorCodeRefSelfCycle               = "ref_self_cycle"
	LoadErrorCodeRefNotFound                = "ref_not_found"
	LoadErrorCodeRefCycle                   = "ref_cycle"
	LoadErrorCodeResolveRefFailed           = "resolve_ref_failed"
	LoadErrorCodeDBError                    = "db_error"
	LoadErrorCodeDBResolveModel             = "db_resolve_model"
	LoadErrorCodeDBLookupModelData          = "db_lookup_model_data"
	LoadErrorCodeDBInsertRecord             = "db_insert_record"
	LoadErrorCodeDBInsertModelData          = "db_insert_model_data"
	LoadErrorCodeDBUpdateRecord             = "db_update_record"
	LoadErrorCodeDBUpdateModelDataNoUpdate  = "db_update_model_data_noupdate"
	LoadErrorCodeDBModelTableEmpty          = "db_model_table_empty"
)

// LoadError is a structured, retry-friendly error produced by the data loader.
// It carries stable location info to help pinpoint and fix the offending record.
type LoadError struct {
	Kind        LoadErrorKind
	Code        string
	FilePath    string
	RecordIndex int
	Module      string
	ExternalID  string
	Model       string
	FieldPath   string
	Ref         string
	Message     string
	Cause       error
}

func (e *LoadError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := make([]string, 0, 8)
	if strings.TrimSpace(string(e.Kind)) != "" {
		parts = append(parts, "kind="+string(e.Kind))
	}
	if strings.TrimSpace(e.Code) != "" {
		parts = append(parts, "code="+strings.TrimSpace(e.Code))
	}
	if strings.TrimSpace(e.FilePath) != "" {
		parts = append(parts, "file="+e.FilePath)
	}
	if e.RecordIndex >= 0 {
		parts = append(parts, "recordIndex="+strconv.Itoa(e.RecordIndex))
	}
	if strings.TrimSpace(e.Module) != "" || strings.TrimSpace(e.ExternalID) != "" {
		parts = append(parts, "external_id="+strings.TrimSpace(e.Module)+"."+strings.TrimSpace(e.ExternalID))
	}
	if strings.TrimSpace(e.Model) != "" {
		parts = append(parts, "model="+strings.TrimSpace(e.Model))
	}
	if strings.TrimSpace(e.FieldPath) != "" {
		parts = append(parts, "field="+strings.TrimSpace(e.FieldPath))
	}
	if strings.TrimSpace(e.Ref) != "" {
		parts = append(parts, "ref="+strings.TrimSpace(e.Ref))
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "data load error"
	}
	if len(parts) == 0 {
		return msg
	}
	return msg + " (" + strings.Join(parts, ", ") + ")"
}

func (e *LoadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func wrapLoadError(err error, filePath string, recordIndex int, rec record, message string) error {
	if err == nil {
		return nil
	}
	var le *LoadError
	if errors.As(err, &le) {
		return err
	}
	return &LoadError{
		Kind:        LoadErrorKindDB,
		Code:        LoadErrorCodeDBError,
		FilePath:    filePath,
		RecordIndex: recordIndex,
		Module:      strings.TrimSpace(rec.Module),
		ExternalID:  strings.TrimSpace(rec.ExternalID),
		Model:       strings.TrimSpace(rec.Model),
		Message:     message,
		Cause:       err,
	}
}

func wrapLoadErrorWithCode(err error, filePath string, recordIndex int, rec record, kind LoadErrorKind, code string, message string) error {
	if err == nil {
		return nil
	}
	var le *LoadError
	if errors.As(err, &le) {
		return err
	}
	return &LoadError{
		Kind:        kind,
		Code:        code,
		FilePath:    filePath,
		RecordIndex: recordIndex,
		Module:      strings.TrimSpace(rec.Module),
		ExternalID:  strings.TrimSpace(rec.ExternalID),
		Model:       strings.TrimSpace(rec.Model),
		Message:     message,
		Cause:       err,
	}
}

func (l *Loader) ApplyModule(ctx context.Context, mod *meta.IrModule, opts ApplyOptions) error {
	if l == nil || l.runtimeScope == nil {
		return xfmt.Errorf("nil loader")
	}
	if mod == nil {
		return nil
	}
	if strings.TrimSpace(mod.Path) == "" {
		return xfmt.Errorf("module %s has empty Path", mod.Name)
	}

	dataFiles, err := decodeStringArray(mod.DataStr)
	if err != nil {
		return xfmt.Errorf("decode manifest data for %s: %w", mod.Name, err)
	}
	demoFiles, err := decodeStringArray(mod.DemoStr)
	if err != nil {
		return xfmt.Errorf("decode manifest demo for %s: %w", mod.Name, err)
	}

	if err := l.applyFiles(ctx, mod, dataFiles); err != nil {
		return err
	}
	if opts.WithDemo {
		if err := l.applyFiles(ctx, mod, demoFiles); err != nil {
			return err
		}
	}
	return nil
}

// ApplyFiles applies the given relative data files (relative to the module root).
// It enforces the same ownership and validation rules as ApplyModule.
func (l *Loader) ApplyFiles(ctx context.Context, mod *meta.IrModule, files []string) error {
	if l == nil || l.runtimeScope == nil {
		return xfmt.Errorf("nil loader")
	}
	if mod == nil {
		return nil
	}
	if strings.TrimSpace(mod.Path) == "" {
		return xfmt.Errorf("module %s has empty Path", mod.Name)
	}
	return l.applyFiles(ctx, mod, files)
}

func decodeStringArray(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (l *Loader) withApplyTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	txRoot := l.runtimeScope
	if rebound := l.runtimeScope.WithContext(ctx); rebound != nil {
		txRoot = rebound
	}

	return txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
		db, err := loaderTransactionDB(txScope, tx)
		if err != nil {
			return err
		}
		return fn(db)
	})
}

func loaderTransactionDB(runtimeScope scope.Scope, tx scope.Transaction) (*gorm.DB, error) {
	effectiveCtx := context.Background()
	if runtimeScope != nil && runtimeScope.Context() != nil {
		effectiveCtx = runtimeScope.Context()
	}
	if tx != nil {
		if txCtx := tx.Context(); txCtx != nil {
			effectiveCtx = txCtx
		}
		if sess := tx.Session(); sess != nil && sess.DB != nil {
			return sess.DB.WithContext(effectiveCtx), nil
		}
	}
	if runtimeScope == nil || runtimeScope.Session() == nil || runtimeScope.Session().DB == nil {
		return nil, xfmt.Errorf("nil db session")
	}
	return runtimeScope.Session().DB.WithContext(effectiveCtx), nil
}

func (l *Loader) applyFiles(ctx context.Context, mod *meta.IrModule, relPaths []string) error {
	if len(relPaths) == 0 {
		return nil
	}

	// Parse all files first so we can validate refs across files and build a
	// single dependency graph for deterministic ordering and cross-file cycles.
	batch := make([]batchRecord, 0, 64)
	cleaned := make([]string, 0, len(relPaths))
	for _, rel := range relPaths {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		rel = filepath.Clean(rel)
		if rel == "." {
			continue
		}
		cleaned = append(cleaned, rel)
	}
	sort.Strings(cleaned)
	for _, rel := range cleaned {
		abs := filepath.Join(mod.Path, rel)
		b, err := os.ReadFile(abs)
		if err != nil {
			return xfmt.Errorf("read data file %s: %w", abs, err)
		}
		var df dataFile
		if err := json.Unmarshal(b, &df); err != nil {
			return xfmt.Errorf("parse data file %s: %w", abs, err)
		}
		for idx, rec := range df.Records {
			batch = append(batch, batchRecord{FileRel: rel, FilePath: abs, RecordIndex: idx, Rec: rec})
		}
	}
	if len(batch) == 0 {
		return nil
	}

	return l.withApplyTransaction(ctx, func(tx *gorm.DB) error {
		order, err := l.planBatchRecordOrder(tx, mod, batch)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, idx := range order {
			br := batch[idx]
			if err := l.applyRecord(tx, mod, br.FilePath, br.RecordIndex, br.Rec, now); err != nil {
				return err
			}
		}
		return nil
	})
}

// planBatchRecordOrder performs a fail-fast preflight validation before writing and
// returns a stable topological order for applying records across multiple files.
//
// Forward refs within the same apply batch are supported by reordering.
// Cycles (including cross-file) are rejected with a structured error including the cycle chain.
func (l *Loader) planBatchRecordOrder(tx *gorm.DB, owner *meta.IrModule, records []batchRecord) ([]int, error) {
	if owner == nil {
		return nil, xfmt.Errorf("nil owner module")
	}
	rules, err := buildModuleRules(tx, owner)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	indexByKey := make(map[string]int, len(records))
	for idx, br := range records {
		rec := br.Rec
		if err := validateRecordModule(rules, br.FilePath, br.RecordIndex, rec); err != nil {
			return nil, err
		}
		moduleName := strings.TrimSpace(rec.Module)
		externalID := strings.TrimSpace(rec.ExternalID)
		if externalID == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingExternalID, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, Message: "missing external_id"}
		}
		modelFull := strings.TrimSpace(rec.Model)
		if modelFull == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModel, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, ExternalID: externalID, Message: "missing model"}
		}
		if rec.Values == nil {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingValues, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, ExternalID: externalID, Model: modelFull, Message: "missing values"}
		}

		key := moduleName + "." + externalID
		if prevIdx, ok := indexByKey[key]; ok {
			prev := records[prevIdx]
			prevFile := prev.FileRel
			if strings.TrimSpace(prevFile) == "" {
				prevFile = filepath.Base(prev.FilePath)
			}
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeDuplicateExternalIDInInput, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, ExternalID: externalID, Model: modelFull, Message: "duplicate external_id in input (first at file=" + prevFile + ", recordIndex=" + strconv.Itoa(prev.RecordIndex) + ")"}
		}
		indexByKey[key] = idx
	}

	dep := make([][]int, len(records))
	adj := make([][]int, len(records))
	indeg := make([]int, len(records))
	edgeInfo := make(map[[2]int]refOccurrence)

	neededByModule := map[string]map[string]struct{}{}
	for idx, br := range records {
		rec := br.Rec
		occ := collectRefOccurrences(rec.Values)
		for _, o := range occ {
			mod, externalID, err := splitRef(o.Ref)
			if err != nil {
				return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeInvalidRef, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "invalid ref", Cause: err}
			}
			key := mod + "." + externalID
			if targetIdx, ok := indexByKey[key]; ok {
				if targetIdx == idx {
					return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefSelfCycle, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "ref points to itself (cycle)"}
				}
				dep[idx] = append(dep[idx], targetIdx)
				adj[targetIdx] = append(adj[targetIdx], idx)
				indeg[idx]++
				ek := [2]int{idx, targetIdx}
				if _, exists := edgeInfo[ek]; !exists {
					edgeInfo[ek] = o
				}
				continue
			}
			m := neededByModule[mod]
			if m == nil {
				m = map[string]struct{}{}
				neededByModule[mod] = m
			}
			m[externalID] = struct{}{}
		}
	}

	if len(neededByModule) > 0 {
		existing := map[string]map[string]struct{}{}
		modules := make([]string, 0, len(neededByModule))
		for mod := range neededByModule {
			modules = append(modules, mod)
		}
		sort.Strings(modules)
		for _, mod := range modules {
			ids := make([]string, 0, len(neededByModule[mod]))
			for id := range neededByModule[mod] {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			var rows []metadata.IrModelData
			if err := tx.Model(&metadata.IrModelData{}).Select("external_id").Where("module = ? AND external_id IN ?", mod, ids).Find(&rows).Error; err != nil {
				return nil, xfmt.Errorf("lookup ir_model_data for refs: %w", err)
			}
			m := map[string]struct{}{}
			for _, r := range rows {
				m[r.ExternalID] = struct{}{}
			}
			existing[mod] = m
		}

		for _, br := range records {
			rec := br.Rec
			occ := collectRefOccurrences(rec.Values)
			for _, o := range occ {
				mod, externalID, err := splitRef(o.Ref)
				if err != nil {
					continue
				}
				key := mod + "." + externalID
				if _, ok := indexByKey[key]; ok {
					continue
				}
				if _, ok := existing[mod][externalID]; !ok {
					return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefNotFound, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: refNotFoundMessage(rules.OwnerName, o.Ref)}
				}
			}
		}
	}

	order, err := topoOrderOrCycleBatch(records, dep, adj, indeg, edgeInfo)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func topoOrderOrCycleBatch(records []batchRecord, dep [][]int, adj [][]int, indeg []int, edgeInfo map[[2]int]refOccurrence) ([]int, error) {
	keyOf := func(i int) string {
		br := records[i]
		rec := br.Rec
		// Use a stable key for deterministic ordering independent of input slice order.
		fileID := strings.TrimSpace(br.FileRel)
		if fileID == "" {
			fileID = filepath.Base(br.FilePath)
		}
		return fileID + "/" + strconv.Itoa(br.RecordIndex) + "/" + strings.TrimSpace(rec.Module) + "." + strings.TrimSpace(rec.ExternalID)
	}
	less := func(i, j int) bool { return keyOf(i) < keyOf(j) }

	// Normalize edge iteration order for determinism.
	for i := range dep {
		if len(dep[i]) > 1 {
			sort.Slice(dep[i], func(a, b int) bool { return less(dep[i][a], dep[i][b]) })
		}
	}
	for i := range adj {
		if len(adj[i]) > 1 {
			sort.Slice(adj[i], func(a, b int) bool { return less(adj[i][a], adj[i][b]) })
		}
	}

	queue := make([]int, 0, len(records))
	for i := range records {
		if indeg[i] == 0 {
			queue = append(queue, i)
		}
	}
	sort.Slice(queue, func(a, b int) bool { return less(queue[a], queue[b]) })
	order := make([]int, 0, len(records))
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		order = append(order, n)
		for _, m := range adj[n] {
			indeg[m]--
			if indeg[m] == 0 {
				queue = append(queue, m)
			}
		}
		sort.Slice(queue, func(a, b int) bool { return less(queue[a], queue[b]) })
	}
	if len(order) == len(records) {
		return order, nil
	}

	cycle := findCycle(dep)
	if len(cycle) == 0 {
		return nil, xfmt.Errorf("circular ref detected")
	}
	// Canonicalize cycle start for stable error strings.
	if len(cycle) >= 2 {
		base := cycle[:len(cycle)-1]
		minPos := 0
		minKey := keyOf(base[0])
		for i := 1; i < len(base); i++ {
			k := keyOf(base[i])
			if k < minKey {
				minKey = k
				minPos = i
			}
		}
		if minPos != 0 {
			rot := append(append([]int{}, base[minPos:]...), base[:minPos]...)
			cycle = append(rot, rot[0])
		}
	}
	chain := make([]string, 0, len(cycle))
	for _, idx := range cycle {
		br := records[idx]
		rec := br.Rec
		fileID := strings.TrimSpace(br.FileRel)
		if fileID == "" {
			fileID = filepath.Base(br.FilePath)
		}
		chain = append(chain, strings.TrimSpace(rec.Module)+"."+strings.TrimSpace(rec.ExternalID)+"(file="+fileID+",recordIndex="+strconv.Itoa(br.RecordIndex)+")")
	}
	msg := "circular ref detected: " + strings.Join(chain, " -> ")

	first := records[cycle[0]]
	le := &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefCycle, FilePath: first.FilePath, RecordIndex: first.RecordIndex, Module: strings.TrimSpace(first.Rec.Module), ExternalID: strings.TrimSpace(first.Rec.ExternalID), Model: strings.TrimSpace(first.Rec.Model), Message: msg}
	for i := 0; i+1 < len(cycle); i++ {
		ek := [2]int{cycle[i], cycle[i+1]}
		if info, ok := edgeInfo[ek]; ok {
			br := records[cycle[i]]
			le.FilePath = br.FilePath
			le.RecordIndex = br.RecordIndex
			le.Module = strings.TrimSpace(br.Rec.Module)
			le.ExternalID = strings.TrimSpace(br.Rec.ExternalID)
			le.Model = strings.TrimSpace(br.Rec.Model)
			le.FieldPath = info.FieldPath
			le.Ref = info.Ref
			break
		}
	}
	return nil, le
}

func (l *Loader) applyFile(ctx context.Context, mod *meta.IrModule, absPath string) error {
	b, err := os.ReadFile(absPath)
	if err != nil {
		return xfmt.Errorf("read data file %s: %w", absPath, err)
	}
	var df dataFile
	if err := json.Unmarshal(b, &df); err != nil {
		return xfmt.Errorf("parse data file %s: %w", absPath, err)
	}

	return l.withApplyTransaction(ctx, func(tx *gorm.DB) error {
		order, err := l.planRecordOrder(tx, mod, absPath, df.Records)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, idx := range order {
			rec := df.Records[idx]
			if err := l.applyRecord(tx, mod, absPath, idx, rec, now); err != nil {
				return err
			}
		}
		return nil
	})
}

// planRecordOrder performs a fail-fast preflight validation before writing and
// returns a stable topological order for applying records.
//
// Forward refs within the same file are supported by reordering.
// Cycles are rejected with a structured error including the cycle chain.
func (l *Loader) planRecordOrder(tx *gorm.DB, owner *meta.IrModule, filePath string, records []record) ([]int, error) {
	if owner == nil {
		return nil, xfmt.Errorf("nil owner module")
	}
	rules, err := buildModuleRules(tx, owner)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	indexByKey := make(map[string]int, len(records))
	for idx, rec := range records {
		if err := validateRecordModule(rules, filePath, idx, rec); err != nil {
			return nil, err
		}
		moduleName := strings.TrimSpace(rec.Module)
		externalID := strings.TrimSpace(rec.ExternalID)
		if externalID == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingExternalID, FilePath: filePath, RecordIndex: idx, Module: moduleName, Message: "missing external_id"}
		}
		modelFull := strings.TrimSpace(rec.Model)
		if modelFull == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModel, FilePath: filePath, RecordIndex: idx, Module: moduleName, ExternalID: externalID, Message: "missing model"}
		}
		if rec.Values == nil {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingValues, FilePath: filePath, RecordIndex: idx, Module: moduleName, ExternalID: externalID, Model: modelFull, Message: "missing values"}
		}

		key := moduleName + "." + externalID
		if prevIdx, ok := indexByKey[key]; ok {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeDuplicateExternalIDInInput, FilePath: filePath, RecordIndex: idx, Module: moduleName, ExternalID: externalID, Model: modelFull, Message: "duplicate external_id in input (first at recordIndex=" + strconv.Itoa(prevIdx) + ")"}
		}
		indexByKey[key] = idx
	}

	// Build internal dependency graph for topological ordering.
	// dep[i] contains the records that i depends on (i -> dep).
	dep := make([][]int, len(records))
	// adj[j] contains dependents of j (j -> dependent). Used for Kahn.
	adj := make([][]int, len(records))
	indeg := make([]int, len(records))
	edgeInfo := make(map[[2]int]refOccurrence)

	// Collect ref usages and validate them.
	neededByModule := map[string]map[string]struct{}{}
	for idx, rec := range records {
		occ := collectRefOccurrences(rec.Values)
		for _, o := range occ {
			mod, externalID, err := splitRef(o.Ref)
			if err != nil {
				return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeInvalidRef, FilePath: filePath, RecordIndex: idx, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "invalid ref", Cause: err}
			}
			key := mod + "." + externalID
			if targetIdx, ok := indexByKey[key]; ok {
				if targetIdx == idx {
					return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefSelfCycle, FilePath: filePath, RecordIndex: idx, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "ref points to itself (cycle)"}
				}
				// Internal ref within the same file: record idx depends on targetIdx.
				dep[idx] = append(dep[idx], targetIdx)
				adj[targetIdx] = append(adj[targetIdx], idx)
				indeg[idx]++
				ek := [2]int{idx, targetIdx}
				if _, exists := edgeInfo[ek]; !exists {
					edgeInfo[ek] = o
				}
				continue
			}
			m := neededByModule[mod]
			if m == nil {
				m = map[string]struct{}{}
				neededByModule[mod] = m
			}
			m[externalID] = struct{}{}
		}
	}

	if len(neededByModule) == 0 {
		order, err := topoOrderOrCycle(records, dep, adj, indeg, edgeInfo, filePath)
		if err != nil {
			return nil, err
		}
		return order, nil
	}

	// Batch lookup existing mappings per module.
	existing := map[string]map[string]struct{}{}
	modules := make([]string, 0, len(neededByModule))
	for mod := range neededByModule {
		modules = append(modules, mod)
	}
	sort.Strings(modules)
	for _, mod := range modules {
		ids := make([]string, 0, len(neededByModule[mod]))
		for id := range neededByModule[mod] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		var rows []metadata.IrModelData
		if err := tx.Model(&metadata.IrModelData{}).Select("external_id").Where("module = ? AND external_id IN ?", mod, ids).Find(&rows).Error; err != nil {
			return nil, xfmt.Errorf("lookup ir_model_data for refs: %w", err)
		}
		m := map[string]struct{}{}
		for _, r := range rows {
			m[r.ExternalID] = struct{}{}
		}
		existing[mod] = m
	}

	// Report the first missing ref with stable location info.
	for idx, rec := range records {
		occ := collectRefOccurrences(rec.Values)
		for _, o := range occ {
			mod, externalID, err := splitRef(o.Ref)
			if err != nil {
				continue
			}
			key := mod + "." + externalID
			if _, ok := indexByKey[key]; ok {
				continue
			}
			if _, ok := existing[mod][externalID]; !ok {
				return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefNotFound, FilePath: filePath, RecordIndex: idx, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: refNotFoundMessage(rules.OwnerName, o.Ref)}
			}
		}
	}

	order, err := topoOrderOrCycle(records, dep, adj, indeg, edgeInfo, filePath)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func topoOrderOrCycle(records []record, dep [][]int, adj [][]int, indeg []int, edgeInfo map[[2]int]refOccurrence, filePath string) ([]int, error) {
	queue := make([]int, 0, len(records))
	for i := range records {
		if indeg[i] == 0 {
			queue = append(queue, i)
		}
	}
	sort.Ints(queue)
	order := make([]int, 0, len(records))
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		order = append(order, n)
		for _, m := range adj[n] {
			indeg[m]--
			if indeg[m] == 0 {
				queue = append(queue, m)
			}
		}
		sort.Ints(queue)
	}
	if len(order) == len(records) {
		return order, nil
	}

	cycle := findCycle(dep)
	if len(cycle) == 0 {
		return nil, xfmt.Errorf("circular ref detected")
	}
	chain := make([]string, 0, len(cycle))
	for _, idx := range cycle {
		rec := records[idx]
		chain = append(chain, strings.TrimSpace(rec.Module)+"."+strings.TrimSpace(rec.ExternalID)+"(recordIndex="+strconv.Itoa(idx)+")")
	}
	msg := "circular ref detected: " + strings.Join(chain, " -> ")

	le := &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefCycle, FilePath: filePath, RecordIndex: cycle[0], Module: strings.TrimSpace(records[cycle[0]].Module), ExternalID: strings.TrimSpace(records[cycle[0]].ExternalID), Model: strings.TrimSpace(records[cycle[0]].Model), Message: msg}
	// Attach a concrete edge (field/ref) from within the cycle when available.
	// This makes the error more actionable and more stable across internal changes.
	for i := 0; i+1 < len(cycle); i++ {
		ek := [2]int{cycle[i], cycle[i+1]}
		if info, ok := edgeInfo[ek]; ok {
			le.RecordIndex = cycle[i]
			le.Module = strings.TrimSpace(records[cycle[i]].Module)
			le.ExternalID = strings.TrimSpace(records[cycle[i]].ExternalID)
			le.Model = strings.TrimSpace(records[cycle[i]].Model)
			le.FieldPath = info.FieldPath
			le.Ref = info.Ref
			break
		}
	}
	return nil, le
}

// findCycle returns a cycle as a list of record indices in order, and repeats
// the starting node at the end (e.g. [a,b,c,a]) for easier display.
func findCycle(dep [][]int) []int {
	n := len(dep)
	color := make([]uint8, n) // 0=unvisited,1=visiting,2=done
	stack := make([]int, 0, n)
	pos := make([]int, n)
	for i := range pos {
		pos[i] = -1
	}

	var dfs func(int) []int
	dfs = func(v int) []int {
		color[v] = 1
		pos[v] = len(stack)
		stack = append(stack, v)
		for _, to := range dep[v] {
			if color[to] == 0 {
				if cyc := dfs(to); len(cyc) > 0 {
					return cyc
				}
				continue
			}
			if color[to] == 1 {
				start := pos[to]
				if start >= 0 && start < len(stack) {
					cycle := append([]int{}, stack[start:]...)
					cycle = append(cycle, to)
					return cycle
				}
				return []int{to, to}
			}
		}
		color[v] = 2
		stack = stack[:len(stack)-1]
		pos[v] = -1
		return nil
	}

	for i := 0; i < n; i++ {
		if color[i] == 0 {
			if cyc := dfs(i); len(cyc) > 0 {
				return cyc
			}
		}
	}
	return nil
}

type refOccurrence struct {
	FieldPath string
	Ref       string
}

func collectRefOccurrences(values map[string]any) []refOccurrence {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]refOccurrence, 0, 4)
	for _, fieldName := range keys {
		raw := values[fieldName]
		fieldPath := "values." + fieldName
		out = append(out, collectRefOccurrencesAny(raw, fieldPath)...)
	}
	return out
}

func collectRefOccurrencesAny(v any, fieldPath string) []refOccurrence {
	if v == nil {
		return nil
	}
	if ref, ok := extractRef(v); ok {
		return []refOccurrence{{FieldPath: fieldPath, Ref: ref}}
	}
	switch t := v.(type) {
	case []any:
		out := make([]refOccurrence, 0, len(t))
		for i, item := range t {
			out = append(out, collectRefOccurrencesAny(item, fieldPath+"["+strconv.Itoa(i)+"]")...)
		}
		return out
	default:
		return nil
	}
}

func (l *Loader) applyRecord(tx *gorm.DB, owner *meta.IrModule, filePath string, recordIndex int, rec record, now time.Time) error {
	moduleName := strings.TrimSpace(rec.Module)
	if moduleName == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModule, FilePath: filePath, RecordIndex: recordIndex, Message: "missing module"}
	}
	externalID := strings.TrimSpace(rec.ExternalID)
	if externalID == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingExternalID, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Message: "missing external_id"}
	}
	modelFull := strings.TrimSpace(rec.Model)
	if modelFull == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModel, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, ExternalID: externalID, Message: "missing model"}
	}

	app, modelName, err := splitModel(modelFull)
	if err != nil {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeInvalidModel, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, ExternalID: externalID, Model: modelFull, Message: "invalid model", Cause: err}
	}
	model := &meta.IrModel{}
	if err := tx.Where("application = ? AND name = ?", app, modelName).First(model).Error; err != nil {
		return wrapLoadErrorWithCode(xfmt.Errorf("resolve model %s: %w", modelFull, err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBResolveModel, "resolve model")
	}
	if strings.TrimSpace(model.ModelTable) == "" {
		return &LoadError{Kind: LoadErrorKindDB, Code: LoadErrorCodeDBModelTableEmpty, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, ExternalID: externalID, Model: modelFull, Message: "model has empty model_table"}
	}
	tableName := model.ModelTable

	noUpdate := false
	if rec.NoUpdate != nil {
		noUpdate = *rec.NoUpdate
	}

	columns, err := l.resolveAndMapValues(tx, filePath, recordIndex, rec, rec.Values)
	if err != nil {
		return err
	}

	mapping := &metadata.IrModelData{}
	err = tx.Where("module = ? AND external_id = ?", moduleName, externalID).First(mapping).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return wrapLoadErrorWithCode(xfmt.Errorf("lookup ir_model_data: %w", err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBLookupModelData, "lookup ir_model_data")
		}

		resID := xid.New().String()
		columns["id"] = resID
		columns["created_at"] = now
		columns["updated_at"] = now

		if err := tx.Table(tableName).Create(columns).Error; err != nil {
			return wrapLoadErrorWithCode(xfmt.Errorf("insert into %s: %w", tableName, err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBInsertRecord, "insert record")
		}
		mapping.Module = moduleName
		mapping.ExternalID = externalID
		mapping.Model = modelFull
		mapping.ResID = resID
		mapping.NoUpdate = noUpdate
		if err := tx.Create(mapping).Error; err != nil {
			return wrapLoadErrorWithCode(xfmt.Errorf("insert ir_model_data: %w", err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBInsertModelData, "insert ir_model_data")
		}
		return nil
	}

	if mapping.NoUpdate {
		return nil
	}

	columns["updated_at"] = now
	if err := tx.Table(tableName).Where("id = ?", mapping.ResID).Updates(columns).Error; err != nil {
		return wrapLoadErrorWithCode(xfmt.Errorf("update %s id=%s: %w", tableName, mapping.ResID, err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBUpdateRecord, "update record")
	}

	if noUpdate && !mapping.NoUpdate {
		if err := tx.Model(mapping).Update("no_update", true).Error; err != nil {
			return wrapLoadErrorWithCode(xfmt.Errorf("update ir_model_data.noupdate: %w", err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBUpdateModelDataNoUpdate, "update ir_model_data.noupdate")
		}
	}
	return nil
}

func (l *Loader) resolveAndMapValues(tx *gorm.DB, filePath string, recordIndex int, rec record, values map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(values))
	keys := make([]string, 0, len(values))
	for k := range values {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, fieldName := range keys {
		raw := values[fieldName]
		if fieldName == "Id" || fieldName == "CreatedAt" || fieldName == "UpdatedAt" || fieldName == "DeletedAt" {
			continue
		}

		resolved, err := l.resolveValue(tx, filePath, recordIndex, rec, "values."+fieldName, raw)
		if err != nil {
			return nil, err
		}

		// NOTE: We insert via tx.Table(tableName).Create(map[string]any), which bypasses
		// model-level serializers. For JSON-ish values (arrays/objects), marshal them
		// to a JSON string explicitly so drivers like sqlite don't treat slices as
		// row-value tuples ("row value misused").
		resolved, err = normalizeSeedDBValue(resolved)
		if err != nil {
			return nil, &LoadError{
				Kind:        LoadErrorKindValidation,
				Code:        LoadErrorCodeMissingValues,
				FilePath:    filePath,
				RecordIndex: recordIndex,
				Module:      strings.TrimSpace(rec.Module),
				ExternalID:  strings.TrimSpace(rec.ExternalID),
				Model:       strings.TrimSpace(rec.Model),
				FieldPath:   "values." + fieldName,
				Message:     "failed to normalize seed value",
				Cause:       err,
			}
		}
		out[strcase.ToSnake(fieldName)] = resolved
	}
	return out, nil
}

func normalizeSeedDBValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch t := v.(type) {
	case []any, map[string]any:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(b), nil
	default:
		return t, nil
	}
}

func (l *Loader) resolveValue(tx *gorm.DB, filePath string, recordIndex int, rec record, fieldPath string, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	if modelFull, field, value, ok := extractRefBy(v); ok {
		resID, err := l.resolveRefBy(tx, modelFull, field, value)
		if err != nil {
			return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeResolveRefFailed, FilePath: filePath, RecordIndex: recordIndex, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: fieldPath, Ref: "refBy", Message: "resolve refBy failed", Cause: err}
		}
		return resID, nil
	}
	if ref, ok := extractRef(v); ok {
		resID, err := l.resolveRef(tx, ref)
		if err != nil {
			return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeResolveRefFailed, FilePath: filePath, RecordIndex: recordIndex, Module: strings.TrimSpace(rec.Module), ExternalID: strings.TrimSpace(rec.ExternalID), Model: strings.TrimSpace(rec.Model), FieldPath: fieldPath, Ref: ref, Message: "resolve ref failed", Cause: err}
		}
		return resID, nil
	}

	switch t := v.(type) {
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			resolved, err := l.resolveValue(tx, filePath, recordIndex, rec, fieldPath, item)
			if err != nil {
				return nil, err
			}
			out = append(out, resolved)
		}
		return out, nil
	default:
		return v, nil
	}
}

func extractRef(v any) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	raw, ok := m["ref"]
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func extractRefBy(v any) (modelFull string, field string, value any, ok bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", "", nil, false
	}

	raw, ok := m["refBy"]
	if !ok {
		raw, ok = m["ref_by"]
		if !ok {
			return "", "", nil, false
		}
	}

	mm, ok := raw.(map[string]any)
	if !ok {
		return "", "", nil, false
	}

	modelRaw, ok := mm["model"]
	if !ok {
		return "", "", nil, false
	}
	model, ok := modelRaw.(string)
	if !ok {
		return "", "", nil, false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", nil, false
	}

	fieldRaw, ok := mm["field"]
	if !ok {
		return "", "", nil, false
	}
	f, ok := fieldRaw.(string)
	if !ok {
		return "", "", nil, false
	}
	f = strings.TrimSpace(f)
	if f == "" {
		return "", "", nil, false
	}

	val, ok := mm["value"]
	if !ok {
		return "", "", nil, false
	}

	return model, f, val, true
}

func (l *Loader) resolveRefBy(tx *gorm.DB, modelFull string, field string, value any) (string, error) {
	modelFull = strings.TrimSpace(modelFull)
	field = strings.TrimSpace(field)
	if modelFull == "" || field == "" {
		return "", xfmt.Errorf("empty refBy model/field")
	}

	app, modelName, err := splitModel(modelFull)
	if err != nil {
		return "", err
	}
	model := &meta.IrModel{}
	if err := tx.Where("application = ? AND name = ?", app, modelName).First(model).Error; err != nil {
		return "", xfmt.Errorf("resolve refBy model %s: %w", modelFull, err)
	}
	tableName := strings.TrimSpace(model.ModelTable)
	if tableName == "" {
		return "", xfmt.Errorf("refBy model %s has empty model_table", modelFull)
	}

	col := strcase.ToSnake(field)
	type row struct {
		ID string `gorm:"column:id"`
	}
	var r row
	if err := tx.Table(tableName).Select("id").Where(col+" = ?", value).Limit(1).Scan(&r).Error; err != nil {
		return "", err
	}
	if strings.TrimSpace(r.ID) == "" {
		return "", xfmt.Errorf("refBy not found: %s where %s=%v", modelFull, field, value)
	}
	return strings.TrimSpace(r.ID), nil
}

func (l *Loader) resolveRef(tx *gorm.DB, ref string) (string, error) {
	mod, externalID, err := splitRef(ref)
	if err != nil {
		return "", err
	}
	mapping := &metadata.IrModelData{}
	if err := tx.Where("module = ? AND external_id = ?", mod, externalID).First(mapping).Error; err != nil {
		return "", xfmt.Errorf("resolve ref %s: %w", ref, err)
	}
	return mapping.ResID, nil
}

func splitRef(s string) (string, string, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return "", "", xfmt.Errorf("invalid ref %q (expected module.external_id)", s)
	}
	mod := strings.TrimSpace(parts[0])
	externalID := strings.TrimSpace(parts[1])
	if mod == "" || externalID == "" {
		return "", "", xfmt.Errorf("invalid ref %q (empty module or external_id)", s)
	}
	return mod, externalID, nil
}

func refNotFoundMessage(ownerModule string, ref string) string {
	ownerModule = strings.TrimSpace(ownerModule)
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "ref not found"
	}
	mod, _, err := splitRef(ref)
	if err != nil {
		return "ref not found"
	}
	if ownerModule != "" && mod != "" && mod != ownerModule {
		return "ref not found (possible missing dependency or install order: ensure module " + mod + " is installed before " + ownerModule + ")"
	}
	return "ref not found"
}

func splitModel(s string) (string, string, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return "", "", xfmt.Errorf("invalid model %q (expected app.Model)", s)
	}
	app := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if app == "" || name == "" {
		return "", "", xfmt.Errorf("invalid model %q (empty app or name)", s)
	}
	return app, name, nil
}
