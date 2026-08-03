// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

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
	runtimeScope          scope.Scope
	mu                    sync.RWMutex
	modelCache            map[string]string
	fieldCardinalityCache map[string]refCardinality
}

func New(runtimeScope scope.Scope) *Loader {
	return &Loader{
		runtimeScope:          runtimeScope,
		modelCache:            make(map[string]string),
		fieldCardinalityCache: make(map[string]refCardinality),
	}
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

func buildModuleRules(tx *gorm.DB, owner *meta.Module) (*moduleRules, error) {
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

func buildModuleRulesFromOwner(owner *meta.Module) (*moduleRules, error) {
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
	queue := make([]*meta.Module, 0, len(owner.Dependencies))
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
	if err := tx.Model(&meta.Module{}).Select("id", "name", "application_str").Find(&rows).Error; err != nil {
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
	if err := tx.Table("meta_module_dependencies").Select("module_id", "depend_module_id").Find(&rows).Error; err != nil {
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
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeModuleNotFound, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), Message: "record.module not found in registry"}
	}
	if strings.TrimSpace(info.Application) != rules.OwnerApp {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeModuleCrossApplication, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), Message: "record.module belongs to a different application"}
	}
	if _, ok := rules.Allowed[moduleName]; !ok {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeModuleNotInDependencyChain, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), Message: "record.module is outside owner dependency closure"}
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
	Module   string         `json:"module"`
	Name     string         `json:"name"`
	Model    string         `json:"model"`
	NoUpdate *bool          `json:"noupdate,omitempty"`
	Values   map[string]any `json:"values"`
}

// refQuerySpecKind identifies the type of reference query in data file values.
type refQuerySpecKind string

const (
	refQuerySpecKindRef        refQuerySpecKind = "ref"
	refQuerySpecKindRefBy      refQuerySpecKind = "refBy"
	refQuerySpecKindSearch     refQuerySpecKind = "search"
	refQuerySpecKindModelRef   refQuerySpecKind = "modelRef"
	refQuerySpecKindServiceRef refQuerySpecKind = "serviceRef"
)

// refQuerySpec is the unified representation of a reference query.
type refQuerySpec struct {
	Kind       refQuerySpecKind
	Ref        string     // for ref kind: "module.name"
	RefBy      refBySpec  // for refBy kind
	Search     searchSpec // for search kind
	ModelRef   string     // for modelRef kind: "app.Model"
	ServiceRef string     // for serviceRef kind: "app.Model/Method"
}

// refBySpec captures a single-field lookup reference.
type refBySpec struct {
	Model string
	Field string
	Value any
}

// searchSpec captures a domain-based search reference.
type searchSpec struct {
	Model   string // app.Model
	Domain  any    // domain expression
	OrderBy string
	Limit   int
}

// refCardinality describes how many results a reference field expects.
type refCardinality int

const (
	refCardinalityManyToOne  refCardinality = iota // default: exactly one result required
	refCardinalityManyToMany                       // multiple results allowed, returned as a collection
	refCardinalityFirst                            // explicit limit=1 + orderBy: take first, error on zero
)

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
	LoadErrorCodeMissingName                = "missing_name"
	LoadErrorCodeMissingModel               = "missing_model"
	LoadErrorCodeInvalidModel               = "invalid_model"
	LoadErrorCodeMissingValues              = "missing_values"
	LoadErrorCodeDuplicateNameInInput       = "duplicate_name_in_input"
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
	LoadErrorCodeRefSearchInvalidShape      = "ref_search_invalid_shape"
	LoadErrorCodeRefSearchUnsupportedOp     = "ref_search_unsupported_op"
	LoadErrorCodeRefSearchNotFound          = "ref_search_not_found"
	LoadErrorCodeRefSearchNotUnique         = "ref_search_not_unique"
)

// LoadError is a structured, retry-friendly error produced by the data loader.
// It carries stable location info to help pinpoint and fix the offending record.
type LoadError struct {
	Kind        LoadErrorKind
	Code        string
	FilePath    string
	RecordIndex int
	Module      string
	Name        string
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
	if strings.TrimSpace(e.Module) != "" || strings.TrimSpace(e.Name) != "" {
		parts = append(parts, "name="+strings.TrimSpace(e.Module)+"."+strings.TrimSpace(e.Name))
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
		Name:        strings.TrimSpace(rec.Name),
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
		Name:        strings.TrimSpace(rec.Name),
		Model:       strings.TrimSpace(rec.Model),
		Message:     message,
		Cause:       err,
	}
}

// newRefLoadError constructs a LoadError with all location fields pre-filled.
func newRefLoadError(kind LoadErrorKind, code string, filePath string, recordIndex int, rec record, fieldPath string, ref string, message string, cause error) error {
	return &LoadError{
		Kind:        kind,
		Code:        code,
		FilePath:    filePath,
		RecordIndex: recordIndex,
		Module:      strings.TrimSpace(rec.Module),
		Name:        strings.TrimSpace(rec.Name),
		Model:       strings.TrimSpace(rec.Model),
		FieldPath:   fieldPath,
		Ref:         ref,
		Message:     message,
		Cause:       cause,
	}
}

