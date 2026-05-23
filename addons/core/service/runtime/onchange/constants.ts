// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Onchange runtime constants (Phase 1).
 * Later phases may override these values through configuration or environment variables.
 */

import { getRuntimeOnchangeFlagOverridesValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';

/** Maximum single-hop path depth. Phase 1 only supports 1, meaning Root.Field. */
export const MAX_PATH_DEPTH = 1;

/** Maximum number of times a field may be scheduled within one Onchange run. */
export const DEFAULT_LOOP_THRESHOLD = 1;

/** Default handler priority. Smaller values run earlier. */
export const ONCHANGE_DEFAULT_PRIORITY = 100;

/** Whether multi-hop path prefetching is enabled. This can be turned on in Phase 2 or 3. */
export const ENABLE_MULTI_HOP_PREVIEW = true;

/** Whether collection path reads such as Lines.Product.Name are enabled. This can be turned on in Phase 3. */
export const ENABLE_COLLECTION_PATH_READS = true;

/** Iteration cap used to prevent infinite loops. */
export const MAX_ITERATIONS = 10;

// ===== Phase 2 feature flags =====

/** Whether to collect Onchange diagnostics such as missing-field counts or prefetch timings. */
export const ENABLE_DIAGNOSTICS = true;

/** Whether to enable path-prefetch plan caching from signature to plan skeleton. */
export const PLAN_CACHE_ENABLED = true;

/** Maximum multi-hop depth. When multi-hop is disabled, this effectively stays aligned with MAX_PATH_DEPTH at 1. */
export const MAX_MULTI_HOP_DEPTH = 2; // Enabled only when ENABLE_MULTI_HOP_PREVIEW is active.

/** Diagnostic output detail level. 'info' reports basics, while 'debug' includes more internal detail. */
export type DiagnosticsLevel = 'info' | 'debug';
export const DIAGNOSTICS_LEVEL: DiagnosticsLevel = 'info';

/** Whether to record per-layer batch and row diagnostics for execution. */
export const DIAG_BATCH_STATS_ENABLED = true;

// ===== Phase 3 feature flags and parameters =====

/** Whether to enable request-scoped result caching during plan execution for a single Onchange call. */
export const REQUEST_CACHE_ENABLED = true; // Phase 3 enabled.

/**
 * Request-scoped result cache with the lifetime of one Onchange call.
 *
 * Features:
 * - Reuse results when the same model and field set are queried repeatedly within one execute().
 * - Use cache keys in the form "ModelName#Field1,Field2" with Id-level hits.
 *
 * Benefits:
 * - Reduce 30-50% of queries when multiple handlers or computes touch the same path.
 * - Reuse intermediate multi-hop layers, such as sharing the A.B query for A.B.C and A.B.D.
 *
 * Risk: none, because the cache is destroyed automatically at the end of execute().
 * Memory: the data volume of one Onchange call is limited, typically under 10 MB.
 */

/** Prefetch batch size, grouped by model and level and chunked by Id count. */
export const PREFETCH_BATCH_SIZE = 200;

/** Cross-request LRU cache size in entries. A value of 0 disables it. */
export const LRU_CACHE_SIZE = (() => {
  return 500; // Default to 500 entries.
})();

/**
 * LRU cache size for cross-request reuse, measured in entries.
 *
 * Configuration guidance:
 * - Development: 500 by default.
 * - Production: tune based on monitoring, usually between 200 and 1000.
 *
 * Memory estimate:
 * - One entry is roughly 10 rows × 100 bytes, or about 1 KB.
 * - 500 entries are about 500 KB of raw data.
 * - Real usage is closer to 2-5 MB after V8 object overhead.
 *
 * Invalidation strategy:
 * - TTL is 5 minutes with automatic expiration.
 * - Clear by model at a coarse grain when the model is updated.
 * - Apply a 20 MB memory cap so lru-cache can evict automatically.
 *
 * Monitoring:
 * - Use OnchangeCacheManager.getStats() to inspect hit rate.
 * - Override the default with the CHOYSUM_LRU_CACHE_SIZE environment variable.
 */

/** Plan signature version. Increment this when the signature shape changes to force cache invalidation. */
export const PLAN_SIGNATURE_VERSION = 1;

/** Maximum depth for parent-to-child preview cascades within one RPC call. */
export const PREVIEW_CASCADE_MAX_DEPTH = 5;

/** Whether child-model @Onchange handlers run during preview. Compute always runs. */
export const ENABLE_CHILD_ONCHANGE_IN_PREVIEW = true;

/**
 * Whether kernel validation is enabled during preview.
 * It is enabled by default and scoped by PREVIEW_KERNEL_RULES.
 */
export const ENABLE_PREVIEW_KERNEL_VALIDATION = true;

/**
 * Subset of kernel rules used during preview to avoid noisy required checks on drafts.
 */
export const PREVIEW_KERNEL_RULES = ['int', 'selection', 'decimal', 'relationShape'] as const;

export type OnchangeRuntimeFlags = {
  ENABLE_MULTI_HOP_PREVIEW: boolean;
  MAX_PATH_DEPTH: number;
  MAX_MULTI_HOP_DEPTH: number;
  PLAN_CACHE_ENABLED: boolean;
  PLAN_SIGNATURE_VERSION: number;
};

function getRuntimeFlagOverrides(): Partial<OnchangeRuntimeFlags> | undefined {
  const candidate = asObjectRecord(getRuntimeOnchangeFlagOverridesValue());
  if (!candidate) return undefined;
  return candidate as Partial<OnchangeRuntimeFlags>;
}

export function getOnchangeRuntimeFlags(): OnchangeRuntimeFlags {
  const overrides = getRuntimeFlagOverrides();
  return {
    ENABLE_MULTI_HOP_PREVIEW: typeof overrides?.ENABLE_MULTI_HOP_PREVIEW === 'boolean' ? overrides.ENABLE_MULTI_HOP_PREVIEW : ENABLE_MULTI_HOP_PREVIEW,
    MAX_PATH_DEPTH: typeof overrides?.MAX_PATH_DEPTH === 'number' ? overrides.MAX_PATH_DEPTH : MAX_PATH_DEPTH,
    MAX_MULTI_HOP_DEPTH: typeof overrides?.MAX_MULTI_HOP_DEPTH === 'number' ? overrides.MAX_MULTI_HOP_DEPTH : MAX_MULTI_HOP_DEPTH,
    PLAN_CACHE_ENABLED: typeof overrides?.PLAN_CACHE_ENABLED === 'boolean' ? overrides.PLAN_CACHE_ENABLED : PLAN_CACHE_ENABLED,
    PLAN_SIGNATURE_VERSION: typeof overrides?.PLAN_SIGNATURE_VERSION === 'number' ? overrides.PLAN_SIGNATURE_VERSION : PLAN_SIGNATURE_VERSION,
  };
}
