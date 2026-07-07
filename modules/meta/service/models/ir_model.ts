// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type { EffectiveConstraintMeta } from '@/core/service/api/constraint';
import { MetadataStorage, type EffectiveOnchangeMeta } from '@/core/service/api/metadata';
import { normalizePagination, paginateAndWrap } from '@/core/service/utils/pagination';
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
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 512, notNull: true } })
  Path!: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Application?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ClassName?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ModelTable?: string;

  @Field({ type: 'boolean', column: { default: () => false } })
  Abstract?: boolean;

  @Field({ type: 'boolean', column: { default: () => true } })
  AutoMigrate?: boolean;

  @Field({ type: 'boolean', column: { default: () => false } })
  Readonly?: boolean;

  @Field({ type: 'varchar', column: { size: 255 } })
  RawExtends?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Extends?: string;

  @Field({ type: 'boolean' })
  CompanyScoped?: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModule } })
  ModuleId?: IrModule;

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrDecorator, inverseField: 'ModelId' } })
  Decorators?: IrDecorator[];

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrService, inverseField: 'ModelId' } })
  Services?: IrService[];

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrField, inverseField: 'ModelId' } })
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
    const key = String(modelIdentifier || '').trim();
    if (!key) {
      throw new Error('modelIdentifier cannot be empty');
    }

    const ctor = BaseModel.resolveModelConstructor(key);
    if (!ctor) {
      throw new Error(`model not found: ${key}`);
    }

    const meta = MetadataStorage.instance.getModelMetadata(ctor);
    const model =
      String(meta.fullModelName || '').trim() || String(meta.modelName || '').trim() || String(meta.name || '').trim() || String(ctor.name || '').trim() || key;

    const normalizedPrefix = String(options?.methodPrefix || '')
      .trim()
      .toLowerCase();
    const hasPreviewFilter = typeof options?.preview === 'boolean';
    const hasAlwaysOnCreateFilter = typeof options?.alwaysOnCreate === 'boolean';
    const pagination = normalizePagination(options);
    const hasMinPriority = typeof options?.minPriority === 'number' && Number.isFinite(options.minPriority);
    const hasMaxPriority = typeof options?.maxPriority === 'number' && Number.isFinite(options.maxPriority);
    const normalizedMinPriority = hasMinPriority ? Number(options?.minPriority) : undefined;
    const normalizedMaxPriority = hasMaxPriority ? Number(options?.maxPriority) : undefined;

    const effective = ctor.EffectiveConstraints() as EffectiveConstraintMeta[];
    const filtered = effective.filter(item => {
      if (hasPreviewFilter && item.preview !== Boolean(options?.preview)) return false;
      if (hasAlwaysOnCreateFilter && item.alwaysOnCreate !== Boolean(options?.alwaysOnCreate)) return false;
      const priority = typeof item.priority === 'number' && Number.isFinite(item.priority) ? item.priority : 0;
      if (normalizedMinPriority !== undefined && priority < normalizedMinPriority) return false;
      if (normalizedMaxPriority !== undefined && priority > normalizedMaxPriority) return false;
      if (
        normalizedPrefix &&
        !String(item.method || '')
          .toLowerCase()
          .startsWith(normalizedPrefix)
      )
        return false;
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
    const key = String(modelIdentifier || '').trim();
    if (!key) {
      throw new Error('modelIdentifier cannot be empty');
    }

    const ctor = BaseModel.resolveModelConstructor(key);
    if (!ctor) {
      throw new Error(`model not found: ${key}`);
    }

    const meta = MetadataStorage.instance.getModelMetadata(ctor);
    const model =
      String(meta.fullModelName || '').trim() || String(meta.modelName || '').trim() || String(meta.name || '').trim() || String(ctor.name || '').trim() || key;

    const normalizedPrefix = String(options?.methodPrefix || '')
      .trim()
      .toLowerCase();
    const normalizedTrigger = String(options?.triggerField || '')
      .trim()
      .toLowerCase();
    const pagination = normalizePagination(options);
    const hasMinPriority = typeof options?.minPriority === 'number' && Number.isFinite(options.minPriority);
    const hasMaxPriority = typeof options?.maxPriority === 'number' && Number.isFinite(options.maxPriority);
    const normalizedMinPriority = hasMinPriority ? Number(options?.minPriority) : undefined;
    const normalizedMaxPriority = hasMaxPriority ? Number(options?.maxPriority) : undefined;

    const effective = ctor.EffectiveOnchange() as EffectiveOnchangeMeta[];
    const filtered = effective.filter(item => {
      const priority = typeof item.priority === 'number' && Number.isFinite(item.priority) ? item.priority : 0;
      if (normalizedMinPriority !== undefined && priority < normalizedMinPriority) return false;
      if (normalizedMaxPriority !== undefined && priority > normalizedMaxPriority) return false;
      if (
        normalizedPrefix &&
        !String(item.method || '')
          .toLowerCase()
          .startsWith(normalizedPrefix)
      )
        return false;
      if (normalizedTrigger && !item.triggers.some(trigger => trigger.toLowerCase() === normalizedTrigger)) return false;
      return true;
    });

    return paginateAndWrap(filtered, 'onchanges', pagination, effective.length, { model }) as any;
  }
}
