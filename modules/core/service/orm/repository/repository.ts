// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import BaseModel from '../model/model';
import {
  Compilable,
  Entity,
  DeleteResult,
  UpdateResult,
  SearchOptions,
  BaseQueryCondition,
  ConditionEnvelope,
  SimplifyResult,
  RepoReadGroupOptions,
  RepoReadGroupRow,
  RepoReadTotalsOptions,
  RepoReadTotalsRow,
  RepoReadGroupCountOptions,
  RecordRuleOp,
} from './types';
import { ChoysumDialect, ChoysumDatabase, ChoysumCamelCasePlugin, ChoysumDeduplicateJoinsPlugin, ChoysumParseJSONResultsPlugin } from '../../infra/database';
import type { DialectName } from './repository_dialect';
import { FieldMetadata } from '../metadata';
import { ModelMetadata } from '../metadata';
import { REL_ALIAS_PREFIX } from '../relation/relation_alias';
import { buildHiddenScaleAlias } from './hidden_scale_alias';
import {
  aliasSelection as aliasSelectionExternal,
  buildRelationJsonSelect as buildRelationJsonSelectExternal,
  buildSelectionTree as buildSelectionTreeExternal,
  decodeRowWithTree as decodeRowWithTreeExternal,
  getScalarFields as getScalarFieldsExternal,
} from './projection';
import { decodeFromDb as decodeFromDbExternal, encodeForDb as encodeForDbExternal } from './projection/row_codec';
import {
  applyRepositoryDefaultLayers,
  applyRepositorySoftDeleteLayer,
  countRepositoryConditionMatches,
  convertCondition as convertConditionExternal,
  isEmptyRepositoryCondition,
  locateRepositoryIdsForCondition,
  makeSelectCtx as makeSelectCtxExternal,
  normalizeOrderBy as normalizeOrderByExternal,
  repositorySoftDeleteEnabled,
  resolveEffectiveOrder as resolveEffectiveOrderExternal,
  applyOrderByToQuery as applyOrderByToQueryExternal,
  hasRepositorySqlComputeExpression,
  resolveRepositorySqlComputeExpression,
} from './query';
import type { RepositoryPredicateBuilder, RepositoryPredicate } from './query/predicate_builder_adapter';
import type { SelectionNode, SelectionRelationEntry } from './projection';
import { ValidationPipelineError, type ConstraintMode } from '../metadata';
import {
  createRepositoryConditionQueryDeps,
  createRepositoryReadAggregateFacadeDeps,
  createRepositoryReadQueryFacadeDeps,
  createRepositorySearchFacadeDeps,
  executeRepositoryReadGroup,
  executeRepositoryReadGroupCount,
  executeRepositoryReadTotals,
  executeRepositorySearch,
} from './read';
import { ChoysumError } from '@/core/service/error';
import { getRuntimeEnvFlag } from '@/core/utils/env';
import { getRepositoryReadonlyCtx, getRepositoryUserId, invalidateRepositoryRuntimeCache, type RepositoryRuntimeContext } from './repository_runtime_bridge';
import {
  assertRepositoryCompanyWriteAccessForCondition,
  assertRepositoryFieldRuleWriteAllowed,
  assertRepositoryRecordRuleAllCreatedAllowed,
  assertRepositoryRecordRuleAllTargetsAllowed,
  assertRepositoryRecordRuleCreateAllowed,
  createRepositoryAuthzContextDeps,
  createRepositoryCompanyScopeDeps,
  createRepositoryCompanyScopePolicyDeps,
  createRepositoryCompanyScopeQueryDeps,
  createRepositoryFieldRuleDeps,
  createRepositoryFieldRulePolicyDeps,
  createRepositoryFieldRuleSelectionDeps,
  createRepositoryPermissionDeniedError,
  createRepositoryRecordRuleCoordinatorDeps,
  createRepositoryRecordRuleCoordinatorPolicyDeps,
  createRepositoryRecordRuleDeps,
  createRepositoryRecordRulePolicyDeps,
  emitRepositoryAuthzDecisionSummary,
  getOrInitRepositoryReqServiceState,
  getRepositoryCompanyScopeFacts,
  getRepositoryCurrentReq,
  getRepositoryCurrentReqWrapper,
  getRepositoryFieldRuleSpec,
  getRecordRuleBypassDepth as readRecordRuleBypassDepth,
  getRepositoryRecordRuleEnvelope,
  getRepositoryReqMethodMeta,
  getValidationBypassDepth as readValidationBypassDepth,
  isRepositoryTopLevelGrpcCall,
  normalizeRepositoryCompanyIdForWrite,
  normalizeRepositoryCompanyIds,
  applyRepositoryCompanyLayer,
  applyRepositoryDefaultCompanyIdOnCreate,
  applyRepositoryDefaultCompanyIdOnUpdate,
  applyRepositoryRecordRuleToCondition,
  pruneRepositorySelectionTreeForFieldRule,
  replaceRepositoryRecordRuleTokens,
  repositoryCompanyFieldEnabled,
  repositoryFieldRuleEnabled,
  validateRepositoryCompanyIdInScope,
  withFieldRuleBypass as runWithFieldRuleBypass,
  withRecordRuleBypass as runWithRecordRuleBypass,
  withValidationBypass as runWithValidationBypass,
} from './authz';
import type { RepositoryAuthzDecisionSummary } from './authz';
import {
  createRepositoryCreateFacadeDeps,
  createRepositoryDeleteWriteDeps,
  createRepositoryMutationWriteFacadeDeps,
  createRepositoryDeleteChild,
  createRepositoryUpdateFacadeDeps,
  executeRepositoryCreate,
  executeRepositoryDelete,
  executeRepositoryHardDelete,
  executeRepositoryUpdate,
} from './write';
import { wrapRepositoryValidationError, throwRepositorySqlWriteError, validateRepositoryWrite } from './validation';
import { ComputeEngine } from '../../runtime/compute/engine';
import type { ObjectRecord } from '../../../utils/types';

export const db = new ChoysumDatabase<ObjectRecord>({
  dialect: new ChoysumDialect(),
  plugins: [new ChoysumCamelCasePlugin(), new ChoysumParseJSONResultsPlugin(), new ChoysumDeduplicateJoinsPlugin()],
});

export class Repository {
  private db = db;
  private table: string;

  // Soft-delete controls.
  private includeDeleted = false; // Skip the default DeletedAt is null filter for reads and updates.
  private onlyDeletedMode = false; // Only query soft-deleted rows, where DeletedAt is not null.

