// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Author-facing metadata surface (SF-1). Engine-only SelectCtx / Flat* /
// ComputeGraph / ComputeDeps stay on orm/metadata deep imports.

export type {
  ModelCtor,
  StandardFields,
  ManyToOneMetadata,
  OneToManyMetadata,
  ManyToManyMetadata,
  BaseFieldOptions,
  SelectionItem,
  FieldType,
  RelationFieldType,
  ToManyRelationFieldType,
  M2OScalarPath,
  FieldMetadata,
  FieldPathType,
  FieldPath,
  ComputeDep,
  OnchangeTrigger,
  FieldOptions,
  ConstraintMode,
  ConstraintField,
  ConstraintOptions,
  ConstraintMeta,
  EffectiveConstraintMeta,
  ValidationIssue,
  ConstraintContext,
  ConstraintMethod,
  ConstraintMethodFn,
  InstanceConstraintMethod,
  PathDep,
  CollectionPathDep,
  OnchangeHandlerMeta,
  EffectiveOnchangeMeta,
  ModelMetadata,
  ValueType,
  ParamMetadata,
  ServiceMetadata,
} from '../orm/metadata';
export { ValidationPipelineError, MetadataStorage, getEffectiveConstraints, getEffectiveOnchange } from '../orm/metadata';
