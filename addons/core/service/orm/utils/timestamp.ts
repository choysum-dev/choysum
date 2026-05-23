// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Insertable, Updateable } from '../repository/types';
import { asObjectRecord } from '../../../utils/object';

function readOwnField(input: unknown, key: string): unknown {
  const record = asObjectRecord(input);
  return record?.[key];
}

/**
 * Timestamp helpers for model create and update flows.
 */
export class TimestampUtils {
  /**
   * Add timestamps for create operations, including CreatedAt and UpdatedAt.
   *
   * @param value Input value to enrich.
   * @returns Value with timestamps applied.
   */
  static addTimestamps<T>(value: Partial<Insertable<T>>): Partial<Insertable<T>> {
    const now = new Date();
    return {
      ...value,
      CreatedAt: readOwnField(value, 'CreatedAt') || now,
      UpdatedAt: readOwnField(value, 'UpdatedAt') || now,
    };
  }

  /**
   * Add the update timestamp for update operations.
   * UpdatedAt is only injected when the caller did not provide one.
   *
   * @param value Input value to enrich.
   * @returns Value with UpdatedAt applied.
   */
  static addUpdateTimestamp<T>(value: Partial<Updateable<T>>): Partial<Updateable<T>> {
    return {
      ...value,
      UpdatedAt: readOwnField(value, 'UpdatedAt') || new Date(),
    };
  }
}
