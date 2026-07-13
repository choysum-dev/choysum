// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Entity } from '../repository';
import type { Repository } from '../repository';
import { Field, SqlCompute } from '../decorator';
import {
  QueryCondition,
  BaseQueryCondition,
  Operator,
  SearchOptions,
  Insertable,
  Updateable,
  Selectable,
  FieldSelection,
  SoftDeleteOptions,
  CountOptions,
  UpdateOptions,
  DeleteOptions,
  ReadGroupOptions,
  ReadGroupResult,
  ReadGroupCountOptions,
  GroupBySpec,
} from '../repository/types';
import { EntityConverter } from '../utils/converter';
import type { OnchangeTrigger } from '../metadata/field';
import type { RuntimeModelCtor } from './types';
import type { OnchangeDraft, OnchangeResult } from '../../runtime/onchange/types';
import type { Context } from '../../runtime/context';
import type { ObjectRecord } from '../../../utils/types';
import { MetadataStorage, getEffectiveConstraints, getEffectiveOnchange } from '../metadata/storage';
import type { EffectiveConstraintMeta, EffectiveOnchangeMeta } from '../metadata';
import {
  getModelContext,
  getModelCompanyId,
  getModelCompanyIds,
  getModelLang,
  getModelTimezone,
  getModelUserId,
  withModelContext,
  getInstanceModelContext,
  getInstanceModelCompanyId,
  getInstanceModelCompanyIds,
  getInstanceModelLang,
  getInstanceModelTimezone,
  getInstanceModelUserId,
  withInstanceModelContext,
} from './model_context_facade';
import {
  updateModelInstance,
  deleteModelInstance,
  loadModelInstance,
  reloadModelInstance,
  toTransportObject as toTransportObjectExternal,
} from './model_instance';
import { browseModel, browseManyModels, searchModels, countModels, readGroupedModels, countGroupedModels } from './model_read_facade';
import { withModelSavepoint, hydrateModelFacade, toPlainObject as toPlainObjectExternal, toEntity as toEntityExternal } from './model_edge_facade';
import { getModelRepository } from './model_internal_facade';
import { defaultModelValues, runModelOnchange } from './model_runtime_service_facade';
import { deleteModels, deleteModelById } from './model_delete_service_facade';
import { createModel, createManyModels } from './model_create_service_facade';
import { updateModels, updateModelById } from './model_update_service_facade';
import { currentBridgeFrame } from '../../runtime/compute/bridge';

// Delegated implementation.

type BaseModelCtor<T extends BaseModel> = { new (factoryToken: Symbol, entity: Entity, fields?: unknown): T };
type ModelLoadFieldSelection = FieldSelection<ObjectRecord>;

export type SqlComputeCtx<TModel extends BaseModel = BaseModel> = {
  field: {
    (path: string): unknown;
    (model: Function, path: string): unknown;
  };
  fieldExist: {
    (path: string): boolean;
    (model: Function, path: string): boolean;
  };
  model?: Function;
  str: {
    concat: (...items: unknown[]) => unknown;
    concatWs?: (separator: string, ...items: unknown[]) => unknown;
    lower: (value: unknown) => unknown;
  };
  fn: {
    coalesce: (...items: unknown[]) => unknown;
  };
  col?: (table: string, column: string) => unknown;
  selectFrom?: (table: string) => unknown;
};

export type SearchCtx<TModel extends BaseModel = BaseModel> = {
  field: <K extends Extract<keyof TModel, string>>(name: K) => string;
  op: () => Operator;
  value: <V = unknown>() => V;
  and: (clauses: BaseQueryCondition[]) => BaseQueryCondition;
  or: (clauses: BaseQueryCondition[]) => BaseQueryCondition;
  cmp: (left: unknown, op: Operator, right: unknown) => BaseQueryCondition;
};

export type InverseCtx<TModel extends BaseModel = BaseModel> = {
  value: <V = unknown>() => V;
  writePath: (path: string, value: unknown) => void;
  readPath: (path: string) => unknown;
  record: TModel;
};

export type DecoratorRuntimeBridge<TModel extends BaseModel = BaseModel> = {
  readonly $sql: SqlComputeCtx<TModel>;
  readonly $search: SearchCtx<TModel>;
  readonly $inverse: InverseCtx<TModel>;
};

/**
 * BaseModel exposes the caller-facing ORM facade shared by runtime models.
 * It provides request context accessors, persistence helpers, query entry points,
 * and serialization helpers while delegating implementation details to dedicated services.
 */
class BaseModel {
  /**
   * Returns the request context for the current static model call.
   */
  static get ctx(): Context {
    return getModelContext();
  }