func (l *Loader) ApplyModule(ctx context.Context, mod *meta.Module, opts ApplyOptions) error {
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
func (l *Loader) ApplyFiles(ctx context.Context, mod *meta.Module, files []string) error {
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

func (l *Loader) applyFiles(ctx context.Context, mod *meta.Module, relPaths []string) error {
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
			if err := l.applyRecord(tx, br.FilePath, br.RecordIndex, br.Rec, now); err != nil {
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
func (l *Loader) planBatchRecordOrder(tx *gorm.DB, owner *meta.Module, records []batchRecord) ([]int, error) {
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
		localName := strings.TrimSpace(rec.Name)
		if localName == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingName, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, Message: "missing name"}
		}
		modelFull := strings.TrimSpace(rec.Model)
		if modelFull == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModel, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, Name: localName, Message: "missing model"}
		}
		if rec.Values == nil {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingValues, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, Name: localName, Model: modelFull, Message: "missing values"}
		}

		key := moduleName + "." + localName
		if prevIdx, ok := indexByKey[key]; ok {
			prev := records[prevIdx]
			prevFile := prev.FileRel
			if strings.TrimSpace(prevFile) == "" {
				prevFile = filepath.Base(prev.FilePath)
			}
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeDuplicateNameInInput, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: moduleName, Name: localName, Model: modelFull, Message: "duplicate name in input (first at file=" + prevFile + ", recordIndex=" + strconv.Itoa(prev.RecordIndex) + ")"}
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
			mod, localName, err := splitRef(o.Ref)
			if err != nil {
				return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeInvalidRef, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: strings.TrimSpace(rec.Module), Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "invalid ref", Cause: err}
			}
			key := mod + "." + localName
			if targetIdx, ok := indexByKey[key]; ok {
				if targetIdx == idx {
					return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefSelfCycle, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: strings.TrimSpace(rec.Module), Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "ref points to itself (cycle)"}
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
			m[localName] = struct{}{}
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
			var rows []metadata.ModelData
			if err := tx.Model(&metadata.ModelData{}).Select("name").Where("module = ? AND name IN ?", mod, ids).Find(&rows).Error; err != nil {
				return nil, xfmt.Errorf("lookup model_data for refs: %w", err)
			}
			m := map[string]struct{}{}
			for _, r := range rows {
				m[r.Name] = struct{}{}
			}
			existing[mod] = m
		}

		for _, br := range records {
			rec := br.Rec
			occ := collectRefOccurrences(rec.Values)
			for _, o := range occ {
				mod, localName, err := splitRef(o.Ref)
				if err != nil {
					continue
				}
				key := mod + "." + localName
				if _, ok := indexByKey[key]; ok {
					continue
				}
				if _, ok := existing[mod][localName]; !ok {
					return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefNotFound, FilePath: br.FilePath, RecordIndex: br.RecordIndex, Module: strings.TrimSpace(rec.Module), Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: refNotFoundMessage(rules.OwnerName, o.Ref)}
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
		return fileID + "/" + strconv.Itoa(br.RecordIndex) + "/" + strings.TrimSpace(rec.Module) + "." + strings.TrimSpace(rec.Name)
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
		chain = append(chain, strings.TrimSpace(rec.Module)+"."+strings.TrimSpace(rec.Name)+"(file="+fileID+",recordIndex="+strconv.Itoa(br.RecordIndex)+")")
	}
	msg := "circular ref detected: " + strings.Join(chain, " -> ")

	first := records[cycle[0]]
	le := &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefCycle, FilePath: first.FilePath, RecordIndex: first.RecordIndex, Module: strings.TrimSpace(first.Rec.Module), Name: strings.TrimSpace(first.Rec.Name), Model: strings.TrimSpace(first.Rec.Model), Message: msg}
	for i := 0; i+1 < len(cycle); i++ {
		ek := [2]int{cycle[i], cycle[i+1]}
		if info, ok := edgeInfo[ek]; ok {
			br := records[cycle[i]]
			le.FilePath = br.FilePath
			le.RecordIndex = br.RecordIndex
			le.Module = strings.TrimSpace(br.Rec.Module)
			le.Name = strings.TrimSpace(br.Rec.Name)
			le.Model = strings.TrimSpace(br.Rec.Model)
			le.FieldPath = info.FieldPath
			le.Ref = info.Ref
			break
		}
	}
	return nil, le
}

