// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata, ModelMetadata } from '../../metadata';
import { assertSafeInt } from '../../utils/int-guard';
import type { Entity } from '../types';
import { normalizeDecimalByMeta } from '@/core/utils/decimal';
import { asObjectRecord, hasOwnKey } from '../../../../utils/object';
import type { ObjectRecord } from '../../../../utils/types';
import { _t } from '@/core/service/i18n_binder';
import { resolveDecimalScaleForWrite } from '../projection/row_codec';

export type ValidationMode = 'create' | 'update' | 'preview';

export type KernelValidationRule = 'int' | 'selection' | 'required' | 'decimal' | 'relationShape';

export type KernelIssueCode =
  | 'kernel_int_invalid'
  | 'kernel_selection_invalid'
  | 'kernel_required_missing'
  | 'kernel_required_null'
  | 'kernel_decimal_invalid'
  | 'kernel_relation_shape_invalid';

export class KernelValidationError extends Error {
  public readonly code: KernelIssueCode;
  public readonly field?: string;
  public readonly detail?: ObjectRecord;

  constructor(code: KernelIssueCode, message: string, options?: { field?: string; detail?: ObjectRecord }) {
    super(message);
    this.name = 'KernelValidationError';
    this.code = code;
    this.field = options?.field;
    this.detail = options?.detail;
  }
}

export function validateIntFields(meta: ModelMetadata, vals: Entity) {
  const values = asObjectRecord(vals);
  if (!values) return;

  meta.fields.forEach((fieldMeta, fieldName) => {
    if (fieldMeta.type === 'int' && hasOwnKey(values, fieldName)) {
      try {
        assertSafeInt(values[fieldName], fieldName);
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : _t('Field "%s" is not a valid int', { scope: 'service/orm/repository/validation/kernel' }, String(fieldName));
        throw new KernelValidationError('kernel_int_invalid', message, { field: String(fieldName) });
      }
    }
  });
}

export function validateSelectionFields(meta: ModelMetadata, input: Entity): void {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord) return;

  meta.fields.forEach((fieldMeta, fieldName) => {
    if (fieldMeta.type !== 'selection' || !Array.isArray(fieldMeta.selection)) {
      return;
    }

    const value = inputRecord[fieldName];
    if (value == null) return;

    const validValues = new Set(fieldMeta.selection.map(item => item.value));
    const inputValue = String(value);

    if (!validValues.has(inputValue)) {
      throw new KernelValidationError(
        'kernel_selection_invalid',
        _t(
          'Field "%s" has value "%s" outside the allowed selection values. Allowed values: %s',
          { scope: 'service/orm/repository/validation/kernel' },
          fieldName,
          inputValue,
          Array.from(validValues).join(', ')
        ),
        {
          field: String(fieldName),
          detail: {
            allowed: Array.from(validValues),
            actual: inputValue,
          },
        }
      );
    }
  });
}

function hasDefaultValue(fieldMeta: FieldMetadata | undefined): boolean {
  const column = fieldMeta?.column;
  return Boolean(column && Object.prototype.hasOwnProperty.call(column, 'default') && column.default !== undefined);
}

function isPrimaryKeyField(fieldMeta: FieldMetadata | undefined): boolean {
  return fieldMeta?.column?.primaryKey === true;
}

export function validateRequiredFields(meta: ModelMetadata, input: Entity, mode: ValidationMode): void {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord) return;

  meta.fields.forEach((fieldMeta, fieldName) => {
    const notNull = fieldMeta.column?.notNull === true;
    if (!notNull) return;

    const hasKey = hasOwnKey(inputRecord, fieldName);
    const value = hasKey ? inputRecord[fieldName] : undefined;
    const isNullish = value === null || value === undefined;

    if (mode === 'create') {
      if (hasKey) {
        if (isNullish) {
          throw new KernelValidationError(
            'kernel_required_null',
            _t('Field "%s" cannot be null', { scope: 'service/orm/repository/validation/kernel' }, fieldName),
            {
              field: String(fieldName),
            }
          );
        }
        return;
      }

      if (hasDefaultValue(fieldMeta) || isPrimaryKeyField(fieldMeta)) {
        return;
      }

      throw new KernelValidationError(
        'kernel_required_missing',
        _t('Field "%s" is required', { scope: 'service/orm/repository/validation/kernel' }, fieldName),
        {
          field: String(fieldName),
        }
      );
    }

    if (mode === 'update' && hasKey && isNullish) {
      throw new KernelValidationError(
        'kernel_required_null',
        _t('Field "%s" cannot be null', { scope: 'service/orm/repository/validation/kernel' }, fieldName),
        {
          field: String(fieldName),
        }
      );
    }
  });
}

