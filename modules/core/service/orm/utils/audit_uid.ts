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
 * Audit-uid helpers for BaseModel CreatedUid / UpdatedUid / DeletedUid (AU6–AU8).
 * Called from repository write prepare (create/update) and soft-delete SET — not from the model layer.
 * Wall-clock *At columns stay in {@link TimestampUtils}.
 */
export class AuditUidUtils {
  /**
   * Default CreatedUid / UpdatedUid on create when a request actor is present.
   * Preserves explicit client values (setdefault semantics).
   */
  static addCreateUids<T>(value: Partial<Insertable<T>>): Partial<Insertable<T>> {
    const actor = resolveActorUid();
    if (!actor) return value;
    const out: Record<string, unknown> = { ...(asObjectRecord(value) || {}) };
    if (!readOwnField(value, 'CreatedUid')) out.CreatedUid = actor;
    if (!readOwnField(value, 'UpdatedUid')) out.UpdatedUid = actor;
    return out as Partial<Insertable<T>>;
  }

  /**
   * Default UpdatedUid on update when a request actor is present and the caller
   * did not provide one.
   */
  static addUpdateUid<T>(value: Partial<Updateable<T>>): Partial<Updateable<T>> {
    const actor = resolveActorUid();
    if (!actor || readOwnField(value, 'UpdatedUid')) return value;
    return {
      ...(asObjectRecord(value) || {}),
      UpdatedUid: actor,
    } as Partial<Updateable<T>>;
  }

  /**
   * Apply audit-uid rules for a scalar update payload (AU7 / AU8):
   * - strip client CreatedUid writes
   * - refresh UpdatedUid when actor present
   * - on restore (DeletedAt cleared): clear DeletedUid; otherwise strip client DeletedUid
   *
   * Mutates `value` in place and returns it.
   */
  static applyOnUpdate(value: Record<string, unknown>): Record<string, unknown> {
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

  /**
   * Stamp DeletedUid + UpdatedUid on a soft-delete SET payload when actor present.
   * Mutates `value` in place and returns it.
   */
  static applyOnSoftDelete(value: Record<string, unknown>): Record<string, unknown> {
    const actor = resolveActorUid();
    if (actor) {
      value.UpdatedUid = actor;
      value.DeletedUid = actor;
    }
    return value;
  }
}
