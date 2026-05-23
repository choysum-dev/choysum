// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseQueryCondition } from '../repository/types';

/**
 * Optimistic-lock utilities for concurrency-control helpers.
 */
export class LockUtils {
  /**
   * Build an optimistic-lock condition.
   * When currentUpdatedAt is provided, this creates a composite condition containing both Id and UpdatedAt.
   * Otherwise only the Id condition is used.
   *
   * @param id Record Id.
   * @param currentUpdatedAt Current update timestamp, when available.
   * @returns Query condition.
   */
  static buildOptimisticLockCondition(id: string, currentUpdatedAt?: Date): BaseQueryCondition {
    if (!currentUpdatedAt) {
      return ['Id', '=', id];
    }

    return {
      And: [
        ['Id', '=', id],
        ['UpdatedAt', '=', currentUpdatedAt],
      ],
    };
  }

  /**
   * Validate the update result.
   * Throws an optimistic-lock conflict when no rows were updated.
   *
   * @param result Result array returned by the update operation.
   * @throws When the update fails because the optimistic lock no longer matches.
   */
  static validateUpdateResult(result: unknown[] | undefined): void {
    if (!result || result.length === 0) {
      throw new Error('Update failed: This record has been modified by another user. ' + 'Please reload the record and try again.');
    }
  }

  /**
   * Check whether an error is an optimistic-lock conflict.
   *
   * @param error Error object.
   * @returns True when the error matches the optimistic-lock conflict message.
   */
  static isOptimisticLockError(error: Error): boolean {
    return error.message.includes('has been modified');
  }

  /**
   * Format an optimistic-lock error.
   *
   * @param error Original error.
   * @returns Formatted error object.
   */
  static formatLockError(error: Error): Error {
    if (this.isOptimisticLockError(error)) {
      return error; // Return optimistic-lock errors unchanged.
    }
    return new Error(`Update failed: ${error.message}`);
  }
}