  constructor(private meta: ModelMetadata) {
    this.table = meta.tableName();
  }

  // Read the effective soft-delete toggle from global and model configuration.
  private softDeleteEnabled(): boolean {
    return repositorySoftDeleteEnabled(this.meta);
  }

  private readonly softField = 'DeletedAt';

  /* ----------------------- AuthZ decision summary (Phase C) ----------------------- */

  private emitAuthzDecisionSummary(summary: RepositoryAuthzDecisionSummary): void {
    emitRepositoryAuthzDecisionSummary(summary);
  }

  private getReqMethodMeta(): { fullMethod: string; method: string; companyMode: string; recordRuleMode: string; fieldRuleMode: string } {
    return getRepositoryReqMethodMeta();
  }

  private getCompanyScopeFacts(): { activeCompanyId: string; enabledCompanyIds: string[] } {
    return getRepositoryCompanyScopeFacts(this.ctx, this.normalizeCompanyIds());
  }

  /* ----------------------------- RecordRule (P2-2) ----------------------------- */

  /**
   * Control-plane metadata models (application === 'meta') must skip RecordRule injection.
   * - Avoid recursion when GetRecordRuleCondition or CheckMethodAccess depends on metadata queries.
   * - Keep Smart Routing local and remote behavior aligned.
   */
  private isControlPlaneMetaModel(): boolean {
    const app = String(this.meta.application || '').trim();
    if (app === 'meta') return true;

    const full = String(this.meta.fullModelName || '').trim();
    return full.startsWith('meta.');
  }

  /**
   * FieldRule control-plane models must not be constrained by FieldRule itself.
   * Otherwise a deny rule can block the later allow exception and lock the system out.
   * These models should instead be protected by higher-level mechanisms such as MethodAccess and RecordRule.
   */
  private isFieldRuleControlPlaneModel(): boolean {
    const app = String(this.meta.application || '').trim();
    const name = String(this.meta.modelName || this.meta.name || '').trim();
    const full = String(this.meta.fullModelName || '').trim();
    if (full === 'auth.RoleFieldRule') return true;
    return app === 'auth' && name === 'RoleFieldRule';
  }

  private recordRuleEnabled(): boolean {
    return getRuntimeEnvFlag('CHOYSUM_GRPC_RECORD_RULE_ENABLED', true);
  }

  private getCurrentReq(): ObjectRecord | undefined {
    return getRepositoryCurrentReq();
  }

  private getCurrentReqWrapper(): ObjectRecord | undefined {
    return getRepositoryCurrentReqWrapper();
  }

  private isTopLevelGrpcCall(): boolean {
    return isRepositoryTopLevelGrpcCall();
  }

  private getOrInitReqServiceState(req: unknown): ObjectRecord | undefined {
    return getOrInitRepositoryReqServiceState(req) as ObjectRecord | undefined;
  }

  private getRecordRuleBypassDepth(): number {
    return readRecordRuleBypassDepth();
  }

  private async withRecordRuleBypass<T>(fn: () => Promise<T>): Promise<T> {
    return await runWithRecordRuleBypass(fn);
  }

  private getTopLevelCompanyMode(): string {
    const req = this.getCurrentReq();
    const depth = typeof req?.depth === 'number' ? req.depth : 0;
    if (depth !== 0) return '';
    return typeof req?.companyMode === 'string' ? String(req.companyMode).trim() : '';
  }

  private companyLayerSkipped(): boolean {
    return this.getTopLevelCompanyMode() === 'skip';
  }

  private async getRecordRuleEnvelope(op: RecordRuleOp): Promise<ConditionEnvelope> {
    return await getRepositoryRecordRuleEnvelope(this.createRecordRuleDeps(), op);
  }

  private replaceRecordRuleTokens(condition: BaseQueryCondition): BaseQueryCondition {
    return replaceRepositoryRecordRuleTokens(this.createRecordRuleDeps(), condition);
  }

  private async applyRecordRuleToCondition(condition: BaseQueryCondition, op: RecordRuleOp): Promise<BaseQueryCondition> {
    if (this.isControlPlaneMetaModel()) return condition;

    return await applyRepositoryRecordRuleToCondition(this.createRecordRuleCoordinatorDeps(), condition, op);
  }

  private async locateIdsForCondition(condition: BaseQueryCondition): Promise<string[]> {
    return await locateRepositoryIdsForCondition(
      this.createConditionQueryDeps(queryCondition => this.applySoftLayer(queryCondition)),
      condition
    );
  }

  private async countConditionMatches(condition: BaseQueryCondition): Promise<number> {
    return await countRepositoryConditionMatches(
      this.createConditionQueryDeps(queryCondition => this.applySoftLayer(this.applyCompanyLayer(queryCondition))),
      condition
    );
  }

  private createConditionQueryDeps(applyConditionLayers: (condition: BaseQueryCondition) => BaseQueryCondition) {
    return createRepositoryConditionQueryDeps({
      db: this.db,
      table: this.table,
      applyConditionLayers,
      isEmptyCondition: queryCondition => this.isEmptyCondition(queryCondition),
      convertCondition: (eb, queryCondition, selfTable) => this.convertCondition(eb as RepositoryPredicateBuilder, queryCondition, selfTable),
      execute: async <T = unknown>(query: unknown) => (await this.execute(query as Compilable<T>)) as unknown as T[],
    });
  }

  private async assertRecordRuleAllTargetsAllowed(op: Extract<RecordRuleOp, 'write' | 'delete'>, targetIds: string[]): Promise<void> {
    await assertRepositoryRecordRuleAllTargetsAllowed(this.createRecordRuleCoordinatorDeps(), op, targetIds);
  }

  private async assertRecordRuleCreateAllowed(): Promise<void> {
    await assertRepositoryRecordRuleCreateAllowed(this.createRecordRuleCoordinatorDeps());
  }

  private async assertRecordRuleAllCreatedAllowed(createdIds: string[], env: ConditionEnvelope): Promise<void> {
    await assertRepositoryRecordRuleAllCreatedAllowed(this.createRecordRuleCoordinatorDeps(), createdIds, env);
  }

  /* ----------------------------- FieldRule（P4） ----------------------------- */

  private getValidationBypassDepth(): number {
    return readValidationBypassDepth();
  }

  public async withValidationBypass<T>(fn: () => Promise<T>): Promise<T> {
    return await runWithValidationBypass(fn);
  }

  private async withFieldRuleBypass<T>(fn: () => Promise<T>): Promise<T> {
    return await runWithFieldRuleBypass(fn);
  }

