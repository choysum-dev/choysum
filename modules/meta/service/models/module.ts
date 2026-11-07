// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaApplication from './application';
import MetaComponent from './component';
import MetaModel from './model';
import MetaUiResource from './ui_resource';

@Model('MetaModule', {
  tableName: 'meta_module',
  autoMigrate: false,
})
export default class MetaModule extends BaseModel {
  @Field({ type: 'varchar', size: 255, unique: true, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaModule.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 1024, string: _lt('Short Description', { scope: 'meta.model.MetaModule.fields' }) })
  ShortDesc?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Version', { scope: 'meta.model.MetaModule.fields' }) })
  Version?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Tarball', { scope: 'meta.model.MetaModule.fields' }) })
  Tarball?: string;

  @Field({ type: 'text', string: _lt('Summary', { scope: 'meta.model.MetaModule.fields' }) })
  Summary?: string;

  @Field({ type: 'text', string: _lt('Description', { scope: 'meta.model.MetaModule.fields' }) })
  Description?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Application', { scope: 'meta.model.MetaModule.fields' }) })
  ApplicationStr?: string;

  @Field({ type: 'jsonobject', string: _lt('Entry Points', { scope: 'meta.model.MetaModule.fields' }) })
  EntryPoints?: Record<string, unknown> | null;

  @Field({ type: 'varchar', size: 512, string: _lt('Web Entry Point', { scope: 'meta.model.MetaModule.fields' }) })
  WebEntryPoint?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Service Entry Point', { scope: 'meta.model.MetaModule.fields' }) })
  ServiceEntryPoint?: string;

  @Field({ type: 'jsonobject', string: _lt('Dependencies', { scope: 'meta.model.MetaModule.fields' }) })
  DependsStr?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'jsonobject', string: _lt('External Dependencies', { scope: 'meta.model.MetaModule.fields' }) })
  ExternalDependencies?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'varchar', size: 255, string: _lt('Author', { scope: 'meta.model.MetaModule.fields' }) })
  Author?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('License', { scope: 'meta.model.MetaModule.fields' }) })
  License?: string;

  @Field({ type: 'text', string: _lt('Homepage', { scope: 'meta.model.MetaModule.fields' }) })
  Homepage?: string;

  @Field({ type: 'text', string: _lt('Repository', { scope: 'meta.model.MetaModule.fields' }) })
  Repository?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Path', { scope: 'meta.model.MetaModule.fields' }) })
  Path?: string;

  @Field({ type: 'varchar', size: 64, string: _lt('Status', { scope: 'meta.model.MetaModule.fields' }) })
  Status?: string;

  @Field({ type: 'varchar', size: 255, index: true, string: _lt('Category', { scope: 'meta.model.MetaModule.fields' }) })
  Category?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaApplication }, string: _lt('Application', { scope: 'meta.model.MetaModule.fields' }) })
  ApplicationId?: MetaApplication;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaModel, inverseField: 'ModuleId' },
    string: _lt('Models', { scope: 'meta.model.MetaModule.fields' }),
  })
  Models?: MetaModel[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaComponent, inverseField: 'ModuleId' },
    string: _lt('Components', { scope: 'meta.model.MetaModule.fields' }),
  })
  Components?: MetaComponent[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaUiResource, inverseField: 'ModuleId' },
    string: _lt('UI Resources', { scope: 'meta.model.MetaModule.fields' }),
  })
  UiResources?: MetaUiResource[];
}
