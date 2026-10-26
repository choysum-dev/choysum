// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
import { _lt } from '../i18n';
import SandboxRecord from './sandbox_record';

const LINE_STATES = [
  { value: 'todo', label: 'Todo' },
  { value: 'doing', label: 'Doing' },
  { value: 'done', label: 'Done' },
] as const;

/** OneToMany line for subtable and Kanban hand tests. */
@Model('SandboxLine')
export default class SandboxLine extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxRecord, onDelete: 'CASCADE' },
    notNull: true,
    index: true,
    string: _lt('Record', { scope: 'sandbox.model.SandboxLine.fields' }),
  })
  RecordId: SandboxRecord;

  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    string: _lt('Name', { scope: 'sandbox.model.SandboxLine.fields' }),
  })
  Name: string;

  @Field({
    type: 'int',
    notNull: true,
    default: () => 1,
    string: _lt('Quantity', { scope: 'sandbox.model.SandboxLine.fields' }),
    help: _lt('Onchange clamps Quantity to 1…999 and warns when adjusted.', {
      scope: 'sandbox.model.SandboxLine.fields',
    }),
  })
  Quantity: number;

  @Field({
    type: 'selection',
    selection: () => [...LINE_STATES],
    size: 16,
    notNull: true,
    default: () => 'todo',
    index: true,
    string: _lt('State', { scope: 'sandbox.model.SandboxLine.fields' }),
  })
  State: string;

  @Field({
    type: 'int',
    notNull: true,
    default: () => 10,
    index: true,
    string: _lt('Sequence', { scope: 'sandbox.model.SandboxLine.fields' }),
  })
  Sequence: number;

  /**
   * Keep Quantity in 1…999 for kitchen-sink onchange demos.
   */
  @Onchange<SandboxLine>('Quantity')
  OnchangeQuantity() {
    const raw = Number(this.Quantity);
    const qty = Number.isFinite(raw) ? Math.trunc(raw) : 1;
    if (qty < 1) {
      this.Quantity = 1;
      return {
        messages: [
          {
            level: 'warn' as const,
            // Plain string: onchange engine only keeps string messages.
            message: 'Quantity must be at least 1; reset to 1.',
          },
        ],
      };
    }
    if (qty > 999) {
      this.Quantity = 999;
      return {
        messages: [
          {
            level: 'warn' as const,
            message: 'Quantity capped at 999.',
          },
        ],
      };
    }
    this.Quantity = qty;
  }
}
