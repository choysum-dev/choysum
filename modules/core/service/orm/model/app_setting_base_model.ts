// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { MetadataStorage } from '../metadata/storage';
import { raiseDomainError } from '@/core/service/error';
import { deleteReqStateKeysByPrefix, memoizeInReqState } from '../../runtime/context';
import {
  getOrInitRepositoryReqServiceState,
  getRepositoryCurrentReq,
} from '../repository/authz';
import BaseModel from './model';
import type { InstantiableModelCtor } from './types';

/** Minimal surface for `pool<AppSettingModelCtor>('AppSetting')` typing. */
export type AppSettingModelCtor = {
  Get(key: string, defaultValue?: string | null): Promise<string | null>;
  Set(key: string, value: string | null | undefined): Promise<string | null>;
};

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function storeMeta(ctor: InstantiableModelCtor<AppSettingBaseModel>) {
  return MetadataStorage.instance.getModelMetadata(ctor as any);
}

function storeApplication(ctor: InstantiableModelCtor<AppSettingBaseModel>): string {
  return String(storeMeta(ctor).application || '').trim();
}

function normalizeKey(key: string): string {
  const k = String(key ?? '').trim();
  if (!k) {
    fail('APP_SETTING_INVALID_KEY', 'AppSetting key must not be empty');
  }
  return k;
}

function appSettingMemoKey(application: string, key: string): string {
  return `appSetting:${application}:${key}`;
}

function appSettingReqState(): Record<string, unknown> | undefined {
  return getOrInitRepositoryReqServiceState(getRepositoryCurrentReq()) as Record<string, unknown> | undefined;
}

function invalidateAppSettingMemo(application: string, key: string): void {
  const app = String(application || '').trim();
  const k = String(key || '').trim();
  if (!app || !k) return;
  deleteReqStateKeysByPrefix(appSettingReqState(), appSettingMemoKey(app, k));
}

async function findByKey(
  ctor: InstantiableModelCtor<AppSettingBaseModel>,
  key: string
): Promise<AppSettingBaseModel | undefined> {
  const rows = await (ctor as any).Search(
    { And: [['Key', '=', key]] } as any,
    { fields: ['Id', 'Key', 'Value'] as any, limit: 2 } as any
  );
  return (rows && rows[0]) || undefined;
}

/**
 * Per-application AppSetting store base (no `@Model`, no table).
 *
 * Thin app classes (hand-written or C2):
 * `@Model('AppSetting', { softDelete: false }) export default class AppSetting extends AppSettingBaseModel {}`
 *
 * Prefer `softDelete: false` so `Set(key, null)` hard-deletes and unique `key` can be reused.
 * Access via `SomeModel.pool<AppSettingModelCtor>('AppSetting').Get/Set` — do not import C2 thin classes.
 */
export default class AppSettingBaseModel extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, unique: true, index: true })
  Key!: string;

  @Field({ type: 'text', notNull: true })
  Value!: string;

  /**
   * Read setting by key. Missing row → `defaultValue` (default `null`).
   * Memoized per request on `(application, key)`.
   */
  static async Get(
    this: InstantiableModelCtor<AppSettingBaseModel>,
    key: string,
    defaultValue: string | null = null
  ): Promise<string | null> {
    const k = normalizeKey(key);
    const application = storeApplication(this);
    if (!application || application === 'core') {
      return defaultValue;
    }

    const memoKey = appSettingMemoKey(application, k);
    const found = await memoizeInReqState(appSettingReqState(), memoKey, async () => {
      const row = await findByKey(this, k);
      if (!row) return { miss: true as const };
      return { miss: false as const, value: String((row as any).Value ?? '') };
    });

    if (!found || (found as { miss?: boolean }).miss) {
      return defaultValue;
    }
    return (found as { value: string }).value;
  }

  /**
   * Upsert setting. `null`/`undefined` → hard-delete row when present; returns previous value or `null`.
   */
  static async Set(
    this: InstantiableModelCtor<AppSettingBaseModel>,
    key: string,
    value: string | null | undefined
  ): Promise<string | null> {
    const k = normalizeKey(key);
    const application = storeApplication(this);

    const existing = await findByKey(this, k);
    const previous = existing ? String((existing as any).Value ?? '') : null;

    if (value === null || value === undefined) {
      if (existing?.Id) {
        // Prefer hard delete: thin AppSetting models should set softDelete: false (see class JSDoc).
        await (this as any).DeleteById(existing.Id);
      }
      if (application) invalidateAppSettingMemo(application, k);
      return previous;
    }

    const stored = String(value);
    if (existing?.Id) {
      if (previous === stored) {
        if (application) invalidateAppSettingMemo(application, k);
        return previous;
      }
      await (this as any).UpdateById(existing.Id, { Value: stored } as any);
    } else {
      await (this as any).Create({ Key: k, Value: stored } as any);
    }
    if (application) invalidateAppSettingMemo(application, k);
    return previous;
  }
}
