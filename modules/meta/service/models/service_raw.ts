// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModelRaw from './model_raw';
import MetaDecoratorRaw from './decorator_raw';
import MetaParameterRaw from './parameter_raw';
import MetaTypeParameterRaw from './type_parameter_raw';

@Model('MetaServiceRaw', {
  tableName: 'meta_raw_service',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaServiceRaw extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaServiceRaw.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Origin Model Path', { scope: 'meta.model.MetaServiceRaw.fields' }) })
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Accessibility Modifier', { scope: 'meta.model.MetaServiceRaw.fields' }) })
  AccessibilityModifier?: string;

  @Field({ type: 'boolean', string: _lt('Static', { scope: 'meta.model.MetaServiceRaw.fields' }) })
  IsStatic?: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModelRaw }, string: _lt('Model', { scope: 'meta.model.MetaServiceRaw.fields' }) })
  ModelId?: MetaModelRaw;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaTypeParameterRaw, inverseField: 'ServiceId' },
    string: _lt('Type Parameters', { scope: 'meta.model.MetaServiceRaw.fields' }),
  })
  TypeParameters?: MetaTypeParameterRaw[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaParameterRaw, inverseField: 'ServiceId' },
    string: _lt('Parameters', { scope: 'meta.model.MetaServiceRaw.fields' }),
  })
  Parameters?: MetaParameterRaw[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaDecoratorRaw, inverseField: 'ServiceId' },
    string: _lt('Decorators', { scope: 'meta.model.MetaServiceRaw.fields' }),
  })
  Decorators?: MetaDecoratorRaw[];
}
