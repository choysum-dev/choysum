// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata';
import { FieldOptions, FieldMetadata, FieldType, type SelectionItem, type SelectionDeclaration } from '../metadata/field';
import type { ModelCtor } from '../metadata/field';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';
import { isTermReference, type TermReference } from '../../i18n';

const scalarTypes = new Set<FieldType>([
  'char',
  'varchar',
  'text',
  'binary',
  'image',
  'int',
  'bigint',
  'number',
  'decimal',
  'boolean',
  'datetime',
  'date',
  'time',
  'jsonobject',
  'selection',
  'ManyToOneRef',
  'ManyToManyRef',
]);
const relationTypes = new Set<FieldType>(['ManyToOne', 'OneToMany', 'ManyToMany']);

type FieldDecoratorOptionBag = {
  type?: FieldType;
  string?: unknown;
  select?: unknown;
  column?: unknown;
  /** Static array | method name | callable (see SelectionDeclaration). */
  selection?: SelectionDeclaration;
  targetModel?: unknown;
  relation?: unknown;
  related?: unknown;
  required?: unknown;
  notNull?: unknown;
  indexed?: unknown;
  index?: unknown;
  size?: unknown;
  precision?: unknown;
  scale?: unknown;
  scaleField?: unknown;
  primaryKey?: unknown;
  unique?: unknown;
  uniqueIndex?: unknown;
  checkConstraint?: unknown;
  default?: unknown;
  round?: unknown;
};

function normalizeFieldString(name: string, value: unknown): { string?: string; stringText?: TermReference } {
  if (value === undefined || value === null) {
    return {};
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      throw new Error(`@Field(${name}) string must be a non-empty string or term reference`);
    }
    return { string: trimmed };
  }
  if (isTermReference(value)) {
    const src = typeof value.src === 'string' ? value.src.trim() : '';
    if (!src) {
      throw new Error(`@Field(${name}) string term reference requires a non-empty src`);
    }
    return { string: src, stringText: { ...value } };
  }
  throw new Error(`@Field(${name}) string must be a string or term reference`);
}

function toFieldDecoratorOptionBag(value: unknown): FieldDecoratorOptionBag {
  const record = asObjectRecord(value);
  return (record || {}) as FieldDecoratorOptionBag;
}

/**
 * Declares model field metadata for persistence, relations, selections, and compute behavior.
 *
 * @param options Field metadata to register on the decorated property.
 * @returns A property decorator that records the field definition in metadata storage.
 */