  /**
   * P4 (Milestone 5): provide a read-only entry point for top-level output pruning through denyReadFields.
   * - Reuse the gate, skip, meta-model, and cache logic inside getFieldRuleSpec.
   * - Only strip keys at the current response layer without recursive relation pruning.
   */
  public async getDenyReadFields(): Promise<{ denyReadFields: string[]; reason?: string }> {
    const spec = await getRepositoryFieldRuleSpec(this.createFieldRuleDeps());
    return { denyReadFields: spec.denyReadFields, reason: spec.reason };
  }

  /**
   * FieldsGet write-ACL surface: fields the current principal may not write.
   * Same field-rule spec / cache as getDenyReadFields (P5 → isReadonly).
   */
  public async getDenyWriteFields(): Promise<{ denyWriteFields: string[]; reason?: string }> {
    const spec = await getRepositoryFieldRuleSpec(this.createFieldRuleDeps());
    return { denyWriteFields: spec.denyWriteFields, reason: spec.reason };
  }

  private async assertFieldRuleWriteAllowed(payload: Entity): Promise<void> {
    await assertRepositoryFieldRuleWriteAllowed({
      ...this.createFieldRuleDeps(),
      payload,
    });
  }

  /* ----------------------------- Company scope (P2-1) ----------------------------- */

  private companyFieldEnabled(): boolean {
    return repositoryCompanyFieldEnabled(this.createCompanyScopeDeps());
  }

  private permissionDenied(code: string, message: string, metadata?: Record<string, string>): ChoysumError {
    return createRepositoryPermissionDeniedError(this.createAuthzContextDeps(), code, message, metadata);
  }

  private wrapValidationError(error: ValidationPipelineError, mode: ConstraintMode): ChoysumError {
    return wrapRepositoryValidationError(this.meta, error, mode);
  }

  private wrapSqlWriteError(error: unknown, mode: ConstraintMode): never {
    throwRepositorySqlWriteError(this.meta, error, mode);
  }

  private normalizeCompanyIds(): string[] {
    return normalizeRepositoryCompanyIds(this.ctx);
  }

  private createAuthzContextParams() {
    return {
      meta: this.meta,
      userId: this.userId,
      getReqMethodMeta: () => this.getReqMethodMeta(),
      getCompanyScopeFacts: () => this.getCompanyScopeFacts(),
      emitAuthzDecisionSummary: (summary: RepositoryAuthzDecisionSummary) => this.emitAuthzDecisionSummary(summary),
    };
  }

  private createAuthPolicyCommonParams() {
    return {
      meta: this.meta,
      userId: this.userId,
      requestContext: this.ctx,
      normalizeCompanyIds: () => this.normalizeCompanyIds(),
      permissionDenied: (code: string, message: string, metadata?: Record<string, string>) => this.permissionDenied(code, message, metadata),
    };
  }

  private createRepositoryQueryBridgeDeps() {
    return {
      db: this.db,
      table: this.table,
      isEmptyCondition: (condition: BaseQueryCondition) => this.isEmptyCondition(condition),
      convertCondition: (eb: unknown, condition: BaseQueryCondition, selfTable?: string) =>
        this.convertCondition(eb as RepositoryPredicateBuilder, condition, selfTable),
      execute: async <T = unknown>(query: unknown) => (await this.execute(query as Compilable<T>)) as unknown as T[],
    };
  }

  private createAuthzContextDeps() {
    return createRepositoryAuthzContextDeps(this.createAuthzContextParams());
  }

  private createRecordRuleDeps() {
    const common = this.createAuthPolicyCommonParams();
    return createRepositoryRecordRuleDeps(
      createRepositoryRecordRulePolicyDeps({
        ...common,
        normalizeCompanyIdForWrite: () => this.normalizeCompanyIdForWrite(),
        isControlPlaneMetaModel: () => this.isControlPlaneMetaModel(),
        recordRuleEnabled: () => this.recordRuleEnabled(),
        getRecordRuleBypassDepth: () => this.getRecordRuleBypassDepth(),
        withRecordRuleBypass: fn => this.withRecordRuleBypass(fn),
      })
    );
  }
  private createRecordRuleCoordinatorDeps() {
    const authzContext = this.createAuthzContextParams();
    return createRepositoryRecordRuleCoordinatorDeps(
      createRepositoryRecordRuleCoordinatorPolicyDeps({
        ...authzContext,
        recordRuleEnabled: () => this.recordRuleEnabled(),
        getRecordRuleEnvelope: op => this.getRecordRuleEnvelope(op),
        replaceRecordRuleTokens: condition => this.replaceRecordRuleTokens(condition),
        permissionDenied: (code, message, metadata) => this.permissionDenied(code, message, metadata),
        countConditionMatches: condition => this.countConditionMatches(condition),
      })
    );
  }

  private createFieldRuleDeps() {
    const common = this.createAuthPolicyCommonParams();
    return createRepositoryFieldRuleDeps(
      createRepositoryFieldRulePolicyDeps({
        ...common,
        isControlPlaneMetaModel: () => this.isControlPlaneMetaModel(),
        isFieldRuleControlPlaneModel: () => this.isFieldRuleControlPlaneModel(),
        withRecordRuleBypass: fn => this.withRecordRuleBypass(fn),
        withFieldRuleBypass: fn => this.withFieldRuleBypass(fn),
      })
    );
  }

  private createDeleteChildRepository(meta: ModelMetadata) {
    const childRepo = new Repository(meta);
    return createRepositoryDeleteChild({
      softDeleteEnabled: () => childRepo.softDeleteEnabled(),
      delete: condition => childRepo.delete(condition),
      hardDelete: condition => childRepo.hardDelete(condition),
      count: condition => childRepo.count(condition),
      withFieldRuleBypass: fn => childRepo.withFieldRuleBypass(fn),
      update: (vals, condition) => childRepo.update(vals, condition),
    });
  }

  private createCompanyScopeDeps() {
    const authzContext = this.createAuthzContextParams();
    return createRepositoryCompanyScopeDeps(
      createRepositoryCompanyScopePolicyDeps({
        ...authzContext,
        ctx: this.ctx,
        companyLayerSkipped: () => this.companyLayerSkipped(),
        permissionDenied: (code, message, metadata) => this.permissionDenied(code, message, metadata),
      })
    );
  }

  private createCompanyScopeQueryDeps() {
    const queryBridge = this.createRepositoryQueryBridgeDeps();
    return createRepositoryCompanyScopeQueryDeps({
      ...this.createCompanyScopeDeps(),
      ...queryBridge,
      applySoftLayer: condition => this.applySoftLayer(condition),
    });
  }