func (l *Loader) applyFile(ctx context.Context, mod *meta.Module, absPath string) error {
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
			if err := l.applyRecord(tx, absPath, idx, rec, now); err != nil {
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
func (l *Loader) planRecordOrder(tx *gorm.DB, owner *meta.Module, filePath string, records []record) ([]int, error) {
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
		localName := strings.TrimSpace(rec.Name)
		if localName == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingName, FilePath: filePath, RecordIndex: idx, Module: moduleName, Message: "missing name"}
		}
		modelFull := strings.TrimSpace(rec.Model)
		if modelFull == "" {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModel, FilePath: filePath, RecordIndex: idx, Module: moduleName, Name: localName, Message: "missing model"}
		}
		if rec.Values == nil {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingValues, FilePath: filePath, RecordIndex: idx, Module: moduleName, Name: localName, Model: modelFull, Message: "missing values"}
		}

		key := moduleName + "." + localName
		if prevIdx, ok := indexByKey[key]; ok {
			return nil, &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeDuplicateNameInInput, FilePath: filePath, RecordIndex: idx, Module: moduleName, Name: localName, Model: modelFull, Message: "duplicate name in input (first at recordIndex=" + strconv.Itoa(prevIdx) + ")"}
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
			mod, localName, err := splitRef(o.Ref)
			if err != nil {
				return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeInvalidRef, FilePath: filePath, RecordIndex: idx, Module: strings.TrimSpace(rec.Module), Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "invalid ref", Cause: err}
			}
			key := mod + "." + localName
			if targetIdx, ok := indexByKey[key]; ok {
				if targetIdx == idx {
					return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefSelfCycle, FilePath: filePath, RecordIndex: idx, Module: strings.TrimSpace(rec.Module), Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: "ref points to itself (cycle)"}
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
			m[localName] = struct{}{}
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
		var rows []metadata.ModelData
		if err := tx.Model(&metadata.ModelData{}).Select("name").Where("module = ? AND name IN ?", mod, ids).Find(&rows).Error; err != nil {
			return nil, xfmt.Errorf("lookup model_data for refs: %w", err)
		}
		m := map[string]struct{}{}
		for _, r := range rows {
			m[r.Name] = struct{}{}
		}
		existing[mod] = m
	}

	// Report the first missing ref with stable location info.
	for idx, rec := range records {
		occ := collectRefOccurrences(rec.Values)
		for _, o := range occ {
			mod, localName, err := splitRef(o.Ref)
			if err != nil {
				continue
			}
			key := mod + "." + localName
			if _, ok := indexByKey[key]; ok {
				continue
			}
			if _, ok := existing[mod][localName]; !ok {
				return nil, &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefNotFound, FilePath: filePath, RecordIndex: idx, Module: strings.TrimSpace(rec.Module), Name: strings.TrimSpace(rec.Name), Model: strings.TrimSpace(rec.Model), FieldPath: o.FieldPath, Ref: o.Ref, Message: refNotFoundMessage(rules.OwnerName, o.Ref)}
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
		chain = append(chain, strings.TrimSpace(rec.Module)+"."+strings.TrimSpace(rec.Name)+"(recordIndex="+strconv.Itoa(idx)+")")
	}
	msg := "circular ref detected: " + strings.Join(chain, " -> ")

	le := &LoadError{Kind: LoadErrorKindRef, Code: LoadErrorCodeRefCycle, FilePath: filePath, RecordIndex: cycle[0], Module: strings.TrimSpace(records[cycle[0]].Module), Name: strings.TrimSpace(records[cycle[0]].Name), Model: strings.TrimSpace(records[cycle[0]].Model), Message: msg}
	// Attach a concrete edge (field/ref) from within the cycle when available.
	// This makes the error more actionable and more stable across internal changes.
	for i := 0; i+1 < len(cycle); i++ {
		ek := [2]int{cycle[i], cycle[i+1]}
		if info, ok := edgeInfo[ek]; ok {
			le.RecordIndex = cycle[i]
			le.Module = strings.TrimSpace(records[cycle[i]].Module)
			le.Name = strings.TrimSpace(records[cycle[i]].Name)
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

func (l *Loader) applyRecord(tx *gorm.DB, filePath string, recordIndex int, rec record, now time.Time) error {
	moduleName := strings.TrimSpace(rec.Module)
	if moduleName == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModule, FilePath: filePath, RecordIndex: recordIndex, Message: "missing module"}
	}
	localName := strings.TrimSpace(rec.Name)
	if localName == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingName, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Message: "missing name"}
	}
	modelFull := strings.TrimSpace(rec.Model)
	if modelFull == "" {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeMissingModel, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Name: localName, Message: "missing model"}
	}

	app, modelName, err := splitModel(modelFull)
	if err != nil {
		return &LoadError{Kind: LoadErrorKindValidation, Code: LoadErrorCodeInvalidModel, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Name: localName, Model: modelFull, Message: "invalid model", Cause: err}
	}
	model := &meta.Model{}
	if err := tx.Where("application = ? AND name = ?", app, modelName).First(model).Error; err != nil {
		return wrapLoadErrorWithCode(xfmt.Errorf("resolve model %s: %w", modelFull, err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBResolveModel, "resolve model")
	}
	if strings.TrimSpace(model.ModelTable) == "" {
		return &LoadError{Kind: LoadErrorKindDB, Code: LoadErrorCodeDBModelTableEmpty, FilePath: filePath, RecordIndex: recordIndex, Module: moduleName, Name: localName, Model: modelFull, Message: "model has empty model_table"}
	}
	tableName := model.ModelTable

	noUpdate := false
	if rec.NoUpdate != nil {
		noUpdate = *rec.NoUpdate
	}

	columns, err := l.resolveAndMapValues(tx, filePath, recordIndex, rec, model, rec.Values)
	if err != nil {
		return err
	}

	mapping := &metadata.ModelData{}
	err = tx.Where("module = ? AND name = ?", moduleName, localName).First(mapping).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return wrapLoadErrorWithCode(xfmt.Errorf("lookup model_data: %w", err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBLookupModelData, "lookup model_data")
		}

		resID := xid.New().String()
		columns["id"] = resID
		columns["created_at"] = now
		columns["updated_at"] = now

		if err := tx.Table(tableName).Create(columns).Error; err != nil {
			return wrapLoadErrorWithCode(xfmt.Errorf("insert into %s: %w", tableName, err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBInsertRecord, "insert record")
		}
		mapping.Module = moduleName
		mapping.Name = localName
		mapping.Model = modelFull
		mapping.ResID = resID
		mapping.NoUpdate = noUpdate
		if err := tx.Create(mapping).Error; err != nil {
			return wrapLoadErrorWithCode(xfmt.Errorf("insert model_data: %w", err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBInsertModelData, "insert model_data")
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
			return wrapLoadErrorWithCode(xfmt.Errorf("update model_data.noupdate: %w", err), filePath, recordIndex, rec, LoadErrorKindDB, LoadErrorCodeDBUpdateModelDataNoUpdate, "update model_data.noupdate")
		}
	}
	return nil
}

func (l *Loader) resolveAndMapValues(tx *gorm.DB, filePath string, recordIndex int, rec record, model *meta.Model, values map[string]any) (map[string]any, error) {
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

		// Enforce reference cardinality on search results.
		if ids, ok := resolved.([]string); ok {
			cardinality := l.detectSearchCardinality(tx, rec.Model, fieldName, raw)
			resolved, err = enforceReferenceCardinality(ids, cardinality, filePath, recordIndex, rec, "values."+fieldName, "search")
			if err != nil {
				return nil, err
			}
		}

		resolved, err = l.normalizeTranslatedSeedValue(tx, filePath, recordIndex, rec, model, fieldName, resolved, values)
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
				Name:        strings.TrimSpace(rec.Name),
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
	case []string:
		b, err := json.Marshal(t)
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

	// Unified ref query dispatch — handles ref, refBy, search, modelRef, serviceRef.
	if spec, ok, err := parseRefQuerySpec(v); ok {
		if err != nil {
			return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeRefSearchInvalidShape, filePath, recordIndex, rec, fieldPath, "", "invalid ref query shape", err)
		}
		switch spec.Kind {
		case refQuerySpecKindRef:
			resID, err := l.resolveRef(tx, spec.Ref)
			if err != nil {
				return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeResolveRefFailed, filePath, recordIndex, rec, fieldPath, spec.Ref, "resolve ref failed", err)
			}
			return resID, nil
		case refQuerySpecKindRefBy:
			resID, err := l.resolveRefBy(tx, spec.RefBy.Model, spec.RefBy.Field, spec.RefBy.Value)
			if err != nil {
				return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeResolveRefFailed, filePath, recordIndex, rec, fieldPath, "refBy", "resolve refBy failed", err)
			}
			return resID, nil
		case refQuerySpecKindSearch:
			domain, err := l.resolveSearchDomainRefs(tx, filePath, recordIndex, rec, fieldPath+".domain", spec.Search.Domain)
			if err != nil {
				return nil, err
			}
			spec.Search.Domain = domain
			ids, err := l.resolveRefBySearch(tx, spec.Search)
			if err != nil {
				return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeResolveRefFailed, filePath, recordIndex, rec, fieldPath, "search", "resolve search failed", err)
			}
			// Return raw []string; cardinality enforcement happens in resolveAndMapValues.
			return ids, nil
		case refQuerySpecKindModelRef:
			resID, err := l.resolveModelRef(tx, spec.ModelRef)
			if err != nil {
				return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeResolveRefFailed, filePath, recordIndex, rec, fieldPath, "modelRef", "resolve modelRef failed", err)
			}
			return resID, nil
		case refQuerySpecKindServiceRef:
			resID, err := l.resolveServiceRef(tx, spec.ServiceRef)
			if err != nil {
				return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeResolveRefFailed, filePath, recordIndex, rec, fieldPath, "serviceRef", "resolve serviceRef failed", err)
			}
			return resID, nil
		}
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
	spec, ok, _ := parseRefQuerySpec(v)
	if !ok || spec.Kind != refQuerySpecKindRef {
		return "", false
	}
	return spec.Ref, true
}

