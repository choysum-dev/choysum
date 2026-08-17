// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata';
import {
  FieldOptions,
  FieldMetadata,
  FieldType,
  RELATIONAL_CONDITION_TYPES,
  type SelectionItem,
  type SelectionDeclaration,
  type FlatManyToOneFieldOptions,
  type FlatOneToManyFieldOptions,
  type FlatManyToManyFieldOptions,
  type FlatManyToOneRefFieldOptions,
  type FlatManyToManyRefFieldOptions,
} from '../metadata/field';
import type { ModelCtor } from '../metadata/field';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';
import { isTermReference, type TermReference } from '../../i18n';
import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from '../upload_limits';
import { mergeSelectionByValue } from '../metadata/selection_merge';

const scalarTypes = new Set<FieldType>([
  'char',
  'varchar',
  'text',
  'html',
  'binary',
  'image',
  'int',
  'bigint',
  'number',
  'decimal',
  'monetary',
  'boolean',
  'datetime',
  'date',
  'time',
  'jsonobject',
  'properties',
  'selection',
  'ManyToOneRef',
  'ManyToManyRef',
]);
const relationTypes = new Set<FieldType>(['ManyToOne', 'OneToMany', 'ManyToMany']);

type FieldDecoratorOptionBag = {
  type?: FieldType;
  string?: unknown;
  help?: unknown;
  select?: unknown;
  column?: unknown;
  /** Static array | method name | callable (see SelectionDeclaration). */
  selection?: SelectionDeclaration;
  /** Append static options onto an inherited selection field (PR-P2-F4). */
  selectionAdd?: unknown;
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
  currencyField?: unknown;
  /** Properties parent container field name (PP6); omit = App-level. */
  definition?: unknown;
  primaryKey?: unknown;
  unique?: unknown;
  uniqueIndex?: unknown;
  checkConstraint?: unknown;
  default?: unknown;
  round?: unknown;
  /** Data i18n: store per-language values as a JSON/JSONB lang map. */
  translate?: unknown;
  /** Company-dependent: store per-company values as a JSON/JSONB company map. */
  companyDependent?: unknown;
  /** Whether the field participates in Model.Copy (default true when omitted). */
  copy?: unknown;
  /** Declarative field readonly (PR-P2-F2); wire still uses isReadonly. */
  readonly?: unknown;
  /** When true, Write path dials audit.FieldChange (AU3 / PR-P3-A2). */
  tracking?: unknown;
  /**
   * Odoo-style check_company for ManyToOne / ManyToOneRef (parent↔related CompanyId).
   */
  checkCompany?: unknown;
  /** Relational default condition (static tree or callable); relation field types only. */
  condition?: unknown;
  maxUploadBytes?: unknown;
  maxWidth?: unknown;
  maxHeight?: unknown;
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

/** Normalize help like string, but blank/whitespace omits (D6) instead of throwing. */
function normalizeFieldHelp(name: string, value: unknown): { help?: string; helpText?: TermReference } {
  if (value === undefined || value === null) {
    return {};
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) return {};
    return { help: trimmed };
  }
  if (isTermReference(value)) {
    // isTermReference already requires src: string; blank/whitespace still omits via throw.
    const src = value.src.trim();
    if (!src) {
      throw new Error(`@Field(${name}) help term reference requires a non-empty src`);
    }
    return { help: src, helpText: { ...value } };
  }
  throw new Error(`@Field(${name}) help must be a string or term reference`);
}

function toFieldDecoratorOptionBag(value: unknown): FieldDecoratorOptionBag {
  const record = asObjectRecord(value);
  return (record || {}) as FieldDecoratorOptionBag;
}

function parsePositiveIntFieldOption(fieldName: string, optionName: string, value: unknown): number | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new Error(`@Field(${fieldName}) ${optionName} must be a positive integer`);
  }
  return value;
}

