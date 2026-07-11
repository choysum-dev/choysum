// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import type { Repository } from '../repository/repository';
import type { ModelMetadata } from './model';
import type { FieldPath, ModelCtor } from './field';
import type { ObjectRecord } from '../../../utils/types';

/**
 * Supported execution modes for runtime constraint validation.
 */
export type ConstraintMode = 'create' | 'update' | 'preview';

/**
 * Field names or field paths that can trigger a constraint.
 */
export type ConstraintField<TModel extends BaseModel = BaseModel> = Extract<keyof TModel, string> | FieldPath<TModel, unknown>;

/**
 * Runtime options that control when and how a constraint handler executes.
 */
export interface ConstraintOptions<TModel extends BaseModel = BaseModel> {
  fields?: ConstraintField<TModel>[];
  preview?: boolean;
  alwaysOnCreate?: boolean;
  priority?: number;
}

/**
 * Persisted metadata for one registered constraint handler.
 */
export interface ConstraintMeta {
  method: string;
  fields: string[];
  preview: boolean;
  alwaysOnCreate: boolean;
  priority: number;
  isStatic: boolean;
}

/**
 * Constraint metadata annotated with the source that registered it.
 */
export interface EffectiveConstraintMeta extends ConstraintMeta {
  source: string;
}

/**
 * Normalized validation issue emitted by kernel, platform, constraint, or SQL checks.
 */
export interface ValidationIssue {
  scope: 'kernel' | 'platform' | 'constraint' | 'sql';
  field?: string;
  method?: string;
  code: string;
  message: string;
  severity: 'error' | 'warning';
  meta?: ObjectRecord;
}

/**
 * Execution context passed to runtime constraint handlers.
 */
export interface ConstraintContext<TModel extends BaseModel = BaseModel> {
  mode: ConstraintMode;
  model: ModelCtor<TModel> & typeof BaseModel;
  metadata: ModelMetadata;
  self?: TModel;
  current?: ObjectRecord;
  values: ObjectRecord;
  changedFields: Set<string>;
  repository: Repository;
  requestContext?: unknown;
}

/**
 * Legacy constraint handler signature (static methods).
 *
 * Static handlers receive the merged `self` record and the full
 * {@link ConstraintContext}, and are responsible for manually
 * writing back any normalization results via `writeConstraintFields`.
 *
 * @deprecated Prefer {@link InstanceConstraintMethod} for new code.
 */
export type LegacyConstraintMethod<TModel extends BaseModel = BaseModel> = (self: TModel, ctx: ConstraintContext<TModel>) => void | Promise<void>;

/**
 * Instance constraint handler signature (non-static methods).
 *
 * Instance handlers do NOT receive parameters.  The runtime engine
 * binds `this` to a draft proxy so that field reads resolve through
 * `changes → ctx.values → original self` and field writes are
 * automatically collected and written back to `ctx.values`.
 *
 * This is the **preferred signature** for all new constraint methods.
 */
export type InstanceConstraintMethod<TModel extends BaseModel = BaseModel> = (this: TModel) => void | Promise<void>;

/**
 * Constraint handler signature (compatibility union).
 *
 * Accepts both {@link LegacyConstraintMethod} and {@link InstanceConstraintMethod}.
 * The runtime engine dispatches by {@link ConstraintMeta.isStatic}.
 */
export type ConstraintMethod<TModel extends BaseModel = BaseModel> = LegacyConstraintMethod<TModel> | InstanceConstraintMethod<TModel>;

/**
 * Aggregates validation issues that should be surfaced as a single pipeline failure.
 */
export class ValidationPipelineError extends Error {
  /**
   * Issues collected during the failed validation run.
   */
  public readonly issues: ValidationIssue[];

  constructor(message: string, issues: ValidationIssue[]) {
    super(message);
    this.name = 'ValidationPipelineError';
    this.issues = issues;
  }
}
