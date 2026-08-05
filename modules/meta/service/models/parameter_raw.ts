// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaServiceRaw from './service_raw';

@Model('MetaParameterRaw', {
  tableName: 'meta_raw_parameter',
  autoMigrate: false,
})
export default class MetaParameterRaw extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaParameterRaw.fields' }) })
  Name!: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaServiceRaw }, string: _lt('Service', { scope: 'meta.model.MetaParameterRaw.fields' }) })
  ServiceId?: MetaServiceRaw;
}