function isValidReferenceObject(value: unknown): boolean {
  const record = asObjectRecord(value);
  if (!record) return false;
  if (!hasOwnKey(record, 'Id') && !hasOwnKey(record, 'id')) return false;
  const id = hasOwnKey(record, 'Id') ? record.Id : record.id;
  if (id === null || id === undefined) return false;
  if (typeof id === 'string') return id.trim().length > 0;
  return typeof id === 'number' || typeof id === 'bigint';
}

function isValidReferenceScalar(value: unknown): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === 'string') return value.trim().length > 0;
  return typeof value === 'number' || typeof value === 'bigint';
}

export function validateRelationShapeFields(meta: ModelMetadata, input: Entity): void {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord) return;

  meta.fields.forEach((fieldMeta, fieldName) => {
    if (!hasOwnKey(inputRecord, fieldName)) return;
    const value = inputRecord[fieldName];
    if (value === null || value === undefined) return;

    const fieldType = fieldMeta.type;
    if (fieldType !== 'ManyToOne' && fieldType !== 'ManyToOneRef') return;

    if (isValidReferenceScalar(value) || isValidReferenceObject(value)) {
      return;
    }

    throw new KernelValidationError(
      'kernel_relation_shape_invalid',
      _t(
        'Field "%s" has an invalid reference shape and must be an Id or an object containing Id/id',
        { scope: 'service/orm/repository/validation/kernel' },
        fieldName
      ),
      {
        field: String(fieldName),
      }
    );
  });
}

export function validateDecimalFields(meta: ModelMetadata, input: Entity): void {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord) return;

  meta.fields.forEach((fieldMeta, fieldName) => {
    if (fieldMeta.type !== 'decimal' && fieldMeta.type !== 'monetary') return;
    if (!hasOwnKey(inputRecord, fieldName)) return;

    const value = inputRecord[fieldName];
    if (value === null || value === undefined || value === '') return;

    let metaForNormalize = fieldMeta;
    if (fieldMeta.type === 'monetary') {
      // Prefer scale stamped by stampMonetaryScalesForWrite (hidden alias / currency digits).
      const scale = resolveDecimalScaleForWrite({ ...fieldMeta, name: fieldName }, input);
      if (typeof scale === 'number') {
        metaForNormalize = { column: { ...(fieldMeta.column || {}), scale, round: 'ROUND_HALF_UP' } } as typeof fieldMeta;
      }
    }

    const normalized = normalizeDecimalByMeta(metaForNormalize, value);
    if (!normalized) {
      throw new KernelValidationError(
        'kernel_decimal_invalid',
        _t(
          'Field "%s" must be a valid decimal that satisfies the precision and scale limits',
          { scope: 'service/orm/repository/validation/kernel' },
          fieldName
        ),
        {
          field: String(fieldName),
        }
      );
    }
  });
}

export function validateFields(
  meta: ModelMetadata,
  input: Entity,
  options?: {
    mode?: ValidationMode;
    rules?: KernelValidationRule[];
  }
): void {
  const mode = options?.mode || 'update';
  const enabledRules = new Set<KernelValidationRule>(
    options?.rules && options.rules.length > 0 ? options.rules : ['int', 'selection', 'required', 'decimal', 'relationShape']
  );

  if (enabledRules.has('int')) validateIntFields(meta, input);
  if (enabledRules.has('selection')) validateSelectionFields(meta, input);
  if (enabledRules.has('required')) validateRequiredFields(meta, input, mode);
  if (enabledRules.has('decimal')) validateDecimalFields(meta, input);
  if (enabledRules.has('relationShape')) validateRelationShapeFields(meta, input);
}
