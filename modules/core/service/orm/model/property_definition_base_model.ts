// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { raiseDomainError } from '@/core/service/error';
import { MetadataStorage } from '../metadata/storage';
import BaseModel from './model';
import { registerLogicalModelName } from './logical_model_registry';
import { assertValidPropertyDefinitionItems } from './properties_types';
import type { InstantiableModelCtor } from './types';
import type { Insertable, Updateable, FieldSelection, QueryCondition, UpdateOptions } from '../repository/types';

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function normalizeDefinitionOnVals(vals: Record<string, unknown> | undefined): void {
  if (!vals || !Object.prototype.hasOwnProperty.call(vals, 'Definition')) return;
  try {
    vals.Definition = assertValidPropertyDefinitionItems(vals.Definition);
  } catch (err) {
    fail('PROPERTY_DEFINITION_INVALID', errorMessage(err));
  }
}

const ensuredUniqueIndexTables = new Set<string>();

/** Test-only: clear ensured unique-index table cache. */
export function __resetPropertyDefinitionUniqueIndexTablesForTest(): void {
  ensuredUniqueIndexTables.clear();
}

/** Test-only: run unique-index ensure for a PropertyDefinition ctor. */
export async function __ensureDefinitionUniqueIndexForTest(
  ctor: InstantiableModelCtor<PropertyDefinitionBaseModel>
): Promise<void> {
  await ensureDefinitionUniqueIndex(ctor);
}

/** Test-only: run Definition normalize/validate on a vals bag. */
export function __normalizeDefinitionOnValsForTest(vals: Record<string, unknown> | undefined): void {
  normalizeDefinitionOnVals(vals);
}

/** Test-only: expose error message coercion used by Definition/index failures. */
export function __errorMessageForTest(err: unknown): string {
  return errorMessage(err);
}

/** Test-only: expose scope-touch predicate used by UpdateById. */
export function __touchesDefinitionScopeForTest(vals: Record<string, unknown> | undefined): boolean {
  return touchesDefinitionScope(vals);
}

function storeMeta(ctor: InstantiableModelCtor<PropertyDefinitionBaseModel>) {
  return MetadataStorage.instance.getModelMetadata(ctor as any);
}

function nullScope(value: unknown): string | null {
  if (value == null) return null;
  const s = String(value).trim();
  return s || null;
}

function scopeEq(field: string, value: string | null): unknown[] {
  return value == null ? [field, '=', null] : [field, '=', value];
}

/**
 * Application-level uniqueness for (TargetModel, PropertiesField, ContainerModel, ContainerId).
 * Complements the DB unique index and covers environments where DDL ensure is unavailable.
 */
async function assertUniqueDefinitionScope(
  ctor: InstantiableModelCtor<PropertyDefinitionBaseModel>,
  vals: Record<string, unknown>,
  excludeId?: string
): Promise<void> {
  const targetModel = String(vals.TargetModel ?? '').trim();
  const propertiesField = String(vals.PropertiesField ?? '').trim();
  if (!targetModel || !propertiesField) return;

  const containerModel = nullScope(vals.ContainerModel);
  const containerId = nullScope(vals.ContainerId);
  const And: unknown[] = [
    ['TargetModel', '=', targetModel],
    ['PropertiesField', '=', propertiesField],
    scopeEq('ContainerModel', containerModel),
    scopeEq('ContainerId', containerId),
  ];
  if (excludeId) {
    And.push(['Id', '!=', excludeId]);
  }

  const rows = await (ctor as any).Search({ And } as any, {
    fields: ['Id'] as any,
    limit: 1,
  } as any);
  if (rows && rows.length > 0) {
    fail(
      'PROPERTY_DEFINITION_DUPLICATE_SCOPE',
      `PropertyDefinition scope already exists for ${targetModel}.${propertiesField}`
    );
  }
}

function touchesDefinitionScope(vals: Record<string, unknown> | undefined): boolean {
  if (!vals) return false;
  return (
    Object.prototype.hasOwnProperty.call(vals, 'TargetModel') ||
    Object.prototype.hasOwnProperty.call(vals, 'PropertiesField') ||
    Object.prototype.hasOwnProperty.call(vals, 'ContainerModel') ||
    Object.prototype.hasOwnProperty.call(vals, 'ContainerId')
  );
}

/**
 * Composite uniqueness for (TargetModel, PropertiesField, ContainerModel, ContainerId).
 * Uses COALESCE for all dialects so PostgreSQL 14 and older remain compatible
 * (avoids PG15-only NULLS NOT DISTINCT). DDL failures propagate.
 */