func extractRefBy(v any) (modelFull string, field string, value any, ok bool) {
	spec, ok, _ := parseRefQuerySpec(v)
	if !ok || spec.Kind != refQuerySpecKindRefBy {
		return "", "", nil, false
	}
	return spec.RefBy.Model, spec.RefBy.Field, spec.RefBy.Value, true
}

func (l *Loader) resolveRefBy(tx *gorm.DB, modelFull string, field string, value any) (string, error) {
	modelFull = strings.TrimSpace(modelFull)
	field = strings.TrimSpace(field)
	if modelFull == "" || field == "" {
		return "", xfmt.Errorf("empty refBy model/field")
	}

	spec := searchSpec{
		Model:   modelFull,
		Domain:  []any{[]any{field, "=", value}},
		Limit:   1,
		OrderBy: "id ASC",
	}
	ids, err := l.resolveRefBySearch(tx, spec)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", xfmt.Errorf("refBy not found: %s where %s=%v", modelFull, field, value)
	}
	return ids[0], nil
}

func (l *Loader) resolveRef(tx *gorm.DB, ref string) (string, error) {
	mod, localName, err := splitRef(ref)
	if err != nil {
		return "", err
	}
	mapping := &metadata.ModelData{}
	if err := tx.Where("module = ? AND name = ?", mod, localName).First(mapping).Error; err != nil {
		return "", xfmt.Errorf("resolve ref %s: %w", ref, err)
	}
	return mapping.ResID, nil
}

func splitRef(s string) (string, string, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return "", "", xfmt.Errorf("invalid ref %q (expected module.name)", s)
	}
	mod := strings.TrimSpace(parts[0])
	localName := strings.TrimSpace(parts[1])
	if mod == "" || localName == "" {
		return "", "", xfmt.Errorf("invalid ref %q (empty module or name)", s)
	}
	return mod, localName, nil
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

