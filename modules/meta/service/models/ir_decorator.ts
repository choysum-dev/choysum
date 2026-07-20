// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrComponent from './ir_component';
import IrField from './ir_field';
import IrModel from './ir_model';
import IrService from './ir_service';
import IrArgument from './ir_argument';

@Model('IrDecorator', {
  tableName: 'meta_ir_decorator',
  autoMigrate: false,
})
export default class IrDecorator extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrDecorator.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Module Spec Path', { scope: 'meta.model.IrDecorator.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Reference Ident', { scope: 'meta.model.IrDecorator.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModel }, string: _lt('Model', { scope: 'meta.model.IrDecorator.fields' }) })
  ModelId?: IrModel;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrService }, string: _lt('Service', { scope: 'meta.model.IrDecorator.fields' }) })
  ServiceId?: IrService;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrField }, string: _lt('Field', { scope: 'meta.model.IrDecorator.fields' }) })
  FieldId?: IrField;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrComponent }, string: _lt('Component', { scope: 'meta.model.IrDecorator.fields' }) })
  ComponentId?: IrComponent;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrArgument, inverseField: 'DecoratorId' },
    string: _lt('Arguments', { scope: 'meta.model.IrDecorator.fields' }),
  })
  Arguments?: IrArgument[];
}
