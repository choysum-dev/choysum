// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getEffectiveConstraints, type EffectiveConstraintMeta } from '@/core/service/api/constraint';
import { MetadataStorage } from '@/core/service/api/metadata';
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

  private static resolveModelCtor(modelIdentifier: string): (new (...args: any[]) => BaseModel) | undefined {
    const key = String(modelIdentifier || '').trim();
    if (!key) return undefined;

    const pool = (globalThis as any)?.pool;
    if (pool && typeof pool.get === 'function') {
      const ctor = pool.get(key);
      if (ctor && typeof ctor === 'function') {
        return ctor as new (...args: any[]) => BaseModel;
      }
    }

    const models = (MetadataStorage.instance as any)?.models as Map<new (...args: any[]) => BaseModel, any> | undefined;
    if (!models || typeof models.entries !== 'function') return undefined;

    for (const [ctor, meta] of models.entries()) {
      const fullModelName = String(meta?.fullModelName || '').trim();
      const modelName = String(meta?.modelName || '').trim();
      const name = String(meta?.name || '').trim();
      const className = String((ctor as any)?.name || '').trim();
      if (key === fullModelName || key === modelName || key === name || key === className) {
        return ctor;
      }
    }

    return undefined;
  }

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

    const ctor = this.resolveModelCtor(key);
    if (!ctor) {
      throw new Error(`model not found: ${key}`);
    }

    const meta = MetadataStorage.instance.getModelMetadata(ctor as any);
    const model =
      String(meta.fullModelName || '').trim() ||
      String(meta.modelName || '').trim() ||
      String(meta.name || '').trim() ||
      String((ctor as any)?.name || '').trim() ||
      key;

    const normalizedPrefix = String(options?.methodPrefix || '')
      .trim()
      .toLowerCase();
    const hasPreviewFilter = typeof options?.preview === 'boolean';
    const hasAlwaysOnCreateFilter = typeof options?.alwaysOnCreate === 'boolean';
    const normalizedLimit = typeof options?.limit === 'number' && Number.isFinite(options.limit) && options.limit > 0 ? Math.floor(options.limit) : undefined;
    const normalizedOffset = typeof options?.offset === 'number' && Number.isFinite(options.offset) && options.offset > 0 ? Math.floor(options.offset) : 0;
    const hasMinPriority = typeof options?.minPriority === 'number' && Number.isFinite(options.minPriority);
    const hasMaxPriority = typeof options?.maxPriority === 'number' && Number.isFinite(options.maxPriority);
    const normalizedMinPriority = hasMinPriority ? Number(options?.minPriority) : undefined;
    const normalizedMaxPriority = hasMaxPriority ? Number(options?.maxPriority) : undefined;

    const effective = getEffectiveConstraints(ctor as any);
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

    const paged = normalizedLimit ? filtered.slice(normalizedOffset, normalizedOffset + normalizedLimit) : filtered.slice(normalizedOffset);

    return {
      model,
      constraints: paged,
      total: effective.length,
      filtered: filtered.length,
      offset: normalizedOffset,
      limit: normalizedLimit,
      returned: paged.length,
    };
  }
}
