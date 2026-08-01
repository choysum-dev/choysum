// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaDecorator from './decorator';
import MetaModel from './model';

@Model('MetaField', {
  tableName: 'meta_field',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaField extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaField.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 1000, string: _lt('TS Type Annotation', { scope: 'meta.model.MetaField.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 1000, string: _lt('TS Type Reference', { scope: 'meta.model.MetaField.fields' }) })
  TsTypeReference?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Origin Model Path', { scope: 'meta.model.MetaField.fields' }) })
  OriginModelPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Field Type', { scope: 'meta.model.MetaField.fields' }) })
  FieldType?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation', { scope: 'meta.model.MetaField.fields' }) })
  Relation?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Model', { scope: 'meta.model.MetaField.fields' }) })
  RelationModel?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Filter', { scope: 'meta.model.MetaField.fields' }) })
  RelationFilter?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Inverse Field', { scope: 'meta.model.MetaField.fields' }) })
  RelationInverseField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Join Model', { scope: 'meta.model.MetaField.fields' }) })
  RelationJoinModel?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Join Field', { scope: 'meta.model.MetaField.fields' }) })
  RelationJoinField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Inverse Join Field', { scope: 'meta.model.MetaField.fields' }) })
  RelationInverseJoinField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Relation Model Parent Field', { scope: 'meta.model.MetaField.fields' }) })
  RelationModelParentField?: string;

  @Field({ type: 'text', string: _lt('Selection', { scope: 'meta.model.MetaField.fields' }) })
  Selection?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Title', { scope: 'meta.model.MetaField.fields' }) })
  FieldString?: string;

  @Field({ type: 'text', string: _lt('Title Text', { scope: 'meta.model.MetaField.fields' }) })
  StringText?: string;

  @Field({ type: 'varchar', string: _lt('Reference Ident', { scope: 'meta.model.MetaField.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'varchar', string: _lt('Module Spec Path', { scope: 'meta.model.MetaField.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Accessibility Modifier', { scope: 'meta.model.MetaField.fields' }) })
  AccessibilityModifier?: string;

  @Field({ type: 'boolean', string: _lt('Static', { scope: 'meta.model.MetaField.fields' }) })
  IsStatic?: boolean;

  @Field({ type: 'boolean', string: _lt('Readonly', { scope: 'meta.model.MetaField.fields' }) })
  IsReadonly?: boolean;

  @Field({ type: 'boolean', string: _lt('Indexed', { scope: 'meta.model.MetaField.fields' }) })
  Indexed?: boolean;

  @Field({ type: 'boolean', string: _lt('Required', { scope: 'meta.model.MetaField.fields' }) })
  NotNull?: boolean;

  @Field({ type: 'int', string: _lt('Size', { scope: 'meta.model.MetaField.fields' }) })
  Size?: number;

  @Field({ type: 'int', string: _lt('Precision', { scope: 'meta.model.MetaField.fields' }) })
  Precision?: number;

  @Field({ type: 'int', string: _lt('Scale', { scope: 'meta.model.MetaField.fields' }) })
  Scale?: number;

  @Field({ type: 'varchar', size: 255, string: _lt('Scale Field', { scope: 'meta.model.MetaField.fields' }) })
  ScaleField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Currency Field', { scope: 'meta.model.MetaField.fields' }) })
  CurrencyField?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Round', { scope: 'meta.model.MetaField.fields' }) })
  Round?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModel }, string: _lt('Model', { scope: 'meta.model.MetaField.fields' }) })
  ModelId?: MetaModel;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaDecorator, inverseField: 'FieldId' },
    string: _lt('Decorators', { scope: 'meta.model.MetaField.fields' }),
  })
  Decorators?: MetaDecorator[];
}
