// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../../orm/model/model';
import { KernelValidationError, validateFields as validateKernelFields } from '../../orm/repository/validation';
import type { KernelValidationRule } from '../../orm/repository/validation';
import type { ConstraintContext, ConstraintMeta, ConstraintMethod, ValidationIssue } from '../../orm/metadata/constraint';
import { ValidationPipelineError } from '../../orm/metadata/constraint';
import { MetadataStorage } from '../../orm/metadata';
import type { FieldMetadata, ModelCtor } from '../../orm/metadata/field';
import type { ModelMetadata } from '../../orm/metadata/model';
import type { BaseQueryCondition, SearchOptions } from '../../orm/repository/types';
import { getRuntimeRepository } from '../runtime_repository_facade';
import type { ObjectRecord } from '../../../utils/types';

type ReferenceModelMeta = Pick<FieldMetadata, 'relation' | 'targetModel'>;
type RuntimeModelNameMeta = Pick<ModelMetadata, 'fullModelName' | 'application' | 'modelName'>;

type GlobalPool = {
  get?: (name: string) => unknown;
};

/**
 * ValidationEngine orchestrates kernel, platform, and constraint validation passes.
 */
export class ValidationEngine {
  /**
   * Runs the enabled validation stages and returns the collected issues.
   */
  static async validate(
    ctx: ConstraintContext,
    options?: {
      includeKernel?: boolean;
      includePlatform?: boolean;
      includeConstraints?: boolean;
      kernelRules?: KernelValidationRule[];
      platformCreateWriteWhitelist?: string[];
      platformRejectUnknownFields?: boolean;
      onPlatformCreateWhitelistHit?: (fields: string[]) => void;
    }
  ): Promise<ValidationIssue[]> {
    const includeKernel = options?.includeKernel ?? true;
    const includePlatform = options?.includePlatform ?? true;
    const includeConstraints = options?.includeConstraints ?? true;

    const issues: ValidationIssue[] = [];
    if (includeKernel) {
      issues.push(...(await this.validateKernelRules(ctx, options?.kernelRules)));
    }
    if (includePlatform) {
      issues.push(
        ...(await this.validatePlatformRules(ctx, {
          createWriteWhitelist: options?.platformCreateWriteWhitelist,
          rejectUnknownFields: options?.platformRejectUnknownFields,
          onCreateWhitelistHit: options?.onPlatformCreateWhitelistHit,
        }))
      );
    }
    if (includeConstraints) {
      issues.push(...(await this.validateConstraintMethods(ctx)));
    }

    return issues;
  }

  /**
   * Runs validation and throws when any error-severity issue is returned.
   */
  static async validateOrThrow(
    ctx: ConstraintContext,
    options?: {
      includeKernel?: boolean;
      includePlatform?: boolean;
      includeConstraints?: boolean;
      kernelRules?: KernelValidationRule[];
      platformCreateWriteWhitelist?: string[];
      platformRejectUnknownFields?: boolean;
      onPlatformCreateWhitelistHit?: (fields: string[]) => void;
    }
  ): Promise<void> {
    const issues = await this.validate(ctx, options);

    if (issues.some(issue => issue.severity === 'error')) {
      throw new ValidationPipelineError('validation pipeline failed', issues);
    }
  }

  /**
   * Runs kernel-level validation rules.
   */
  static async validateKernelRules(ctx: ConstraintContext, rules?: KernelValidationRule[]): Promise<ValidationIssue[]> {
    try {
      validateKernelFields(ctx.metadata, ctx.values, { mode: ctx.mode, rules });
      return [];
    } catch (error) {
      if (error instanceof KernelValidationError) {
        return [
          {
            scope: 'kernel',
            field: error.field,
            code: error.code,
            message: error.message,
            severity: 'error',
            meta: {
              legacyCode: 'kernel_validation_failed',
              ...(error.detail || {}),
            },
          },
        ];
      }

      const message = error instanceof Error ? error.message : String(error);
      return [
        {
          scope: 'kernel',
          code: 'kernel_validation_failed',
          message,
          severity: 'error',
        },
      ];
    }
  }

