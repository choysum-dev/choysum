// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export {
  createRepositoryConditionQueryDeps,
  createRepositoryReadConditionDeps,
  createRepositoryReadOrderDeps,
  createRepositoryReadQueryFacadeDeps,
  createRepositorySearchFacadeDeps,
  createRepositoryReadAggregateDeps,
  createRepositoryReadAggregateFacadeDeps,
} from './deps';
export { executeRepositorySearch } from './search';
export { executeRepositoryReadGroup, executeRepositoryReadTotals, executeRepositoryReadGroupCount } from './read_aggregate';
export {
  buildRepositoryReadAggregateGroupExprs,
  buildRepositoryReadAggregateSelections,
  buildRepositoryReadAggregateTotalSelections,
  applyRepositoryReadAggregateCondition,
  resolveRepositoryReadAggregateKnownAliases,
  normalizeRepositoryAggregateDecimals,
} from './read_aggregate_helpers';
export type { NormalizedGroupSpec, NormalizedCompositeGroupSpec, NormalizedAgg } from './group_spec';
export {
  normalizeGroupBySpec,
  normalizeGroupBySpecs,
  normalizeFieldAggregation,
  rebuildGroupSpec,
  rebuildCompositeGroupSpec,
  rebuildAggFields,
} from './group_spec';
export { coerceToBucketStart, nextBucket, enumerateBuckets } from './time_bucket_runtime';