function validateUploadLimitOptions(
  fieldName: string,
  type: FieldType,
  optionBag: FieldDecoratorOptionBag
): Pick<FieldMetadata, 'maxUploadBytes' | 'maxWidth' | 'maxHeight'> {
  const hasAny =
    optionBag.maxUploadBytes !== undefined || optionBag.maxWidth !== undefined || optionBag.maxHeight !== undefined;
  if (!hasAny) {
    return {};
  }
  if (type !== 'image' && type !== 'binary') {
    throw new Error(`@Field(${fieldName}) maxUploadBytes/maxWidth/maxHeight are only allowed on image/binary fields`);
  }

  const maxUploadBytes = parsePositiveIntFieldOption(fieldName, 'maxUploadBytes', optionBag.maxUploadBytes);
  const maxWidth = parsePositiveIntFieldOption(fieldName, 'maxWidth', optionBag.maxWidth);
  const maxHeight = parsePositiveIntFieldOption(fieldName, 'maxHeight', optionBag.maxHeight);

  if (type === 'binary' && (maxWidth !== undefined || maxHeight !== undefined)) {
    throw new Error(`@Field(${fieldName}) maxWidth/maxHeight are only allowed on image fields`);
  }
  if (maxUploadBytes !== undefined && maxUploadBytes > DEFAULT_GLOBAL_MAX_UPLOAD_BYTES) {
    throw new Error(
      `@Field(${fieldName}) maxUploadBytes cannot exceed global upload cap (${DEFAULT_GLOBAL_MAX_UPLOAD_BYTES} bytes)`
    );
  }

  const out: Pick<FieldMetadata, 'maxUploadBytes' | 'maxWidth' | 'maxHeight'> = {};
  if (maxUploadBytes !== undefined) out.maxUploadBytes = maxUploadBytes;
  if (maxWidth !== undefined) out.maxWidth = maxWidth;
  if (maxHeight !== undefined) out.maxHeight = maxHeight;
  return out;
}

/**
 * Declares model field metadata for persistence, relations, selections, and compute behavior.
 *
 * Overloads infer condition target fields from ctor `targetModel` for object relations.
 * For string Ref, pass Field<TTarget> (import type) to tighten condition; omit to keep BaseQueryCondition.
 *
 * @param options Field metadata to register on the decorated property.
 * @returns A property decorator that records the field definition in metadata storage.
 */
