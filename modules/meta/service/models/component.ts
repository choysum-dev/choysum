// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModule from './module';

@Model('MetaComponent', {
  tableName: 'meta_component',
  autoMigrate: false,
})
export default class MetaComponent extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaComponent.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, notNull: true, string: _lt('Path', { scope: 'meta.model.MetaComponent.fields' }) })
  Path!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Raw Extends', { scope: 'meta.model.MetaComponent.fields' }) })
  RawExtends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Extends', { scope: 'meta.model.MetaComponent.fields' }) })
  Extends?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModule }, string: _lt('Module', { scope: 'meta.model.MetaComponent.fields' }) })
  ModuleId?: MetaModule;
}
