// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { FieldMetadata } from './field';
import { ServiceMetadata } from './service';
import type { ConstraintMeta } from './constraint';
import { OrderBy } from '../repository';
import type BaseModel from '../model/model';
import type { ObjectRecord } from '../../../utils/types';

type ModelOrderBy = OrderBy<ObjectRecord>;

/**
 * Parsed dependency that starts from a scalar or ManyToOne root and stores the remaining chain.
 */
export type PathDep = { root: string; chain: string[] };

/**
 * Parsed dependency that starts from a collection root and stores the remaining chain.
 */
export type CollectionPathDep = { collection: string; chain: string[] };

/**
 * Metadata recorded for one onchange handler.
 */
export interface OnchangeHandlerMeta {
  method: string;
  triggers: string[]; // Trigger fields.
  priority?: number; // Optional; defaults to 100.
  reads?: string[]; // Optional read-only dependencies, either scalar fields or paths.
}

/**
 * Parsed compute dependency categories kept in sync with compute/parser.ts.
 */
export type ParsedDep =
  | { kind: 'scalar'; field: string }
  | { kind: 'path'; root: string; chain: string[] }
  | { kind: 'collection'; collection: string }
  | { kind: 'collectionPath'; collection: string; chain: string[] };

/**
 * Parent-model trigger rules for the reverse trigger index.
 *
 * When child-model fields change or child rows are added or removed, the reverse
 * index finds every parent-model compute field that depends on the child model
 * and triggers recomputation for the parent record.
 *
 * @example
 * // Parent model SaleOrder depends on Lines.Quantity.
 * // Register it in SaleOrderLine.computeGraph.reverseComputeIndex:
 * {
 *   parentModelCtor: SaleOrder,
 *   parentComputeField: 'TotalAmount',
 *   inverseField: 'OrderId',
 *   triggerMode: 'field-change'
 * }
 */
export interface ParentComputeTrigger {
  /** Parent model constructor, stored so the parent model can be accessed directly. */
  parentModelCtor: typeof BaseModel;

  /** Parent compute field that must be recomputed. */
  parentComputeField: string;

  /** Foreign-key field on the child model that points to the parent record, such as 'OrderId'. */
  inverseField: string;

  /** Collection field name on the parent model, such as 'Lines'. */
  collectionField: string;

  /**
   * Trigger mode:
   * - 'field-change': child field changes trigger recomputation and map to collectionPath deps.
   * - 'lifecycle': child row create/delete events trigger recomputation and map to collection root deps.
   * - 'membership-change': child-parent membership changes trigger recomputation when inverseField changes.
   */
  triggerMode: 'field-change' | 'lifecycle' | 'membership-change';
}

// Compute graph structure.
export interface ModelComputeGraph {
  order: string[];
  reverseDeps: Map<string, Set<string>>;
  parsedDeps: Map<string, ParsedDep[]>;
  collectionTouchIndex: Map<string, Set<string>>;
  computeFields: Set<string>;
  persistedComputeFields?: Set<string>;
  virtualComputeFields?: Set<string>;
  orderIndex: Map<string, number>;
  fastReverseDeps: Map<string, string[]>;
  fastPersistReverseDeps?: Map<string, string[]>;

  // NEW: Scalar dependencies that can be queried directly from the primary table for each compute field.
  // This includes scalar deps plus the in-model portion of path root and chain segments.
  computeScalarDeps?: Map<string, Set<string>>;

  // NEW: Path dependency list for each compute field. Only path deps are retained for later planBuilder merging.
  computePathDeps?: Map<string, PathDep[]>;

  // NEW: Collection-path dependency list for each compute field. Only collectionPath deps are retained for later planBuilder merging.
  computeCollectionPathDeps?: Map<string, CollectionPathDep[]>;

  /**
   * NEW: Reverse trigger index when this model acts as a child model.
   *
   * Key: a field name such as 'Quantity' or '__lifecycle' for lifecycle events.
   * Value: all parent-model trigger rules that depend on that field.
   *
   * @remarks
   * - Lazily built in buildComputeGraph on the first getMetadata call.
   * - Walks all registered models and finds collection deps on the current model in computeGraph.parsedDeps.
   * - Aggregated by field name for fast lookup.
   *
   * @example
   * // Reverse index for the SaleOrderLine model:
   * {
   *   'Quantity': [
   *     { parentModelCtor: SaleOrder, parentComputeField: 'TotalAmount', inverseField: 'OrderId', triggerMode: 'field-change' },
   *     { parentModelCtor: SaleOrder, parentComputeField: 'AvgQuantity', inverseField: 'OrderId', triggerMode: 'field-change' }
   *   ],
   *   '__lifecycle': [
   *     { parentModelCtor: SaleOrder, parentComputeField: 'LineCount', inverseField: 'OrderId', triggerMode: 'lifecycle' }
   *   ]
   * }
   */
  reverseComputeIndex?: Map<string, ParentComputeTrigger[]>;
}

/**
 * Full runtime metadata recorded for a model class.
 */
export interface ModelMetadata {
  name: string;
  modelName?: string;
  fullModelName?: string;
  application?: string;
  className?: string;
  tableName: () => string;
  type: Function & { name: string };
  fields: Map<string, FieldMetadata>;
  services: Map<string, ServiceMetadata>;
  orderBy?: ModelOrderBy | ModelOrderBy[];
  softDelete?: boolean;

  /**
   * Enables default company filtering (P2-1).
   * - true: inject the company filter by default in the repository layer.
   * - false: do not inject it, for example in global or security-context flows.
   */
  companyScoped?: boolean;

  // Added migration and read-only controls.
  autoMigrate?: boolean;
  readonly?: boolean;

  // Only parentField remains configurable; ParentPath stays fixed.
  parentField?: string;

  // Onchange
  onchangeHandlers?: OnchangeHandlerMeta[]; // May be an empty array; merge logic must deduplicate by method.

  // Constraint
  constraintHandlers?: ConstraintMeta[]; // May be an empty array; merge logic must deduplicate by method.

  // Compute
  computeGraph?: ModelComputeGraph;
}
