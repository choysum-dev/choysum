// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaDecorator from './decorator';
import MetaModel from './model';
import MetaParameter from './parameter';
import MetaTypeParameter from './type_parameter';

@Model('MetaService', {
  tableName: 'meta_service',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaService extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaService.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Origin Model Path', { scope: 'meta.model.MetaService.fields' }) })
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Accessibility Modifier', { scope: 'meta.model.MetaService.fields' }) })
  AccessibilityModifier?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('TS Type Annotation', { scope: 'meta.model.MetaService.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Protobuf Type', { scope: 'meta.model.MetaService.fields' }) })
  ProtobufType?: string;

  @Field({ type: 'boolean', string: _lt('Static', { scope: 'meta.model.MetaService.fields' }) })
  IsStatic?: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModel }, string: _lt('Model', { scope: 'meta.model.MetaService.fields' }) })
  ModelId?: MetaModel;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaTypeParameter, inverseField: 'ServiceId' },
    string: _lt('Type Parameters', { scope: 'meta.model.MetaService.fields' }),
  })
  TypeParameters?: MetaTypeParameter[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaParameter, inverseField: 'ServiceId' },
    string: _lt('Parameters', { scope: 'meta.model.MetaService.fields' }),
  })
  Parameters?: MetaParameter[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaDecorator, inverseField: 'ServiceId' },
    string: _lt('Decorators', { scope: 'meta.model.MetaService.fields' }),
  })
  Decorators?: MetaDecorator[];
}