  /**
   * Runs platform-level validation rules such as readonly, field shape, and company-scope checks.
   */
  static async validatePlatformRules(
    _ctx: ConstraintContext,
    options?: {
      createWriteWhitelist?: string[];
      rejectUnknownFields?: boolean;
      onCreateWhitelistHit?: (fields: string[]) => void;
    }
  ): Promise<ValidationIssue[]> {
    const ctx = _ctx;
    const issues: ValidationIssue[] = [];

    if ((ctx.mode === 'create' || ctx.mode === 'update') && ctx.metadata.readonly) {
      const modelName = String(ctx.metadata.fullModelName || ctx.metadata.modelName || ctx.metadata.name || '').trim() || 'unknown';
      return [
        {
          scope: 'platform',
          code: 'platform_model_readonly',
          message: `model "${modelName}" is readonly and cannot be written`,
          severity: 'error',
        },
      ];
    }

    const fields = ctx.changedFields.size > 0 ? Array.from(ctx.changedFields) : Object.keys(ctx.values || {});
    const createWriteWhitelist = new Set((options?.createWriteWhitelist || []).map(field => String(field || '').trim()).filter(Boolean));
    const whitelistedHits: string[] = [];

    for (const field of fields) {
      if (!field || field.startsWith('__')) {
        continue;
      }

      const meta = ctx.metadata.fields.get(field);
      if (!meta) {
        if ((ctx.mode === 'create' || ctx.mode === 'update') && options?.rejectUnknownFields) {
          issues.push({
            scope: 'platform',
            field,
            code: 'platform_unknown_field',
            message: `field "${field}" does not exist on model metadata and cannot be written`,
            severity: 'error',
          });
        }
        continue;
      }

      const writeToSelectOnlyField = meta.select && !meta.column;
      const writeToComputedField = Boolean(meta.column?.compute);
      const shouldCheckWriteScope = ctx.mode === 'create' || ctx.mode === 'update';
      const isWhitelistedOnCreate = shouldCheckWriteScope && createWriteWhitelist.has(field);
      if (isWhitelistedOnCreate) {
        whitelistedHits.push(field);
      }

      if (shouldCheckWriteScope && writeToSelectOnlyField && !isWhitelistedOnCreate) {
        issues.push({
          scope: 'platform',
          field,
          code: 'platform_write_to_select_field',
          message: `field "${field}" is select-only and cannot be written`,
          severity: 'error',
        });
        continue;
      }

      if (shouldCheckWriteScope && writeToComputedField && !isWhitelistedOnCreate) {
        issues.push({
          scope: 'platform',
          field,
          code: 'platform_write_to_computed_field',
          message: `field "${field}" is computed and cannot be written directly`,
          severity: 'error',
        });
      }

      if (ctx.mode === 'preview') {
        continue;
      }

      if (meta.type !== 'ManyToOne' && meta.type !== 'ManyToOneRef') {
        continue;
      }

      const refId = this.resolveReferenceId(ctx.values?.[field]);
      if (!refId) {
        continue;
      }

      const enabledCompanyIds = this.extractEnabledCompanyIds(ctx.requestContext);
      if (this.isBaseCompanyTarget(meta)) {
        if (enabledCompanyIds.length > 0 && !enabledCompanyIds.includes(refId)) {
          issues.push({
            scope: 'platform',
            field,
            code: 'platform_cross_company_reference_violation',
            message: `reference "${field}" points to company "${refId}", which is outside ctx.enabledCompanyIds`,
            severity: 'error',
          });
        }
        continue;
      }

      const targetCtor = this.resolveReferenceTargetCtor(meta);
      if (!targetCtor) {
        continue;
      }

      const targetMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
      if (this.isBaseCompanyTarget(meta, targetMeta)) {
        if (enabledCompanyIds.length > 0 && !enabledCompanyIds.includes(refId)) {
          issues.push({
            scope: 'platform',
            field,
            code: 'platform_cross_company_reference_violation',
            message: `reference "${field}" points to company "${refId}", which is outside ctx.enabledCompanyIds`,
            severity: 'error',
          });
        }
        continue;
      }

      const targetIsCompanyScoped = Boolean(targetMeta.companyScoped && targetMeta.fields?.has('CompanyId'));
      if (!targetIsCompanyScoped) {
        continue;
      }

      const targetRepo = getRuntimeRepository(targetCtor);
      const condition: BaseQueryCondition = ['Id', '=', refId];
      const searchOptions: SearchOptions<ObjectRecord> = {
        fields: ['Id', 'CompanyId'],
      };
      const rows = await targetRepo.withDeleted().search(condition, searchOptions);

      if (!rows?.length) {
        issues.push({
          scope: 'platform',
          field,
          code: 'platform_cross_company_reference_not_visible',
          message: `reference "${field}" with id "${refId}" is not visible in current company scope`,
          severity: 'error',
        });
        continue;
      }

      const firstRow = (rows[0] ?? {}) as ObjectRecord;
      const targetCompanyId = String(firstRow.CompanyId ?? '').trim();

      if (targetCompanyId && enabledCompanyIds.length > 0 && !enabledCompanyIds.includes(targetCompanyId)) {
        issues.push({
          scope: 'platform',
          field,
          code: 'platform_cross_company_reference_violation',
          message: `reference "${field}" points to company "${targetCompanyId}", which is outside ctx.enabledCompanyIds`,
          severity: 'error',
        });
      }
    }

    if (whitelistedHits.length > 0 && options?.onCreateWhitelistHit) {
      options.onCreateWhitelistHit(Array.from(new Set(whitelistedHits)).sort());
    }

    return issues;
  }

