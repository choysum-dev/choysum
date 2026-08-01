// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type { EffectiveConstraintMeta } from '@/core/service/api/constraint';
import { MetadataStorage, type EffectiveOnchangeMeta } from '@/core/service/api/metadata';
import { normalizePagination, paginateAndWrap } from '@/core/service/utils/pagination';
import { resolveEffectiveModel, normalizePriorityRange, priorityInRange, matchesMethodPrefix } from '@/core/service/orm/metadata/effective_query_helper';
import { _lt } from '../i18n';
import MetaDecorator from './decorator';
import MetaField from './field';
import MetaModule from './module';
import MetaService from './service';

type EffectiveConstraintsQueryOptions = {
  preview?: boolean;
  alwaysOnCreate?: boolean;
  methodPrefix?: string;
  limit?: number;
  offset?: number;
  minPriority?: number;
  maxPriority?: number;
};

type EffectiveOnchangeQueryOptions = {
  methodPrefix?: string;
  triggerField?: string;
  limit?: number;
  offset?: number;
  minPriority?: number;
  maxPriority?: number;
};

@Model('MetaModel', {
  tableName: 'meta_model',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class MetaModel extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaModel.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, notNull: true, string: _lt('Path', { scope: 'meta.model.MetaModel.fields' }) })
  Path!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Application', { scope: 'meta.model.MetaModel.fields' }) })
  Application?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Class Name', { scope: 'meta.model.MetaModel.fields' }) })
  ClassName?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Table Name', { scope: 'meta.model.MetaModel.fields' }) })
  ModelTable?: string;

  @Field({ type: 'boolean', default: () => false, string: _lt('Abstract', { scope: 'meta.model.MetaModel.fields' }) })
  Abstract?: boolean;

  @Field({ type: 'boolean', default: () => true, string: _lt('Auto Migrate', { scope: 'meta.model.MetaModel.fields' }) })
  AutoMigrate?: boolean;

  @Field({ type: 'boolean', default: () => false, string: _lt('Readonly', { scope: 'meta.model.MetaModel.fields' }) })
  Readonly?: boolean;

  @Field({ type: 'varchar', size: 255, string: _lt('Raw Extends', { scope: 'meta.model.MetaModel.fields' }) })
  RawExtends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Extends', { scope: 'meta.model.MetaModel.fields' }) })
  Extends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Company Field', { scope: 'meta.model.MetaModel.fields' }) })
  CompanyField?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaModule }, string: _lt('Module', { scope: 'meta.model.MetaModel.fields' }) })
  ModuleId?: MetaModule;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaDecorator, inverseField: 'ModelId' },
    string: _lt('Decorators', { scope: 'meta.model.MetaModel.fields' }),
  })
  Decorators?: MetaDecorator[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaService, inverseField: 'ModelId' },
    string: _lt('Services', { scope: 'meta.model.MetaModel.fields' }),
  })
  Services?: MetaService[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaField, inverseField: 'ModelId' },
    string: _lt('Fields', { scope: 'meta.model.MetaModel.fields' }),
  })
  Fields?: MetaField[];

  static async GetEffectiveConstraints(
    modelIdentifier: string,
    options?: EffectiveConstraintsQueryOptions
  ): Promise<{
    model: string;
    constraints: EffectiveConstraintMeta[];
    total: number;
    filtered: number;
    offset: number;
    limit?: number;
    returned: number;
  }> {
    const { ctor, model } = resolveEffectiveModel(modelIdentifier);

    const hasPreviewFilter = typeof options?.preview === 'boolean';
    const hasAlwaysOnCreateFilter = typeof options?.alwaysOnCreate === 'boolean';
    const pagination = normalizePagination(options);
    const priorityRange = normalizePriorityRange(options);

    const effective = ctor.EffectiveConstraints() as EffectiveConstraintMeta[];
    const filtered = effective.filter(item => {
      if (hasPreviewFilter && item.preview !== Boolean(options?.preview)) return false;
      if (hasAlwaysOnCreateFilter && item.alwaysOnCreate !== Boolean(options?.alwaysOnCreate)) return false;
      if (!priorityInRange(item, priorityRange)) return false;
      if (!matchesMethodPrefix(item, options?.methodPrefix)) return false;
      return true;
    });

    return paginateAndWrap(filtered, 'constraints', pagination, effective.length, { model }) as any;
  }

  static async GetEffectiveOnchange(
    modelIdentifier: string,
    options?: EffectiveOnchangeQueryOptions
  ): Promise<{
    model: string;
    onchanges: EffectiveOnchangeMeta[];
    total: number;
    filtered: number;
    offset: number;
    limit?: number;
    returned: number;
  }> {
    const { ctor, model } = resolveEffectiveModel(modelIdentifier);

    const normalizedTrigger = String(options?.triggerField || '')
      .trim()
      .toLowerCase();
    const pagination = normalizePagination(options);
    const priorityRange = normalizePriorityRange(options);

    const effective = ctor.EffectiveOnchange() as EffectiveOnchangeMeta[];
    const filtered = effective.filter(item => {
      if (!priorityInRange(item, priorityRange)) return false;
      if (!matchesMethodPrefix(item, options?.methodPrefix)) return false;
      if (normalizedTrigger && !item.triggers.some(trigger => trigger.toLowerCase() === normalizedTrigger)) return false;
      return true;
    });

    return paginateAndWrap(filtered, 'onchanges', pagination, effective.length, { model }) as any;
  }
}
