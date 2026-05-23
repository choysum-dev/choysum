// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../../orm/metadata/model';
import type { PathPrefetchPlan, PrefetchExecStats } from '../types';
import {
  finalizePlan,
  mergeCollectionsIntoPlan,
  mergeComputePathsIntoPlan,
  mergeM2OChainsIntoPlan,
  mergeOnchangeReadsExIntoPlan,
  mergeOnchangeReadsIntoPlan,
} from './builder';
import { computePlanDepth, getCachedOrBuildPlanV2 } from './cache';
import { extractComputeCollectionPathDeps, extractComputePathDeps } from './compute_deps';
import { PathPlanExecutor } from './executor';
import type { ModelCtor } from './shared';
import type { UnknownRecord } from '../../../../utils/types';

export { extractComputeCollectionPathDeps, extractComputePathDeps };

/**
 * PathPlanBuilder builds and executes onchange prefetch plans.
 */
export class PathPlanBuilder {
  private readonly plan: PathPrefetchPlan;
  private pathDepthMax = 1;

  // Level-1 request-scoped cache keyed by modelKey#fieldsSig and storing Map<Id, row>.
  private requestCache?: Map<string, Map<string, UnknownRecord>>;

  private constructor() {
    this.plan = {
      rootManyToOne: new Map(),
      m2oChains: new Map(),
      collections: new Map(),
    };
  }

  /**
   * Creates an empty path prefetch plan.
   */
  static createEmptyPlan(): PathPrefetchPlan {
    return {
      rootManyToOne: new Map(),
      m2oChains: new Map(),
      collections: new Map(),
    };
  }

  /**
   * Creates a mutable builder instance.
   */
  static builder(): PathPlanBuilder {
    return new PathPlanBuilder();
  }

  /**
   * Returns the current in-progress plan.
   */
  getPlan(): PathPrefetchPlan {
    return this.plan;
  }

  /**
   * Returns the maximum path depth recorded on the finalized plan.
   */
  getPathDepthMax(): number {
    return this.pathDepthMax;
  }

  /**
   * Get a prefetch plan from cache or build a new V2 plan.
   * Strategy A uses a conservative approach for computeCollectionPathDeps by loading only the first segment after the
   * collection, which means child-row Id plus the direct foreign key without drilling deeper.
   */
  static getCachedOrBuildV2(
    modelCtor: ModelCtor,
    m2oReads: Map<string, string[][]>,
    collectionReads: Map<string, string[][]>,
    computeM2oPaths: Map<string, string[][]>,
    computeCollectionPaths: Map<string, string[][]>
  ): { plan: PathPrefetchPlan; fromCache: boolean; signature: string; pathDepthMax: number } {
    return getCachedOrBuildPlanV2(modelCtor, m2oReads, collectionReads, computeM2oPaths, computeCollectionPaths, (m2oChains, collections) => {
      const builder = PathPlanBuilder.builder().mergeM2OChains(m2oChains).mergeCollections(collections).finalize();
      return { plan: builder.getPlan(), pathDepthMax: builder.getPathDepthMax() };
    });
  }

  /**
   * Merges parsed onchange reads that include both ManyToOne and collection paths.
   */
  mergeOnchangeReadsEx(parsed: { m2o: Map<string, string[][]>; collections: Map<string, string[][]> }): this {
    mergeOnchangeReadsExIntoPlan(this.plan, parsed);

    return this;
  }

  /**
   * Merges ManyToOne-only onchange reads into the plan.
   */
  mergeOnchangeReads(readsMap: Map<string, string[][]>): this {
    mergeOnchangeReadsIntoPlan(this.plan, readsMap);
    return this;
  }

  /**
   * Merges compute-derived path requirements into the plan.
   */
  mergeComputePaths(m2oPaths: Map<string, string[][]>, collectionPaths?: Map<string, string[][]>): this {
    mergeComputePathsIntoPlan(this.plan, m2oPaths, collectionPaths);
    return this;
  }

  /**
   * Finalizes the plan and records its maximum depth.
   */
  finalize(): this {
    this.pathDepthMax = finalizePlan(this.plan);
    return this;
  }

  /**
   * Computes the effective depth for a completed plan.
   */
  static computeDepth(plan: PathPrefetchPlan): number {
    return computePlanDepth(plan);
  }

  /**
   * Executes the current plan against a draft payload.
   */
  async execute(_modelCtor: ModelCtor, meta: ModelMetadata, draft: UnknownRecord): Promise<PrefetchExecStats> {
    return await new PathPlanExecutor(this.plan).execute(meta, draft);
  }

  /**
   * Executes a provided plan without allocating a mutable builder instance.
   */
  static async executeWithPlan(modelCtor: ModelCtor, meta: ModelMetadata, draft: UnknownRecord, plan: PathPrefetchPlan): Promise<PrefetchExecStats> {
    const normalizedPlan: PathPrefetchPlan = {
      rootManyToOne: plan.rootManyToOne,
      m2oChains: plan.m2oChains || new Map(),
      collections: plan.collections || new Map(),
    };
    return await new PathPlanExecutor(normalizedPlan).execute(meta, draft);
  }

  /**
   * Merges ManyToOne chains into the current plan.
   */
  mergeM2OChains(m2o: Map<string, string[][]>): this {
    mergeM2OChainsIntoPlan(this.plan, m2o);
    return this;
  }

  /**
   * Merges collection chains into the current plan.
   */
  mergeCollections(colls: Map<string, string[][]>): this {
    mergeCollectionsIntoPlan(this.plan, colls);
    return this;
  }
}
