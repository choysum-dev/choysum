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
   * Stamp CreatedUid / UpdatedUid on create when a request actor is present.
   * CreatedUid uses setdefault (keeps explicit values for trusted/seed paths).
   * UpdatedUid always refreshes from actor when present.
   */
  static addCreateUids<T>(value: Partial<Insertable<T>>): Partial<Insertable<T>> {
    const actor = resolveActorUid();
    if (!actor) return value;
    const out: Record<string, unknown> = { ...(asObjectRecord(value) || {}) };
    if (!readOwnField(value, 'CreatedUid')) out.CreatedUid = actor;
    out.UpdatedUid = actor;
    return out as Partial<Insertable<T>>;
  }

  /**
   * Stamp UpdatedUid on update when a request actor is present.
   * Always strips client UpdatedUid first so untrusted input cannot persist without an actor.
   */
  static addUpdateUid<T>(value: Partial<Updateable<T>>): Partial<Updateable<T>> {
    const out: Record<string, unknown> = { ...(asObjectRecord(value) || {}) };
    delete out.UpdatedUid;
    const actor = resolveActorUid();
    if (actor) out.UpdatedUid = actor;
    return out as Partial<Updateable<T>>;
  }

  /**
   * Apply audit-uid rules for a scalar update payload (AU7 / AU8):
   * - strip client CreatedUid writes
   * - strip client UpdatedUid, then refresh from actor when present
   * - DeletedAt cleared (restore): clear DeletedUid
   * - DeletedAt set (soft-delete via Update): stamp DeletedUid from actor
   * - otherwise strip client DeletedUid
   *
   * Mutates `value` in place and returns it.
   */
  static applyOnUpdate(value: Record<string, unknown>): Record<string, unknown> {
    delete value.CreatedUid;
    delete value.UpdatedUid;

    const actor = resolveActorUid();
    const hasDeletedAt = Object.prototype.hasOwnProperty.call(value, 'DeletedAt');
    if (hasDeletedAt) {
      if (value.DeletedAt == null) {
        value.DeletedUid = null;
      } else if (actor) {
        value.DeletedUid = actor;
      } else {
        delete value.DeletedUid;
      }
    } else {
      delete value.DeletedUid;
    }

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
