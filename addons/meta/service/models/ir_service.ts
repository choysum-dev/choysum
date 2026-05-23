// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
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
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 512 } })
  OriginModelPath?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  AccessibilityModifier?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ProtobufType?: string;

  @Field({ type: 'boolean' })
  IsStatic?: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModel } })
  ModelId?: IrModel;

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrTypeParameter, inverseField: 'ServiceId' } })
  TypeParameters?: IrTypeParameter[];

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrParameter, inverseField: 'ServiceId' } })
  Parameters?: IrParameter[];

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrDecorator, inverseField: 'ServiceId' } })
  Decorators?: IrDecorator[];
}
