// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModelRaw from './model_raw';
import MetaServiceRaw from './service_raw';
import MetaFieldRaw from './field_raw';
import MetaArgumentRaw from './argument_raw';

@Model('MetaDecoratorRaw', {
  tableName: 'meta_raw_decorator',
  autoMigrate: false,
})
export default class MetaDecoratorRaw extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaDecoratorRaw.fields' }) })
  Name!: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModelRaw }, string: _lt('Model', { scope: 'meta.model.MetaDecoratorRaw.fields' }) })
  ModelId?: MetaModelRaw;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaServiceRaw }, string: _lt('Service', { scope: 'meta.model.MetaDecoratorRaw.fields' }) })
  ServiceId?: MetaServiceRaw;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaFieldRaw }, string: _lt('Field', { scope: 'meta.model.MetaDecoratorRaw.fields' }) })
  FieldId?: MetaFieldRaw;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaArgumentRaw, inverseField: 'DecoratorId' },
    string: _lt('Arguments', { scope: 'meta.model.MetaDecoratorRaw.fields' }),
  })
  Arguments?: MetaArgumentRaw[];
}
