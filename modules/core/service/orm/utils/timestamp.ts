// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getUserId } from '../../runtime/context';
import { Insertable, Updateable } from '../repository/types';
import { asObjectRecord } from '../../../utils/object';

function readOwnField(input: unknown, key: string): unknown {
  const record = asObjectRecord(input);
  return record?.[key];
}

function resolveActorUid(): string {
  return String(getUserId() || '').trim();
}

/**
 * Timestamp + audit-uid helpers for model create and update flows.
 */
export class TimestampUtils {
  /**
   * Add timestamps for create operations, including CreatedAt and UpdatedAt.
   * When a request actor is present, also default CreatedUid / UpdatedUid.
   *
   * @param value Input value to enrich.
   * @returns Value with timestamps applied.
   */
  static addTimestamps<T>(value: Partial<Insertable<T>>): Partial<Insertable<T>> {
    const now = new Date();
    const actor = resolveActorUid();
    const out: Record<string, unknown> = {
      ...(asObjectRecord(value) || {}),
      CreatedAt: readOwnField(value, 'CreatedAt') || now,
      UpdatedAt: readOwnField(value, 'UpdatedAt') || now,
    };
    if (actor) {
      if (!readOwnField(value, 'CreatedUid')) out.CreatedUid = actor;
      if (!readOwnField(value, 'UpdatedUid')) out.UpdatedUid = actor;
    }
    return out as Partial<Insertable<T>>;
  }

  /**
   * Add the update timestamp for update operations.
   * UpdatedAt is only injected when the caller did not provide one.
   * When a request actor is present, also default UpdatedUid.
   *
   * @param value Input value to enrich.
   * @returns Value with UpdatedAt applied.
   */
  static addUpdateTimestamp<T>(value: Partial<Updateable<T>>): Partial<Updateable<T>> {
    const actor = resolveActorUid();
    const out: Record<string, unknown> = {
      ...(asObjectRecord(value) || {}),
      UpdatedAt: readOwnField(value, 'UpdatedAt') || new Date(),
    };
    if (actor && !readOwnField(value, 'UpdatedUid')) {
      out.UpdatedUid = actor;
    }
    return out as Partial<Updateable<T>>;
  }

  /**
   * Apply BaseModel audit-uid rules for a scalar update payload (AU7 / AU8):
   * - strip client CreatedUid writes
   * - refresh UpdatedUid when actor present
   * - on restore (DeletedAt cleared): clear DeletedUid; otherwise strip client DeletedUid
   *
   * Mutates `value` in place and returns it.
   */
  static applyAuditUidOnUpdate(value: Record<string, unknown>): Record<string, unknown> {
    delete value.CreatedUid;

    const restoring = Object.prototype.hasOwnProperty.call(value, 'DeletedAt') && value.DeletedAt == null;
    if (restoring) {
      value.DeletedUid = null;
    } else {
      delete value.DeletedUid;
    }

    const actor = resolveActorUid();
    if (actor) {
      value.UpdatedUid = actor;
    }
    return value;
  }
}