  /**
   * Returns the request context bound to this model instance.
   */
  get ctx(): Context {
    return getInstanceModelContext(this);
  }

  /**
   * Returns the active company Id from the current request scope.
   */
  static get companyId(): string | undefined {
    return getModelCompanyId();
  }

  /**
   * Returns the active company Id for this model instance.
   */
  get companyId(): string | undefined {
    return getInstanceModelCompanyId(this);
  }

  /**
   * Returns all enabled company Ids from the current request scope.
   */
  static get companyIds(): string[] {
    return getModelCompanyIds();
  }

  /**
   * Returns all enabled company Ids for this model instance.
   */
  get companyIds(): string[] {
    return getInstanceModelCompanyIds(this);
  }

  /**
   * Returns the current request language.
   */
  static get lang(): string | undefined {
    return getModelLang();
  }

  /**
   * Returns the current language for this model instance.
   */
  get lang(): string | undefined {
    return getInstanceModelLang(this);
  }

  /**
   * Returns the current request timezone.
   */
  static get tz(): string | undefined {
    return getModelTimezone();
  }

  /**
   * Returns the current timezone for this model instance.
   */
  get tz(): string | undefined {
    return getInstanceModelTimezone(this);
  }

  /**
   * Returns the current request user Id.
   */
  static get userId(): string | undefined {
    return getModelUserId();
  }

  /**
   * Returns the current user Id for this model instance.
   */
  get userId(): string | undefined {
    return getInstanceModelUserId(this);
  }

  /**
   * Bridge slot for @SqlCompute runtime context (wired in runtime phase).
   */
  get $sql(): SqlComputeCtx<this> {
    return currentBridgeFrame<SqlComputeCtx<this>>(this, 'sql');
  }

  /**
   * Bridge slot for @Search runtime context (wired in runtime phase).
   */
  get $search(): SearchCtx<this> {
    return currentBridgeFrame<SearchCtx<this>>(this, 'search');
  }

  /**
   * Bridge slot for @Inverse runtime context (wired in runtime phase).
   */
  get $inverse(): InverseCtx<this> {
    return currentBridgeFrame<InverseCtx<this>>(this, 'inverse');
  }

  /**
   * Returns the current request user Id, throwing when no user identity
   * can be resolved.
   */
  static ensureUserId(): string {
    const id = String(this.userId || '').trim();
    if (!id) {
      throw new Error('current user is required');
    }
    return id;
  }

  /**
   * Resolve a model constructor by its identifier.
   *
   * The identifier can be a full model name (e.g. "meta.IrModule"),
   * a short model name ("IrModule"), the metadata name, or the
   * constructor class name.
   */
  static resolveModelConstructor(identifier: string): typeof BaseModel | undefined {
    const key = String(identifier || '').trim();
    if (!key) return undefined;

    const pool = (globalThis as any)?.pool;
    if (pool && typeof pool.get === 'function') {
      const ctor = pool.get(key);
      if (ctor && typeof ctor === 'function') {
        return ctor as typeof BaseModel;
      }
    }

    const models = (MetadataStorage.instance as any)?.models as Map<typeof BaseModel, any> | undefined;
    if (!models || typeof models.entries !== 'function') return undefined;

    for (const [ctor, meta] of models.entries()) {
      if (!ctor) continue;
      const fullModelName = String(meta?.fullModelName || '').trim();
      const modelName = String(meta?.modelName || '').trim();
      const name = String(meta?.name || '').trim();
      const className = String(ctor.name || '').trim();
      if (key === fullModelName || key === modelName || key === name || key === className) {
        return ctor;
      }
    }

    return undefined;
  }

