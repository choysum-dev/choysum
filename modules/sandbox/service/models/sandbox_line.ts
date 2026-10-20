// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
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
    relation: { targetModel: () => SandboxRecord },
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
}
