// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { MetadataStorage } from '../metadata/storage';
import type { FieldMetadata, FieldType } from '../metadata/field';
import type { ModelMetadata } from '../metadata/model';
import { raiseDomainError } from '@/core/service/error';
import BaseModel from './model';
import { resolveEffectiveFieldDefaults } from './field_default_resolve';
import type { InstantiableModelCtor } from './types';

/** Align Odoo: False→global; True→current; id→specific. */
export type FieldDefaultScopeDim = string | boolean | null | undefined;

export type FieldDefaultScopeOpts = {
  userId?: FieldDefaultScopeDim;
  companyId?: FieldDefaultScopeDim;
};

const SUPPORTED_DEFAULT_TYPES = new Set<FieldType>([
  'char',
  'varchar',
  'text',
  'html',
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
  'selection',
  'ManyToOne',
  'ManyToOneRef',
]);

const UNSUPPORTED_DEFAULT_TYPES = new Set<FieldType>([
  'OneToMany',
  'ManyToMany',
  'ManyToManyRef',
  'binary',
  'image',
]);

const ensuredUniqueIndexTables = new Set<string>();

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function resolveScopeDim(dim: FieldDefaultScopeDim, current: string | undefined, label: string): string | null {
  if (dim === undefined || dim === null || dim === false) return null;
  if (dim === true) {
    const id = String(current || '').trim();
    if (!id) {
      fail('FIELD_DEFAULT_INVALID_VALUE', `${label}=true requires a current ${label} in context`);
    }
    return id;
  }
  const id = String(dim).trim();
  if (!id) {
    fail('FIELD_DEFAULT_INVALID_VALUE', `${label} scope id cannot be empty`);
  }
  return id;
}

function scopeCondition(field: string, value: string | null): any {
  return value == null ? [field, 'is', null] : [field, '=', value];
}

function storeMeta(ctor: InstantiableModelCtor<FieldDefaultBaseModel>) {
  return MetadataStorage.instance.getModelMetadata(ctor as any);
}

function resolveTargetModel(
  ctor: InstantiableModelCtor<FieldDefaultBaseModel>,
  modelShortName: string
): { ctor: typeof BaseModel; targetMeta: ModelMetadata } {
  const short = String(modelShortName || '').trim();
  if (!short) {
    fail('FIELD_DEFAULT_UNKNOWN_FIELD', 'model is required');
  }

  const meta = storeMeta(ctor);
  const application = String(meta.application || '').trim();
  if (!application || application === 'core') {
    fail('FIELD_DEFAULT_CROSS_APP_MODEL', 'FieldDefault store application is invalid');
  }

  const fullName = `${application}.${short}`;
  const Target = BaseModel.resolveModelConstructor(fullName) || BaseModel.resolveModelConstructor(short);
  if (!Target) {
    fail('FIELD_DEFAULT_CROSS_APP_MODEL', `Model ${short} is not registered`);
  }

  const targetMeta = MetadataStorage.instance.getModelMetadata(Target as any) as ModelMetadata;
  const targetApp = String(targetMeta.application || '').trim();
  if (targetApp !== application) {
    fail('FIELD_DEFAULT_CROSS_APP_MODEL', `Model ${short} does not belong to application ${application}`);
  }

  return { ctor: Target as typeof BaseModel, targetMeta };
}

function resolveTargetField(targetMeta: ModelMetadata, fieldName: string): FieldMetadata {
  const name = String(fieldName || '').trim();
  if (!name) {
    fail('FIELD_DEFAULT_UNKNOWN_FIELD', 'field is required');
  }
  const field = targetMeta.fields?.get?.(name);
  if (!field) {
    fail('FIELD_DEFAULT_UNKNOWN_FIELD', `Field ${name} is unknown on target model`);
  }
  const type = field.type as FieldType;
  if (UNSUPPORTED_DEFAULT_TYPES.has(type) || !SUPPORTED_DEFAULT_TYPES.has(type)) {
    fail('FIELD_DEFAULT_INVALID_VALUE', `Field type ${type} is not supported as a FieldDefault value`);
  }
  return field;
}

