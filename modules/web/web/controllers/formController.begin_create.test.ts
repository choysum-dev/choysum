// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, test, vi } from 'vitest';

vi.mock('@/web/web/query/context', () => ({
  buildBrowseContext: vi.fn(() => ({ kind: 'browse_ctx' })),
}));

vi.mock('@/web/web/query/planner', () => ({
  buildPlan: vi.fn(() => ({ kind: 'browse_plan' })),
}));

vi.mock('@/web/web/query/executor', () => ({
  execute: vi.fn(async () => ({ kind: 'search', rows: [], total: 0, ts: Date.now() })),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => ({})),
}));

vi.mock('@/web/web/query/utils/handoff', () => ({
  handoffCache: { set: vi.fn() },
  flashRead: vi.fn(() => undefined),
}));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection: vi.fn(async () => {}),
}));

import { createFormController } from './formController';

function newStore(defaultGet: ReturnType<typeof vi.fn>) {
  return {
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget',
    fieldsMetadata: {
      Name: { type: 'varchar' },
      Code: { type: 'varchar' },
    },
    state: {},
    getContext: () => ({}),
    DefaultGet: defaultGet,
  } as any;
}

describe('formController beginCreate DefaultGet prefetch (FD-4)', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  test('prefetch success merges server defaults while keeping seed keys', async () => {
    const DefaultGet = vi.fn(async (seed: any) => ({
      Name: 'from-server',
      Code: 'server-code',
      ...(seed || {}),
    }));
    const controller = createFormController(newStore(DefaultGet));

    await controller.beginCreate({ Name: 'seed-name' });

    expect(controller.vm.mode).toBe('create');
    expect(controller.vm.original).toBeNull();
    expect(DefaultGet).toHaveBeenCalledTimes(1);
    expect(DefaultGet.mock.calls[0]?.[0]).toEqual({ Name: 'seed-name' });
    expect(controller.vm.draft).toEqual({
      Name: 'seed-name',
      Code: 'server-code',
    });
  });

  test('DefaultGet failure keeps seed draft and create mode', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const DefaultGet = vi.fn(async () => {
      throw new Error('rpc down');
    });
    const controller = createFormController(newStore(DefaultGet));

    await expect(controller.beginCreate({ Name: 'seed-only' })).resolves.toBeUndefined();

    expect(controller.vm.mode).toBe('create');
    expect(controller.vm.draft).toEqual({ Name: 'seed-only' });
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  test('copy→create seed wins over overlapping server defaults', async () => {
    const DefaultGet = vi.fn(async () => ({
      Name: 'server-name',
      Code: 'server-code',
      Note: 'server-note',
    }));
    const controller = createFormController(newStore(DefaultGet));

    // Simulate copy: seed carries prior field values without Id.
    await controller.beginCreate({ Name: 'copied-name', Code: 'copied-code' });

    expect(controller.vm.mode).toBe('create');
    expect(controller.vm.draft).toEqual({
      Name: 'copied-name',
      Code: 'copied-code',
      Note: 'server-note',
    });
  });

  test('explicit null in seed is preserved over server default', async () => {
    const DefaultGet = vi.fn(async () => ({
      Name: 'server-name',
      Code: 'server-code',
    }));
    const controller = createFormController(newStore(DefaultGet));

    await controller.beginCreate({ Name: null });

    expect(controller.vm.draft).toEqual({
      Name: null,
      Code: 'server-code',
    });
  });

  test('missing DefaultGet on store still opens create with seed', async () => {
    const store = newStore(vi.fn());
    delete (store as any).DefaultGet;
    const controller = createFormController(store);

    await controller.beginCreate({ Name: 'local' });

    expect(controller.vm.mode).toBe('create');
    expect(controller.vm.draft).toEqual({ Name: 'local' });
  });
});
