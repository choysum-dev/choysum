// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared public types for the Onchange engine.
 *
 * This file contains the core interfaces and types exposed to engine, context, diagnostics,
 * and other shared modules. Internal implementation types such as previewProxy or writeProxy
 * remain in their owning modules.
 */

import type { OnchangeHandlerMeta } from '../../orm/metadata';
import BaseModel from '../../orm/model/model';
import type { QueryCondition, Selectable, Insertable } from '../../orm/repository/types';
import type { ObjectRecord } from '../../../utils/types';

// ============================================================================
// Utility types
// ============================================================================

type SelectableKeys<T extends BaseModel> = keyof Selectable<T>;

type RelationSelectableKeys<T extends BaseModel> = Extract<
  {
    [K in SelectableKeys<T>]: K extends keyof T
      ? T[K] extends BaseModel
        ? K
        : T[K] extends Array<infer R>
          ? R extends BaseModel
            ? K
            : never
          : never
      : never;
  }[SelectableKeys<T>],
  string
>;

type RelationModelOf<T extends BaseModel, K extends RelationSelectableKeys<T>> = K extends keyof T
  ? T[K] extends BaseModel
    ? T[K]
    : T[K] extends Array<infer R>
      ? R extends BaseModel
        ? R
        : never
      : never
  : never;

type RelationConditionEntry<T extends BaseModel, K extends RelationSelectableKeys<T>> = {
  field: K;
  condition: QueryCondition<RelationModelOf<T, K>>;
};

type CommonCondition<T extends BaseModel> = {
  field: SelectableKeys<T> & string;
  condition: QueryCondition<T>;
};

type OnchangeConditionUnion<T extends BaseModel> =
  RelationSelectableKeys<T> extends never
    ? never
    : {
        [K in RelationSelectableKeys<T>]: RelationConditionEntry<T, K>;
      };

export type OnchangeCondition<T extends BaseModel = BaseModel> =
  RelationSelectableKeys<T> extends never ? CommonCondition<T> : OnchangeConditionUnion<T>[RelationSelectableKeys<T>] | CommonCondition<T>;

// ============================================================================
// Selection condition types
// ============================================================================

/**
 * Selection field filter condition.
 *
 * Design principles:
 * 1. The selection array represents both filtered options and display order.
 * 2. The frontend automatically filters values outside metadata scope while preserving order.
 * 3. Partial disable lists are supported when needed.
 *
 * @example
 * // Scenario 1: filter only while preserving explicit order.
 * { field: 'PaymentMethod', selection: ['transfer', 'credit', 'cash', 'card'] }
 *
 * // Scenario 2: filter plus disable.
 * { field: 'DiscountType', selection: ['none', 'percentage', 'fixed', 'vip'], disabled: ['fixed', 'vip'] }
 *
 * // Scenario 3: reorder with urgent options first.
 * { field: 'Priority', selection: ['urgent', 'high', 'normal', 'low'] }
 */
export interface SelectionCondition {
  /**
   * Selection field name.
   * @example 'PaymentMethod'
   */
  field: string;

  /**
   * Filtered option list shown in this exact order.
   * - Array order becomes frontend display order.
   * - Returned values must exist in metadata-defined options.
   * - The frontend automatically filters out-of-range values while preserving order.
   *
   * @example
   * // VIP customers: show urgent options first.
   * ['urgent', 'high', 'normal', 'low']
   *
   * // Regular customers: keep the normal order.
   * ['low', 'normal']
   */
  selection: string[];

  /**
   * Optional list of disabled option values.
   * - These options remain visible but are marked disabled.
   * - Values must exist in the selection list.
   *
   * @example
   * // Show all options but disable advanced discounts.
   * selection: ['none', 'percentage', 'fixed', 'vip']
   * disabled: ['fixed', 'vip']
   */
  disabled?: string[];
}

// ============================================================================
// Message types
// ============================================================================

export type MessageLevel = 'info' | 'warn' | 'error';

export interface OnchangeMessage<T extends BaseModel = BaseModel> {
  level: MessageLevel;
  message: string;
  field?: SelectableKeys<T>;
  blocking?: boolean;
  title?: string;
}

// ============================================================================
// Value types
// ============================================================================

export type OnchangeValue<T extends BaseModel = BaseModel> = Partial<Insertable<T>>;

export type OnchangeDraft = ObjectRecord;
export type OnchangeDiagnosticMessage = ObjectRecord;

// ============================================================================
// Builder types used by OnchangeContext
// ============================================================================

export type MsgBuilder<T extends BaseModel = BaseModel> = (
  fieldOrLevel: SelectableKeys<T> | MessageLevel,
  levelOrMessage: MessageLevel | string,
  messageOrOptions?: string | { message?: string; blocking?: boolean; title?: string }
) => OnchangeMessage<T>;