// parseRefQuerySpec detects and parses a reference query from a data file value.
// It returns (spec, true, nil) when the value is a recognized ref query shape.
// It returns (spec, true, err) when the shape is recognized but invalid.
// It returns (zero, false, nil) when the value is not a ref query at all.
func parseRefQuerySpec(v any) (refQuerySpec, bool, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return refQuerySpec{}, false, nil
	}

	// ref: "module.name"
	if raw, ok := m["ref"]; ok {
		s, ok := raw.(string)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("ref value must be a string")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return refQuerySpec{}, true, xfmt.Errorf("ref value must not be empty")
		}
		return refQuerySpec{Kind: refQuerySpecKindRef, Ref: s}, true, nil
	}

	// refBy / ref_by
	raw, hasRefBy := m["refBy"]
	if !hasRefBy {
		raw, hasRefBy = m["ref_by"]
	}
	if hasRefBy {
		mm, ok := raw.(map[string]any)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("refBy value must be an object")
		}
		modelRaw, ok := mm["model"]
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("refBy missing model")
		}
		model, ok := modelRaw.(string)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("refBy model must be a string")
		}
		model = strings.TrimSpace(model)
		if model == "" {
			return refQuerySpec{}, true, xfmt.Errorf("refBy model must not be empty")
		}
		fieldRaw, ok := mm["field"]
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("refBy missing field")
		}
		field, ok := fieldRaw.(string)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("refBy field must be a string")
		}
		field = strings.TrimSpace(field)
		if field == "" {
			return refQuerySpec{}, true, xfmt.Errorf("refBy field must not be empty")
		}
		val, ok := mm["value"]
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("refBy missing value")
		}
		return refQuerySpec{
			Kind: refQuerySpecKindRefBy,
			RefBy: refBySpec{
				Model: model,
				Field: field,
				Value: val,
			},
		}, true, nil
	}

	// search
	if raw, ok := m["search"]; ok {
		mm, ok := raw.(map[string]any)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("search value must be an object")
		}
		spec, err := parseSearchSpec(mm)
		if err != nil {
			return refQuerySpec{}, true, err
		}
		return refQuerySpec{Kind: refQuerySpecKindSearch, Search: spec}, true, nil
	}

	// modelRef: "app.Model"
	if raw, ok := m["modelRef"]; ok {
		s, ok := raw.(string)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("modelRef value must be a string")
		}
		s, err := parseModelRefSpec(s)
		if err != nil {
			return refQuerySpec{}, true, err
		}
		return refQuerySpec{Kind: refQuerySpecKindModelRef, ModelRef: s}, true, nil
	}

	// serviceRef: "app.Model/Method"
	if raw, ok := m["serviceRef"]; ok {
		s, ok := raw.(string)
		if !ok {
			return refQuerySpec{}, true, xfmt.Errorf("serviceRef value must be a string")
		}
		s, err := parseServiceRefSpec(s)
		if err != nil {
			return refQuerySpec{}, true, err
		}
		return refQuerySpec{Kind: refQuerySpecKindServiceRef, ServiceRef: s}, true, nil
	}

	return refQuerySpec{}, false, nil
}

// parseModelRefSpec validates a modelRef shortcut string ("app.Model").
func parseModelRefSpec(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", xfmt.Errorf("modelRef value must not be empty")
	}
	return s, nil
}

// parseServiceRefSpec validates a serviceRef shortcut string ("app.Model/Method").
func parseServiceRefSpec(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", xfmt.Errorf("serviceRef value must not be empty")
	}
	return s, nil
}

// parseSearchSpec validates and normalizes a search spec object.
func parseSearchSpec(raw map[string]any) (searchSpec, error) {
	modelRaw, ok := raw["model"]
	if !ok {
		return searchSpec{}, xfmt.Errorf("search missing model")
	}
	model, ok := modelRaw.(string)
	if !ok {
		return searchSpec{}, xfmt.Errorf("search model must be a string")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return searchSpec{}, xfmt.Errorf("search model must not be empty")
	}

	spec := searchSpec{Model: model}

	if domainRaw, ok := raw["domain"]; ok {
		spec.Domain = domainRaw
	}
	if orderByRaw, ok := raw["orderBy"]; ok {
		orderBy, ok := orderByRaw.(string)
		if !ok {
			return searchSpec{}, xfmt.Errorf("search orderBy must be a string")
		}
		orderBy = strings.TrimSpace(orderBy)
		if orderBy != "" && !orderByRegexp.MatchString(orderBy) {
			return searchSpec{}, xfmt.Errorf("invalid search orderBy: %q", orderBy)
		}
		spec.OrderBy = orderBy
	}
	if limitRaw, ok := raw["limit"]; ok {
		// JSON numbers decode as float64.
		switch v := limitRaw.(type) {
		case float64:
			spec.Limit = int(v)
		case int:
			spec.Limit = v
		case int64:
			spec.Limit = int(v)
		default:
			return searchSpec{}, xfmt.Errorf("search limit must be a number")
		}
		if spec.Limit < 0 {
			return searchSpec{}, xfmt.Errorf("search limit must be >= 0")
		}
	}

	return spec, nil
}

// --- Search query executor ---------------------------------------------------------

// resolveSearchDomainRefs walks a search domain tree and resolves nested ref queries
// (ref / refBy / search / modelRef / serviceRef) in leaf values so seeds can write
// e.g. ["ModelId", "=", {"modelRef": "auth.User"}].
func (l *Loader) resolveSearchDomainRefs(tx *gorm.DB, filePath string, recordIndex int, rec record, fieldPath string, domain any) (any, error) {
	if domain == nil {
		return nil, nil
	}
	arr, ok := domain.([]any)
	if !ok {
		return domain, nil
	}
	if len(arr) == 0 {
		return arr, nil
	}

	firstStr, firstIsStr := arr[0].(string)
	if firstIsStr {
		switch firstStr {
		case "&", "|":
			out := make([]any, len(arr))
			out[0] = firstStr
			for i, child := range arr[1:] {
				resolved, err := l.resolveSearchDomainRefs(tx, filePath, recordIndex, rec, fieldPath, child)
				if err != nil {
					return nil, err
				}
				out[i+1] = resolved
			}
			return out, nil
		case "!":
			if len(arr) < 2 {
				return arr, nil
			}
			resolved, err := l.resolveSearchDomainRefs(tx, filePath, recordIndex, rec, fieldPath, arr[1])
			if err != nil {
				return nil, err
			}
			return []any{firstStr, resolved}, nil
		}
	}

	// Leaf: [field, operator, value?]
	if firstIsStr && len(arr) >= 2 {
		if op, ok := arr[1].(string); ok && validateSearchOperator(op) {
			out := make([]any, len(arr))
			out[0] = arr[0]
			out[1] = arr[1]
			if len(arr) >= 3 {
				resolved, err := l.resolveValue(tx, filePath, recordIndex, rec, fieldPath, arr[2])
				if err != nil {
					return nil, err
				}
				// Collapse singleton search results so "=" comparisons receive a scalar id.
				if ids, ok := resolved.([]string); ok && len(ids) == 1 {
					resolved = ids[0]
				}
				out[2] = resolved
				for i := 3; i < len(arr); i++ {
					out[i] = arr[i]
				}
			}
			return out, nil
		}
	}

	// Implicit AND: [node1, node2, ...]
	out := make([]any, len(arr))
	for i, child := range arr {
		resolved, err := l.resolveSearchDomainRefs(tx, filePath, recordIndex, rec, fieldPath, child)
		if err != nil {
			return nil, err
		}
		out[i] = resolved
	}
	return out, nil
}

