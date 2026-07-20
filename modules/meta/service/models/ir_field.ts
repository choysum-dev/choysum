// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrDecorator from './ir_decorator';
import IrModel from './ir_model';

@Model('IrField', {
  tableName: 'meta_ir_field',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class IrField extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrField.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 1000, string: _lt('TS Type Annotation', { scope: 'meta.model.IrField.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 1000, string: _lt('TS Type Reference', { scope: 'meta.model.IrField.fields' }) })
  TsTypeReference?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Origin Model Path', { scope: 'meta.model.IrField.fields' }) })
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Field Type', { scope: 'meta.model.IrField.fields' }) })
  FieldType?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation', { scope: 'meta.model.IrField.fields' }) })
  Relation?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Model', { scope: 'meta.model.IrField.fields' }) })
  RelationModel?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Filter', { scope: 'meta.model.IrField.fields' }) })
  RelationFilter?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Inverse Field', { scope: 'meta.model.IrField.fields' }) })
  RelationInverseField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Join Model', { scope: 'meta.model.IrField.fields' }) })
  RelationJoinModel?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Join Field', { scope: 'meta.model.IrField.fields' }) })
  RelationJoinField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Inverse Join Field', { scope: 'meta.model.IrField.fields' }) })
  RelationInverseJoinField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Model Parent Field', { scope: 'meta.model.IrField.fields' }) })
  RelationModelParentField?: string;

  @Field({ type: 'text', string: _lt('Selection', { scope: 'meta.model.IrField.fields' }) })
  Selection?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Title', { scope: 'meta.model.IrField.fields' }) })
  FieldString?: string;

  @Field({ type: 'text', string: _lt('Title Text', { scope: 'meta.model.IrField.fields' }) })
  StringText?: string;

  @Field({ type: 'varchar', string: _lt('Reference Ident', { scope: 'meta.model.IrField.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'varchar', string: _lt('Module Spec Path', { scope: 'meta.model.IrField.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Accessibility Modifier', { scope: 'meta.model.IrField.fields' }) })
  AccessibilityModifier?: string;

  @Field({ type: 'boolean', string: _lt('Static', { scope: 'meta.model.IrField.fields' }) })
  IsStatic?: boolean;

  @Field({ type: 'boolean', string: _lt('Readonly', { scope: 'meta.model.IrField.fields' }) })
  IsReadonly?: boolean;

  @Field({ type: 'boolean', string: _lt('Indexed', { scope: 'meta.model.IrField.fields' }) })
  Indexed?: boolean;

  @Field({ type: 'boolean', string: _lt('Required', { scope: 'meta.model.IrField.fields' }) })
  NotNull?: boolean;

  @Field({ type: 'int', string: _lt('Size', { scope: 'meta.model.IrField.fields' }) })
  Size?: number;

  @Field({ type: 'int', string: _lt('Precision', { scope: 'meta.model.IrField.fields' }) })
  Precision?: number;

  @Field({ type: 'int', string: _lt('Scale', { scope: 'meta.model.IrField.fields' }) })
  Scale?: number;

  @Field({ type: 'varchar', size: 255, string: _lt('Scale Field', { scope: 'meta.model.IrField.fields' }) })
  ScaleField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Round', { scope: 'meta.model.IrField.fields' }) })
  Round?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModel }, string: _lt('Model', { scope: 'meta.model.IrField.fields' }) })
  ModelId?: IrModel;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrDecorator, inverseField: 'FieldId' },
    string: _lt('Decorators', { scope: 'meta.model.IrField.fields' }),
  })
  Decorators?: IrDecorator[];
}
