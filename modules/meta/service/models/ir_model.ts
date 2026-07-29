// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type { EffectiveConstraintMeta } from '@/core/service/api/constraint';
import { MetadataStorage, type EffectiveOnchangeMeta } from '@/core/service/api/metadata';
import { normalizePagination, paginateAndWrap } from '@/core/service/utils/pagination';
import { resolveEffectiveModel, normalizePriorityRange, priorityInRange, matchesMethodPrefix } from '@/core/service/orm/metadata/effective_query_helper';
import { _lt } from '../i18n';
import IrDecorator from './ir_decorator';
import IrField from './ir_field';
import IrModule from './ir_module';
import IrService from './ir_service';

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

@Model('IrModel', {
  tableName: 'meta_ir_model',
  autoMigrate: false,
  orderBy: { field: 'Id', order: 'asc' },
})
export default class IrModel extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrModel.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 512, notNull: true, string: _lt('Path', { scope: 'meta.model.IrModel.fields' }) })
  Path!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Application', { scope: 'meta.model.IrModel.fields' }) })
  Application?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Class Name', { scope: 'meta.model.IrModel.fields' }) })
  ClassName?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Table Name', { scope: 'meta.model.IrModel.fields' }) })
  ModelTable?: string;

  @Field({ type: 'boolean', default: () => false, string: _lt('Abstract', { scope: 'meta.model.IrModel.fields' }) })
  Abstract?: boolean;

  @Field({ type: 'boolean', default: () => true, string: _lt('Auto Migrate', { scope: 'meta.model.IrModel.fields' }) })
  AutoMigrate?: boolean;

  @Field({ type: 'boolean', default: () => false, string: _lt('Readonly', { scope: 'meta.model.IrModel.fields' }) })
  Readonly?: boolean;

  @Field({ type: 'varchar', size: 255, string: _lt('Raw Extends', { scope: 'meta.model.IrModel.fields' }) })
  RawExtends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Extends', { scope: 'meta.model.IrModel.fields' }) })
  Extends?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Company Field', { scope: 'meta.model.IrModel.fields' }) })
  CompanyField?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModule }, string: _lt('Module', { scope: 'meta.model.IrModel.fields' }) })
  ModuleId?: IrModule;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrDecorator, inverseField: 'ModelId' },
    string: _lt('Decorators', { scope: 'meta.model.IrModel.fields' }),
  })
  Decorators?: IrDecorator[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrService, inverseField: 'ModelId' },
    string: _lt('Services', { scope: 'meta.model.IrModel.fields' }),
  })
  Services?: IrService[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrField, inverseField: 'ModelId' },
    string: _lt('Fields', { scope: 'meta.model.IrModel.fields' }),
  })
  Fields?: IrField[];

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
