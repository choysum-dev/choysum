// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrDecorator from './ir_decorator';

@Model('IrArgument', {
  tableName: 'meta_ir_argument',
  autoMigrate: false,
})
export default class IrArgument extends BaseModel {
  @Field({ type: 'varchar' })
  Type?: string;

  @Field({ type: 'varchar' })
  Value?: string;

  @Field({ type: 'varchar' })
  ReferenceIdent?: string;

  @Field({ type: 'varchar' })
  ModuleSpecPath?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrDecorator } })
  DecoratorId?: IrDecorator;
}