  /**
   * Returns the resolved effective constraints for the calling model constructor.
   */
  static EffectiveConstraints<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel): EffectiveConstraintMeta[] {
    return getEffectiveConstraints(this as any);
  }

  /**
   * Returns the resolved effective onchange handlers for the calling model constructor.
   */
  static EffectiveOnchange<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel): EffectiveOnchangeMeta[] {
    return getEffectiveOnchange(this as any);
  }

  /**
   * Runs a function with additional static model context.
   */
  static withContext<R>(ctx: Partial<Context> | (() => Partial<Context>), fn: () => R, opts?: { merge?: boolean }): R {
    return withModelContext(ctx, fn, opts);
  }

  /**
   * Runs a function with additional context bound to this model instance.
   */
  withContext<R>(ctx: Partial<Context> | (() => Partial<Context>), fn: () => R, opts?: { merge?: boolean }): R {
    return withInstanceModelContext(this, ctx, fn, opts);
  }

  /**
   * Primary key for the model instance.
   */
  @Field({ type: 'char', size: 20, primaryKey: true })
  public readonly Id: string;

  /**
   * Display label resolved for the record.
   */
  @Field({
    type: 'varchar',
    size: 255,
  })
  public readonly DisplayName!: string;

  @SqlCompute<BaseModel>('DisplayName')
  sqlDisplayName() {
    if (this.$sql.fieldExist('Name')) {
      return this.$sql.field('Name');
    }
    if (this.$sql.fieldExist('Username')) {
      return this.$sql.field('Username');
    }
    return this.$sql.field('Id');
  }

  /**
   * Timestamp recorded when the record was created.
   */
  @Field({ type: 'datetime', index: true })
  public readonly CreatedAt: Date;

  /**
   * Timestamp recorded when the record was last updated.
   */
  @Field({ type: 'datetime', index: true })
  public UpdatedAt: Date;

  /**
   * Soft-delete timestamp for the record.
   */
  @Field({ type: 'datetime', index: true })
  public DeletedAt: Date;

  // The model_runtime layer stores the metadata cache on the constructor static.
  private static metadata: unknown;

  // Internal factory token; direct construction is forbidden.
  private static readonly FACTORY_TOKEN = Symbol('FACTORY_TOKEN');

  // fields stays mutable so callers can append to it when needed.
  /**
   * Internal constructor used by runtime factories and hydration helpers.
   * Direct instantiation is blocked by the factory token guard.
   */
  constructor(
    factoryToken: Symbol,
    private readonly entity: Entity,
    private fields?: unknown
  ) {
    if (factoryToken !== BaseModel.FACTORY_TOKEN) {
      throw new Error('Models cannot be directly instantiated. Use factory methods like Create(), Browse(), or Search() instead.');
    }
    EntityConverter.entityToModel(this, entity);
  }

  /**
   * Returns the repository bound to the current model constructor.
   */
  static getRepository<T extends BaseModel>(this: BaseModelCtor<T>): Repository {
    return getModelRepository(this as unknown as RuntimeModelCtor<T>);
  }

  /**
   * Updates this model instance in place and returns the refreshed instance.
   */
  async update(options?: UpdateOptions): Promise<this> {
    return await updateModelInstance(this, options);
  }

  /**
   * Deletes this model instance.
   */
  async delete(options?: DeleteOptions): Promise<void> {
    await deleteModelInstance(this, options);
  }

  /**
   * Loads additional fields onto this model instance.
   */
  async load(fields?: ModelLoadFieldSelection, options?: SoftDeleteOptions): Promise<this> {
    return await loadModelInstance(this, fields, options);
  }

  /**
   * Reloads this model instance from storage.
   */
  async reload(options?: SoftDeleteOptions): Promise<this> {
    return await reloadModelInstance(this, options);
  }

  /**
   * Resolves default values for a pending create payload.
   */
  static async DefaultGet<T extends BaseModel>(this: BaseModelCtor<T>, value: Partial<Insertable<T & BaseModel>>): Promise<Partial<Insertable<T & BaseModel>>> {
    return await defaultModelValues<T>(this as unknown as RuntimeModelCtor<T>, value);
  }

  /**
   * Creates one record and optionally returns a selected field projection.
   */
  static async Create<T extends BaseModel>(this: BaseModelCtor<T>, value: Partial<Insertable<T & BaseModel>>, returnFields?: FieldSelection<T>): Promise<T> {
    return await createModel<T>(this as unknown as RuntimeModelCtor<T>, value, returnFields);
  }

  /**
   * Creates multiple records and optionally returns a selected field projection.
   */
  static async CreateMany<T extends BaseModel>(
    this: BaseModelCtor<T>,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    return await createManyModels<T>(this as unknown as RuntimeModelCtor<T>, values, returnFields);
  }

  /**
   * Loads a single record by Id.
   */
  static async Browse<T extends BaseModel>(this: BaseModelCtor<T>, id: string, fields?: FieldSelection<T>, options?: SoftDeleteOptions): Promise<T> {
    return await browseModel<T>(this as unknown as RuntimeModelCtor<T>, id, fields, options);
  }

  /**
   * Loads multiple records by Id while preserving BrowseMany compatibility.
   */
  static async BrowseMany<T extends BaseModel>(
    this: BaseModelCtor<T>,
    ids: string[],
    fields?: (keyof Selectable<T>)[],
    options?: SoftDeleteOptions
  ): Promise<T[]> {
    return await browseManyModels<T>(this as unknown as RuntimeModelCtor<T>, ids, fields, options);
  }

  /**
   * Searches for records matching a query condition.
   */
  static async Search<T extends BaseModel>(this: BaseModelCtor<T>, condition: QueryCondition<T> | [] = [], options?: SearchOptions<T>): Promise<T[]> {
    return await searchModels<T>(this as unknown as RuntimeModelCtor<T>, condition, options);
  }

  /**
   * Counts records matching a query condition.
   */
  static async Count<T extends BaseModel>(this: BaseModelCtor<T>, condition: QueryCondition<T> | [] = [], options?: CountOptions): Promise<number> {
    return await countModels<T>(this as unknown as RuntimeModelCtor<T>, condition, options);
  }

  /**
   * Executes a grouped read and returns plain grouped results.
   */
  static async ReadGroup<T extends BaseModel>(
    this: BaseModelCtor<T>,
    groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
    condition: QueryCondition<T> | [] = [],
    options: ReadGroupOptions<T> = {}
  ): Promise<ReadGroupResult> {
    return await readGroupedModels<T>(this as unknown as RuntimeModelCtor<T>, groupby, condition, options);
  }

  /**
   * Counts top-level groups for a grouped read query.
   */
  static async ReadGroupCount<T extends BaseModel>(
    this: BaseModelCtor<T>,
    groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
    condition: QueryCondition<T> | [] = [],
    options: ReadGroupCountOptions<T> = {}
  ): Promise<number> {
    return await countGroupedModels<T>(this as unknown as RuntimeModelCtor<T>, groupby, condition, options);
  }

  /**
   * Updates all records matching a condition and optionally returns selected fields.
   */
  static async Update<T extends BaseModel>(
    this: BaseModelCtor<T>,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>[]> {
    return await updateModels<T>(this as unknown as RuntimeModelCtor<T>, condition, values, returnFields, options);
  }

  /**
   * Updates a single record by Id and optionally returns selected fields.
   */
  static async UpdateById<T extends BaseModel>(
    this: BaseModelCtor<T>,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    return await updateModelById<T>(this as unknown as RuntimeModelCtor<T>, id, values, returnFields, options);
  }

  /**
   * Deletes all records matching a condition.
   */
  static async Delete<T extends BaseModel>(this: BaseModelCtor<T>, condition: QueryCondition<T>, options?: DeleteOptions): Promise<number> {
    return await deleteModels<T>(this as unknown as RuntimeModelCtor<T>, condition, options);
  }

  /**
   * Deletes a single record by Id.
   */
  static async DeleteById<T extends BaseModel>(this: BaseModelCtor<T>, id: string, options?: DeleteOptions): Promise<number> {
    return await deleteModelById<T>(this as unknown as RuntimeModelCtor<T>, id, options);
  }

  /**
   * Runs onchange handlers for a draft payload and returns the accumulated result.
   */
  static async Onchange<T extends BaseModel>(
    this: BaseModelCtor<T>,
    draft: OnchangeDraft,
    changed: OnchangeTrigger<T>[],
    opts?: {
      withCompute?: boolean;
      maxIterations?: number;
      loopThreshold?: number;
    }
  ): Promise<OnchangeResult> {
    return await runModelOnchange<T>(this as unknown as RuntimeModelCtor<T>, draft, changed, opts);
  }

  /**
   * Protect an operation with a savepoint. Throwing rolls back to that savepoint.
   * Note: this is a convenience entry point that delegates to Repository.withSavepoint.
   */
  static async withSavepoint<T extends BaseModel, R>(this: BaseModelCtor<T>, fn: () => Promise<R>, name?: string): Promise<R> {
    return await withModelSavepoint<T, R>(this as unknown as RuntimeModelCtor<T>, fn, name);
  }

  /**
   * Transport serialization for top-level gRPC responses:
   * - Uses the EntityConverter plan path over entity plus fields with only the required normalization.
   * - Marks the return value with __choysum_plain so the shared service-runtime finalize step can short-circuit.
   */
  public toTransportObject(): ObjectRecord {
    return toTransportObjectExternal(this);
  }

  /**
   * Converts the model instance into a plain object.
   */
  public toPlainObject(): ObjectRecord {
    return toPlainObjectExternal(this);
  }

  /**
   * Converts the model instance into an entity-shaped object.
   */
  public toEntity(): ObjectRecord {
    return toEntityExternal(this);
  }

  /**
   * Hydrates a model instance from an entity payload.
   */
  static Hydrate<T extends BaseModel>(this: BaseModelCtor<T>, entity: ObjectRecord, fields?: FieldSelection<T>): T {
    return hydrateModelFacade<T>(this as unknown as RuntimeModelCtor<T>, entity, fields);
  }
}

export default BaseModel;