  private createMutationWriteFacadeDeps() {
    const queryBridge = this.createRepositoryQueryBridgeDeps();
    return createRepositoryMutationWriteFacadeDeps({
      meta: this.meta,
      table: queryBridge.table,
      locateIdsForCondition: condition => this.locateIdsForCondition(condition),
      assertCompanyWriteAccessForCondition: condition => this.assertCompanyWriteAccessForCondition(condition),
      assertRecordRuleAllTargetsAllowed: (op, targetIds) => this.assertRecordRuleAllTargetsAllowed(op, targetIds),
      applyRecordRuleToCondition: (condition, op) => this.applyRecordRuleToCondition(condition, op),
      applyDefaultLayers: condition => this.applyDefaultLayers(condition),
      isEmptyCondition: queryBridge.isEmptyCondition,
      convertCondition: queryBridge.convertCondition,
    });
  }

  private createDeleteWriteTargetDeps() {
    return {
      meta: this.meta,
      locateIdsForCondition: (condition: BaseQueryCondition) => this.locateIdsForCondition(condition),
      assertCompanyWriteAccessForCondition: (condition: BaseQueryCondition) => this.assertCompanyWriteAccessForCondition(condition),
      assertRecordRuleAllTargetsAllowed: (op: 'delete', targetIds: string[]) => this.assertRecordRuleAllTargetsAllowed(op, targetIds),
    };
  }

  private createDeleteWriteSoftDeletePreWriteDeps() {
    const queryBridge = this.createRepositoryQueryBridgeDeps();
    return {
      meta: this.meta,
      db: this.db,
      table: queryBridge.table,
      softField: this.softField,
      softDeleteEnabled: () => this.softDeleteEnabled(),
      createRepository: (meta: ModelMetadata) => this.createDeleteChildRepository(meta),
      applySoftLayer: (condition: BaseQueryCondition) => this.applySoftLayer(condition),
      isEmptyCondition: queryBridge.isEmptyCondition,
      convertCondition: queryBridge.convertCondition,
    };
  }

  private createDeleteWriteQueryPrepareDeps() {
    const queryBridge = this.createRepositoryQueryBridgeDeps();
    return {
      db: this.db,
      table: queryBridge.table,
      applyRecordRuleToCondition: (condition: BaseQueryCondition, op: 'delete') => this.applyRecordRuleToCondition(condition, op),
      applyDefaultLayers: (condition: BaseQueryCondition) => this.applyDefaultLayers(condition),
      isEmptyCondition: queryBridge.isEmptyCondition,
      convertCondition: queryBridge.convertCondition,
    };
  }

  private createDeleteWriteRuntimeDeps() {
    return {
      execute: <R>(query: Compilable<R>) => this.execute(query),
      wrapSqlWriteError: (error: unknown, mode: ConstraintMode) => this.wrapSqlWriteError(error, mode),
    };
  }

  private createDeleteWritePostWriteDeps() {
    return {
      invalidateCache: () => this.invalidateCache(),
    };
  }

  private createUpdateWriteProjectionDeps() {
    return {
      getScalarFields: (meta: ModelMetadata) => this.getScalarFields(meta),
      makeSelectCtx: (builder: unknown, selfTable: string, curMeta?: ModelMetadata) => this.makeSelectCtx(builder, selfTable, curMeta),
      aliasSelection: (selection: unknown, alias: string) => this.aliasSelection(selection, alias),
      decodeFromDb: (row: Entity) => this.decodeFromDb(row),
    };
  }

  private createUpdateWritePayloadDeps() {
    return {
      assertFieldRuleWriteAllowed: (payload: Entity) => this.assertFieldRuleWriteAllowed(payload),
      applyDefaultCompanyIdOnUpdate: (payload: Entity) => this.applyDefaultCompanyIdOnUpdate(payload),
      validateFields: (input: Entity, mode: ConstraintMode, current?: ObjectRecord) => this.validateFields(input, mode, current),
      encodeForDb: (input: Entity) => this.encodeForDb(input),
    };
  }

  private createUpdateWriteSoftFilterDeps() {
    return {
      applySoftLayer: (condition: BaseQueryCondition) => this.applySoftLayer(condition),
    };
  }

  private createUpdateWriteRuntimeDeps() {
    return {
      execute: <R>(query: Compilable<R>) => this.execute(query),
    };
  }

  private createUpdateWritePostWriteDeps() {
    return {
      invalidateCache: () => this.invalidateCache(),
      recomputePersistForUpdate: async (payload: { targetIds: string[]; sanitized: Entity }) => {
        await this.recomputePersistForUpdate(payload);
      },
    };
  }

  private createReadQueryFacadeDeps() {
    return createRepositoryReadQueryFacadeDeps({
      applyRecordRuleToCondition: (condition, op) => this.applyRecordRuleToCondition(condition, op),
      applyDefaultLayers: condition => this.applyDefaultLayers(condition),
      ...this.createRepositoryQueryBridgeDeps(),
      normalizeOrderBy: orderBy => this.normalizeOrderBy(orderBy),
      applyOrderByToQuery: (query, meta, table, orderBy) => this.applyOrderByToQuery(query, meta as ModelMetadata, table, orderBy),
    });
  }

  private createDeleteWriteDeps() {
    const target = this.createDeleteWriteTargetDeps();
    const softDeletePreWrite = this.createDeleteWriteSoftDeletePreWriteDeps();
    const queryPrepare = this.createDeleteWriteQueryPrepareDeps();
    const runtime = this.createDeleteWriteRuntimeDeps();
    const postWrite = this.createDeleteWritePostWriteDeps();

    return createRepositoryDeleteWriteDeps({
      db: queryPrepare.db,
      table: queryPrepare.table,
      meta: target.meta,
      softField: softDeletePreWrite.softField,
      locateIdsForCondition: target.locateIdsForCondition,
      assertCompanyWriteAccessForCondition: target.assertCompanyWriteAccessForCondition,
      assertRecordRuleAllTargetsAllowed: target.assertRecordRuleAllTargetsAllowed,
      softDeleteEnabled: softDeletePreWrite.softDeleteEnabled,
      applySoftLayer: softDeletePreWrite.applySoftLayer,
      applyRecordRuleToCondition: queryPrepare.applyRecordRuleToCondition,
      applyDefaultLayers: queryPrepare.applyDefaultLayers,
      isEmptyCondition: queryPrepare.isEmptyCondition,
      convertCondition: queryPrepare.convertCondition,
      execute: runtime.execute,
      invalidateCache: postWrite.invalidateCache,
      wrapSqlWriteError: runtime.wrapSqlWriteError,
      createRepository: softDeletePreWrite.createRepository,
    });
  }

