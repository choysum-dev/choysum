// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import type { RelationOperations, RelationItem } from '../repository/types';
import type { RelationFieldType } from '../metadata';
import type { ModelCtor } from '../metadata/field';
import type { ObjectRecord } from '../../../utils/types';
import { asObjectRecord } from '../../../utils/object';

export type { RelationFieldType };

export type RelationModelCtor<T extends BaseModel = BaseModel> = ModelCtor<T> & typeof BaseModel;

export type OneToManyRelationConfig = {
  targetModel: () => RelationModelCtor;
  inverseField: string;
};

export type ManyToManyRelationConfig = {
  joinModel: () => RelationModelCtor;
  targetModel: () => RelationModelCtor;
  joinField: string;
  inverseJoinField: string;
};

export type ManyToManyRelationJoinConfig = Pick<ManyToManyRelationConfig, 'joinModel' | 'joinField'>;

export function resolveOneToManyRelationConfig(value: unknown): OneToManyRelationConfig | undefined {
  const relation = asObjectRecord(value);
  const targetModel = relation?.targetModel;
  const inverseField = relation?.inverseField;
  if (typeof targetModel !== 'function' || typeof inverseField !== 'string' || !inverseField) return undefined;
  return {
    targetModel: targetModel as OneToManyRelationConfig['targetModel'],
    inverseField,
  };
}

export function resolveManyToManyRelationConfig(value: unknown): ManyToManyRelationConfig | undefined {
  const relation = asObjectRecord(value);
  const joinModel = relation?.joinModel;
  const targetModel = relation?.targetModel;
  const joinField = relation?.joinField;
  const inverseJoinField = relation?.inverseJoinField;
  if (
    typeof joinModel !== 'function' ||
    typeof targetModel !== 'function' ||
    typeof joinField !== 'string' ||
    !joinField ||
    typeof inverseJoinField !== 'string' ||
    !inverseJoinField
  ) {
    return undefined;
  }
  return {
    joinModel: joinModel as ManyToManyRelationConfig['joinModel'],
    targetModel: targetModel as ManyToManyRelationConfig['targetModel'],
    joinField,
    inverseJoinField,
  };
}

export function resolveManyToManyRelationJoinConfig(value: unknown): ManyToManyRelationJoinConfig | undefined {
  const relation = asObjectRecord(value);
  const joinModel = relation?.joinModel;
  const joinField = relation?.joinField;
  if (typeof joinModel !== 'function' || typeof joinField !== 'string' || !joinField) return undefined;
  return {
    joinModel: joinModel as ManyToManyRelationJoinConfig['joinModel'],
    joinField,
  };
}

/**
 * ManyToOne relation operation.
 */
export interface ManyToOneOperation<T extends BaseModel = BaseModel> {
  /** Relation field name, which is also the foreign-key field on the parent table. */
  fieldName: string;
  /** Relation type. */
  type: 'ManyToOne';
  /** Target model class. */
  targetModel: ModelCtor<T> & typeof BaseModel;
  /** Field value, which may be an Id, a partial object, a model instance, or null to detach the relation. */
  value: RelationItem<T> | null;
}

/**
 * OneToMany relation operation.
 */
export interface OneToManyOperation<T extends BaseModel = BaseModel> {
  /** Field name, which is the relation property on the parent table. */
  fieldName: string;
  /** Relation type. */
  type: 'OneToMany';
  /** Target model class for the child table. */
  targetModel: ModelCtor<T> & typeof BaseModel;
  /** Foreign-key field on the child table that points back to the parent table. */
  inverseField: string;
  /** Relation operations. Arrays mean replace; objects mean create, update, and delete patches. */
  operations: RelationOperations<T> | RelationItem<T>[];
}

/**
 * ManyToMany relation operation.
 */
export interface ManyToManyOperation<T extends BaseModel = BaseModel, J extends BaseModel = BaseModel> {
  /** Relation type. */
  type: 'ManyToMany';
  /** Field name, which is the relation property on the parent table. */
  fieldName: string;
  /** Join-table model class. */
  joinModel: ModelCtor<J> & typeof BaseModel;
  /** Target model class. */
  targetModel: ModelCtor<T> & typeof BaseModel;
  /** Join-table field that references the parent table. */
  joinField: string;
  /** Join-table field that references the target table. */
  inverseJoinField: string;
  /** Relation operations. Arrays mean replace; objects mean create, update, and delete patches. */
  operations: RelationOperations<T> | RelationItem<T>[];
}

/**
 * Union of supported relation operations.
 */
export type RelationOperation<T extends BaseModel = BaseModel> = ManyToOneOperation<T> | OneToManyOperation<T> | ManyToManyOperation<T>;

/**
 * Extracted ToMany relation payloads.
 */
export interface ExtractedRelations {
  /** OneToMany relation operations. */
  oneToManyRelations: OneToManyOperation[];
  /** ManyToMany relation operations. */
  manyToManyRelations: ManyToManyOperation[];
  /**
   * Collection relation field names that were explicitly present in the input object.
   * Used to trigger collection-based compute flows for OneToMany and ManyToMany relations.
   */
  touchedCollections?: Set<string>;
}

/**
 * Create or update preprocessing result.
 */
export interface PrepareResult<T = unknown> {
  /** Processed value object. */
  processedValue: ObjectRecord;
  /** Extracted relation operations. */
  relations: ExtractedRelations;
}

/**
 * Relation-processing result.
 */
export interface RelationProcessingResult<T extends BaseModel = BaseModel> {
  /** Number of successfully processed entities. */
  affectedCount: number;
  /** Target entity Ids after processing. For M2M this refers to target-table Ids. */
  entityIds: string[];
  /** Errors raised while processing. */
  errors: Error[];
  /** Target model class for the relation operation. */
  targetModel: ModelCtor<T> & typeof BaseModel;
  /** Relation type that was processed. */
  relationType: RelationFieldType;
}

/**
 * Batch-processing result.
 */
export interface BatchProcessingResult {
  /** Successfully processed items. */
  success: Array<{
    entityId: string;
    targetModel: string;
    joinModel?: string;
    field?: string;
  }>;

  /** Errors produced during processing. */
  errors: Array<{
    error: Error;
    targetModel: string;
    joinModel?: string;
    entityId?: string;
    field?: string;
  }>;

  /** Summary of the batch-processing result. */
  summary: {
    totalOperations: number;
    successfulOperations: number;
    failedOperations: number;
    relationType: RelationFieldType;
  };
}

/**
 * Supported relation-array mutation methods.
 */
export enum RelationArrayMethod {
  PUSH = 'push',
  POP = 'pop',
  SHIFT = 'shift',
  UNSHIFT = 'unshift',
  SPLICE = 'splice',
  SORT = 'sort',
  REVERSE = 'reverse',
  SET = 'set',
}

/**
 * Relation-array change operation.
 */
export interface RelationChangeOperation {
  method: RelationArrayMethod;
  args: unknown[];
  timestamp: number;
  snapshot?: unknown[];
}

/**
 * Relation change collection.
 */
export interface RelationChangesCollection {
  [fieldName: string]: RelationChangeOperation[];
}
