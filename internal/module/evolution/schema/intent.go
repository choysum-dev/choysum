// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"context"
	"strings"
	"sync"
)

// IntentKind classifies a script-declared schema intent.
type IntentKind string

const (
	IntentRenameColumn   IntentKind = "rename_column"
	IntentDropColumn     IntentKind = "drop_column"
	IntentDropIndex      IntentKind = "drop_index"
	IntentDropCheck      IntentKind = "drop_check"
	IntentDropForeignKey IntentKind = "drop_foreign_key"
)

// Intent records that an upgrade script already handled (or will handle) a Manual op.
type Intent struct {
	Kind  IntentKind
	Table string
	Name  string // column / index / constraint name
	// FromName is set for rename intents (old column).
	FromName string
}

// IntentBag stores process-local schema intents for one upgrade op (PhasePre → schema → PhasePost).
type IntentBag interface {
	Add(intents ...Intent)
	List() []Intent
	Clear()
}

type memoryIntentBag struct {
	mu      sync.Mutex
	intents []Intent
}

// NewMemoryIntentBag returns an empty in-memory IntentBag.
func NewMemoryIntentBag() IntentBag {
	return &memoryIntentBag{}
}

func (b *memoryIntentBag) Add(intents ...Intent) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, in := range intents {
		in.Kind = IntentKind(strings.ToLower(strings.TrimSpace(string(in.Kind))))
		in.Table = strings.TrimSpace(in.Table)
		in.Name = strings.TrimSpace(in.Name)
		in.FromName = strings.TrimSpace(in.FromName)
		if in.Kind == "" || in.Table == "" {
			continue
		}
		if in.Kind == IntentRenameColumn && (in.Name == "" || in.FromName == "") {
			continue
		}
		b.intents = append(b.intents, in)
	}
}

func (b *memoryIntentBag) List() []Intent {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Intent, len(b.intents))
	copy(out, b.intents)
	return out
}

func (b *memoryIntentBag) Clear() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.intents = nil
}

type intentBagContextKey struct{}

// ContextWithIntentBag attaches a bag to ctx for migration script bridges.
func ContextWithIntentBag(ctx context.Context, bag IntentBag) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if bag == nil {
		return ctx
	}
	return context.WithValue(ctx, intentBagContextKey{}, bag)
}

// IntentBagFromContext returns the bag attached to ctx, if any.
func IntentBagFromContext(ctx context.Context) IntentBag {
	if ctx == nil {
		return nil
	}
	bag, _ := ctx.Value(intentBagContextKey{}).(IntentBag)
	return bag
}

// IntentSatisfies reports whether bag covers a Manual/Guarded plan op that scripts already handled.
func IntentSatisfies(op PlanOp, bag IntentBag) bool {
	if bag == nil {
		return false
	}
	table := strings.TrimSpace(op.Table)
	wantKind := intentKindForOp(op)
	if wantKind == "" {
		return false
	}
	from := strings.TrimSpace(op.FromName)
	name := intentOpName(op)
	if wantKind == IntentRenameColumn && (name == "" || from == "") {
		return false
	}
	for _, in := range bag.List() {
		if !strings.EqualFold(in.Table, table) || in.Kind != wantKind {
			continue
		}
		if wantKind == IntentRenameColumn {
			if strings.EqualFold(in.FromName, from) && strings.EqualFold(in.Name, name) {
				return true
			}
			continue
		}
		if name != "" && strings.EqualFold(in.Name, name) {
			return true
		}
	}
	return false
}

func intentOpName(op PlanOp) string {
	if op.CheckName != "" {
		return strings.TrimSpace(op.CheckName)
	}
	if op.IndexName != "" {
		return strings.TrimSpace(op.IndexName)
	}
	if op.Column != nil && strings.TrimSpace(op.Column.Name) != "" {
		return strings.TrimSpace(op.Column.Name)
	}
	detail := strings.TrimSpace(op.Detail)
	lower := strings.ToLower(detail)
	for _, prefix := range []string{
		"drop foreign key ",
		"drop column ",
		"drop index ",
		"drop check ",
	} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(detail[len(prefix):])
		}
	}
	return detail
}

func intentKindForOp(op PlanOp) IntentKind {
	kind := IntentKind(strings.ToLower(strings.TrimSpace(string(op.Kind))))
	switch kind {
	case IntentRenameColumn, IntentDropColumn, IntentDropIndex, IntentDropCheck, IntentDropForeignKey:
		return kind
	case "":
		// Fall through to Detail when Kind is unset (Manual leftovers).
	default:
		// Known non-intent kinds (e.g. alter_column) must not match drop intents via Detail.
		return ""
	}
	detail := strings.ToLower(op.Detail)
	switch {
	case strings.Contains(detail, "drop column"):
		return IntentDropColumn
	case strings.Contains(detail, "drop index"):
		return IntentDropIndex
	case strings.Contains(detail, "drop check"):
		return IntentDropCheck
	case strings.Contains(detail, "drop foreign"):
		return IntentDropForeignKey
	default:
		return ""
	}
}
