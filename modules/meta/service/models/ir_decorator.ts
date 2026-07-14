// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
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
  @Field({ type: 'varchar', size: 255, notNull: true})
  Name!: string;

  @Field({ type: 'varchar', size: 255})
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255})
  ReferenceIdent?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModel } })
  ModelId?: IrModel;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrService } })
  ServiceId?: IrService;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrField } })
  FieldId?: IrField;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrComponent } })
  ComponentId?: IrComponent;

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrArgument, inverseField: 'DecoratorId' } })
  Arguments?: IrArgument[];
}