async function ensureDefinitionUniqueIndex(ctor: InstantiableModelCtor<PropertyDefinitionBaseModel>): Promise<void> {
  const meta = storeMeta(ctor);
  const table = typeof meta.tableName === 'function' ? String(meta.tableName()) : String(meta.tableName || '');
  if (!table || ensuredUniqueIndexTables.has(table)) return;

  const indexName = `uidx_${table}_definition_scope`;
  // Expression unique index: NULL/empty container dims collide (App-level + parent scopes).
  const ddl = `CREATE UNIQUE INDEX IF NOT EXISTS ${indexName} ON ${table} (target_model, properties_field, coalesce(container_model, ''), coalesce(container_id, ''))`;

  const exec = ($choysum as any)?.db?.execute;
  if (typeof exec !== 'function') {
    // No DDL surface (unit harness): uniqueness still enforced in assertUniqueDefinitionScope.
    return;
  }

  try {
    await exec.call(($choysum as any).db, ddl, '[]');
    ensuredUniqueIndexTables.add(table);
  } catch (err) {
    fail('PROPERTY_DEFINITION_INDEX', `Failed to ensure PropertyDefinition unique index on ${table}: ${errorMessage(err)}`);
  }
}

/**
 * Per-application PropertyDefinition store base (no `@Model`, no table).
 *
 * Thin app classes (hand-written or C2 in PP-2):
 * `@Model('PropertyDefinition') export default class PropertyDefinition extends PropertyDefinitionBaseModel {}`
 */
export default class PropertyDefinitionBaseModel extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, index: true })
  TargetModel!: string;

  @Field({ type: 'varchar', size: 255, notNull: true, index: true })
  PropertiesField!: string;

  /** Parent model short name; empty for App-level container. */
  @Field({ type: 'varchar', size: 255 })
  ContainerModel?: string | null;

  /** Parent record id; null for App-level container. */
  @Field({ type: 'varchar', size: 64, index: true })
  ContainerId?: string | null;

  /** Property item schema array (stored as JSON; physical jsonobject). */
  @Field({ type: 'jsonobject', notNull: true })
  Definition!: unknown[];

  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    await ensureDefinitionUniqueIndex(this as any);
    normalizeDefinitionOnVals(value as Record<string, unknown>);
    await assertUniqueDefinitionScope(this as any, value as Record<string, unknown>);
    return super.Create(value as any, returnFields as any) as Promise<T>;
  }

  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    await ensureDefinitionUniqueIndex(this as any);
    const seen = new Set<string>();
    for (const row of values || []) {
      normalizeDefinitionOnVals(row as Record<string, unknown>);
      const rec = row as Record<string, unknown>;
      const key = [
        String(rec.TargetModel ?? '').trim(),
        String(rec.PropertiesField ?? '').trim(),
        nullScope(rec.ContainerModel) ?? '',
        nullScope(rec.ContainerId) ?? '',
      ].join('\0');
      if (seen.has(key)) {
        fail('PROPERTY_DEFINITION_DUPLICATE_SCOPE', `PropertyDefinition CreateMany has duplicate scope "${key.replace(/\0/g, '/')}"`);
      }
      seen.add(key);
      await assertUniqueDefinitionScope(this as any, rec);
    }
    return super.CreateMany(values as any, returnFields as any) as Promise<T[]>;
  }

  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>[]> {
    await ensureDefinitionUniqueIndex(this as any);
    normalizeDefinitionOnVals(values as Record<string, unknown>);
    // Bulk Update cannot cheaply merge per-row scope; DB unique index is the backstop.
    return super.Update(condition as any, values as any, returnFields as any, options as any) as Promise<Partial<T>[]>;
  }

  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    await ensureDefinitionUniqueIndex(this as any);
    normalizeDefinitionOnVals(values as Record<string, unknown>);
    if (touchesDefinitionScope(values as Record<string, unknown>)) {
      const currentRows = await (this as any).Search(
        { And: [['Id', '=', id]] } as any,
        { fields: ['Id', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'] as any, limit: 1 } as any
      );
      const current = (currentRows && currentRows[0]) || {};
      await assertUniqueDefinitionScope(this as any, { ...current, ...(values as object) }, id);
    }
    return super.UpdateById(id, values as any, returnFields as any, options as any) as Promise<Partial<T>>;
  }
}

registerLogicalModelName('PropertyDefinition');