  private createCreateWriteAuthzDeps() {
    return {
      getRecordRuleEnvelope: (op: 'create') => this.getRecordRuleEnvelope(op),
      permissionDenied: (code: string, message: string, metadata?: Record<string, string>) => this.permissionDenied(code, message, metadata),
    };
  }

  private createCreateWritePostWriteDeps() {
    return {
      assertRecordRuleAllCreatedAllowed: (createdIds: string[], env: ConditionEnvelope) => this.assertRecordRuleAllCreatedAllowed(createdIds, env),
      recomputePersistForCreate: async (createdIds: string[], sanitizedEntities: Entity[]) => {
        await this.recomputePersistForCreate(createdIds, sanitizedEntities);
      },
    };
  }

  private createCreateWritePayloadDeps() {
    return {
      assertFieldRuleWriteAllowed: (payload: Entity) => this.assertFieldRuleWriteAllowed(payload),
      applyDefaultCompanyIdOnCreate: (entity: Entity) => this.applyDefaultCompanyIdOnCreate(entity),
      validateFields: (input: Entity, mode: 'create') => this.validateFields(input, mode),
      encodeForDb: (input: Entity) => this.encodeForDb(input),
    };
  }

  private createCreateWriteRuntimeDeps() {
    return {
      generateId: () => $choysum.xid.New(),
      execute: <R>(query: Compilable<R>) => this.execute(query),
      wrapSqlWriteError: (error: unknown, mode: 'create') => this.wrapSqlWriteError(error, mode),
    };
  }

  private createCreateWriteDeps() {
    return createRepositoryCreateFacadeDeps({
      db: this.db,
      table: this.table,
      meta: this.meta,
      ...this.createCreateWriteAuthzDeps(),
      ...this.createCreateWritePayloadDeps(),
      ...this.createCreateWriteRuntimeDeps(),
      ...this.createCreateWritePostWriteDeps(),
    });
  }

  private createUpdateWriteDeps() {
    return createRepositoryUpdateFacadeDeps({
      db: this.db,
      ...this.createUpdateWriteProjectionDeps(),
      ...this.createUpdateWriteSoftFilterDeps(),
      ...this.createUpdateWriteRuntimeDeps(),
      ...this.createUpdateWritePostWriteDeps(),
      ...this.createMutationWriteFacadeDeps(),
      ...this.createUpdateWritePayloadDeps(),
    });
  }

  // Build the company filter layer shared by search, count, update, delete, and readGroup.
  // Semantics: ownership field != NULL means company-owned rows; NULL means shared rows.
  private applyCompanyLayer(condition: BaseQueryCondition): BaseQueryCondition {
    return applyRepositoryCompanyLayer(this.createCompanyScopeDeps(), condition);
  }

  private applyDefaultLayers(condition: BaseQueryCondition): BaseQueryCondition {
    return applyRepositoryDefaultLayers(
      {
        meta: this.meta,
        softField: this.softField,
        includeDeleted: this.includeDeleted,
        onlyDeletedMode: this.onlyDeletedMode,
        applyCompanyLayer: conditionValue => this.applyCompanyLayer(conditionValue),
      },
      condition
    );
  }

  // Apply the default ownership company on write paths only for company-isolated models.
  private normalizeCompanyIdForWrite(): string | undefined {
    return normalizeRepositoryCompanyIdForWrite(this.ctx);
  }

  private applyDefaultCompanyIdOnCreate(entity: Entity): Entity {
    return applyRepositoryDefaultCompanyIdOnCreate(this.createCompanyScopeDeps(), entity);
  }

  private applyDefaultCompanyIdOnUpdate(vals: Entity): Entity {
    return applyRepositoryDefaultCompanyIdOnUpdate(this.createCompanyScopeDeps(), vals);
  }

  private validateCompanyIdInScope(companyId: unknown, companyIds: string[]): void {
    validateRepositoryCompanyIdInScope(this.createCompanyScopeDeps(), companyId, companyIds);
  }

  private async assertCompanyWriteAccessForCondition(condition: BaseQueryCondition): Promise<string[]> {
    return await assertRepositoryCompanyWriteAccessForCondition(this.createCompanyScopeQueryDeps(), condition);
  }

  /**
   * Assert the current principal may write the given row ids (company scope).
   * Empty id list is a no-op.
   */
  public async assertCompanyWriteAccessForIds(targetIds: string[]): Promise<void> {
    const ids = (targetIds || []).map(id => String(id || '').trim()).filter(Boolean);
    if (!ids.length) return;
    await this.assertCompanyWriteAccessForCondition({ And: [['Id', 'in', ids]] } as BaseQueryCondition);
  }

  /**
   * Assert RecordRule allows the given op on all target ids.
   * Empty id list is a no-op. When record rules are disabled, this is a no-op.
   */
  public async assertRecordRuleTargetsAllowed(op: Extract<RecordRuleOp, 'write' | 'delete'>, targetIds: string[]): Promise<void> {
    await this.assertRecordRuleAllTargetsAllowed(op, targetIds);
  }

  // Build the soft-delete filter layer shared by search, count, and update.
  private applySoftLayer(condition: BaseQueryCondition): BaseQueryCondition {
    return applyRepositorySoftDeleteLayer(
      {
        meta: this.meta,
        softField: this.softField,
        includeDeleted: this.includeDeleted,
        onlyDeletedMode: this.onlyDeletedMode,
      },
      condition
    );
  }

  // Public mode toggles.
  public withDeleted(): Repository {
    const r = new Repository(this.meta);
    r.includeDeleted = true;
    r.onlyDeletedMode = false;
    return r;
  }

  public onlyDeleted(): Repository {
    const r = new Repository(this.meta);
    r.includeDeleted = true; // Skip the default "is null" filter.
    r.onlyDeletedMode = true; // Force an "is not null" filter.
    return r;
  }

  /* ----------------------------- Core helpers ----------------------------- */

  // Read the current request read-only context, sharing the same reference as BaseModel.ctx.
  public get ctx(): RepositoryRuntimeContext {
    return getRepositoryReadonlyCtx();
  }

  // Convenience accessor for the current user Id.
  public get userId(): string | undefined {
    return getRepositoryUserId();
  }

  private isEmptyCondition(condition: BaseQueryCondition): boolean {
    return isEmptyRepositoryCondition(condition);
  }

  private getDialect(): DialectName {
    return ($choysum.db?.dialectName || 'postgres') as DialectName;
  }

