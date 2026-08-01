// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaComponent from './component';
import MetaField from './field';
import MetaModel from './model';
import MetaService from './service';
import MetaArgument from './argument';

@Model('MetaDecorator', {
  tableName: 'meta_decorator',
  autoMigrate: false,
})
export default class MetaDecorator extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaDecorator.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Module Spec Path', { scope: 'meta.model.MetaDecorator.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Reference Ident', { scope: 'meta.model.MetaDecorator.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModel }, string: _lt('Model', { scope: 'meta.model.MetaDecorator.fields' }) })
  ModelId?: MetaModel;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaService }, string: _lt('Service', { scope: 'meta.model.MetaDecorator.fields' }) })
  ServiceId?: MetaService;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaField }, string: _lt('Field', { scope: 'meta.model.MetaDecorator.fields' }) })
  FieldId?: MetaField;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaComponent }, string: _lt('Component', { scope: 'meta.model.MetaDecorator.fields' }) })
  ComponentId?: MetaComponent;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaArgument, inverseField: 'DecoratorId' },
    string: _lt('Arguments', { scope: 'meta.model.MetaDecorator.fields' }),
  })
  Arguments?: MetaArgument[];
}