  private static resolveReferenceId(raw: unknown): string | undefined {
    if (raw === null || raw === undefined) return undefined;
    if (typeof raw === 'string' || typeof raw === 'number' || typeof raw === 'bigint') {
      const id = String(raw).trim();
      return id || undefined;
    }
    if (typeof raw === 'object') {
      const record = raw as { Id?: unknown; id?: unknown };
      const id = record.Id ?? record.id;
      if (id === null || id === undefined) return undefined;
      const normalized = String(id).trim();
      return normalized || undefined;
    }
    return undefined;
  }

  private static isRuntimeModelCtor(value: unknown): value is ModelCtor<BaseModel> & typeof BaseModel {
    return typeof value === 'function';
  }

  private static hasConstructPrototype(value: unknown): value is ModelCtor<BaseModel> & typeof BaseModel {
    if (typeof value !== 'function') {
      return false;
    }
    const maybeWithProto = value as { prototype?: unknown };
    return maybeWithProto.prototype !== undefined;
  }

  private static getGlobalPool(): GlobalPool | undefined {
    const root = globalThis as unknown as ObjectRecord;
    const pool = root.pool;
    if (!pool || typeof pool !== 'object') {
      return undefined;
    }
    return pool as GlobalPool;
  }

  private static resolveReferenceTargetCtor(meta: ReferenceModelMeta): (ModelCtor<BaseModel> & typeof BaseModel) | undefined {
    const resolver = meta?.relation?.targetModel ?? meta?.targetModel;
    if (!resolver) return undefined;

    if (typeof resolver === 'function') {
      try {
        const ctor = (resolver as () => unknown)();
        if (ValidationEngine.isRuntimeModelCtor(ctor)) return ctor;
      } catch {
        // ignore and try direct constructor branch
      }

      if (ValidationEngine.hasConstructPrototype(resolver)) {
        return resolver;
      }
      return undefined;
    }

    if (typeof resolver === 'string') {
      const pool = ValidationEngine.getGlobalPool();
      const ctor = pool?.get?.(resolver);
      if (ValidationEngine.isRuntimeModelCtor(ctor)) {
        return ctor;
      }
    }

    return undefined;
  }

  private static isBaseCompanyTarget(meta: ReferenceModelMeta, targetMeta?: RuntimeModelNameMeta): boolean {
    if (targetMeta) {
      const fullModelName = String(targetMeta.fullModelName || '').trim();
      if (fullModelName === 'base.Company') return true;

      const app = String(targetMeta.application || '').trim();
      const modelName = String(targetMeta.modelName || '').trim();
      if (app === 'base' && modelName === 'Company') return true;
    }

    const resolver = meta?.relation?.targetModel || meta?.targetModel;
    if (typeof resolver === 'string') {
      return resolver.trim() === 'base.Company';
    }
    return false;
  }