export function Field<TTarget extends BaseModel>(options: FlatManyToOneFieldOptions<TTarget>): PropertyDecorator;
export function Field<TTarget extends BaseModel>(options: FlatOneToManyFieldOptions<TTarget>): PropertyDecorator;
export function Field<TJoin extends BaseModel, TTarget extends BaseModel>(
  options: FlatManyToManyFieldOptions<TJoin, TTarget>
): PropertyDecorator;
export function Field<TTarget extends BaseModel>(options: FlatManyToOneRefFieldOptions<TTarget>): PropertyDecorator;
export function Field<TTarget extends BaseModel>(options: FlatManyToManyRefFieldOptions<TTarget>): PropertyDecorator;
export function Field(options: FlatManyToOneRefFieldOptions): PropertyDecorator;
export function Field(options: FlatManyToManyRefFieldOptions): PropertyDecorator;
export function Field(options: FieldOptions): PropertyDecorator;
// Implementation: accept typed Ref options (TTarget) that are not assignable into default FieldOptions.
export function Field(
  options: FieldOptions | FlatManyToOneRefFieldOptions<BaseModel> | FlatManyToManyRefFieldOptions<BaseModel>
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
    let selectionAddItems: SelectionItem[] | undefined;
    let conditionKind: FieldMetadata['conditionKind'];
    let conditionStatic: FieldMetadata['condition'];
    let conditionCallable: FieldMetadata['conditionCallable'];
    let normalizedColumn: ObjectRecord | undefined;

    const isRelation = relationTypes.has(type);

    if (optionBag.translate !== undefined && typeof optionBag.translate !== 'boolean') {
      throw new Error(`@Field(${name}) translate must be a boolean`);
    }
    const translate = optionBag.translate === true;
    if (optionBag.companyDependent !== undefined && typeof optionBag.companyDependent !== 'boolean') {
      throw new Error(`@Field(${name}) companyDependent must be a boolean`);
    }
    const companyDependent = optionBag.companyDependent === true;
    if (translate && companyDependent) {
      throw new Error(`@Field(${name}) cannot combine translate and companyDependent`);
    }
    if (optionBag.copy !== undefined && typeof optionBag.copy !== 'boolean') {
      throw new Error(`@Field(${name}) copy must be a boolean`);
    }
    const copyFlag =
      optionBag.copy === false
        ? false
        : optionBag.copy === true
          ? true
          : companyDependent
            ? false
            : undefined;
    if (optionBag.readonly !== undefined && typeof optionBag.readonly !== 'boolean') {
      throw new Error(`@Field(${name}) readonly must be a boolean`);
    }
    const readonlyFlag = optionBag.readonly === true;
    if (optionBag.tracking !== undefined && typeof optionBag.tracking !== 'boolean') {
      throw new Error(`@Field(${name}) tracking must be a boolean`);
    }
    const trackingFlag = optionBag.tracking === true;
    if (optionBag.checkCompany !== undefined && typeof optionBag.checkCompany !== 'boolean') {
      throw new Error(`@Field(${name}) checkCompany must be a boolean`);
    }
    if (optionBag.checkCompany === true && type !== 'ManyToOne' && type !== 'ManyToOneRef') {
      throw new Error(`@Field(${name}) checkCompany is only supported on ManyToOne / ManyToOneRef fields`);
    }
    const checkCompany = optionBag.checkCompany === true;
    const uploadLimits = validateUploadLimitOptions(name, type, optionBag);
    if (translate) {
      if (type !== 'char' && type !== 'varchar' && type !== 'text') {
        throw new Error(`@Field(${name}) translate is only supported on char/varchar/text fields`);
      }
      const uniqueIndexOn =
        optionBag.uniqueIndex === true ||
        (typeof optionBag.uniqueIndex === 'string' && optionBag.uniqueIndex.trim().length > 0);
      if (optionBag.unique === true || uniqueIndexOn) {
        throw new Error(`@Field(${name}) translate cannot be combined with unique/uniqueIndex`);
      }
      // Translate fields only accept optional index: 'trigram' (data-i18n-design §7.1); never btree.
      if (optionBag.indexed === true) {
        throw new Error(`@Field(${name}) translate cannot use indexed/index btree; use index: 'trigram' or omit`);
      }
      if (optionBag.index !== undefined && optionBag.index !== 'trigram') {
        throw new Error(`@Field(${name}) translate only supports index: 'trigram' (or omit index)`);
      }
    }
    if (companyDependent) {
      // Canonical FieldTypes only (int/number — not float/integer aliases).
      const allowed = new Set([
        'char',
        'varchar',
        'text',
        'html',
        'boolean',
        'int',
        'number',
        'decimal',
        'monetary',
        'date',
        'datetime',
        'selection',
        'ManyToOne',
        'ManyToOneRef',
      ]);
      if (!allowed.has(String(type))) {
        throw new Error(
          `@Field(${name}) companyDependent is not supported on type "${type}"`
        );
      }
      const uniqueIndexOn =
        optionBag.uniqueIndex === true ||
        (typeof optionBag.uniqueIndex === 'string' && optionBag.uniqueIndex.trim().length > 0);
      if (optionBag.unique === true || uniqueIndexOn) {
        throw new Error(`@Field(${name}) companyDependent cannot be combined with unique/uniqueIndex`);
      }
      // Migration strips physical indexes for company-dependent JSON maps; reject so FieldsGet
      // cannot advertise an index the database does not have.
      const hasPhysicalIndex =
        optionBag.indexed === true ||
        optionBag.index === true ||
        (typeof optionBag.index === 'string' && optionBag.index.trim().length > 0);
      if (hasPhysicalIndex) {
        throw new Error(`@Field(${name}) companyDependent does not support indexed/index`);
      }
    }

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
      optionBag.scaleField !== undefined ||
      optionBag.currencyField !== undefined;
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
        if (type === 'monetary') {
          throw new Error(`@Field(${name}) monetary forbids scale, scaleField, and precision (use Currency.DecimalDigits)`);
        }
        if (!isInt(optionBag.precision) || optionBag.precision < 1 || optionBag.precision > 38) {
          throw new Error(`@Field(${name}) precision must be in 1..38`);
        }
        if (type !== 'decimal') {
          throw new Error(`@Field(${name}) precision is only supported on decimal fields`);
        }
        hints.precision = optionBag.precision;
      }

      if (optionBag.scale !== undefined) {
        if (type === 'monetary') {
          throw new Error(`@Field(${name}) monetary forbids scale, scaleField, and precision (use Currency.DecimalDigits)`);
        }
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
      // translate / companyDependent: size is a logical value limit only; do not set physical varchar(n).
      if (normalizedStorageHints?.size != null && !translate && !companyDependent) {
        normalizedColumn.size = normalizedStorageHints.size;
      }
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
      if (optionBag.currencyField !== undefined) normalizedColumn.currencyField = optionBag.currencyField;
    }

    const hasColumn = normalizedColumn !== undefined;

    if (optionBag.selectionAdd !== undefined && type !== 'selection') {
      throw new Error(`@Field(${name}) selectionAdd is only supported on selection fields`);
    }

    // Selection-specific validation (static array | method name | callable | selectionAdd)
    if (type === 'selection') {
      const selectionRaw = optionBag.selection;
      const hasSelection = selectionRaw !== undefined;
      const hasSelectionAdd = optionBag.selectionAdd !== undefined;

      if (hasSelection && hasSelectionAdd) {
        throw new Error(
          `@Field(${name}) cannot combine selection and selectionAdd; use selectionAdd alone to append, or selection alone to replace`
        );
      }

      if (hasSelectionAdd) {
        selectionAddItems = normalizeSelectionItems(name, optionBag.selectionAdd, 'selectionAdd', {
          allowEmpty: true,
        });
      }

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
        validatedSelection = normalizeSelectionItems(name, selectionRaw, 'selection', { allowEmpty: false });
        selectionKind = 'static';
      } else if (hasSelectionAdd) {
        // Pure append (PR-P2-F4): resolve base static selection from inherited metadata at write time.
        selectionKind = 'static';
      } else if (hasSelection) {
        throw new Error(
          `@Field(${name}) selection must be a non-empty array, method name string, or () => SelectionItem[] callable`
        );
      } else {
        throw new Error(`@Field(${name}) selection type requires selection or selectionAdd`);
      }
    }

    // Relational condition (static QueryCondition | callable; no method-name string)
    if (optionBag.condition !== undefined) {
      if (!RELATIONAL_CONDITION_TYPES.has(type)) {
        throw new Error(
          `@Field(${name}) condition is only supported on ManyToOne / ManyToOneRef / OneToMany / ManyToMany / ManyToManyRef`
        );
      }
      const conditionRaw = optionBag.condition;
      if (typeof conditionRaw === 'function') {
        conditionKind = 'dynamic';
        conditionCallable = conditionRaw as FieldMetadata['conditionCallable'];
      } else if (typeof conditionRaw === 'string') {
        throw new Error(
          `@Field(${name}) condition must not be a method name string; use a QueryCondition tree or () => QueryCondition callable`
        );
      } else if (conditionRaw && typeof conditionRaw === 'object') {
        conditionKind = 'static';
        conditionStatic = conditionRaw as FieldMetadata['condition'];
      } else {
        throw new Error(`@Field(${name}) condition must be a QueryCondition tree or () => QueryCondition callable`);
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

    if (optionBag.currencyField !== undefined && type !== 'monetary') {
      throw new Error(`@Field(${name}) currencyField is only supported on monetary fields`);
    }

    if (optionBag.definition !== undefined && type !== 'properties') {
      throw new Error(`@Field(${name}) definition is only supported on properties fields`);
    }

    if (type === 'properties') {
      const definition = optionBag.definition;
      if (definition !== undefined) {
        if (typeof definition !== 'string' || !definition.trim()) {
          throw new Error(`@Field(${name}) definition must be a non-empty string when provided`);
        }
        if (definition !== definition.trim()) {
          throw new Error(`@Field(${name}) definition must not contain leading or trailing whitespace`);
        }
      }
    }

    if (type === 'monetary') {
      const currencyField = optionBag.currencyField;
      if (typeof currencyField !== 'string' || !currencyField.trim()) {
        throw new Error(`@Field(${name}) monetary requires a non-empty currencyField`);
      }
      if (currencyField !== currencyField.trim()) {
        throw new Error(`@Field(${name}) currencyField must not contain leading or trailing whitespace`);
      }
      // scale/precision/scaleField are rejected earlier in storage-hint / scaleField validation.
      if (optionBag.round !== undefined) {
        throw new Error(`@Field(${name}) monetary does not support round (P0 uses HALF_UP)`);
      }
      // currencyField already forces normalizedColumn via hasFlatColumnOptions; keep trimmed value.
      normalizedColumn!.currencyField = currencyField.trim();
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

    const fieldHelp = normalizeFieldHelp(name, optionBag.help);
    if (fieldHelp.help !== undefined) meta.help = fieldHelp.help;
    if (fieldHelp.helpText !== undefined) meta.helpText = fieldHelp.helpText;

    // Persist selection metadata before final storage/relation defaults.
    if (type === 'selection') {
      if (selectionKind) meta.selectionKind = selectionKind;
      if (validatedSelection) meta.selection = validatedSelection;
      if (selectionMethod) meta.selectionMethod = selectionMethod;
      if (selectionCallable) meta.selectionCallable = selectionCallable;
    }

    if (conditionKind) meta.conditionKind = conditionKind;
    if (conditionStatic) meta.condition = conditionStatic;
    if (conditionCallable) meta.conditionCallable = conditionCallable;

    if (optionBag.relation) meta.relation = optionBag.relation as FieldMetadata['relation'];
    if (normalizedColumn) meta.column = normalizedColumn as FieldMetadata['column'];
    else if (autoColumnScalar || autoColumnManyToOne) meta.column = {};

    if (normalizedRelated) meta.related = normalizedRelated;
    if (normalizedStorageHints) meta.storageHints = normalizedStorageHints;
    if (translate) meta.translate = true;
    if (companyDependent) meta.companyDependent = true;
    if (copyFlag === false) meta.copy = false;
    else if (copyFlag === true) meta.copy = true;
    if (readonlyFlag) meta.readonly = true;
    if (trackingFlag) meta.tracking = true;
    if (checkCompany) meta.checkCompany = true;
    if (type === 'properties' && typeof optionBag.definition === 'string' && optionBag.definition.trim()) {
      meta.definition = optionBag.definition.trim();
    }
    if (uploadLimits.maxUploadBytes !== undefined) meta.maxUploadBytes = uploadLimits.maxUploadBytes;
    if (uploadLimits.maxWidth !== undefined) meta.maxWidth = uploadLimits.maxWidth;
    if (uploadLimits.maxHeight !== undefined) meta.maxHeight = uploadLimits.maxHeight;

    // Write metadata
    const ctor = target.constructor as ModelCtor<BaseModel> & typeof BaseModel;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const existingCompute = prev?.computeHandlers?.get(name);
    const existingSqlCompute = prev?.sqlComputeHandlers?.get(name);

    if (existingCompute?.store === false || !!existingSqlCompute) {
      delete meta.column;
    }

    // PR-P2-F4: selectionAdd-only merges onto the inherited static selection before write.
    if (type === 'selection' && selectionAddItems !== undefined && !validatedSelection) {
      const baseField = prev?.fields?.get(name);
      if (!baseField || baseField.type !== 'selection') {
        throw new Error(`@Field(${name}) selectionAdd requires an inherited selection field`);
      }
      if (baseField.selectionKind === 'dynamic' || !Array.isArray(baseField.selection)) {
        throw new Error(`@Field(${name}) selectionAdd requires an inherited static selection`);
      }
      meta.selectionKind = 'static';
      meta.selection = mergeSelectionByValue(baseField.selection, selectionAddItems);
    }

    const fields = new Map(prev.fields);
    const nextField: FieldMetadata = { ...(fields.get(name) || {}), ...meta };
    if (type === 'selection' && selectionAddItems !== undefined && !validatedSelection) {
      nextField.selectionKind = 'static';
      nextField.selection = meta.selection;
      delete nextField.selectionMethod;
      delete nextField.selectionCallable;
    }
    fields.set(name, nextField);

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      fields,
    });
  };
}

