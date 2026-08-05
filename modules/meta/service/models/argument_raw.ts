// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaDecoratorRaw from './decorator_raw';

@Model('MetaArgumentRaw', {
  tableName: 'meta_raw_argument',
  autoMigrate: false,
})
export default class MetaArgumentRaw extends BaseModel {
  @Field({ type: 'varchar', string: _lt('Type', { scope: 'meta.model.MetaArgumentRaw.fields' }) })
  Type?: string;

  @Field({ type: 'varchar', string: _lt('Value', { scope: 'meta.model.MetaArgumentRaw.fields' }) })
  Value?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaDecoratorRaw }, string: _lt('Decorator', { scope: 'meta.model.MetaArgumentRaw.fields' }) })
  DecoratorId?: MetaDecoratorRaw;
}