  private static extractEnabledCompanyIds(requestContext: unknown): string[] {
    const ctx = (requestContext && typeof requestContext === 'object' ? requestContext : {}) as ObjectRecord;
    const raw = ctx.enabledCompanyIds ?? ctx.EnabledCompanyIds ?? ctx.activeCompanyId ?? ctx.ActiveCompanyId;
    const values = Array.isArray(raw) ? raw : raw == null ? [] : [raw];
    return Array.from(new Set(values.map(v => String(v ?? '').trim()).filter(Boolean)));
  }

  /**
   * Runs registered model constraint methods in priority order.
   *
   * Dispatches by {@link ConstraintMeta.isStatic}:
   * - Static handlers keep the legacy `(self, ctx)` contract.
   * - Instance handlers are invoked with `this` bound to a draft proxy
   *   and mutations are automatically written back to `ctx.values`.
   */
  static async validateConstraintMethods<TModel extends BaseModel>(ctx: ConstraintContext<TModel>): Promise<ValidationIssue[]> {
    const handlers = (ctx.metadata.constraintHandlers || [])
      .filter(handler => this.shouldRunConstraint(handler, ctx))
      .sort((left, right) => left.priority - right.priority || left.method.localeCompare(right.method));

    if (handlers.length === 0) {
      return [];
    }

    const self = this.buildConstraintSelf(ctx);
    const issues: ValidationIssue[] = [];
    const originalChangedFields = new Set(ctx.changedFields);
    const instanceTouched = new Set<string>();
    const instanceChanges: ObjectRecord = {};

    for (const handler of handlers) {
      if (handler.isStatic) {
        const executor = this.resolveConstraintMethod(ctx.model, handler);
        if (!executor) {
          issues.push({
            scope: 'constraint',
            method: handler.method,
            code: 'constraint_method_missing',
            message: `constraint method not found: ${handler.method}`,
            severity: 'error',
          });
          continue;
        }

        try {
          await executor(self, ctx);
        } catch (error) {
          if (error instanceof ValidationPipelineError) {
            issues.push(...error.issues);
            continue;
          }

          const message = error instanceof Error ? error.message : String(error);
          issues.push({
            scope: 'constraint',
            method: handler.method,
            code: 'constraint_execution_failed',
            message,
            severity: 'error',
          });
        }
        continue;
      }

      // Instance constraint: execute with `this` bound to a draft proxy.
      const instanceMethod = this.resolveInstanceConstraintMethod(ctx.model, handler);
      if (!instanceMethod) {
        issues.push({
          scope: 'constraint',
          method: handler.method,
          code: 'constraint_method_missing',
          message: `constraint method not found: ${handler.method}`,
          severity: 'error',
        });
        continue;
      }

      const { draft, changes } = this.createConstraintDraft(self, ctx.values, ctx.metadata.fields);
      try {
        await instanceMethod.call(draft);
        for (const field of Object.keys(changes)) {
          if (changes[field] !== undefined) {
            instanceChanges[field] = changes[field];
            instanceTouched.add(field);
          }
        }
      } catch (error) {
        if (error instanceof ValidationPipelineError) {
          issues.push(...error.issues);
          continue;
        }

        const message = error instanceof Error ? error.message : String(error);
        issues.push({
          scope: 'constraint',
          method: handler.method,
          code: 'constraint_execution_failed',
          message,
          severity: 'error',
        });
      }
    }

    // Write back instance constraint mutations to ctx.values.
    if (instanceTouched.size > 0) {
      this.applyConstraintWriteback(ctx, instanceTouched, instanceChanges);
    }

    // Run post-constraint re-validation for fields newly mutated by instance constraints.
    if (instanceTouched.size > 0) {
      const mutatedFields = new Set([...instanceTouched].filter(f => !originalChangedFields.has(f)));
      if (mutatedFields.size > 0) {
        const postIssues = await this.validatePostConstraintMutations(ctx, mutatedFields);
        issues.push(...postIssues);
      }
    }

    return issues;
  }

  private static shouldRunConstraint(handler: ConstraintMeta, ctx: ConstraintContext): boolean {
    if (ctx.mode === 'preview') {
      return handler.preview;
    }

    if (ctx.mode === 'create' && handler.alwaysOnCreate) {
      return true;
    }

    if (handler.fields.length === 0) {
      return true;
    }

    for (const field of handler.fields) {
      if (ctx.changedFields.has(field)) {
        return true;
      }
    }

    return false;
  }

