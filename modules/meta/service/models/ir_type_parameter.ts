// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrService from './ir_service';

@Model('IrTypeParameter', {
  tableName: 'meta_ir_type_parameter',
  autoMigrate: false,
})
export default class IrTypeParameter extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrTypeParameter.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Module Spec Path', { scope: 'meta.model.IrTypeParameter.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Reference Ident', { scope: 'meta.model.IrTypeParameter.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrService }, string: _lt('Service', { scope: 'meta.model.IrTypeParameter.fields' }) })
  ServiceId?: IrService;
}
