// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModule from './module';
import MetaFieldRaw from './field_raw';
import MetaServiceRaw from './service_raw';
import MetaDecoratorRaw from './decorator_raw';

/**
 * Declaration-layer model (meta_raw_model). IMD may have many rows per (Application, Name).
 * Effective projection is MetaModel / meta_model.
 */
@Model('MetaModelRaw', {
  tableName: 'meta_raw_model',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaModelRaw extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaModelRaw.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, notNull: true, string: _lt('Path', { scope: 'meta.model.MetaModelRaw.fields' }) })
  Path!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Application', { scope: 'meta.model.MetaModelRaw.fields' }) })
  Application?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Class Name', { scope: 'meta.model.MetaModelRaw.fields' }) })
  ClassName?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Table Name', { scope: 'meta.model.MetaModelRaw.fields' }) })
  ModelTable?: string;

  @Field({ type: 'boolean', default: () => false, string: _lt('Abstract', { scope: 'meta.model.MetaModelRaw.fields' }) })
  Abstract?: boolean;

  @Field({ type: 'boolean', default: () => true, string: _lt('Auto Migrate', { scope: 'meta.model.MetaModelRaw.fields' }) })
  AutoMigrate?: boolean;

  @Field({ type: 'boolean', default: () => false, string: _lt('Readonly', { scope: 'meta.model.MetaModelRaw.fields' }) })
  Readonly?: boolean;

  @Field({ type: 'varchar', size: 255, string: _lt('Raw Extends', { scope: 'meta.model.MetaModelRaw.fields' }) })
  RawExtends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Extends', { scope: 'meta.model.MetaModelRaw.fields' }) })
  Extends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Company Field', { scope: 'meta.model.MetaModelRaw.fields' }) })
  CompanyField?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaModule },
    notNull: true,
    string: _lt('Module', { scope: 'meta.model.MetaModelRaw.fields' }),
  })
  ModuleId!: MetaModule;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaDecoratorRaw, inverseField: 'ModelId' },
    string: _lt('Decorators', { scope: 'meta.model.MetaModelRaw.fields' }),
  })
  Decorators?: MetaDecoratorRaw[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaServiceRaw, inverseField: 'ModelId' },
    string: _lt('Services', { scope: 'meta.model.MetaModelRaw.fields' }),
  })
  Services?: MetaServiceRaw[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaFieldRaw, inverseField: 'ModelId' },
    string: _lt('Fields', { scope: 'meta.model.MetaModelRaw.fields' }),
  })
  Fields?: MetaFieldRaw[];
}