  /**
   * Build a SelectCtx that supports:
   * - ctx.field(model, 'A.B.C.X'): path fields along ManyToOne chains, including leaf columns or virtual fields.
   * - ctx.selectFrom(table): subqueries.
   */
  private makeSelectCtx(builder: unknown, selfTable: string, curMeta: ModelMetadata = this.meta) {
    return makeSelectCtxExternal(this.db, () => this.getDialect(), builder, selfTable, curMeta);
  }

  // Apply stable aliases to selections and avoid coercing objects into the string "[object Object]".
  private aliasSelection(sel: unknown, alias: string) {
    return aliasSelectionExternal(sel, alias);
  }

  /* ----------------------------- Condition conversion ----------------------------- */
  public convertCondition(eb: RepositoryPredicateBuilder, condition: BaseQueryCondition, selfTable?: string): RepositoryPredicate {
    return convertConditionExternal(this.db, () => this.getDialect(), this.meta, eb, condition, selfTable);
  }

  /* ----------------------------- Pre-write encoding ----------------------------- */
  private encodeForDb(input: Entity): Entity {
    return encodeForDbExternal(this.meta, input);
  }

  /* ----------------------------- Top-level post-read decoding ----------------------------- */
  private decodeFromDb(row: Entity): Entity {
    return decodeFromDbExternal(this.meta, row);
  }

  /**
   * Recursively decode relations from the selection tree with node-level caching.
   * Cached node metadata avoids repeating the same work for every row:
   * - __columnsArr: column array for the current node.
   * - __decimalCols: columns on the current node that require Decimal conversion.
   * - __relationsArr: flattened relation entries in [key, entry][] form.
   * - __keyMap: Pascal column name to the first-row concrete key that exists in Pascal, camel, or snake form.
   */
  private decodeRowWithTree(meta: ModelMetadata, node: SelectionNode, row: unknown): Entity {
    return decodeRowWithTreeExternal(meta, node, row) as Entity;
  }

  /* ----------------------------- CRUD primitives ----------------------------- */

  public async execute<R>(query: Compilable<R>): Promise<SimplifyResult<R>[]> {
    return this.db.execute(query);
  }

  public insertQueryBuilder() {
    return this.db.insertInto(this.table);
  }

  public updateQueryBuilder() {
    return this.db.updateTable(this.table);
  }

  public deleteQueryBuilder() {
    return this.db.deleteFrom(this.table);
  }

  public selectQueryBuilder() {
    return this.db.selectFrom(this.table);
  }

  /**
   * Unified field-validation entry point.
   */
  private async validateFields(input: Entity, mode: ConstraintMode, current?: ObjectRecord): Promise<void> {
    await validateRepositoryWrite({
      meta: this.meta,
      repository: this,
      requestContext: this.ctx,
      getValidationBypassDepth: () => this.getValidationBypassDepth(),
      input,
      mode,
      current,
    });
  }

  private asRecord(value: unknown): ObjectRecord | undefined {
    if (!value || typeof value !== 'object') return undefined;
    return value as ObjectRecord;
  }

  private getDecimalScaleField(fieldName: string): string | undefined {
    const field = this.meta.fields.get(fieldName) as FieldMetadata | undefined;
    if (!field || field.type !== 'decimal') return undefined;

    const columnSpec = this.asRecord(field.column);
    const scaleField = columnSpec?.scaleField;
    if (typeof scaleField !== 'string') return undefined;

    const normalized = scaleField.trim();
    return normalized || undefined;
  }

  private addPersistComputeField(fields: Set<string>, fieldName: string | undefined): void {
    const normalized = String(fieldName || '').trim();
    if (!normalized) return;

    fields.add(normalized);
    const scaleField = this.getDecimalScaleField(normalized);
    if (scaleField) {
      fields.add(scaleField);
    }
  }

  private getEntityId(value: unknown): string {
    const row = (value ?? {}) as { Id?: unknown };
    return String(row.Id ?? '').trim();
  }

  private resolvePersistComputeSelection(seedFields: Iterable<string>): string[] {
    const fields = new Set<string>(['Id']);
    const normalizedSeed = new Set<string>();

    for (const field of seedFields || []) {
      const normalized = String(field || '').trim();
      if (!normalized) continue;
      normalizedSeed.add(normalized);
      this.addPersistComputeField(fields, normalized);
    }

    const graph = this.meta.computeGraph;
    const persistedComputeFields = graph?.persistedComputeFields;
    if (!graph || !persistedComputeFields?.size || !normalizedSeed.size) {
      return Array.from(fields);
    }

    const queue: string[] = [];
    const toRecompute = new Set<string>();
    const enqueue = (src: string) => {
      const next = graph.fastPersistReverseDeps?.get(src) || graph.fastReverseDeps.get(src);
      if (!next) return;
      for (const computeField of next) {
        if (!persistedComputeFields.has(computeField)) continue;
        if (!toRecompute.has(computeField)) {
          toRecompute.add(computeField);
          queue.push(computeField);
        }
      }
    };

    normalizedSeed.forEach(enqueue);
    for (let i = 0; i < queue.length; i++) enqueue(queue[i]);

    toRecompute.forEach(computeField => {
      this.addPersistComputeField(fields, computeField);

      const scalarDeps = graph.computeScalarDeps?.get(computeField);
      if (scalarDeps) {
        scalarDeps.forEach(dep => this.addPersistComputeField(fields, dep));
      }

      const pathDeps = graph.computePathDeps?.get(computeField) || [];
      for (const dep of pathDeps as Array<{ root?: string }>) {
        this.addPersistComputeField(fields, dep?.root);
      }
    });

    return Array.from(fields);
  }

  private async loadRowsForPersistRecompute(ids: string[], seedFields: Iterable<string>): Promise<Map<string, Entity>> {
    const normalizedIds = Array.from(new Set((ids || []).map(id => String(id || '').trim()).filter(Boolean)));
    if (!normalizedIds.length) {
      return new Map();
    }

    const fields = this.resolvePersistComputeSelection(seedFields);
    const condition: BaseQueryCondition = ['Id', 'in', normalizedIds];
    const searchFields = fields as unknown as SearchOptions<ObjectRecord>['fields'];
    const rows = await this.withRecordRuleBypass(async () => {
      return await this.withFieldRuleBypass(async () => {
        return await this.search(condition, {
          fields: searchFields,
        });
      });
    });

    const byId = new Map<string, Entity>();
    for (const row of rows || []) {
      const id = this.getEntityId(row);
      if (!id) continue;
      byId.set(id, row);
    }
    return byId;
  }

