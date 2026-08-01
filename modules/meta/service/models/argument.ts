// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaDecorator from './decorator';

@Model('MetaArgument', {
  tableName: 'meta_argument',
  autoMigrate: false,
})
export default class MetaArgument extends BaseModel {
  @Field({ type: 'varchar', string: _lt('Type', { scope: 'meta.model.MetaArgument.fields' }) })
  Type?: string;

  @Field({ type: 'varchar', string: _lt('Value', { scope: 'meta.model.MetaArgument.fields' }) })
  Value?: string;

  @Field({ type: 'varchar', string: _lt('Reference Ident', { scope: 'meta.model.MetaArgument.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'varchar', string: _lt('Module Spec Path', { scope: 'meta.model.MetaArgument.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaDecorator }, string: _lt('Decorator', { scope: 'meta.model.MetaArgument.fields' }) })
  DecoratorId?: MetaDecorator;
}
