// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { raiseDomainError } from '@/core/service/error';
import { MetadataStorage } from '../metadata/storage';
import BaseModel from './model';
import { registerLogicalModelName } from './logical_model_registry';
import { assertValidPropertyDefinitionItems } from './properties_types';
import type { ModelCtor, RowOf } from './types';
import type { Insertable, Updateable, FieldSelection,
  Projected,
  RowOrProjected, QueryCondition, UpdateOptions, DeleteOptions } from '../repository/types';
import {
  assertPropertyDefinitionParentWritable,
  collectParentScopesToProbe,
  normalizeDefinitionContainerScopeOnVals,
  parentScopeKey,
} from './properties_definition_acl';
import { getChoysumRuntime } from '../../runtime/choysum_root';

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
  ctor: ModelCtor<PropertyDefinitionBaseModel>
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

function storeMeta(ctor: ModelCtor<PropertyDefinitionBaseModel>) {
  return MetadataStorage.instance.getModelMetadata(ctor);
}

function asDefinitionCtor<C extends ModelCtor>(ctor: C): ModelCtor<PropertyDefinitionBaseModel> {
  return ctor as unknown as ModelCtor<PropertyDefinitionBaseModel>;
}

function nullScope(value: unknown): string | null {
  if (value == null) return null;
  const s = String(value).trim();
  return s || null;
}

function scopeEq(field: string, value: string | null): [string, string, string | null] {
  return value == null ? [field, '=', null] : [field, '=', value];
}

/**
 * Application-level uniqueness for (TargetModel, PropertiesField, ContainerModel, ContainerId).
 * Complements the DB unique index and covers environments where DDL ensure is unavailable.
 */