// resolveRefBySearch resolves a searchSpec against the database and returns matching record IDs.
func (l *Loader) resolveRefBySearch(tx *gorm.DB, spec searchSpec) ([]string, error) {
	_, tableName, err := resolveSearchModel(tx, spec.Model)
	if err != nil {
		return nil, err
	}

	q := tx.Table(tableName).Select("id")

	if spec.Domain != nil {
		q, err = applyDomainToQuery(q, spec.Domain)
		if err != nil {
			return nil, xfmt.Errorf("build search domain for %s: %w", spec.Model, err)
		}
	}

	if spec.OrderBy != "" {
		q = q.Order(spec.OrderBy)
	}
	if spec.Limit > 0 {
		q = q.Limit(spec.Limit)
	}

	type row struct {
		ID string `gorm:"column:id"`
	}
	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, xfmt.Errorf("search %s: %w", spec.Model, err)
	}

	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if id := strings.TrimSpace(r.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// resolveSearchModel looks up a model by app.Model name and returns the model and its table name.
func resolveSearchModel(tx *gorm.DB, modelFull string) (*meta.Model, string, error) {
	app, modelName, err := splitModel(modelFull)
	if err != nil {
		return nil, "", xfmt.Errorf("resolve search model %s: %w", modelFull, err)
	}
	model := &meta.Model{}
	if err := tx.Where("application = ? AND name = ?", app, modelName).First(model).Error; err != nil {
		return nil, "", xfmt.Errorf("resolve search model %s: %w", modelFull, err)
	}
	tableName := strings.TrimSpace(model.ModelTable)
	if tableName == "" {
		return nil, "", xfmt.Errorf("search model %s has empty model_table", modelFull)
	}
	return model, tableName, nil
}

// applyDomainToQuery translates an Odoo-style domain expression into GORM where conditions.
//
// Supported forms:
//   - Leaf: ["field", "operator", value]
//   - Explicit AND: ["&", node1, node2, ...]
//   - Explicit OR:  ["|", node1, node2, ...]
//   - Explicit NOT: ["!", node]
//   - Implicit AND: [node1, node2, ...] (array of arrays)
func applyDomainToQuery(q *gorm.DB, domain any) (*gorm.DB, error) {
	arr, ok := domain.([]any)
	if !ok {
		return nil, xfmt.Errorf("domain node must be an array, got %T", domain)
	}
	if len(arr) == 0 {
		return q, nil
	}

	firstStr, firstIsStr := arr[0].(string)

	// Check for explicit combinators.
	if firstIsStr {
		switch firstStr {
		case "&":
			if len(arr) < 3 {
				return nil, xfmt.Errorf("AND combinator requires at least 2 operands, got %d", len(arr)-1)
			}
			for _, child := range arr[1:] {
				sub := q.Session(&gorm.Session{NewDB: true})
				sub, err := applyDomainToQuery(sub, child)
				if err != nil {
					return nil, err
				}
				q = q.Where(sub)
			}
			return q, nil
		case "|":
			if len(arr) < 3 {
				return nil, xfmt.Errorf("OR combinator requires at least 2 operands, got %d", len(arr)-1)
			}
			group := q.Session(&gorm.Session{NewDB: true})
			for i, child := range arr[1:] {
				sub := q.Session(&gorm.Session{NewDB: true})
				sub, err := applyDomainToQuery(sub, child)
				if err != nil {
					return nil, err
				}
				if i == 0 {
					group = group.Where(sub)
				} else {
					group = group.Or(sub)
				}
			}
			return q.Where(group), nil
		case "!":
			if len(arr) != 2 {
				return nil, xfmt.Errorf("NOT combinator requires exactly 1 operand, got %d", len(arr)-1)
			}
			sub := q.Session(&gorm.Session{NewDB: true})
			sub, err := applyDomainToQuery(sub, arr[1])
			if err != nil {
				return nil, err
			}
			return q.Not(sub), nil
		}
	}

	// Check if it is a leaf: first element is a string field name, second is a recognized operator.
	if firstIsStr && len(arr) >= 2 {
		if op, ok := arr[1].(string); ok {
			if validateSearchOperator(op) {
				return applyLeafNode(q, arr)
			}
			return nil, xfmt.Errorf("unsupported search operator: %q", op)
		}
	}

	// Default: implicit AND of sub-nodes (each element is a leaf or nested domain).
	for _, child := range arr {
		sub := q.Session(&gorm.Session{NewDB: true})
		sub, err := applyDomainToQuery(sub, child)
		if err != nil {
			return nil, err
		}
		q = q.Where(sub)
	}
	return q, nil
}

// applyLeafNode translates a domain leaf [field, operator, value] into a GORM condition.
func applyLeafNode(q *gorm.DB, leaf []any) (*gorm.DB, error) {
	switch len(leaf) {
	case 2, 3:
		// Valid arities.
	default:
		return nil, xfmt.Errorf("invalid domain leaf: expected [field, operator, value], got len=%d", len(leaf))
	}
	field, ok := leaf[0].(string)
	if !ok {
		return nil, xfmt.Errorf("domain leaf field must be a string, got %T", leaf[0])
	}
	field, err := normalizeSearchField(field)
	if err != nil {
		return nil, err
	}
	op, ok := leaf[1].(string)
	if !ok {
		return nil, xfmt.Errorf("domain leaf operator must be a string, got %T", leaf[1])
	}
	if !validateSearchOperator(op) {
		return nil, xfmt.Errorf("unsupported search operator: %q", op)
	}

	hasValue := len(leaf) == 3
	val := any(nil)
	if hasValue {
		val = leaf[2]
	}

	switch op {
	case "=":
		if !hasValue {
			return nil, xfmt.Errorf("operator = requires a value")
		}
		return q.Where(field+" = ?", val), nil
	case "!=":
		if !hasValue {
			return nil, xfmt.Errorf("operator != requires a value")
		}
		return q.Where(field+" != ?", val), nil
	case ">", ">=", "<", "<=":
		if !hasValue {
			return nil, xfmt.Errorf("operator %s requires a value", op)
		}
		return q.Where(field+" "+op+" ?", val), nil
	case "like":
		if !hasValue {
			return nil, xfmt.Errorf("operator like requires a value")
		}
		return q.Where(field+" LIKE ?", val), nil
	case "ilike":
		if !hasValue {
			return nil, xfmt.Errorf("operator ilike requires a value")
		}
		return q.Where("LOWER("+field+") LIKE LOWER(?)", val), nil
	case "in":
		if !hasValue {
			return nil, xfmt.Errorf("operator in requires a value")
		}
		return q.Where(field+" IN ?", val), nil
	case "not_in":
		if !hasValue {
			return nil, xfmt.Errorf("operator not_in requires a value")
		}
		return q.Where(field+" NOT IN ?", val), nil
	case "is_null":
		return q.Where(field + " IS NULL"), nil
	case "is_not_null":
		return q.Where(field + " IS NOT NULL"), nil
	default:
		return nil, xfmt.Errorf("unsupported search operator: %q", op)
	}
}

// validOperators is the whitelist of supported search domain operators.
var validOperators = map[string]bool{
	"=": true, "!=": true,
	">": true, ">=": true, "<": true, "<=": true,
	"in": true, "not_in": true,
	"is_null": true, "is_not_null": true,
	"like": true, "ilike": true,
}

// validateSearchOperator reports whether op is a supported domain operator.
func validateSearchOperator(op string) bool {
	return validOperators[op]
}

// searchFieldRegexp validates that a normalized search field contains only safe characters.
var searchFieldRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// orderByRegexp validates an ORDER BY clause to prevent SQL injection through GORM's Order().
// Allowed: column [ASC|DESC], optionally prefixed with table., comma-separated.
var orderByRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?(\s+(?i:ASC|DESC))?(\s*,\s*[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?(\s+(?i:ASC|DESC))?)*$`)

// normalizeSearchField trims, snake_case-converts, and validates a domain field name.
func normalizeSearchField(field string) (string, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", xfmt.Errorf("search field must not be empty")
	}
	snake := strcase.ToSnake(field)
	if !searchFieldRegexp.MatchString(snake) {
		return "", xfmt.Errorf("invalid search field: %q", field)
	}
	return snake, nil
}

// --- Reference cardinality ---------------------------------------------------------

// detectSearchCardinality determines the expected cardinality for a search result.
// Explicit hints from the search spec (limit=1 + orderBy → First) take precedence;
// otherwise the field type in meta_field is consulted (ManyToMany → ManyToMany,
// default → ManyToOne).
func (l *Loader) detectSearchCardinality(tx *gorm.DB, modelFull string, fieldName string, raw any) refCardinality {
	spec, ok, _ := parseRefQuerySpec(raw)
	if ok && spec.Kind == refQuerySpecKindSearch {
		if spec.Search.Limit == 1 && spec.Search.OrderBy != "" {
			return refCardinalityFirst
		}
	}
	return l.detectFieldCardinality(tx, modelFull, fieldName)
}

// detectFieldCardinality looks up the field type from meta_field and returns the
// corresponding cardinality. It defaults to ManyToOne when the field or model cannot
// be resolved. Results are cached per loader instance since model/field metadata is
// static during a bootstrap apply run.
func (l *Loader) detectFieldCardinality(tx *gorm.DB, modelFull string, fieldName string) refCardinality {
	snake := strcase.ToSnake(fieldName)
	cacheKey := modelFull + "." + snake

	l.mu.RLock()
	if c, ok := l.fieldCardinalityCache[cacheKey]; ok {
		l.mu.RUnlock()
		return c
	}
	l.mu.RUnlock()

	app, modelName, err := splitModel(modelFull)
	if err != nil {
		l.mu.Lock()
		l.fieldCardinalityCache[cacheKey] = refCardinalityManyToOne
		l.mu.Unlock()
		return refCardinalityManyToOne
	}

	// Reuse modelCache to avoid a duplicate meta_model query.
	modelCacheKey := strcase.ToSnake(modelFull)

	l.mu.RLock()
	modelID, ok := l.modelCache[modelCacheKey]
	l.mu.RUnlock()

	if !ok {
		model := &meta.Model{}
		if err := tx.Where("application = ? AND name = ?", app, modelName).First(model).Error; err != nil {
			l.mu.Lock()
			l.fieldCardinalityCache[cacheKey] = refCardinalityManyToOne
			l.mu.Unlock()
			return refCardinalityManyToOne
		}
		modelID = strings.TrimSpace(model.Id.String)
		l.mu.Lock()
		l.modelCache[modelCacheKey] = modelID
		l.mu.Unlock()
	}

	field := &meta.Field{}
	if err := tx.Where("model_id = ? AND name = ?", modelID, snake).First(field).Error; err != nil {
		l.mu.Lock()
		l.fieldCardinalityCache[cacheKey] = refCardinalityManyToOne
		l.mu.Unlock()
		return refCardinalityManyToOne
	}
	var c refCardinality
	switch field.FieldType {
	case "ManyToMany", "OneToMany":
		c = refCardinalityManyToMany
	default:
		c = refCardinalityManyToOne
	}
	l.mu.Lock()
	l.fieldCardinalityCache[cacheKey] = c
	l.mu.Unlock()
	return c
}

// enforceReferenceCardinality validates and coerces search results according to the
// expected cardinality.
//
//   - ManyToOne: exactly one result required; 0 → not_found, >1 → not_unique.
//   - First: at least one result required (limit=1 + orderBy); returns the first.
//   - ManyToMany: any number of results allowed; returned as []string (raw slice).
func enforceReferenceCardinality(ids []string, cardinality refCardinality, filePath string, recordIndex int, rec record, fieldPath string, ref string) (any, error) {
	switch cardinality {
	case refCardinalityManyToOne:
		if len(ids) == 0 {
			return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeRefSearchNotFound, filePath, recordIndex, rec, fieldPath, ref, "search returned no results (expected exactly one)", nil)
		}
		if len(ids) > 1 {
			return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeRefSearchNotUnique, filePath, recordIndex, rec, fieldPath, ref, "search returned multiple results (expected exactly one)", nil)
		}
		return ids[0], nil
	case refCardinalityFirst:
		if len(ids) == 0 {
			return nil, newRefLoadError(LoadErrorKindRef, LoadErrorCodeRefSearchNotFound, filePath, recordIndex, rec, fieldPath, ref, "search returned no results (expected at least one with limit=1)", nil)
		}
		return ids[0], nil
	case refCardinalityManyToMany:
		return ids, nil
	default:
		return ids, nil
	}
}

// --- Shortcut reference resolvers -------------------------------------------------

// resolveModelRef resolves a modelRef shortcut ("app.Model") to the Model ID.
func (l *Loader) resolveModelRef(tx *gorm.DB, modelRef string) (string, error) {
	key := strcase.ToSnake(modelRef)
	l.mu.RLock()
	if id, ok := l.modelCache[key]; ok {
		l.mu.RUnlock()
		return id, nil
	}
	l.mu.RUnlock()

	app, modelName, err := splitModel(modelRef)
	if err != nil {
		return "", xfmt.Errorf("resolve modelRef %s: %w", modelRef, err)
	}
	model := &meta.Model{}
	if err := tx.Where("application = ? AND name = ?", app, modelName).First(model).Error; err != nil {
		return "", xfmt.Errorf("resolve modelRef %s: %w", modelRef, err)
	}
	id := strings.TrimSpace(model.Id.String)
	l.mu.Lock()
	l.modelCache[key] = id
	l.mu.Unlock()
	return id, nil
}

// resolveServiceRef resolves a serviceRef shortcut ("app.Model/Method") to the Service ID.
// It first resolves the model via resolveModelRef, then searches meta_service by model_id + name.
func (l *Loader) resolveServiceRef(tx *gorm.DB, serviceRef string) (string, error) {
	modelFull, method, err := splitServiceRef(serviceRef)
	if err != nil {
		return "", xfmt.Errorf("resolve serviceRef %s: %w", serviceRef, err)
	}
	modelID, err := l.resolveModelRef(tx, modelFull)
	if err != nil {
		return "", xfmt.Errorf("resolve serviceRef %s: %w", serviceRef, err)
	}
	spec := serviceRefToSearchSpec(modelID, method)
	ids, err := l.resolveRefBySearch(tx, spec)
	if err != nil {
		return "", xfmt.Errorf("resolve serviceRef %s: %w", serviceRef, err)
	}
	if len(ids) == 0 {
		return "", xfmt.Errorf("serviceRef not found: %s", serviceRef)
	}
	return ids[0], nil
}

// splitServiceRef splits "app.Model/Method" into ("app.Model", "Method").
func splitServiceRef(s string) (modelFull string, method string, err error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return "", "", xfmt.Errorf("invalid serviceRef %q (expected app.Model/Method)", s)
	}
	modelFull = strings.TrimSpace(parts[0])
	method = strings.TrimSpace(parts[1])
	if modelFull == "" || method == "" {
		return "", "", xfmt.Errorf("invalid serviceRef %q (empty model or method)", s)
	}
	return modelFull, method, nil
}

// serviceRefToSearchSpec builds a searchSpec for looking up meta_service by model ID + method name.
func serviceRefToSearchSpec(modelID string, method string) searchSpec {
	return searchSpec{
		Model: "meta.MetaService",
		Domain: []any{
			"&",
			[]any{"model_id", "=", modelID},
			[]any{"name", "=", method},
		},
	}
}