  private async applyPersistComputeFollowUps(followUps: Array<{ id: string; values: Entity }>): Promise<void> {
    if (!followUps.length) return;

    await this.withValidationBypass(async () => {
      for (const item of followUps) {
        const id = String(item?.id || '').trim();
        const values = (item?.values || {}) as Entity;
        if (!id || !Object.keys(values).length) continue;

        const idCondition: BaseQueryCondition = ['Id', '=', id];

        const query = this.db
          .updateTable(this.table)
          .set(values)
          .where(({ eb }: { eb: RepositoryPredicateBuilder }) => this.convertCondition(eb, idCondition, this.table));

        try {
          await this.execute(query);
        } catch (error) {
          this.wrapSqlWriteError(error, 'update');
        }
      }
    });
  }

  private async recomputePersistForCreate(createdIds: string[], sanitizedEntities: Entity[]): Promise<void> {
    const graph = this.meta.computeGraph;
    if (!graph || !graph.persistedComputeFields?.size) return;

    const normalizedIds = Array.from(new Set((createdIds || []).map(id => String(id || '').trim()).filter(Boolean)));
    if (!normalizedIds.length) return;

    const seedById = new Map<string, Set<string>>();
    const mergedSeed = new Set<string>();
    for (const entity of sanitizedEntities || []) {
      const id = this.getEntityId(entity);
      if (!id) continue;

      const seed = new Set<string>(Object.keys(entity || {}));
      seedById.set(id, seed);
      seed.forEach(field => mergedSeed.add(field));
    }
    if (!mergedSeed.size) return;

    const rowsById = await this.loadRowsForPersistRecompute(normalizedIds, mergedSeed);
    const followUps: Array<{ id: string; values: Entity }> = [];

    for (const id of normalizedIds) {
      const entity = rowsById.get(id);
      if (!entity) continue;

      const baseSeed = new Set<string>(seedById.get(id) ?? mergedSeed);
      if (!baseSeed.size) continue;

      const changed = new Set<string>(baseSeed);
      await ComputeEngine.recompute(this.meta, entity, changed, 'persist');

      const followUp: Entity = {};
      changed.forEach(field => {
        if (baseSeed.has(field)) return;
        if (Object.prototype.hasOwnProperty.call(entity, field)) {
          followUp[field] = entity[field];
        }
      });

      if (Object.keys(followUp).length) {
        followUps.push({ id, values: followUp });
      }
    }

    await this.applyPersistComputeFollowUps(followUps);
  }

  private async recomputePersistForUpdate(payload: { targetIds: string[]; sanitized: Entity }): Promise<void> {
    const graph = this.meta.computeGraph;
    if (!graph || !graph.persistedComputeFields?.size) return;

    const normalizedIds = Array.from(new Set((payload?.targetIds || []).map(id => String(id || '').trim()).filter(Boolean)));
    if (!normalizedIds.length) return;

    const baseSeed = new Set<string>(Object.keys((payload?.sanitized || {}) as Entity));
    if (!baseSeed.size) return;

    const rowsById = await this.loadRowsForPersistRecompute(normalizedIds, baseSeed);
    const followUps: Array<{ id: string; values: Entity }> = [];

    for (const id of normalizedIds) {
      const entity = rowsById.get(id);
      if (!entity) continue;

      const changed = new Set<string>(baseSeed);
      await ComputeEngine.recompute(this.meta, entity, changed, 'persist');

      const followUp: Entity = {};
      changed.forEach(field => {
        if (baseSeed.has(field)) return;
        if (Object.prototype.hasOwnProperty.call(entity, field)) {
          followUp[field] = entity[field];
        }
      });

      if (Object.keys(followUp).length) {
        followUps.push({ id, values: followUp });
      }
    }

    await this.applyPersistComputeFollowUps(followUps);
  }

  /**
   * Create records.
   */
  public async create(value: Entity[]): Promise<string[]> {
    return await executeRepositoryCreate(this.createCreateWriteDeps(), value);
  }

  /**
   * Invalidate repository runtime cache.
   */
  private invalidateCache(): void {
    try {
      const modelCtor = this.meta.type as unknown as typeof BaseModel;
      if (modelCtor && typeof modelCtor === 'function') {
        invalidateRepositoryRuntimeCache(modelCtor);
      }
    } catch (error) {
      console.warn('[Repository] LRU cache invalidation failed:', error);
    }
  }

  /**
   * Delete records.
   */
  public async delete(condition: BaseQueryCondition): Promise<DeleteResult[]> {
    return await executeRepositoryDelete(this.createDeleteWriteDeps(), condition);
  }

  /**
   * Update records.
   */
  public async update(vals: Entity, condition: BaseQueryCondition): Promise<UpdateResult[]> {
    return await executeRepositoryUpdate(this.createUpdateWriteDeps(), vals, condition);
  }

  public async count(condition: BaseQueryCondition): Promise<number> {
    const condWithRR = await this.applyRecordRuleToCondition(condition, 'read');
    return await countRepositoryConditionMatches(
      this.createConditionQueryDeps(queryCondition => this.applyDefaultLayers(queryCondition)),
      condWithRR
    );
  }

  /* ----------------------------- Shared ordering helpers ----------------------------- */

  private normalizeOrderBy(input: unknown): Array<{ field: string; order: 'asc' | 'desc' }> | undefined {
    return normalizeOrderByExternal(input);
  }

  private resolveEffectiveOrder(
    overrideOrder: Array<{ field: string; order: 'asc' | 'desc' }> | undefined | null,
    metaOrder: Array<{ field: string; order: 'asc' | 'desc' }> | undefined | null,
    meta: ModelMetadata
  ): Array<{ field: string; order: 'asc' | 'desc' }> {
    return resolveEffectiveOrderExternal(overrideOrder, metaOrder, meta);
  }

  private applyOrderByToQuery<T>(query: T, targetMeta: ModelMetadata, targetTable: string, orderList: Array<{ field: string; order: 'asc' | 'desc' }>): T {
    return applyOrderByToQueryExternal(query, targetMeta, targetTable, orderList, {
      getDialect: () => this.getDialect(),
      resolvePathField: (builder, field) => {
        const ctx = this.makeSelectCtx(builder, targetTable, targetMeta);
        const fieldResolver = (ctx as { field?: (model: unknown, path: string) => unknown }).field;
        if (typeof fieldResolver !== 'function') {
          throw new Error(`SelectCtx.field is missing for ${targetMeta.fullModelName || targetMeta.modelName || targetMeta.name}`);
        }
        return fieldResolver(targetMeta.type, field);
      },
      resolveSelectField: (builder, field, _fieldMeta) => {
        if (!hasRepositorySqlComputeExpression(targetMeta, field)) {
          throw new Error(`field sql compute handler is missing: ${targetMeta.fullModelName || targetMeta.modelName || targetMeta.name}.${field}`);
        }

        const out = resolveRepositorySqlComputeExpression(targetMeta, field, this.makeSelectCtx(builder, targetTable, targetMeta));
        if (out === undefined) {
          throw new Error(`field sql compute handler is missing: ${targetMeta.fullModelName || targetMeta.modelName || targetMeta.name}.${field}`);
        }
        return out;
      },
    });
  }

