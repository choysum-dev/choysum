// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { raiseDomainError } from '@/core/service/error';
import BaseModel from './model';
import { registerLogicalModelName } from './logical_model_registry';
import { assertValidPropertyDefinitionItems } from './properties_types';
import type { Insertable, Updateable, FieldSelection, QueryCondition, UpdateOptions } from '../repository/types';

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function normalizeDefinitionOnVals(vals: Record<string, unknown> | undefined): void {
  if (!vals || !Object.prototype.hasOwnProperty.call(vals, 'Definition')) return;
  try {
    vals.Definition = assertValidPropertyDefinitionItems(vals.Definition);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    fail('PROPERTY_DEFINITION_INVALID', message);
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
    normalizeDefinitionOnVals(value as Record<string, unknown>);
    return super.Create(value as any, returnFields as any) as Promise<T>;
  }

  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    for (const row of values || []) {
      normalizeDefinitionOnVals(row as Record<string, unknown>);
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
    normalizeDefinitionOnVals(values as Record<string, unknown>);
    return super.Update(condition as any, values as any, returnFields as any, options as any) as Promise<Partial<T>[]>;
  }

  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    normalizeDefinitionOnVals(values as Record<string, unknown>);
    return super.UpdateById(id, values as any, returnFields as any, options as any) as Promise<Partial<T>>;
  }
}

registerLogicalModelName('PropertyDefinition');
