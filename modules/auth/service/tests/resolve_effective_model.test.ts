// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';
import {
  resolveEffectiveApplicationId,
  resolveEffectiveModelId,
  resolveEffectiveModelRow,
} from '../models/_resolve_effective_model';

const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');

test('resolveEffectiveModelRow prefers empty ModuleId over newer shell', async () => {
  const orig = (MetaModel as any).Search;
  try {
    (MetaModel as any).Search = async () => [
      {
        Id: 'shell',
        ModuleId: 'mod-1',
        UpdatedAt: '2026-08-05T12:00:00.000Z',
        Name: 'Partner',
      },
      {
        Id: 'eff',
        ModuleId: null,
        UpdatedAt: '2026-08-05T10:00:00.000Z',
        Name: 'Partner',
      },
    ];
    const row = await resolveEffectiveModelRow('partner', 'Partner', ['Name']);
    expect(row?.Id).toBe('eff');
    expect(await resolveEffectiveModelId('partner', 'Partner')).toBe('eff');
  } finally {
    (MetaModel as any).Search = orig;
  }
});

test('resolveEffectiveModelRow ModuleId object / whitespace / UpdatedAt / Id tie-break', async () => {
  const orig = (MetaModel as any).Search;
  try {
    // Object ModuleId with empty Id counts as empty.
    (MetaModel as any).Search = async () => [
      { Id: 'shell', ModuleId: { Id: 'm1' }, UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { Id: 'eff-obj', ModuleId: { Id: '' }, UpdatedAt: '2026-08-05T09:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('eff-obj');

    // Whitespace ModuleId string counts as empty.
    (MetaModel as any).Search = async () => [
      { Id: 'shell', module_id: 'mod', updated_at: '2026-08-05T12:00:00.000Z' },
      { Id: 'ws', module_id: '  ', updated_at: '2026-08-05T08:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('ws');

    // Both empty ModuleId → prefer newer UpdatedAt.
    (MetaModel as any).Search = async () => [
      { Id: 'old', ModuleId: null, UpdatedAt: '2026-08-05T08:00:00.000Z' },
      { Id: 'new', ModuleId: '', UpdatedAt: '2026-08-05T12:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('new');

    // Equal UpdatedAt → prefer larger Id.
    (MetaModel as any).Search = async () => [
      { Id: 'aaa', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { Id: 'zzz', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('zzz');

    // Number UpdatedAt + invalid date parse on empty-ModuleId rows (hits Number.isFinite false).
    (MetaModel as any).Search = async () => [
      { Id: 'n1', ModuleId: null, UpdatedAt: 100 },
      { Id: 'n2', ModuleId: null, UpdatedAt: 200 },
      { Id: 'bad', ModuleId: null, UpdatedAt: 'not-a-date' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('n2');

    // Single row short-circuit; rows without Id filtered out.
    (MetaModel as any).Search = async () => [{ Id: 'solo', ModuleId: 'm', UpdatedAt: null }];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('solo');

    (MetaModel as any).Search = async () => [{ Name: 'no-id' }, null];
    expect(await resolveEffectiveModelRow('a', 'M')).toBeUndefined();
    expect(await resolveEffectiveModelId('a', 'M')).toBe('');

    (MetaModel as any).Search = async () => null;
    expect(await resolveEffectiveModelRow('a', 'M')).toBeUndefined();
  } finally {
    (MetaModel as any).Search = orig;
  }
});

test('resolveEffectiveModelRow covers remaining ModuleId/Id/UpdatedAt branches', async () => {
  const orig = (MetaModel as any).Search;
  try {
    // ModuleID key + object ModuleId.id (lowercase) empty.
    (MetaModel as any).Search = async () => [
      { Id: 'shell', ModuleID: 'mod-x', UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { Id: 'eff-id-key', ModuleId: { id: null }, UpdatedAt: '2026-08-05T09:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('eff-id-key');

    // Object ModuleId with lowercase id whitespace-only counts as empty; row id via lowercase `id`.
    (MetaModel as any).Search = async () => [
      { Id: 'shell', ModuleId: { Id: 'm' }, UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { id: 'eff-ws-obj', ModuleId: { id: '  ' }, UpdatedAt: '2026-08-05T08:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.id).toBe('eff-ws-obj');

    // Empty UpdatedAt string → 0; older candidate must not beat newer best.
    (MetaModel as any).Search = async () => [
      { Id: 'newer', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { Id: 'older', ModuleId: null, UpdatedAt: '' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('newer');

    // UpdatedAt null with no updated_at → raw == null path in rowUpdatedAt.
    (MetaModel as any).Search = async () => [
      { Id: 'keep', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { Id: 'null-ts', ModuleId: null, UpdatedAt: null },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('keep');

    // UpdatedAt null falls through to updated_at.
    (MetaModel as any).Search = async () => [
      { Id: 'via-snake', ModuleId: null, UpdatedAt: null, updated_at: '2026-08-05T13:00:00.000Z' },
      { Id: 'older-iso', ModuleId: null, UpdatedAt: '2026-08-05T10:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('via-snake');

    // Equal UpdatedAt but smaller Id must not replace best.
    (MetaModel as any).Search = async () => [
      { Id: 'zzz', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' },
      { Id: 'aaa', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('zzz');

    // Shell after effective already selected must not steal (empty=false, bestEmpty=true).
    (MetaModel as any).Search = async () => [
      { Id: 'eff-first', ModuleId: null, UpdatedAt: '2026-08-05T08:00:00.000Z' },
      { Id: 'shell-later', ModuleId: 'mod', UpdatedAt: '2026-08-05T12:00:00.000Z' },
    ];
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('eff-first');

    // Default fields argument (omit third param).
    (MetaModel as any).Search = async (_d: any, opts: any) => {
      expect(opts.fields.includes('CompanyField')).toBe(true);
      return [{ Id: 'solo-default', ModuleId: null, UpdatedAt: 1 }];
    };
    expect((await resolveEffectiveModelRow('a', 'M'))?.Id).toBe('solo-default');
  } finally {
    (MetaModel as any).Search = orig;
  }
});

test('resolveEffectiveApplicationId returns tip Id or empty', async () => {
  const orig = (MetaApplication as any).Search;
  try {
    (MetaApplication as any).Search = async () => [{ Id: 'app-1' }];
    expect(await resolveEffectiveApplicationId('auth')).toBe('app-1');

    (MetaApplication as any).Search = async () => [];
    expect(await resolveEffectiveApplicationId('missing')).toBe('');

    (MetaApplication as any).Search = async () => null;
    expect(await resolveEffectiveApplicationId('missing')).toBe('');
  } finally {
    (MetaApplication as any).Search = orig;
  }
});
