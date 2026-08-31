// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import type { FieldSelection } from '@/core/service/api/selection';

/**
 * Audit-local isomorphic copy of message PolymorphicRecordModel.
 * Kept in-module so audit does not depend on the message package.
 * Must be the module default export so `@Model` classes can `extends` it.
 */
export default abstract class PolymorphicRecordModel extends BaseModel {
  /** Order field for SearchByRecord (FieldChange uses At). */
  protected static polymorphicOrderByField(): string {
    return 'At';
  }

  /** Denied message passed to the module target-record probe. */
  protected static polymorphicDeniedMessage(): string {
    return 'Access is not allowed for this record';
  }

  /** Raise domain INVALID_ARGUMENT when Model/ResId are missing. */
  protected static raisePolymorphicInvalidArgument(_message: string): never {
    throw new Error('raisePolymorphicInvalidArgument must be overridden');
  }

  /** Assert the caller can Search the underlying business record. */
  protected static async assertPolymorphicTargetReadable(_model: string, _resId: string): Promise<void> {
    throw new Error('assertPolymorphicTargetReadable must be overridden');
  }

  /**
   * Searches this model's rows for one target record after a readability probe.
   */
  public static async SearchByRecord(
    this: typeof PolymorphicRecordModel,
    model: string,
    resId: string,
    fields?: FieldSelection<any>
  ): Promise<Partial<any>[]> {
    const m = String(model || '').trim();
    const id = String(resId || '').trim();
    if (!m || !id) {
      this.raisePolymorphicInvalidArgument('SearchByRecord requires Model and ResId');
    }
    await this.assertPolymorphicTargetReadable(m, id);
    return await (this as any).Search(
      {
        And: [
          ['Model', '=', m],
          ['ResId', '=', id],
        ],
      },
      {
        fields,
        orderBy: { field: this.polymorphicOrderByField(), order: 'asc' },
      }
    );
  }
}
