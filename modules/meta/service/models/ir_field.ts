// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrDecorator from './ir_decorator';
import IrModel from './ir_model';

@Model('IrField', {
  tableName: 'meta_ir_field',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class IrField extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true})
  Name!: string;

  @Field({ type: 'varchar', size: 1000})
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 1000})
  TsTypeReference?: string;

  @Field({ type: 'varchar', size: 512})
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255})
  FieldType?: string;

  @Field({ type: 'varchar', size: 255})
  Relation?: string;

  @Field({ type: 'varchar', size: 255})
  RelationModel?: string;

  @Field({ type: 'varchar', size: 255})
  RelationFilter?: string;

  @Field({ type: 'varchar', size: 255})
  RelationInverseField?: string;

  @Field({ type: 'varchar', size: 255})
  RelationJoinModel?: string;

  @Field({ type: 'varchar', size: 255})
  RelationJoinField?: string;

  @Field({ type: 'varchar', size: 255})
  RelationInverseJoinField?: string;

  @Field({ type: 'varchar', size: 255})
  RelationModelParentField?: string;

  @Field({ type: 'text' })
  Selection?: string;

  @Field({ type: 'varchar', size: 512 })
  FieldString?: string;

  @Field({ type: 'text' })
  StringText?: string;

  @Field({ type: 'varchar' })
  ReferenceIdent?: string;

  @Field({ type: 'varchar' })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255})
  AccessibilityModifier?: string;

  @Field({ type: 'boolean' })
  IsStatic?: boolean;

  @Field({ type: 'boolean' })
  IsReadonly?: boolean;

  @Field({ type: 'boolean' })
  Indexed?: boolean;

  @Field({ type: 'boolean' })
  NotNull?: boolean;

  @Field({ type: 'int' })
  Size?: number;

  @Field({ type: 'int' })
  Precision?: number;

  @Field({ type: 'int' })
  Scale?: number;

  @Field({ type: 'varchar', size: 255})
  ScaleField?: string;

  @Field({ type: 'varchar', size: 255})
  Round?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModel } })
  ModelId?: IrModel;

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrDecorator, inverseField: 'FieldId' } })
  Decorators?: IrDecorator[];
}