export function Field<T extends BaseModel, R extends keyof T = keyof T, TJoin extends BaseModel = BaseModel, TTarget extends BaseModel = BaseModel>(
  options: FieldOptions<T, R, TJoin, TTarget>
): PropertyDecorator {
  return function (target: Object, propertyKey: string | symbol) {
    const name = propertyKey as string;
    const optionBag = toFieldDecoratorOptionBag(options);
    const type = optionBag.type;
    if (!type) throw new Error(`@Field(${name}) missing type`);

    const hasLegacySelect = optionBag.select !== undefined;
    const hasLegacyColumn = optionBag.column !== undefined;
    if (hasLegacySelect || hasLegacyColumn) {
      throw new Error(`@Field(${name}) column/select syntax is forbidden; use flat field options and behavior decorators`);
    }

    let validatedSelection: FieldMetadata['selection'];
    let selectionKind: FieldMetadata['selectionKind'];
    let selectionMethod: FieldMetadata['selectionMethod'];
    let selectionCallable: FieldMetadata['selectionCallable'];
    let normalizedColumn: ObjectRecord | undefined;

    const isRelation = relationTypes.has(type);

    const hasFlatStorageHints =
      optionBag.required !== undefined ||
      optionBag.notNull !== undefined ||
      optionBag.indexed !== undefined ||
      optionBag.index !== undefined ||
      optionBag.size !== undefined ||
      optionBag.precision !== undefined ||
      optionBag.scale !== undefined;
    const hasFlatColumnOptions =
      optionBag.primaryKey !== undefined ||
      optionBag.unique !== undefined ||
      optionBag.uniqueIndex !== undefined ||
      optionBag.checkConstraint !== undefined ||
      optionBag.default !== undefined ||
      optionBag.round !== undefined ||
      optionBag.scaleField !== undefined;
    const hasRelated = optionBag.related !== undefined;

    let normalizedStorageHints: FieldMetadata['storageHints'] | undefined;
    if (hasFlatStorageHints) {
      const hints: NonNullable<FieldMetadata['storageHints']> = {};
      const isInt = (x: unknown): x is number => typeof x === 'number' && Number.isInteger(x);

      if (optionBag.required !== undefined) {
        if (typeof optionBag.required !== 'boolean') {
          throw new Error(`@Field(${name}) required must be a boolean`);
        }
        hints.required = optionBag.required;
      }

      if (optionBag.notNull !== undefined) {
        if (typeof optionBag.notNull !== 'boolean') {
          throw new Error(`@Field(${name}) notNull must be a boolean`);
        }
        hints.required = optionBag.notNull;
      }

      if (optionBag.indexed !== undefined) {
        if (typeof optionBag.indexed !== 'boolean') {
          throw new Error(`@Field(${name}) indexed must be a boolean`);
        }
        hints.indexed = optionBag.indexed;
      }

      if (optionBag.index !== undefined) {
        if (typeof optionBag.index !== 'boolean' && typeof optionBag.index !== 'string') {
          throw new Error(`@Field(${name}) index must be a boolean or string`);
        }
        hints.indexed = optionBag.index === true || (typeof optionBag.index === 'string' && optionBag.index.trim().length > 0);
      }

      if (optionBag.size !== undefined) {
        if (!isInt(optionBag.size) || optionBag.size < 1) {
          throw new Error(`@Field(${name}) size must be a positive integer`);
        }
        if (type !== 'char' && type !== 'varchar' && type !== 'selection' && type !== 'ManyToOneRef' && type !== 'ManyToOne') {
          throw new Error(`@Field(${name}) size is only supported on char/varchar/selection/ManyToOneRef/ManyToOne fields`);
        }
        hints.size = optionBag.size;
      }

      if (optionBag.precision !== undefined) {
        if (!isInt(optionBag.precision) || optionBag.precision < 1 || optionBag.precision > 38) {
          throw new Error(`@Field(${name}) precision must be in 1..38`);
        }
        if (type !== 'decimal') {
          throw new Error(`@Field(${name}) precision is only supported on decimal fields`);
        }
        hints.precision = optionBag.precision;
      }

      if (optionBag.scale !== undefined) {
        if (!isInt(optionBag.scale) || optionBag.scale < 0 || optionBag.scale > 18) {
          throw new Error(`@Field(${name}) scale must be in 0..18`);
        }
        if (type !== 'decimal') {
          throw new Error(`@Field(${name}) scale is only supported on decimal fields`);
        }
        hints.scale = optionBag.scale;
      }

      if (hints.precision != null && hints.scale != null && hints.scale > hints.precision) {
        throw new Error(`@Field(${name}) scale must not be greater than precision (${hints.scale} > ${hints.precision})`);
      }

      normalizedStorageHints = hints;
    }

    let normalizedRelated: FieldMetadata['related'] | undefined;
    if (hasRelated) {
      const related = asObjectRecord(optionBag.related);
      const path = String(related?.path || '').trim();
      if (!path) {
        throw new Error(`@Field(${name}) related.path must be a non-empty string`);
      }

      if (related?.store != null && typeof related.store !== 'boolean') {
        throw new Error(`@Field(${name}) related.store must be a boolean when provided`);
      }

      if (related?.deps != null && !Array.isArray(related.deps)) {
        throw new Error(`@Field(${name}) related.deps must be a string array when provided`);
      }

      const deps = Array.isArray(related?.deps) ? [...new Set(related.deps.map(dep => String(dep || '').trim()).filter(Boolean))] : undefined;

      normalizedRelated = {
        path,
        store: related?.store === true,
        ...(deps && deps.length ? { deps } : {}),
      };
    }

    if (normalizedStorageHints || hasFlatColumnOptions) {
      normalizedColumn = {};
      if (normalizedStorageHints?.required === true) normalizedColumn.notNull = true;
      if (normalizedStorageHints?.indexed === true) normalizedColumn.index = true;
      if (normalizedStorageHints?.size != null) normalizedColumn.size = normalizedStorageHints.size;
      if (normalizedStorageHints?.precision != null) normalizedColumn.precision = normalizedStorageHints.precision;
      if (normalizedStorageHints?.scale != null) normalizedColumn.scale = normalizedStorageHints.scale;

      if (optionBag.index !== undefined) normalizedColumn.index = optionBag.index;
      if (optionBag.primaryKey !== undefined) normalizedColumn.primaryKey = optionBag.primaryKey;
      if (optionBag.unique !== undefined) normalizedColumn.unique = optionBag.unique;
      if (optionBag.uniqueIndex !== undefined) normalizedColumn.uniqueIndex = optionBag.uniqueIndex;
      if (optionBag.checkConstraint !== undefined) normalizedColumn.checkConstraint = optionBag.checkConstraint;
      if (optionBag.default !== undefined) normalizedColumn.default = optionBag.default;
      if (optionBag.round !== undefined) normalizedColumn.round = optionBag.round;
      if (optionBag.scaleField !== undefined) normalizedColumn.scaleField = optionBag.scaleField;
    }

    const hasColumn = normalizedColumn !== undefined;

    // Selection-specific validation (static array | method name | callable)
    if (type === 'selection') {
      const selectionRaw = optionBag.selection;

      if (typeof selectionRaw === 'function') {
        selectionKind = 'dynamic';
        selectionCallable = selectionRaw as FieldMetadata['selectionCallable'];
      } else if (typeof selectionRaw === 'string') {
        const method = selectionRaw.trim();
        if (!method) {
          throw new Error(`@Field(${name}) selection method name must be a non-empty string`);
        }
        selectionKind = 'dynamic';
        selectionMethod = method;
      } else if (Array.isArray(selectionRaw)) {
        if (selectionRaw.length === 0) {
          throw new Error(`@Field(${name}) selection type requires a non-empty selection array`);
        }

        const values = new Set<string>();
        const normalizedSelection: SelectionItem[] = [];
        for (const item of selectionRaw) {
          if (!item || typeof item !== 'object') {
            throw new Error(`@Field(${name}) each selection item must be an object`);
          }
          if (!item.value || typeof item.value !== 'string') {
            throw new Error(`@Field(${name}) each selection item must include a string value field`);
          }
          if ((item as { labelText?: unknown }).labelText != null) {
            throw new Error(
              `@Field(${name}) selection labelText is forbidden; use label: _lt('…') when the option should translate`
            );
          }

          const value = item.value;
          if (values.has(value)) {
            throw new Error(`@Field(${name}) duplicate selection value: ${value}`);
          }
          values.add(value);

          const labelRaw = (item as { label?: unknown }).label;
          if (isTermReference(labelRaw)) {
            const src = String(labelRaw.src || '').trim();
            if (!src) {
              throw new Error(`@Field(${name}) each selection item _lt label must include a non-empty src`);
            }
            normalizedSelection.push({ value, label: src, labelText: labelRaw });
            continue;
          }
          if (!labelRaw || typeof labelRaw !== 'string') {
            throw new Error(
              `@Field(${name}) each selection item label must be a string or TermReference from _lt(...)`
            );
          }
          const label = labelRaw.trim();
          if (!label) {
            throw new Error(`@Field(${name}) each selection item must include a non-empty string label`);
          }
          normalizedSelection.push({ value, label });
        }
        validatedSelection = normalizedSelection;
        selectionKind = 'static';
      } else {
        throw new Error(
          `@Field(${name}) selection must be a non-empty array, method name string, or () => SelectionItem[] callable`
        );
      }
    }

    // ManyToOneRef default physical column: char(20) + index when no explicit storage hints are provided.
    if (type === 'ManyToOneRef' && !hasColumn) {
      normalizedColumn = {
        ...(normalizedColumn || {}),
        size: 20,
        index: true,
      };
    }

    // ManyToManyRef default physical column: jsonobject (actual physical mapping is decided by migrator).
    if (type === 'ManyToManyRef' && !hasColumn) {
      // Do not force a default value; encode/decode layer will fall back to [] on read
      normalizedColumn = {
        ...(normalizedColumn || {}),
      };
    }

    // Validate targetModel for ref types
    if (type === 'ManyToOneRef' || type === 'ManyToManyRef') {
      if (optionBag.targetModel !== undefined) {
        throw new Error(`@Field(${name}) ${type} requires relation.targetModel (top-level targetModel is not supported)`);
      }

      const relation = asObjectRecord(optionBag.relation);
      if (!relation?.targetModel) {
        throw new Error(`@Field(${name}) ${type} requires relation.targetModel`);
      }
    }

    if (optionBag.scaleField !== undefined && type !== 'decimal') {
      throw new Error(`@Field(${name}) scaleField is only supported on decimal fields`);
    }

    // Decimal option validation (DDL stays NUMERIC(38,18); scale metadata is validated here)
    if (type === 'decimal') {
      const branch = normalizedColumn;
      const p = branch?.precision;
      const s = branch?.scale;

      const isInt = (x: unknown): x is number => typeof x === 'number' && Number.isInteger(x);

      // precision: 1..38 (business-level validation only)
      if (p != null) {
        if (!isInt(p) || p < 1 || p > 38) {
          throw new Error(`@Field(${name}) decimal.precision must be in 1..38 (current=${p})`);
        }
      }

      // scale: 0..18 (aligned with DDL upper bound)
      if (s != null) {
        if (!isInt(s) || s < 0 || s > 18) {
          throw new Error(`@Field(${name}) decimal.scale must be in 0..18 (current=${s})`);
        }
      }

      // If both precision and scale are set, require scale <= precision
      if (p != null && s != null && s > p) {
        throw new Error(`@Field(${name}) decimal.scale must not be greater than precision (${s} > ${p})`);
      }

      // scaleField: validates that the companion column name is a non-empty string (only for decimal).
      const sf = optionBag.scaleField;
      if (sf !== undefined) {
        if (typeof sf !== 'string' || !sf.trim()) {
          throw new Error(`@Field(${name}) scaleField must be a non-empty string`);
        }
        if (sf !== sf.trim()) {
          throw new Error(`@Field(${name}) scaleField must not contain leading or trailing whitespace`);
        }
      }
    }

    // Relation constraints
    if (isRelation) {
      if (!optionBag.relation) {
        throw new Error(`@Field(${name}) ${type} requires relation`);
      }
      if ((type === 'OneToMany' || type === 'ManyToMany') && hasColumn) {
        throw new Error(`@Field(${name}) ${type} does not allow column`);
      }
    }

    // Auto-fill column metadata for scalar fields without explicit storage options.
    const autoColumnScalar = !isRelation && scalarTypes.has(type) && !normalizedColumn && name !== 'DisplayName';
    const autoColumnManyToOne = type === 'ManyToOne' && !normalizedColumn;

    const meta: FieldMetadata = { name, type };

    const fieldString = normalizeFieldString(name, optionBag.string);
    if (fieldString.string !== undefined) meta.string = fieldString.string;
    if (fieldString.stringText !== undefined) meta.stringText = fieldString.stringText;

    // Persist selection metadata before final storage/relation defaults.
    if (type === 'selection') {
      if (selectionKind) meta.selectionKind = selectionKind;
      if (validatedSelection) meta.selection = validatedSelection;
      if (selectionMethod) meta.selectionMethod = selectionMethod;
      if (selectionCallable) meta.selectionCallable = selectionCallable;
    }

    if (optionBag.relation) meta.relation = optionBag.relation as FieldMetadata['relation'];
    if (normalizedColumn) meta.column = normalizedColumn as FieldMetadata['column'];
    else if (autoColumnScalar || autoColumnManyToOne) meta.column = {};

    if (normalizedRelated) meta.related = normalizedRelated;
    if (normalizedStorageHints) meta.storageHints = normalizedStorageHints;

    // Write metadata
    const ctor = target.constructor as ModelCtor<BaseModel> & typeof BaseModel;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const existingCompute = prev?.computeHandlers?.get(name);
    const existingSqlCompute = prev?.sqlComputeHandlers?.get(name);

    if (existingCompute?.store === false || !!existingSqlCompute) {
      delete meta.column;
    }

    const fields = new Map(prev?.fields ?? []);
    fields.set(name, { ...(fields.get(name) || {}), ...meta });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      fields,
    });
  };
}