  private static buildConstraintSelf<TModel extends BaseModel>(ctx: ConstraintContext<TModel>): TModel {
    if (ctx.self) {
      return ctx.self;
    }
    return Object.assign(Object.create(ctx.model.prototype), ctx.current || {}, ctx.values) as TModel;
  }

  private static resolveConstraintMethod<TModel extends BaseModel>(
    model: ModelCtor<TModel> & typeof BaseModel,
    handler: ConstraintMeta
  ): ConstraintMethod<TModel> | undefined {
    const owner = (handler.isStatic ? model : model.prototype) as unknown as ObjectRecord;
    const method = owner[handler.method];
    if (typeof method !== 'function') {
      return undefined;
    }
    return method.bind(owner) as ConstraintMethod<TModel>;
  }

  /**
   * Resolves an instance constraint method for `this`-based invocation.
   *
   * Unlike {@link resolveConstraintMethod}, the returned function is NOT pre-bound;
   * it is intended to be called via `fn.call(draftSelf)` so that `this` inside
   * the constraint refers to the draft proxy.
   */
  private static resolveInstanceConstraintMethod<TModel extends BaseModel>(
    model: ModelCtor<TModel> & typeof BaseModel,
    handler: ConstraintMeta
  ): ((this: TModel) => void | Promise<void>) | undefined {
    const owner = model.prototype as unknown as ObjectRecord;
    const method = owner[handler.method];
    if (typeof method !== 'function') {
      return undefined;
    }
    return method as (this: TModel) => void | Promise<void>;
  }

  /**
   * Creates a draft proxy that wraps a constraint `self` object.
   *
   * The proxy intercepts property writes and records them in `changes` / `touched`,
   * while reads follow the priority chain `changes → ctx.values → original self`.
   *
   * @returns The draft proxy and the mutable tracking state.
   */
  private static createConstraintDraft<TModel extends BaseModel>(
    self: TModel,
    payloadValues: ObjectRecord,
    fieldMetadata: Map<string, unknown>
  ): { draft: TModel; changes: ObjectRecord; touched: Set<string> } {
    const changes: ObjectRecord = {};
    const touched = new Set<string>();

    const draft = new Proxy(self as unknown as ObjectRecord, {
      get(_target, prop, receiver) {
        const key = String(prop);
        if (key in changes) return changes[key];
        if (payloadValues && key in payloadValues) return payloadValues[key];
        return Reflect.get(self as unknown as ObjectRecord, prop, receiver);
      },
      set(_target, prop, value, _receiver) {
        const key = String(prop);
        // Reject writes to fields not present in model metadata (prevents typo pollution).
        if (!fieldMetadata.has(key)) {
          return true;
        }
        changes[key] = value;
        touched.add(key);
        return true;
      },
    }) as unknown as TModel;

    return { draft, changes, touched };
  }

  /**
   * Writes back the changes collected by instance constraint draft proxies
   * into the constraint context's `values` payload.
   */
  private static applyConstraintWriteback<TModel extends BaseModel>(ctx: ConstraintContext<TModel>, touched: Set<string>, changes: ObjectRecord): void {
    for (const field of touched) {
      if (ctx.metadata.fields.has(field) || changes[field] !== undefined) {
        ctx.values[field] = changes[field];
      }
    }
  }

  /**
   * Re-runs kernel and platform validation for fields that were newly
   * introduced (mutated) by instance constraint methods, preventing
   * constraint writeback from silently bypassing earlier validation layers.
   */
  private static async validatePostConstraintMutations<TModel extends BaseModel>(
    ctx: ConstraintContext<TModel>,
    mutatedFields: Set<string>
  ): Promise<ValidationIssue[]> {
    if (mutatedFields.size === 0) return [];

    const postCtx: ConstraintContext<TModel> = {
      ...ctx,
      changedFields: mutatedFields,
    };

    const issues: ValidationIssue[] = [];
    issues.push(...(await this.validateKernelRules(postCtx)));
    issues.push(...(await this.validatePlatformRules(postCtx)));
    return issues;
  }
}
