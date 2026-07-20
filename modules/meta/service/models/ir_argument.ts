// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrDecorator from './ir_decorator';

@Model('IrArgument', {
  tableName: 'meta_ir_argument',
  autoMigrate: false,
})
export default class IrArgument extends BaseModel {
  @Field({ type: 'varchar', string: _lt('Type', { scope: 'meta.model.IrArgument.fields' }) })
  Type?: string;

  @Field({ type: 'varchar', string: _lt('Value', { scope: 'meta.model.IrArgument.fields' }) })
  Value?: string;

  @Field({ type: 'varchar', string: _lt('Reference Ident', { scope: 'meta.model.IrArgument.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'varchar', string: _lt('Module Spec Path', { scope: 'meta.model.IrArgument.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrDecorator }, string: _lt('Decorator', { scope: 'meta.model.IrArgument.fields' }) })
  DecoratorId?: IrDecorator;
}