function normalizeSelectionItems(
  name: string,
  raw: unknown,
  optionName: 'selection' | 'selectionAdd',
  opts: { allowEmpty: boolean }
): SelectionItem[] {
  if (!Array.isArray(raw)) {
    throw new Error(`@Field(${name}) ${optionName} must be an array`);
  }
  if (!opts.allowEmpty && raw.length === 0) {
    throw new Error(`@Field(${name}) selection type requires a non-empty selection array`);
  }

  const values = new Set<string>();
  const normalized: SelectionItem[] = [];
  for (const item of raw) {
    if (!item || typeof item !== 'object') {
      throw new Error(`@Field(${name}) each ${optionName} item must be an object`);
    }
    const entry = item as { value?: unknown; label?: unknown; labelText?: unknown };
    if (!entry.value || typeof entry.value !== 'string') {
      throw new Error(`@Field(${name}) each ${optionName} item must include a string value field`);
    }
    if (entry.labelText != null) {
      throw new Error(
        `@Field(${name}) ${optionName} labelText is forbidden; use label: _lt('…') when the option should translate`
      );
    }

    const value = entry.value.trim();
    if (!value) {
      throw new Error(`@Field(${name}) each ${optionName} item must include a non-empty string value`);
    }
    if (values.has(value)) {
      throw new Error(`@Field(${name}) duplicate ${optionName} value: ${value}`);
    }
    values.add(value);

    const labelRaw = entry.label;
    if (isTermReference(labelRaw)) {
      const src = String(labelRaw.src || '').trim();
      if (!src) {
        throw new Error(`@Field(${name}) each ${optionName} item _lt label must include a non-empty src`);
      }
      normalized.push({ value, label: src, labelText: labelRaw });
      continue;
    }
    if (!labelRaw || typeof labelRaw !== 'string') {
      throw new Error(
        `@Field(${name}) each ${optionName} item label must be a string or TermReference from _lt(...)`
      );
    }
    const label = labelRaw.trim();
    if (!label) {
      throw new Error(`@Field(${name}) each ${optionName} item must include a non-empty string label`);
    }
    normalized.push({ value, label });
  }
  return normalized;
}
