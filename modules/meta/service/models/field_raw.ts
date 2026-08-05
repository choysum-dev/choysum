// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModelRaw from './model_raw';
import MetaDecoratorRaw from './decorator_raw';

@Model('MetaFieldRaw', {
  tableName: 'meta_raw_field',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaFieldRaw extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaFieldRaw.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 1000, string: _lt('TS Type Annotation', { scope: 'meta.model.MetaFieldRaw.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Origin Model Path', { scope: 'meta.model.MetaFieldRaw.fields' }) })
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Field Type', { scope: 'meta.model.MetaFieldRaw.fields' }) })
  FieldType?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModelRaw }, string: _lt('Model', { scope: 'meta.model.MetaFieldRaw.fields' }) })
  ModelId?: MetaModelRaw;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaDecoratorRaw, inverseField: 'FieldId' },
    string: _lt('Decorators', { scope: 'meta.model.MetaFieldRaw.fields' }),
  })
  Decorators?: MetaDecoratorRaw[];
}
