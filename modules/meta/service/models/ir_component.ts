// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrModule from './ir_module';

@Model('IrComponent', {
  tableName: 'meta_ir_component',
  autoMigrate: false,
})
export default class IrComponent extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrComponent.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, notNull: true, string: _lt('Path', { scope: 'meta.model.IrComponent.fields' }) })
  Path!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Raw Extends', { scope: 'meta.model.IrComponent.fields' }) })
  RawExtends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Extends', { scope: 'meta.model.IrComponent.fields' }) })
  Extends?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModule }, string: _lt('Module', { scope: 'meta.model.IrComponent.fields' }) })
  ModuleId?: IrModule;
}