function normalizeStoredValue(field: FieldMetadata, value: unknown): unknown {
  if (value === undefined) {
    fail('FIELD_DEFAULT_INVALID_VALUE', 'Value is required');
  }
  const type = field.type as FieldType;
  if (type === 'ManyToOne' || type === 'ManyToOneRef') {
    if (value == null) return null;
    if (typeof value === 'object' && 'Id' in value) {
      const id = String((value as { Id?: unknown }).Id ?? '').trim();
      if (!id) fail('FIELD_DEFAULT_INVALID_VALUE', 'ManyToOne default requires an Id');
      return id;
    }
    const id = String(value).trim();
    if (!id) fail('FIELD_DEFAULT_INVALID_VALUE', 'ManyToOne default requires an Id string');
    return id;
  }
  return value;
}

async function ensureScopeUniqueIndex(ctor: InstantiableModelCtor<FieldDefaultBaseModel>): Promise<void> {
  const meta = storeMeta(ctor);
  const table = typeof meta.tableName === 'function' ? String(meta.tableName()) : String(meta.tableName || '');
  if (!table || ensuredUniqueIndexTables.has(table)) return;

  const dialect = String(($choysum as any)?.db?.dialectName || 'sqlite').toLowerCase();
  const indexName = `uidx_${table}_scope`;
  let ddl = '';
  if (dialect === 'postgres' || dialect === 'postgresql') {
    ddl = `CREATE UNIQUE INDEX IF NOT EXISTS ${indexName} ON ${table} (model, field, user_id, company_id) NULLS NOT DISTINCT`;
  } else {
    // SQLite / others: coalesce NULL to '' so NULL scopes collide.
    ddl = `CREATE UNIQUE INDEX IF NOT EXISTS ${indexName} ON ${table} (model, field, coalesce(user_id, ''), coalesce(company_id, ''))`;
  }

  try {
    const exec = ($choysum as any)?.db?.execute;
    if (typeof exec === 'function') {
      await exec.call(($choysum as any).db, ddl, '[]');
      ensuredUniqueIndexTables.add(table);
    }
  } catch {
    // Best-effort: upsert path still enforces uniqueness in application logic.
  }
}

async function findExactRow(
  ctor: InstantiableModelCtor<FieldDefaultBaseModel>,
  model: string,
  field: string,
  userId: string | null,
  companyId: string | null
): Promise<FieldDefaultBaseModel | undefined> {
  const rows = await (ctor as any).Search(
    {
      And: [['Model', '=', model], ['Field', '=', field], scopeCondition('UserId', userId), scopeCondition('CompanyId', companyId)],
    } as any,
    { fields: ['Id', 'Model', 'Field', 'UserId', 'CompanyId', 'Value'] as any, limit: 2 } as any
  );
  return (rows && rows[0]) || undefined;
}

/**
 * Per-application FieldDefault store base (no `@Model`, no table).
 * Thin app classes: `@Model('FieldDefault') export default class FieldDefault extends FieldDefaultBaseModel {}`
 */
