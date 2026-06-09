// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { QueryCondition, DeleteOptions } from '../repository/types';
import type BaseModel from './model';
import { resolveRepositoryWithSoftDeleteOptions } from './model_soft_delete_scope';
import { collectModelUpstreamInverseFields, triggerModelUpstream } from './model_runtime_service_facade';
import type { RuntimeModelCtor } from './types';
import type { ObjectRecord } from '../../../utils/types';

type DeleteRepositoryLike = {
  search(condition: unknown, options?: { fields?: unknown }): Promise<Array<ObjectRecord>>;
  delete(condition: unknown): Promise<Array<unknown>>;
};

/**
 * DeleteOperations owns model delete flows and upstream recompute propagation.
 */
export class DeleteOperations {
  private static resolveRepository(ModelCtor: RuntimeModelCtor, options?: DeleteOptions): DeleteRepositoryLike {
    return resolveRepositoryWithSoftDeleteOptions(ModelCtor, options) as unknown as DeleteRepositoryLike;
  }

  /**
   * Static delete by condition. Returns the affected row count.
   * - Does not perform cascade or compute handling, matching the existing behavior.
   */
  static async Delete<T extends BaseModel>(ModelCtor: RuntimeModelCtor, condition: QueryCondition<T>, options?: DeleteOptions): Promise<number> {
    const repository = DeleteOperations.resolveRepository(ModelCtor, options);
    const upstreamInverseFields = collectModelUpstreamInverseFields(ModelCtor);
    const snapshotFields = Array.from(new Set<string>(['Id', ...upstreamInverseFields]));
    const oldRows = await repository.search(condition as unknown, {
      fields: snapshotFields as unknown,
    });

    const result = await repository.delete(condition as unknown);

    for (const row of oldRows || []) {
      try {
        await triggerModelUpstream({
          childCtor: ModelCtor,
          operation: 'delete',
          changedFields: [],
          beforeEntity: row,
        });
      } catch (e) {
        if (typeof console !== 'undefined') {
          console.warn('[Delete] upstream recompute failed and was ignored:', e);
        }
      }
    }

    return result.length;
  }

  /**
   * Static delete by Id. Returns the affected row count.
   */
  static async DeleteById<T extends BaseModel>(ModelCtor: RuntimeModelCtor, id: string, options?: DeleteOptions): Promise<number> {
    return await DeleteOperations.Delete<T>(ModelCtor, ['Id', '=', id] as QueryCondition<T>, options);
  }
}
