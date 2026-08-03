// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { _lt, _t } from '../i18n';

const ERROR_DOMAIN = 'meta.MetaModelData';

/** Parse a full xml_id / external id key `module.name` (first `.` splits module vs name; aligns with host splitRef). */
export function parseMetaModelDataKey(xmlId: string): { module: string; name: string } {
  const raw = String(xmlId ?? '').trim();
  if (!raw) {
    throw new ChoysumError({
      domain: ERROR_DOMAIN,
      code: 'EXTERNAL_ID_INVALID_KEY',
      message: _t('external id key must not be empty', { scope: 'service/models/model_data' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  const dot = raw.indexOf('.');
  if (dot < 0) {
    throw new ChoysumError({
      domain: ERROR_DOMAIN,
      code: 'EXTERNAL_ID_INVALID_KEY',
      message: _t('external id key must be module.name', { scope: 'service/models/model_data' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  const module = raw.slice(0, dot).trim();
  const name = raw.slice(dot + 1).trim();
  if (!module || !name) {
    throw new ChoysumError({
      domain: ERROR_DOMAIN,
      code: 'EXTERNAL_ID_INVALID_KEY',
      message: _t('external id key must not have empty module or name', { scope: 'service/models/model_data' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return { module, name };
}

/**
 * Runtime facade over host table `meta_model_data` (Odoo `ir.model.data`).
 * Business code resolves stable `module.name` keys via dial; loaders still write via host Go.
 */
@Model('MetaModelData', {
  tableName: 'meta_model_data',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaModelData extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, index: true, string: _lt('Module', { scope: 'meta.model.MetaModelData.fields' }) })
  Module!: string;

  @Field({ type: 'varchar', size: 255, notNull: true, index: true, string: _lt('Name', { scope: 'meta.model.MetaModelData.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, notNull: true, index: true, string: _lt('Application', { scope: 'meta.model.MetaModelData.fields' }) })
  Application!: string;

  @Field({ type: 'varchar', size: 255, notNull: true, index: true, string: _lt('Model', { scope: 'meta.model.MetaModelData.fields' }) })
  Model!: string;

  @Field({ type: 'varchar', size: 20, notNull: true, index: true, string: _lt('Resource Id', { scope: 'meta.model.MetaModelData.fields' }) })
  ResId!: string;

  @Field({ type: 'boolean', notNull: true, default: false, string: _lt('No Update', { scope: 'meta.model.MetaModelData.fields' }) })
  NoUpdate!: boolean;

  /** Resolve `module.name` → business row id. Missing mapping raises. */
  static async Ref(xmlId: string): Promise<string> {
    const { module, name } = parseMetaModelDataKey(xmlId);
    const resId = await this.lookupResId(module, name);
    if (!resId) {
      throw new ChoysumError({
        domain: ERROR_DOMAIN,
        code: 'EXTERNAL_ID_NOT_FOUND',
        message: _t('external id %s not found', { scope: 'service/models/model_data' }, `${module}.${name}`),
      }).withGrpcCode(GrpcCode.NotFound);
    }
    return resId;
  }

  /** Resolve `module.name` → business row id, or `null` when missing. */
  static async RefOrNull(xmlId: string): Promise<string | null> {
    const { module, name } = parseMetaModelDataKey(xmlId);
    return await this.lookupResId(module, name);
  }

  private static async lookupResId(module: string, name: string): Promise<string | null> {
    const rows = await (this as any).Search(
      {
        And: [
          ['Module', '=', module],
          ['Name', '=', name],
        ],
      } as any,
      { limit: 1, fields: ['ResId'] } as any
    );
    const resId = String(rows?.[0]?.ResId ?? '').trim();
    return resId || null;
  }
}
