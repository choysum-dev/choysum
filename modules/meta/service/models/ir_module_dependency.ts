// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrModule from './ir_module';

@Model('IrModuleDependency', {
  tableName: 'meta_ir_module_dependencies',
  autoMigrate: false,
})
export default class IrModuleDependency extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModule }, string: _lt('Module', { scope: 'meta.model.IrModuleDependency.fields' }) })
  ModuleId!: IrModule;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => IrModule },
    string: _lt('Dependent Module', { scope: 'meta.model.IrModuleDependency.fields' }),
  })
  DependModuleId!: IrModule;
}