export default class FieldDefaultBaseModel extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, index: true })
  Model!: string;

  @Field({ type: 'varchar', size: 255, notNull: true, index: true })
  Field!: string;

  @Field({ type: 'varchar', size: 64, index: true })
  UserId?: string | null;

  @Field({ type: 'varchar', size: 64, index: true })
  CompanyId?: string | null;

  @Field({ type: 'jsonobject', notNull: true })
  Value!: unknown;

  /**
   * Upsert a default for an exact user/company scope (Odoo `ir.default.set`).
   */
  static async Set(
    this: InstantiableModelCtor<FieldDefaultBaseModel>,
    model: string,
    field: string,
    value: unknown,
    opts?: FieldDefaultScopeOpts
  ): Promise<void> {
    const { targetMeta } = resolveTargetModel(this, model);
    const fieldDef = resolveTargetField(targetMeta, field);
    const stored = normalizeStoredValue(fieldDef, value);
    const userId = resolveScopeDim(opts?.userId, (this as any).userId, 'userId');
    const companyId = resolveScopeDim(opts?.companyId, (this as any).companyId, 'companyId');
    const modelShort = String(model).trim();
    const fieldName = String(field).trim();

    await ensureScopeUniqueIndex(this);

    try {
      await (this as any).withSavepoint(async () => {
        const existing = await findExactRow(this, modelShort, fieldName, userId, companyId);
        if (existing?.Id) {
          await (this as any).UpdateById(existing.Id, { Value: stored } as any);
          return;
        }
        await (this as any).Create({
          Model: modelShort,
          Field: fieldName,
          UserId: userId,
          CompanyId: companyId,
          Value: stored,
        } as any);
      });
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      // Map only uniqueness violations — not every "constraint" failure (FK, NOT NULL, …).
      if (/unique constraint|unique index|duplicate key|UNIQUE constraint failed/i.test(msg)) {
        fail('FIELD_DEFAULT_SCOPE_CONFLICT', `FieldDefault scope conflict for ${modelShort}.${fieldName}`);
      }
      throw err;
    }
  }

  /**
   * Read the exact-scope default (Odoo `ir.default._get`). Missing → undefined.
   */
  static async Get(
    this: InstantiableModelCtor<FieldDefaultBaseModel>,
    model: string,
    field: string,
    opts?: FieldDefaultScopeOpts
  ): Promise<unknown | undefined> {
    const { targetMeta } = resolveTargetModel(this, model);
    resolveTargetField(targetMeta, field);
    const userId = resolveScopeDim(opts?.userId, (this as any).userId, 'userId');
    const companyId = resolveScopeDim(opts?.companyId, (this as any).companyId, 'companyId');
    const row = await findExactRow(this, String(model).trim(), String(field).trim(), userId, companyId);
    return row ? (row as any).Value : undefined;
  }

  /**
   * Effective defaults for the current request identity (Odoo `_get_model_defaults`).
   */
  static async GetEffective(
    this: InstantiableModelCtor<FieldDefaultBaseModel>,
    model: string,
    fields?: string[]
  ): Promise<Record<string, unknown>> {
    const { targetMeta } = resolveTargetModel(this, model);
    const modelShort = String(model).trim();
    const uid = String((this as any).userId || '').trim() || null;
    const companyId = String((this as any).companyId || '').trim() || null;

    const and: any[] = [['Model', '=', modelShort]];
    if (fields && fields.length) {
      and.push(['Field', 'in', fields.map(String)]);
    }
    // UserId IS NULL OR UserId = :uid
    and.push({ Or: [scopeCondition('UserId', null), ...(uid ? [['UserId', '=', uid]] : [])] });
    // CompanyId IS NULL OR CompanyId = :companyId (no active company → only NULL)
    if (companyId) {
      and.push({ Or: [scopeCondition('CompanyId', null), ['CompanyId', '=', companyId]] });
    } else {
      and.push(scopeCondition('CompanyId', null));
    }

    const rows = await (this as any).Search(
      { And: and } as any,
      { fields: ['Id', 'Field', 'UserId', 'CompanyId', 'Value'] as any } as any
    );

    const fieldNames =
      fields && fields.length
        ? fields.map(String)
        : [...(targetMeta.fields?.keys?.() || [])].map(String);

    return resolveEffectiveFieldDefaults(rows || [], fieldNames);
  }

  /**
   * Delete the exact-scope default row when present.
   */
  static async Unset(
    this: InstantiableModelCtor<FieldDefaultBaseModel>,
    model: string,
    field: string,
    opts?: FieldDefaultScopeOpts
  ): Promise<void> {
    const { targetMeta } = resolveTargetModel(this, model);
    resolveTargetField(targetMeta, field);
    const userId = resolveScopeDim(opts?.userId, (this as any).userId, 'userId');
    const companyId = resolveScopeDim(opts?.companyId, (this as any).companyId, 'companyId');
    const row = await findExactRow(this, String(model).trim(), String(field).trim(), userId, companyId);
    if (row?.Id) {
      await (this as any).DeleteById(row.Id);
    }
  }
}

/** Test-only: clear process-local unique-index DDL cache. */
export function __resetFieldDefaultUniqueIndexTablesForTest(): void {
  ensuredUniqueIndexTables.clear();
}