export type ConditionBuilder<T extends BaseModel = BaseModel> = <K extends SelectableKeys<T> & string>(
  field: K,
  condition: K extends RelationSelectableKeys<T> ? QueryCondition<RelationModelOf<T, K>> : QueryCondition<T>
) => OnchangeCondition<T>;

/**
 * Selection condition builder.
 *
 * @param field - Field name.
 * @param selection - Filtered option list in display order.
 * @param disabled - Optional list of disabled values.
 *
 * @example
 * // Scenario 1: filter only while preserving explicit order.
 * ctx.sel('PaymentMethod', ['transfer', 'credit', 'cash', 'card'])
 *
 * // Scenario 2: filter plus disable.
 * ctx.sel('DiscountType', ['none', 'percentage', 'fixed', 'vip'], ['fixed', 'vip'])
 *
 * // Scenario 3: reorder with urgent options first.
 * ctx.sel('Priority', ['urgent', 'high', 'normal', 'low'])
 */
export type SelectionBuilder = (field: string, selection: string[], disabled?: string[]) => SelectionCondition;

/**
 * Value builder for ctx.emit(ctx.val(...)).
 *
 * @deprecated Prefer direct `this.Field = value` assignment inside onchange
 * handlers. Since the legacy `ctx` path has been fully retired, `ctx.val` is
 * no longer supported in active runtimes.
 */
export type ValBuilder<T extends BaseModel = BaseModel> = <K extends keyof Insertable<T>>(key: K, value: Insertable<T>[K]) => Pick<OnchangeValue<T>, K>;

// ============================================================================
// Context types
// ============================================================================

export type EmitArg<T extends BaseModel = BaseModel> =
  | OnchangeMessage<T>
  | OnchangeCondition<T>
  | OnchangeValue<T>
  | SelectionCondition
  | Array<OnchangeMessage<T> | OnchangeCondition<T> | OnchangeValue<T> | SelectionCondition>;

export interface OnchangeContext<T extends BaseModel = BaseModel> {
  msg: MsgBuilder<T>;
  cond: ConditionBuilder<T>;
  sel: SelectionBuilder;
  /**
   * @deprecated Prefer direct `this.Field = value` assignment.
   * Since the legacy `ctx` path has been fully retired, `ctx.val` is
   * no longer supported in active runtimes.
   */
  val: ValBuilder<T>;
  draft: T;
  changed: ReadonlySet<string>;
  emit: (payload: EmitArg<T>) => void;
}

// ============================================================================
// Engine runtime types
// ============================================================================

export interface OnchangeRunOptions {
  withCompute: boolean;
  computePreview?: (draft: OnchangeDraft, changed: Set<string>) => void | Promise<void>;
  maxIterations?: number;
  collectField?: (field: string) => void;
  loopThreshold?: number;
  stopOnError?: boolean;
}

export interface OnchangeEngineResult<T extends BaseModel = BaseModel> {
  value?: OnchangeValue<T>; // Optional.
  messages?: OnchangeMessage<T>[]; // Optional.
  condition?: OnchangeCondition<T>[]; // Optional.
  selection?: SelectionCondition[]; // Optional.
  iterations: number; // Required.
  touchedHandlers: string[]; // Required, and an empty array is still meaningful.
  computeRecomputed: string[]; // Required, and an empty array is still meaningful.
  diagnostics?: OnchangeDiagnostics; // Optional.
}

export type OnchangeResult<T extends BaseModel = BaseModel> = Pick<OnchangeEngineResult<T>, 'value' | 'messages' | 'condition' | 'selection'>;
// ============================================================================
// Diagnostics types
// ============================================================================

export interface OnchangeDiagnostics {
  missingCount: number;
  prefetchTimeMs: number;
  pathDepthMax: number;
  computeRecomputed: string[];
  readsRoots: string[];
  changedSeeds: string[];
  iterations: number;
  loopThreshold?: number;
  cachedPlanUsed?: boolean;
  messages: OnchangeDiagnosticMessage[];
}

export interface Timer {
  start(): void;
  stop(): number;
  elapsed(): number;
}

// ============================================================================
// Dependency-analysis types
// ============================================================================

export interface NeededResult {
  needed: Set<string>;
  activeHandlers: OnchangeHandlerMeta[];
}

// ============================================================================
// Prefetch plan types
// ============================================================================

export interface PathPrefetchPlan {
  rootManyToOne: Map<string, Set<string>>;
  m2oChains: Map<string, string[][]>;
  collections: Map<string, { chains: string[][] }>;
}

export interface PrefetchBatchStat {
  phase: 'm2o' | 'collection';
  level: number;
  model: string;
  fields: string[];
  batchCount: number;
  rowCount: number;
  idsSample?: string[];
}

export interface PrefetchExecStats {
  batches: PrefetchBatchStat[];
  totalBatches: number;
  totalRows: number;
}