  /* ----------------------------- Search ----------------------------- */

  public async search(condition: BaseQueryCondition, options?: SearchOptions<ObjectRecord>): Promise<Entity[]> {
    return await executeRepositorySearch(this.createSearchDeps(), condition, options);
  }

  private async pruneSelectionTreeForFieldRule(meta: ModelMetadata, node: SelectionNode, denyCache: Map<unknown, string[]>): Promise<void> {
    await pruneRepositorySelectionTreeForFieldRule(
      createRepositoryFieldRuleSelectionDeps({
        isControlPlaneMetaModel: () => this.isControlPlaneMetaModel(),
      }),
      meta,
      node,
      denyCache
    );
  }

  /* ----------------------- Build selection tree (recursive) ----------------------- */

  private getScalarFields(meta: ModelMetadata) {
    return getScalarFieldsExternal(meta);
  }

  private buildSelectionTree(meta: ModelMetadata, fields: unknown[]) {
    return buildSelectionTreeExternal(meta, fields);
  }

  /* ----------------------- Recursive relation JSON selection ----------------------- */

  private buildRelationJsonSelect(qb: unknown, parentMeta: ModelMetadata, relKey: string, entry: SelectionRelationEntry) {
    return buildRelationJsonSelectExternal(this.db, () => this.getDialect(), parentMeta, relKey, entry);
  }

  private createSearchDeps() {
    return createRepositorySearchFacadeDeps({
      ...this.createRepositoryQueryBridgeDeps(),
      meta: this.meta,
      getDialect: () => this.getDialect(),
      isTopLevelGrpcCall: () => this.isTopLevelGrpcCall(),
      buildSelectionTree: (meta: ModelMetadata, fields: unknown[]) => this.buildSelectionTree(meta, fields),
      getScalarFields: (meta: ModelMetadata) => this.getScalarFields(meta),
      pruneSelectionTreeForFieldRule: (meta: ModelMetadata, node: unknown, denyCache: Map<unknown, string[]>) =>
        this.pruneSelectionTreeForFieldRule(meta, node as SelectionNode, denyCache),
      makeSelectCtx: (builder: unknown, selfTable: string, curMeta?: ModelMetadata) => this.makeSelectCtx(builder, selfTable, curMeta),
      aliasSelection: (selection: unknown, alias: string) => this.aliasSelection(selection, alias),
      buildRelationJsonSelect: (qb: unknown, parentMeta: ModelMetadata, relKey: string, entry: SelectionRelationEntry) =>
        this.buildRelationJsonSelect(qb, parentMeta, relKey, entry),
      ...this.createReadQueryFacadeDeps(),
      resolveEffectiveOrder: (
        overrideOrder: Array<{ field: string; order: 'asc' | 'desc' }> | undefined | null,
        metaOrder: Array<{ field: string; order: 'asc' | 'desc' }> | undefined | null,
        meta: unknown
      ) => this.resolveEffectiveOrder(overrideOrder, metaOrder, meta as ModelMetadata),
      decodeRowWithTree: (meta: ModelMetadata, node: unknown, row: unknown) => this.decodeRowWithTree(meta, node as SelectionNode, row),
    });
  }

  private createReadAggregateDeps(): Parameters<typeof executeRepositoryReadGroup>[0] {
    return createRepositoryReadAggregateFacadeDeps({
      ...this.createRepositoryQueryBridgeDeps(),
      meta: this.meta,
      ctx: this.ctx,
      getDialect: () => this.getDialect(),
      makeSelectCtx: (builder: unknown, table: string, meta?: ModelMetadata) => this.makeSelectCtx(builder, table, meta),
      ...this.createReadQueryFacadeDeps(),
    }) as unknown as Parameters<typeof executeRepositoryReadGroup>[0];
  }

  /**
   * Protect a block of work with a savepoint and roll back to it when an error is thrown.
   */
  public async withSavepoint<T>(fn: () => Promise<T>, name?: string): Promise<T> {
    return this.db.withSavepoint(fn, name);
  }

  // Optional physical delete that bypasses soft-delete logic.
  private async hardDelete(condition: BaseQueryCondition): Promise<DeleteResult[]> {
    return await executeRepositoryHardDelete(this.createDeleteWriteDeps(), condition);
  }

  /**
   * Repository-level single-layer readGroup implementation.
   * - Only a single groupby layer is supported; when an array is provided, only the first item is used.
   * - Relation joins are not performed automatically; only base-table fields and path-expression subqueries are used.
   * - where = applySoftLayer + convertCondition
   * - having supports filters over group aliases, aggregate aliases, and __count.
   * - orderBy supports aliases, aggregate aliases, and __count.
   * - limit and offset only apply to the current layer.
   * - Always returns __count.
   */
  public async readGroup<T>(options: RepoReadGroupOptions<T>): Promise<RepoReadGroupRow[]> {
    return await executeRepositoryReadGroup(this.createReadAggregateDeps(), options);
  }

  /**
   * Totals without groupby.
   * - Only aggregate aliases plus count(*) as "__count" are selected.
   * - where = applySoftLayer + convertCondition
   * - having, orderBy, limit, and offset are not supported.
   */
  public async readTotals<T>(options: RepoReadTotalsOptions<T>): Promise<RepoReadTotalsRow> {
    return await executeRepositoryReadTotals(this.createReadAggregateDeps(), options);
  }

  /**
   * Count top-level groups for readGroup.
   * - Without having, optimize to COUNT(DISTINCT groupExpr).
   * - With having, build a subquery such as select groupExpr, [aggs...], COUNT(*) as "__count" group by groupExpr having ... and count(*) outside it.
   * - where = applySoftLayer + convertCondition
   */
  public async readGroupCount<T>(options: RepoReadGroupCountOptions<T>): Promise<number> {
    return await executeRepositoryReadGroupCount(this.createReadAggregateDeps(), options);
  }
}
