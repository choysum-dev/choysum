// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve `@Field({ condition })` for Search/`forField` and relation load (PR-P1-F4).
 */

import BaseModel from './model';
import { MetadataStorage } from '../metadata';
import type { FieldMetadata, FieldType, ModelCtor } from '../metadata/field';
import type { ModelMetadata } from '../metadata/model';
import type { BaseQueryCondition, ForField } from '../repository/types/query';
import { andRepositoryConditions, isEmptyRepositoryCondition } from '../repository/query/condition_layer';

const RELATIONAL_CONDITION_TYPES = new Set<FieldType>([
  'ManyToOne',
  'ManyToOneRef',
  'OneToMany',
  'ManyToMany',
  'ManyToManyRef',
]);

function trimRequired(label: string, value: unknown): string {
  const trimmed = typeof value === 'string' ? value.trim() : String(value ?? '').trim();
  if (!trimmed) {
    throw new Error(`forField.${label} must be a non-empty string`);
  }
  return trimmed;
}

function resolveRelationTargetFullName(fieldMeta: FieldMetadata): string | undefined {
  const relation = fieldMeta.relation as { targetModel?: unknown } | undefined;
  const targetModel = relation?.targetModel;
  if (typeof targetModel === 'string') {
    const trimmed = targetModel.trim();
    return trimmed || undefined;
  }
  if (typeof targetModel === 'function') {
    try {
      const ctor = (targetModel as () => ModelCtor)();
      if (!ctor) return undefined;
      const meta = MetadataStorage.instance.getModelMetadata(ctor as never);
      const full = String(meta?.fullModelName || meta?.modelName || '').trim();
      return full || undefined;
    } catch {
      return undefined;
    }
  }
  return undefined;
}

function receiverModelKeys(ModelCtor: ModelCtor): Set<string> {
  const keys = new Set<string>();
  try {
    const meta = MetadataStorage.instance.getModelMetadata(ModelCtor as never);
    for (const key of [meta.fullModelName, meta.modelName, meta.name, meta.className, ModelCtor.name]) {
      const trimmed = String(key || '').trim();
      if (trimmed) keys.add(trimmed);
    }
  } catch {
    const name = String(ModelCtor?.name || '').trim();
    if (name) keys.add(name);
  }
  return keys;
}

/**
 * Evaluate a field's stored condition (static or callable). Empty when unset.
 */
export function evaluateFieldRelationalCondition(
  SourceCtor: ModelCtor,
  fieldMeta: FieldMetadata
): BaseQueryCondition | [] {
  if (typeof fieldMeta.conditionCallable === 'function') {
    try {
      const evaluated = fieldMeta.conditionCallable.call(SourceCtor);
      if (!evaluated || typeof evaluated !== 'object') {
        throw new Error('condition callable must return a QueryCondition');
      }
      return evaluated as BaseQueryCondition;
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      throw new Error(
        `Failed to evaluate condition for ${String(SourceCtor?.name || 'Model')}.${fieldMeta.name}: ${detail}`
      );
    }
  }
  if (fieldMeta.condition && typeof fieldMeta.condition === 'object') {
    return fieldMeta.condition;
  }
  return [];
}

/**
 * Resolve condition from parent model metadata + field name (relation load path).
 */
export function resolveParentFieldRelationalCondition(
  parentMeta: ModelMetadata,
  fieldName: string
): BaseQueryCondition | [] {
  const fieldMeta = parentMeta.fields?.get(fieldName) as FieldMetadata | undefined;
  if (!fieldMeta) return [];
  if (!RELATIONAL_CONDITION_TYPES.has(fieldMeta.type)) return [];
  const SourceCtor = parentMeta.type as unknown as ModelCtor;
  return evaluateFieldRelationalCondition(SourceCtor, fieldMeta);
}

/**
 * Resolve and validate `forField`, returning the meta condition to And (or []).
 */
export function resolveForFieldCondition(
  ReceiverCtor: ModelCtor,
  forField: ForField | undefined | null
): BaseQueryCondition | [] {
  if (forField == null) return [];

  const model = trimRequired('model', (forField as ForField).model);
  const field = trimRequired('field', (forField as ForField).field);

  const SourceCtor = BaseModel.resolveModelConstructor(model) as ModelCtor | undefined;
  if (!SourceCtor) {
    throw new Error(`forField.model "${model}" is not a registered model`);
  }

  let sourceMeta: ModelMetadata;
  try {
    sourceMeta = MetadataStorage.instance.getModelMetadata(SourceCtor as never);
  } catch {
    throw new Error(`forField.model "${model}" has no metadata`);
  }

  const fieldMeta = sourceMeta.fields?.get(field) as FieldMetadata | undefined;
  if (!fieldMeta) {
    throw new Error(`forField.field "${field}" does not exist on model "${model}"`);
  }
  if (!RELATIONAL_CONDITION_TYPES.has(fieldMeta.type)) {
    throw new Error(
      `forField.field "${field}" on "${model}" must be a relational field (ManyToOne/Ref, OneToMany, ManyToMany/Ref)`
    );
  }

  const targetFullName = resolveRelationTargetFullName(fieldMeta);
  const receiverKeys = receiverModelKeys(ReceiverCtor);
  if (targetFullName) {
    const targetKeys = new Set<string>([targetFullName]);
    // Also allow short name match against receiver
    const short = targetFullName.includes('.') ? targetFullName.split('.').pop()! : targetFullName;
    if (short) targetKeys.add(short);
    let overlap = false;
    for (const key of targetKeys) {
      if (receiverKeys.has(key)) {
        overlap = true;
        break;
      }
    }
    // Compare full names via receiver meta when available
    if (!overlap) {
      try {
        const recvMeta = MetadataStorage.instance.getModelMetadata(ReceiverCtor as never);
        const recvFull = String(recvMeta.fullModelName || '').trim();
        const recvShort = String(recvMeta.modelName || recvMeta.name || '').trim();
        if (recvFull && (targetFullName === recvFull || short === recvFull)) overlap = true;
        if (recvShort && (targetFullName === recvShort || short === recvShort || targetFullName.endsWith(`.${recvShort}`))) {
          overlap = true;
        }
      } catch {
        /* ignore */
      }
    }
    if (!overlap) {
      throw new Error(
        `forField { model: "${model}", field: "${field}" } targets "${targetFullName}", which does not match the searched model`
      );
    }
  }

  return evaluateFieldRelationalCondition(SourceCtor, fieldMeta);
}

/**
 * And caller condition with forField meta condition.
 */
export function mergeCallerConditionWithForField<T>(
  ReceiverCtor: ModelCtor,
  condition: BaseQueryCondition | [] | undefined,
  forField: ForField | undefined | null
): BaseQueryCondition | [] {
  const metaCondition = resolveForFieldCondition(ReceiverCtor, forField);
  if (isEmptyRepositoryCondition(metaCondition)) {
    return (condition ?? []) as BaseQueryCondition | [];
  }
  return andRepositoryConditions(metaCondition, condition);
}