async function assertUniqueDefinitionScope(
  ctor: ModelCtor<PropertyDefinitionBaseModel>,
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

  const rows = await ctor.Search(
    { And } as QueryCondition<PropertyDefinitionBaseModel>,
    {
    fields: ['Id'],
    limit: 1,
  });
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

async function assertParentsWritableDeduped(
  ctor: ModelCtor<PropertyDefinitionBaseModel>,
  scopes: Record<string, unknown>[]
): Promise<void> {
  const seen = new Set<string>();
  for (const scope of scopes) {
    const key = parentScopeKey(scope);
    if (seen.has(key)) continue;
    seen.add(key);
    await assertPropertyDefinitionParentWritable(ctor, scope);
  }
}

/**
 * Composite uniqueness for (TargetModel, PropertiesField, ContainerModel, ContainerId).
 * Uses COALESCE for all dialects so PostgreSQL 14 and older remain compatible
 * (avoids PG15-only NULLS NOT DISTINCT). DDL failures propagate.
 */
async function ensureDefinitionUniqueIndex(ctor: ModelCtor<PropertyDefinitionBaseModel>): Promise<void> {
  const meta = storeMeta(ctor);
  const table = typeof meta.tableName === 'function' ? String(meta.tableName()) : String(meta.tableName || '');
  if (!table || ensuredUniqueIndexTables.has(table)) return;

  const indexName = `uidx_${table}_definition_scope`;
  // Expression unique index: NULL/empty container dims collide (App-level + parent scopes).
  const ddl = `CREATE UNIQUE INDEX IF NOT EXISTS ${indexName} ON ${table} (target_model, properties_field, coalesce(container_model, ''), coalesce(container_id, ''))`;

  const db = getChoysumRuntime()?.db;
  const exec = db?.execute;
  // QuickJS bridge callables may not report typeof === 'function'; rely on presence + call.
  if (exec == null || db == null) {
    // No DDL surface (unit harness): uniqueness still enforced in assertUniqueDefinitionScope.
    return;
  }

  try {
    await exec.call(db, ddl, '[]');
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

  static override async Create<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    value: Partial<Insertable<RowOf<C>>>,
    returnFields?: F
  ): Promise<RowOrProjected<RowOf<C>, F>> {
    const self = asDefinitionCtor(this);
    const vals = value as Record<string, unknown>;
    await ensureDefinitionUniqueIndex(self);
    normalizeDefinitionContainerScopeOnVals(vals);
    normalizeDefinitionOnVals(vals);
    await assertPropertyDefinitionParentWritable(self, vals);
    await assertUniqueDefinitionScope(self, vals);
    return (await super.Create<C, F>(value, returnFields)) as RowOrProjected<RowOf<C>, F>;
  }

  static override async CreateMany<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    values: Partial<Insertable<RowOf<C>>>[],
    returnFields?: F
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const self = asDefinitionCtor(this);
    await ensureDefinitionUniqueIndex(self);
    const seen = new Set<string>();
    const probed = new Set<string>();
    for (const row of values || []) {
      const rec = row as Record<string, unknown>;
      normalizeDefinitionContainerScopeOnVals(rec);
      normalizeDefinitionOnVals(rec);
      const scopeKey = parentScopeKey(rec);
      if (!probed.has(scopeKey)) {
        await assertPropertyDefinitionParentWritable(self, rec);
        probed.add(scopeKey);
      }
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
      await assertUniqueDefinitionScope(self, rec);
    }
    return (await super.CreateMany<C, F>(values, returnFields)) as Array<
      RowOrProjected<RowOf<C>, F>
    >;
  }

  static override async Update<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<Array<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>>> {
    type UpdatedRows = Array<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>>;
    const self = asDefinitionCtor(this);
    const vals = values as Record<string, unknown>;
    await ensureDefinitionUniqueIndex(self);
    normalizeDefinitionContainerScopeOnVals(vals);
    normalizeDefinitionOnVals(vals);
    // Always re-check parent write for matching rows (any field mutation requires parent write).
    const currentRows = await self.Search(
      condition as QueryCondition<PropertyDefinitionBaseModel>,
      {
      fields: ['Id', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'],
    });
    const scopes: Record<string, unknown>[] = [];
    for (const current of currentRows || []) {
      scopes.push(...collectParentScopesToProbe(current as unknown as Record<string, unknown>, vals));
    }
    await assertParentsWritableDeduped(self, scopes);
    // Bulk Update cannot cheaply merge per-row scope uniqueness; DB unique index is the backstop.
    return (await super.Update<C, F>(condition, values, returnFields, options)) as UpdatedRows;
  }

  static override async UpdateById<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    id: string,
    values: Partial<Updateable<RowOf<C>>>,
    returnFields?: F,
    options?: UpdateOptions
  ): Promise<F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>> {
    type UpdatedRow = F extends FieldSelection<RowOf<C>> ? Projected<RowOf<C>, F> : Partial<RowOf<C>>;
    const self = asDefinitionCtor(this);
    const vals = values as Record<string, unknown>;
    await ensureDefinitionUniqueIndex(self);
    normalizeDefinitionContainerScopeOnVals(vals);
    normalizeDefinitionOnVals(vals);
    const currentRows = await self.Search(
      { And: [['Id', '=', id]] },
      { fields: ['Id', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'], limit: 1 }
    );
    const current = (currentRows && currentRows[0]) || {};
    const merged = { ...(current as unknown as Record<string, unknown>), ...(values as object) } as Record<string, unknown>;
    await assertParentsWritableDeduped(self, collectParentScopesToProbe(current as unknown as Record<string, unknown>, vals));
    if (touchesDefinitionScope(vals)) {
      await assertUniqueDefinitionScope(self, merged, id);
    }
    return (await super.UpdateById<C, F>(id, values, returnFields, options)) as UpdatedRow;
  }

  static override async Delete<C extends ModelCtor>(
    this: C,
    condition: QueryCondition<RowOf<C>>,
    options?: DeleteOptions
  ): Promise<number> {
    const self = asDefinitionCtor(this);
    const rows = await self.Search(
      condition as QueryCondition<PropertyDefinitionBaseModel>,
      {
      fields: ['Id', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'],
    });
    await assertParentsWritableDeduped(self, (rows || []) as unknown as Record<string, unknown>[]);
    return (await super.Delete<C>(condition, options)) as number;
  }

  static override async DeleteById<C extends ModelCtor>(
    this: C,
    id: string,
    options?: DeleteOptions
  ): Promise<number> {
    const self = asDefinitionCtor(this);
    const currentRows = await self.Search(
      { And: [['Id', '=', id]] },
      { fields: ['Id', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'], limit: 1 }
    );
    const current = (currentRows && currentRows[0]) || {};
    await assertPropertyDefinitionParentWritable(self, current as unknown as Record<string, unknown>);
    return (await super.DeleteById<C>(id, options)) as number;
  }
}

registerLogicalModelName('PropertyDefinition');
