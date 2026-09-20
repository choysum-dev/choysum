// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * SF-6: compile-only export-surface freeze.
 *
 * Author / SSOT barrels must not re-export engine symbols (direction §3.2) or
 * hard-cut old names (SF-2…SF-5). Each @ts-expect-error is a regression tripwire.
 */

// --- api root: engine metadata / ducks / prepared ops / kysely ---

// @ts-expect-error SelectCtx stays behind orm/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoSelectCtx = import('@/core/service/api').SelectCtx;

// @ts-expect-error FlatFieldOptions stays behind orm/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoFlatFieldOptions = import('@/core/service/api').FlatFieldOptions;

// @ts-expect-error ModelComputeGraph stays behind orm/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoModelComputeGraph = import('@/core/service/api').ModelComputeGraph;

// @ts-expect-error ComputeDeps stays behind orm/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoComputeDeps = import('@/core/service/api').ComputeDeps;

// @ts-expect-error RepositoryAliasableLike is engine-only
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoRepositoryAliasableLike = import('@/core/service/api').RepositoryAliasableLike;

// @ts-expect-error ExpressionBuilder is engine-only
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoExpressionBuilder = import('@/core/service/api').ExpressionBuilder;

// @ts-expect-error PreparedRelationOp is engine-only
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoPreparedRelationOp = import('@/core/service/api').PreparedRelationOp;

// @ts-expect-error NestedPath stays SSOT/engine, not author root
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoNestedPath = import('@/core/service/api').NestedPath;

// --- api root: hard-cut old names (SF-2…SF-5) ---

// @ts-expect-error Entity was removed (use SelectResult)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoEntity = import('@/core/service/api').Entity;

// @ts-expect-error Queryable was removed (use Selectable)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoQueryable = import('@/core/service/api').Queryable;

// @ts-expect-error RelationOperations was renamed to RelationPatch
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoRelationOperations = import('@/core/service/api').RelationOperations;

// @ts-expect-error ValidationIssueLite was renamed to ParsedValidationIssue
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoValidationIssueLite = import('@/core/service/api').ValidationIssueLite;

// @ts-expect-error RepoReadGroupOptions was renamed to RepositoryReadGroupOptions (engine wire)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoRepoReadGroupOptions = import('@/core/service/api').RepoReadGroupOptions;

// @ts-expect-error BaseQueryCondition was renamed to UntypedQueryCondition
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoBaseQueryCondition = import('@/core/service/api').BaseQueryCondition;

// @ts-expect-error ConditionExpr was removed
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRootNoConditionExpr = import('@/core/service/api').ConditionExpr;

// --- api/metadata: keep engine metadata off the author metadata sub-entry ---

// @ts-expect-error SelectCtx must not reappear on api/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiMetadataNoSelectCtx = import('@/core/service/api/metadata').SelectCtx;

// @ts-expect-error FlatFieldOptions must not reappear on api/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiMetadataNoFlatFieldOptions = import('@/core/service/api/metadata').FlatFieldOptions;

// @ts-expect-error ModelComputeGraph must not reappear on api/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiMetadataNoModelComputeGraph = import('@/core/service/api/metadata').ModelComputeGraph;

// @ts-expect-error ComputeDeps must not reappear on api/metadata
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiMetadataNoComputeDeps = import('@/core/service/api/metadata').ComputeDeps;

// --- api/relation: prepared ops stay engine-only ---

// @ts-expect-error PreparedRelationOp must not appear on api/relation
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRelationNoPreparedRelationOp = import('@/core/service/api/relation').PreparedRelationOp;

// @ts-expect-error RelationOperations hard-cut alias must not return
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type ApiRelationNoRelationOperations = import('@/core/service/api/relation').RelationOperations;

// --- repository/types SSOT: ducks / kysely stay on types/engine ---

// @ts-expect-error RepositoryAliasableLike belongs on types/engine
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type TypesBarrelNoRepositoryAliasableLike = import('@/core/service/orm/repository/types').RepositoryAliasableLike;

// @ts-expect-error ExpressionBuilder belongs on types/engine
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type TypesBarrelNoExpressionBuilder = import('@/core/service/orm/repository/types').ExpressionBuilder;

// @ts-expect-error SelectQueryBuilder belongs on types/engine
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type TypesBarrelNoSelectQueryBuilder = import('@/core/service/orm/repository/types').SelectQueryBuilder;

// @ts-expect-error RepositoryQueryLike belongs on types/engine
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type TypesBarrelNoRepositoryQueryLike = import('@/core/service/orm/repository/types').RepositoryQueryLike;

// @ts-expect-error RepoReadGroupOptions hard-cut old name must stay gone
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type TypesBarrelNoRepoReadGroupOptions = import('@/core/service/orm/repository/types').RepoReadGroupOptions;

// --- repository runtime barrel: no type surface for engine helpers ---

// @ts-expect-error repository index is runtime-only (no RepositoryAliasableLike)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type RepoIndexNoRepositoryAliasableLike = import('@/core/service/orm/repository').RepositoryAliasableLike;

// @ts-expect-error repository index must not expose SelectCtx
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type RepoIndexNoSelectCtx = import('@/core/service/orm/repository').SelectCtx;

test('SF-6 export-surface type deny list compiles (api / metadata / types / repository)', () => {
  expect(true).toBe(true);
});
