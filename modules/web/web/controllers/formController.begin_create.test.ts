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

  test('undefined initial uses empty seed object', async () => {
    const DefaultGet = vi.fn(async () => ({ Name: 'server-only' }));
    const controller = createFormController(newStore(DefaultGet));

    await controller.beginCreate();

    expect(DefaultGet.mock.calls[0]?.[0]).toEqual({});
    expect(controller.vm.draft).toEqual({ Name: 'server-only' });
  });

  test('non-object DefaultGet results collapse to empty server map', async () => {
    for (const bad of [null, 42, 'x', ['Name']]) {
      const DefaultGet = vi.fn(async () => bad as any);
      const controller = createFormController(newStore(DefaultGet));
      await controller.beginCreate({ Name: 'seed' });
      expect(controller.vm.draft).toEqual({ Name: 'seed' });
    }
  });

  test('undefined seed keys are dropped by clone so server defaults remain', async () => {
    const DefaultGet = vi.fn(async () => ({ Name: 'server-name', Code: 'server-code' }));
    const controller = createFormController(newStore(DefaultGet));

    await controller.beginCreate({ Name: undefined, Code: 'seed-code' });

    // JSON clone strips undefined keys from the seed before merge.
    expect(controller.vm.draft).toEqual({
      Name: 'server-name',
      Code: 'seed-code',
    });
  });

  test('null initial uses empty seed object', async () => {
    const DefaultGet = vi.fn(async () => ({ Name: 'server-only' }));
    const controller = createFormController(newStore(DefaultGet));
    await controller.beginCreate(null);
    expect(controller.vm.draft).toEqual({ Name: 'server-only' });
  });

  test('superseded DefaultGet success does not clobber newer create draft', async () => {
    const resolvers: Array<(value: unknown) => void> = [];
    const DefaultGet = vi.fn(
      () =>
        new Promise(resolve => {
          resolvers.push(resolve);
        })
    );
    const controller = createFormController(newStore(DefaultGet));

    const first = controller.beginCreate({ Name: 'first' });
    await Promise.resolve();
    const second = controller.beginCreate({ Name: 'second' });
    await Promise.resolve();

    expect(resolvers.length).toBe(2);
    resolvers[0]!({ Name: 'late-first', Code: 'from-first' });
    await first;
    expect(controller.vm.draft).toEqual({ Name: 'second' });

    resolvers[1]!({ Name: 'from-second', Code: 'server-code' });
    await second;
    expect(controller.vm.draft).toEqual({
      Name: 'second',
      Code: 'server-code',
    });
  });

  test('beginCreate clears loading when it supersedes beginDisplay', async () => {
    let resolveDefaults: ((value: unknown) => void) | undefined;
    const DefaultGet = vi.fn(
      () =>
        new Promise(resolve => {
          resolveDefaults = resolve;
        })
    );
    const controller = createFormController(newStore(DefaultGet));
    controller.vm.loading = true;

    const pending = controller.beginCreate({ Name: 'seed' });
    await Promise.resolve();

    expect(controller.vm.loading).toBe(false);

    resolveDefaults!({});
    await pending;
    expect(controller.vm.loading).toBe(false);
  });

  test('superseded DefaultGet failure does not reset newer create draft', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const rejecters: Array<(reason?: unknown) => void> = [];
    const DefaultGet = vi.fn(
      () =>
        new Promise((_resolve, reject) => {
          rejecters.push(reject);
        })
    );
    const controller = createFormController(newStore(DefaultGet));

    const first = controller.beginCreate({ Name: 'first' });
    await Promise.resolve();
    const second = controller.beginCreate({ Name: 'second' });
    await Promise.resolve();

    rejecters[0]!(new Error('stale'));
    await first;
    expect(controller.vm.draft).toEqual({ Name: 'second' });
    expect(warn).not.toHaveBeenCalled();

    rejecters[1]!(new Error('current'));
    await second;
    expect(controller.vm.draft).toEqual({ Name: 'second' });
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });
});
