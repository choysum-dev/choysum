// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { convertCondition } from './condition_compiler';
export { isEmptyCondition, andAll, toRepoCondition } from './condition_helpers';
export { buildContainsExpression } from './json_contains';
export type { RepositoryVisibilityLayerDeps, RepositoryDefaultLayerDeps } from './condition_layer';
export {
  repositorySoftDeleteEnabled,
  isEmptyRepositoryCondition,
  andRepositoryConditions,
  applyRepositorySoftDeleteLayer,
  applyRepositoryDefaultLayers,
} from './condition_layer';
export { locateRepositoryIdsForCondition, countRepositoryConditionMatches } from './condition_query';
export { convertRepositoryHavingCondition } from './having_condition';
export type { RepositoryOrderSpec } from './ordering';
export { normalizeOrderBy, computeFallbackOrder, resolveEffectiveOrder, applyOrderByToQuery } from './ordering';
export type { DbLike } from './select_context';
export { makeSelectCtx } from './select_context';
export { getStringHelpers } from './string_helpers';
export { hasRepositorySqlComputeExpression, isRepositorySelectableScalarField, resolveRepositorySqlComputeExpression } from './sql_compute_expression';
export type { RepositoryPredicateBuilder, RepositoryPredicate } from './predicate_builder_adapter';
