// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModule from './module';

@Model('MetaModuleDependency', {
  tableName: 'meta_module_dependencies',
  autoMigrate: false,
})
export default class MetaModuleDependency extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModule }, string: _lt('Module', { scope: 'meta.model.MetaModuleDependency.fields' }) })
  ModuleId!: MetaModule;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaModule },
    string: _lt('Dependent Module', { scope: 'meta.model.MetaModuleDependency.fields' }),
  })
  DependModuleId!: MetaModule;
}
