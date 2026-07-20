// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrDecorator from './ir_decorator';
import IrModel from './ir_model';
import IrParameter from './ir_parameter';
import IrTypeParameter from './ir_type_parameter';

@Model('IrService', {
  tableName: 'meta_ir_service',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class IrService extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrService.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Origin Model Path', { scope: 'meta.model.IrService.fields' }) })
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Accessibility Modifier', { scope: 'meta.model.IrService.fields' }) })
  AccessibilityModifier?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('TS Type Annotation', { scope: 'meta.model.IrService.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Protobuf Type', { scope: 'meta.model.IrService.fields' }) })
  ProtobufType?: string;

  @Field({ type: 'boolean', string: _lt('Static', { scope: 'meta.model.IrService.fields' }) })
  IsStatic?: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModel }, string: _lt('Model', { scope: 'meta.model.IrService.fields' }) })
  ModelId?: IrModel;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrTypeParameter, inverseField: 'ServiceId' },
    string: _lt('Type Parameters', { scope: 'meta.model.IrService.fields' }),
  })
  TypeParameters?: IrTypeParameter[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrParameter, inverseField: 'ServiceId' },
    string: _lt('Parameters', { scope: 'meta.model.IrService.fields' }),
  })
  Parameters?: IrParameter[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrDecorator, inverseField: 'ServiceId' },
    string: _lt('Decorators', { scope: 'meta.model.IrService.fields' }),
  })
  Decorators?: IrDecorator[];
}
