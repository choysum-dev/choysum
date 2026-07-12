// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata';
import { FieldOptions, FieldMetadata, FieldType } from '../metadata/field';
import type { ModelCtor } from '../metadata/field';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

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
  select?: unknown;
  column?: ObjectRecord;
  selection?: Array<{ value?: unknown; label?: unknown }>;
  targetModel?: unknown;
  relation?: unknown;
  related?: unknown;
  required?: unknown;
  indexed?: unknown;
  maxLength?: unknown;
  precision?: unknown;
  scale?: unknown;
};

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

    let validatedSelection: FieldMetadata['selection'];

    const isRelation = relationTypes.has(type);
    const hasSelect = optionBag.select !== undefined;
    let hasColumn = optionBag.column !== undefined;
    const columnCompute = asObjectRecord(optionBag.column)?.compute;
    const hasColumnCompute = !!columnCompute;

    const hasFlatStorageHints =
      optionBag.required !== undefined ||
      optionBag.indexed !== undefined ||
      optionBag.maxLength !== undefined ||
      optionBag.precision !== undefined ||
      optionBag.scale !== undefined;
    const hasRelated = optionBag.related !== undefined;
    const hasFlatContract = hasFlatStorageHints || hasRelated;

    if ((hasSelect || hasColumn) && hasFlatContract) {
      throw new Error(`@Field(${name}) flat options cannot be mixed with legacy column/select branches`);
    }

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

      if (optionBag.indexed !== undefined) {
        if (typeof optionBag.indexed !== 'boolean') {
          throw new Error(`@Field(${name}) indexed must be a boolean`);
        }
        hints.indexed = optionBag.indexed;
      }

      if (optionBag.maxLength !== undefined) {
        if (!isInt(optionBag.maxLength) || optionBag.maxLength < 1) {
          throw new Error(`@Field(${name}) maxLength must be a positive integer`);
        }
        if (type !== 'char' && type !== 'varchar') {
          throw new Error(`@Field(${name}) maxLength is only supported on char/varchar fields`);
        }
        hints.maxLength = optionBag.maxLength;
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

    if (!hasSelect && !hasColumn && normalizedStorageHints) {
      const normalizedColumn: ObjectRecord = {};
      if (normalizedStorageHints.required === true) normalizedColumn.notNull = true;
      if (normalizedStorageHints.indexed === true) normalizedColumn.index = true;
      if (normalizedStorageHints.maxLength != null) normalizedColumn.size = normalizedStorageHints.maxLength;
      if (normalizedStorageHints.precision != null) normalizedColumn.precision = normalizedStorageHints.precision;
      if (normalizedStorageHints.scale != null) normalizedColumn.scale = normalizedStorageHints.scale;
      optionBag.column = normalizedColumn;
      hasColumn = true;
    }

    // Selection-specific validation
    if (type === 'selection') {
      const selectionItems = optionBag.selection;

      // 1) selection must be a non-empty array
      if (!Array.isArray(selectionItems) || selectionItems.length === 0) {
        throw new Error(`@Field(${name}) selection type requires a non-empty selection array`);
      }

      // 2) every selection item must contain value and label
      const values = new Set<string>();
      const normalizedSelection: Array<{ value: string; label: string }> = [];
      for (const item of selectionItems) {
        if (!item || typeof item !== 'object') {
          throw new Error(`@Field(${name}) each selection item must be an object`);
        }
        if (!item.value || typeof item.value !== 'string') {
          throw new Error(`@Field(${name}) each selection item must include a string value field`);
        }
        if (!item.label || typeof item.label !== 'string') {
          throw new Error(`@Field(${name}) each selection item must include a string label field`);
        }

        const value = item.value;
        const label = item.label;

        // 3) value must be unique
        if (values.has(value)) {
          throw new Error(`@Field(${name}) duplicate selection value: ${value}`);
        }
        values.add(value);
        normalizedSelection.push({ value, label });
      }
      validatedSelection = normalizedSelection;

      // 4) selection cannot declare both select and column
      if (hasSelect && hasColumn) {
        throw new Error(`@Field(${name}) selection field cannot declare both select and column`);
      }

      // 5) selection defaults to a physical column unless select is explicitly provided
      if (!hasSelect && !hasColumn) {
        // Auto-fill column metadata later
      }
    }

    // Mutual exclusion validation: select vs column (non-selection types)
    if (type !== 'selection') {
      if (hasSelect && hasColumn) {
        throw new Error(`@Field(${name}) select branch cannot declare column`);
      }
      if (hasSelect && hasColumnCompute) {
        throw new Error(`@Field(${name}) select branch cannot declare compute`);
      }
    }

    // ManyToOneRef default physical column: char(20) + index (when column/select is absent)
    if (type === 'ManyToOneRef' && !hasSelect && !hasColumn) {
      optionBag.column = {
        ...(optionBag.column || {}),
        size: 20,
        index: true,
      };
      hasColumn = true;
    }

    // ManyToManyRef default physical column: jsonobject (actual physical mapping is decided by migrator)
    if (type === 'ManyToManyRef' && !hasSelect && !hasColumn) {
      // Do not force a default value; encode/decode layer will fall back to [] on read
      optionBag.column = {
        ...(optionBag.column || {}),
      };
      hasColumn = true;
    }

    // Validate targetModel for ref types
    if (type === 'ManyToOneRef' || type === 'ManyToManyRef') {
      if (!optionBag.targetModel) {
        throw new Error(`@Field(${name}) ${type} requires targetModel`);
      }
    }

    // Decimal option validation (DDL stays NUMERIC(38,18); scale metadata is validated here)
    if (type === 'decimal') {
      const branch = hasSelect ? asObjectRecord(optionBag.select) : hasColumn ? asObjectRecord(optionBag.column) : undefined;
      const p = branch?.precision;
      const s = branch?.scale;
      const sf = branch?.scaleField;

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

      // scale and scaleField are mutually exclusive
      if (s != null && sf != null) {
        throw new Error(`@Field(${name}) decimal cannot declare both scale and scaleField`);
      }

      // scaleField must be a string field name on the same model
      if (sf != null && typeof sf !== 'string') {
        throw new Error(`@Field(${name}) decimal.scaleField must be a string field name on the same model`);
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
      if ((type === 'OneToMany' || type === 'ManyToMany') && hasColumnCompute) {
        throw new Error(`@Field(${name}) ${type} does not support compute`);
      }
    }

    // Compute validation
    if (hasColumnCompute) {
      const compute = asObjectRecord(columnCompute);
      const deps = compute?.deps;

      if (!Array.isArray(deps) || deps.length === 0) {
        throw new Error(`@Field(${name}) compute.deps must not be empty`);
      }

      const store = compute?.store !== false;
      const searchable = compute?.searchable === true;
      const runAs = compute?.runAs;

      const inverse = typeof compute?.inverse === 'string' ? compute.inverse.trim() : '';
      const hasInverse = inverse.length > 0;
      const search = typeof compute?.search === 'string' ? compute.search.trim() : '';
      const hasSearch = search.length > 0;

      if (runAs != null && runAs !== 'user' && runAs !== 'sudo') {
        throw new Error(`@Field(${name}) compute.runAs only supports user|sudo`);
      }

      if (store && compute?.searchable != null) {
        throw new Error(`@Field(${name}) compute.searchable should not be set when store=true`);
      }

      if (!store && hasInverse) {
        throw new Error(`@Field(${name}) compute.inverse is not allowed when store=false`);
      }

      if (!store && searchable && !hasSearch) {
        throw new Error(`@Field(${name}) compute.search is required when store=false and searchable=true`);
      }

      if (!store && !searchable && hasSearch) {
        throw new Error(`@Field(${name}) compute.search is not allowed when store=false and searchable=false`);
      }

      if (compute?.inverse != null && !hasInverse) {
        throw new Error(`@Field(${name}) compute.inverse must not be blank`);
      }

      if (compute?.search != null && !hasSearch) {
        throw new Error(`@Field(${name}) compute.search must not be blank`);
      }
    }

    // Auto-fill column metadata for scalar fields without select/column
    const autoColumnScalar = !isRelation && scalarTypes.has(type) && !hasSelect && !hasColumn;
    const autoColumnManyToOne = type === 'ManyToOne' && !hasSelect && !hasColumn;

    const meta: FieldMetadata = { name, type };

    // Persist selection metadata before column/select handling
    if (type === 'selection') {
      meta.selection = validatedSelection;
    }

    // Persist targetModel metadata for ref types
    if (type === 'ManyToOneRef' || type === 'ManyToManyRef') {
      meta.targetModel = optionBag.targetModel as FieldMetadata['targetModel'];
    }

    if (optionBag.relation) meta.relation = optionBag.relation as FieldMetadata['relation'];
    if (hasColumn) meta.column = optionBag.column as FieldMetadata['column'];
    else if (autoColumnScalar || autoColumnManyToOne) meta.column = {};

    if (normalizedRelated) meta.related = normalizedRelated;
    if (normalizedStorageHints) meta.storageHints = normalizedStorageHints;

    if (hasSelect) meta.select = optionBag.select as FieldMetadata['select'];

    // Write metadata
    const ctor = target.constructor as ModelCtor<BaseModel> & typeof BaseModel;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const fields = new Map(prev?.fields ?? []);
    fields.set(name, { ...(fields.get(name) || {}), ...meta });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      fields,
    });
  };
}
