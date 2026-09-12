// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

// SafetyClass classifies a plan operation for fail-closed policy.
type SafetyClass string

const (
	SafetyAuto    SafetyClass = "auto"
	SafetyGuarded SafetyClass = "guarded"
	SafetyManual  SafetyClass = "manual"
)

// OpKind is a schema plan operation kind.
type OpKind string

const (
	OpCreateTable     OpKind = "create_table"
	OpAddColumn       OpKind = "add_column"
	OpAlterColumn     OpKind = "alter_column"
	OpAddIndex        OpKind = "add_index"
	OpEnsureCheck     OpKind = "ensure_check"
	OpRenameColumn    OpKind = "rename_column"
	OpCreateJoinTable OpKind = "create_join_table"
)

// StorageKind values mirror ResolvedSpec.Migration.StorageKind plus schema-side kinds.
const (
	StoragePhysical         = "physical"
	StorageVirtualSQL       = "virtualSql"
	StorageVirtualRuntime   = "virtualRuntime"
	StorageRelationOnly     = "relationOnly"
	StorageAttachmentBacked = "attachmentBacked"
	StorageJSONMap          = "jsonMap"
)

// ColumnSpec is the normalized desired physical column for one field.
type ColumnSpec struct {
	Table            string
	Name             string // snake_case DB column
	FieldName        string // TS / Go-exported field name
	LogicalType      string
	PhysicalType     string // choysum logical physical (varchar, jsonobject, …)
	Size             *int
	NotNull          bool
	PrimaryKey       bool
	Unique           bool
	Indexed          bool
	IndexName        string // optional named index
	UniqueIndex      bool
	UniqueIndexNames []string
	Default          *string
	CheckExpr        string
	Trigram          bool
	StorageKind      string
	RenameFrom       string
	DropAfter        string
}

// DesiredSchema is the full desired DDL shape for models in one Migrate.
type DesiredSchema struct {
	Tables     map[string][]ColumnSpec // model_table → columns
	JoinTables []JoinTableSpec
}

// JoinEnd describes one side of a ManyToMany join table (FK ensure stays post-schema).
type JoinEnd struct {
	Column      string
	ReferTable  string
	ReferColumn string
}

// JoinTableSpec is a ManyToMany intermediate table to ensure (never auto-dropped).
type JoinTableSpec struct {
	Table string
	Left  JoinEnd
	Right JoinEnd
}

// LiveColumn is one inspected database column.
type LiveColumn struct {
	Name             string
	DatabaseTypeName string
	Length           *int64
	Nullable         *bool
	Default          *string
}

// LiveIndex is one inspected database index.
type LiveIndex struct {
	Name       string
	Columns    []string
	Unique     bool
	PrimaryKey bool
}

// LiveSchema is inspected live state for relevant tables.
type LiveSchema struct {
	Tables   map[string]bool                  // table exists
	Columns  map[string]map[string]LiveColumn // table → column name → meta
	Indexes  map[string][]LiveIndex           // table → indexes (name/columns/unique)
	RowCount map[string]int64                 // table → 0/1 presence indicator (any row?)
}

// PlanOp is one schema change candidate.
type PlanOp struct {
	Kind      OpKind
	Safety    SafetyClass
	Table     string
	Detail    string
	Column    *ColumnSpec
	Columns   []ColumnSpec // create_table
	FromName  string       // rename_column: live column being renamed
	IndexName string       // add_index lookup name
	CheckName string       // ensure_check constraint name
	CheckExpr string       // ensure_check expression
}

// LeftoverKind classifies leftover live objects.
type LeftoverKind string

const (
	LeftoverColumn LeftoverKind = "column"
	LeftoverIndex  LeftoverKind = "index"
)

// Leftover is a live object not present in desired (never auto-dropped).
type Leftover struct {
	Kind         LeftoverKind
	Table        string
	Name         string
	ChoysumOwned bool // index name has idx_ prefix, or column name seen in schema snapshot
}

// SchemaPlan is the diff between desired and live.
type SchemaPlan struct {
	Module   string
	Ops      []PlanOp
	Leftover []Leftover
}
