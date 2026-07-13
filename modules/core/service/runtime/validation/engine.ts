// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../../orm/model/model';
import { KernelValidationError, validateFields as validateKernelFields } from '../../orm/repository/validation';
import type { KernelValidationRule } from '../../orm/repository/validation';
import type { ConstraintContext, ConstraintMeta, InstanceConstraintMethod, LegacyConstraintMethod, ValidationIssue } from '../../orm/metadata/constraint';
import { ValidationPipelineError } from '../../orm/metadata/constraint';
import { MetadataStorage } from '../../orm/metadata';
import type { FieldMetadata, ModelCtor } from '../../orm/metadata/field';
import type { ModelMetadata } from '../../orm/metadata/model';
import type { BaseQueryCondition, SearchOptions } from '../../orm/repository/types';
import { getRuntimeRepository } from '../runtime_repository_facade';
import type { ObjectRecord } from '../../../utils/types';

type ReferenceModelMeta = Pick<FieldMetadata, 'relation'>;
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
      issues.push(...(await this.validateConstraintMethods(ctx, options)));
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

    const fields = Array.from(new Set<string>([...Array.from(ctx.changedFields || []), ...Object.keys(ctx.values || {})]));
    const createWriteWhitelist = new Set((options?.createWriteWhitelist || []).map(field => String(field || '').trim()).filter(Boolean));
    const whitelistedHits: string[] = [];

    for (const field of fields) {
      if (!field || field.startsWith('__')) {
        continue;
      }

      const shouldCheckWriteScope = ctx.mode === 'create' || ctx.mode === 'update';
      const isWhitelistedOnCreate = shouldCheckWriteScope && createWriteWhitelist.has(field);
      if (isWhitelistedOnCreate) {
        whitelistedHits.push(field);
      }

      if (shouldCheckWriteScope && field === 'DisplayName' && !isWhitelistedOnCreate) {
        issues.push({
          scope: 'platform',
          field,
          code: 'platform_write_to_select_field',
          message: `field "${field}" is select-only and cannot be written`,
          severity: 'error',
        });
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

      const computeHandler = ctx.metadata.computeHandlers?.get(field);
      const sqlComputeHandler = ctx.metadata.sqlComputeHandlers?.get(field);
      const isVirtualComputeField = Boolean(ctx.metadata.computeGraph?.virtualComputeFields?.has(field));
      const writeToSelectOnlyField = (Boolean(sqlComputeHandler) && !meta.column) || computeHandler?.store === false || isVirtualComputeField;
      const writeToComputedField = Boolean(computeHandler) || Boolean(sqlComputeHandler);

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
    const resolver = (meta?.relation as { targetModel?: unknown } | undefined)?.targetModel;
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

    const resolver = (meta?.relation as { targetModel?: unknown } | undefined)?.targetModel;
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
  static async validateConstraintMethods<TModel extends BaseModel>(
    ctx: ConstraintContext<TModel>,
    options?: {
      kernelRules?: KernelValidationRule[];
      platformCreateWriteWhitelist?: string[];
      platformRejectUnknownFields?: boolean;
      onPlatformCreateWhitelistHit?: (fields: string[]) => void;
    }
  ): Promise<ValidationIssue[]> {
    const handlers = (ctx.metadata.constraintHandlers || [])
      .filter(handler => this.shouldRunConstraint(handler, ctx))
      .sort((left, right) => left.priority - right.priority || left.method.localeCompare(right.method));

    if (handlers.length === 0) {
      return [];
    }

    const issues: ValidationIssue[] = [];
    const instanceTouched = new Set<string>();

    for (const handler of handlers) {
      // Rebuild `self` on every iteration so that earlier instance-constraint
      // mutations (already flushed to `ctx.values`) are visible to subsequent
      // static handlers.
      const self = this.buildConstraintSelf(ctx);

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

      // Create draft proxy for the current handler.  Previous mutations
      // are already flushed to `ctx.values`, which is passed as the payload
      // source, so the draft naturally sees them through the resolve chain.
      const { draft, changes } = this.createConstraintDraft(self, ctx.values, ctx.metadata.fields);
      try {
        await instanceMethod.call(draft);
        for (const field of Object.keys(changes)) {
          instanceTouched.add(field);
          // Flush to ctx.values immediately so the next handler's draft
          // (which reads ctx.values as a fallback) can observe the update.
          ctx.values[field] = changes[field];
          // When ctx.self is explicitly provided (e.g. preview mode),
          // buildConstraintSelf returns it directly.  Write back so that
          // subsequent static handlers see the mutation through `self`.
          if (ctx.self) {
            (ctx.self as unknown as ObjectRecord)[field] = changes[field];
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

    // Run post-constraint re-validation for fields mutated by instance constraints.
    // Re-validate ALL touched fields — not just the newly introduced ones —
    // because a constraint may have changed a user-submitted value (e.g.
    // normalizing a string) and the mutated value must pass kernel/platform
    // rules before it is persisted.
    if (instanceTouched.size > 0) {
      const postIssues = await this.validatePostConstraintMutations(ctx, instanceTouched, options);
      issues.push(...postIssues);
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

  /**
   * Resolves a legacy constraint method for `(self, ctx)` invocation.
   *
   * When {@link ConstraintMeta.isStatic} is true the method is resolved from the
   * model constructor; otherwise from the prototype.  In both cases the function
   * is pre-bound to its owner so it can be called directly.
   *
   * For inherited constraints the prototype chain naturally resolves the
   * closest override (child prototype before parent prototype).
   */
  private static resolveConstraintMethod<TModel extends BaseModel>(
    model: ModelCtor<TModel> & typeof BaseModel,
    handler: ConstraintMeta
  ): LegacyConstraintMethod<TModel> | undefined {
    const owner = (handler.isStatic ? model : model.prototype) as unknown as ObjectRecord;
    const method = owner[handler.method];
    if (typeof method !== 'function') {
      return undefined;
    }
    return method.bind(owner) as LegacyConstraintMethod<TModel>;
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
  ): InstanceConstraintMethod<TModel> | undefined {
    const owner = model.prototype as unknown as ObjectRecord;
    const method = owner[handler.method];
    if (typeof method !== 'function') {
      return undefined;
    }
    return method as InstanceConstraintMethod<TModel>;
  }

  private static createConstraintDraft<TModel extends BaseModel>(
    self: TModel,
    payloadValues: ObjectRecord,
    fieldMetadata: Map<string, unknown>
  ): { draft: TModel; changes: ObjectRecord } {
    // Use a null-prototype object so that standard prototype properties
    // (e.g. `constructor`, `toString`) never collide with model fields.
    const changes = Object.create(null) as ObjectRecord;

    const resolve = (key: string, receiver?: unknown): unknown => {
      // Only consult changes / payloadValues when the
      // key is a known model field.  Otherwise fall straight through to
      // `self` so that prototype methods and non-metadata properties are
      // read from the real object — the `in` operator would traverse the
      // prototype chain and return incorrect values (e.g. `Object` for
      // `this.constructor`).
      if (fieldMetadata.has(key)) {
        if (key in changes) return changes[key];
        if (payloadValues && Object.prototype.hasOwnProperty.call(payloadValues, key)) return payloadValues[key];
      }
      // Use Reflect.get with the receiver so that prototype getters
      // (e.g. computed fields) run with `this` bound to the draft proxy
      // instead of the raw `self` object.
      return Reflect.get(self as unknown as ObjectRecord, key, receiver ?? self);
    };

    const draft = new Proxy(self as unknown as ObjectRecord, {
      get(_target, prop, receiver) {
        // Symbol properties (e.g. Symbol.iterator) are never model fields;
        // delegate them directly to the target.
        if (typeof prop === 'symbol') {
          return Reflect.get(self as unknown as ObjectRecord, prop, receiver);
        }
        const key = String(prop);
        return resolve(key, receiver);
      },
      set(_target, prop, value, _receiver) {
        if (typeof prop === 'symbol') {
          // Do NOT pass the receiver (the proxy itself) to Reflect.set —
          // that would re-trigger this trap and cause infinite recursion.
          return Reflect.set(self as unknown as ObjectRecord, prop, value);
        }
        const key = String(prop);
        // For non-metadata fields (transient / private properties used during
        // validation), write directly to the target object so they can be read
        // back, but do NOT record them in `changes` (avoid polluting the payload).
        if (!fieldMetadata.has(key)) {
          Reflect.set(self as unknown as ObjectRecord, prop, value);
          return true;
        }
        // Skip recording when the value is already identical to the resolved
        // current value (avoid unnecessary writebacks and re-validation).
        if (resolve(key, _receiver) !== value) {
          changes[key] = value;
        }
        return true;
      },
    }) as unknown as TModel;

    return { draft, changes };
  }

  /**
   * Re-runs kernel and platform validation for fields that were
   * mutated by instance constraint methods, preventing constraint writeback
   * from silently bypassing earlier validation layers.  The optional
   * validation options (kernelRules, whitelists, etc.) are forwarded so
   * the re-validation pass uses the same settings as the primary pass.
   */
  private static async validatePostConstraintMutations<TModel extends BaseModel>(
    ctx: ConstraintContext<TModel>,
    mutatedFields: Set<string>,
    options?: {
      kernelRules?: KernelValidationRule[];
      platformCreateWriteWhitelist?: string[];
      platformRejectUnknownFields?: boolean;
      onPlatformCreateWhitelistHit?: (fields: string[]) => void;
    }
  ): Promise<ValidationIssue[]> {
    if (mutatedFields.size === 0) return [];

    const postCtx: ConstraintContext<TModel> = {
      ...ctx,
      changedFields: mutatedFields,
    };

    const issues: ValidationIssue[] = [];
    issues.push(...(await this.validateKernelRules(postCtx, options?.kernelRules)));
    issues.push(
      ...(await this.validatePlatformRules(postCtx, {
        createWriteWhitelist: options?.platformCreateWriteWhitelist,
        rejectUnknownFields: options?.platformRejectUnknownFields,
        onCreateWhitelistHit: options?.onPlatformCreateWhitelistHit,
      }))
    );
    return issues;
  }
}
