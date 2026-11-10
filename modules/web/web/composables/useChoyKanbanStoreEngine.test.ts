// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h } from 'vue';
import { useChoyKanbanStoreEngine } from './useChoyKanbanStoreEngine';

function store() {
  return {
    storeId: 'demo.Task',
    fieldsMetadata: {},
    state: {
      queryState: { pagination: { limit: 20, offset: 0 }, appliedGroups: ['Stage'] },
      planCache: new Map(),
    },
    Search: async () => [],
    Count: async () => 0,
    ReadGroup: async () => ({ groups: [] }),
  } as any;
}

function groupResult(keys: Array<{ key: string; label: string }>) {
  return {
    kind: 'group',
    rows: keys.map((lane) => ({
      kind: 'group',
      key: lane.key,
      depth: 0,
      label: lane.label,
      count: 1,
      __condition: { Stage: lane.key },
    })),
    visibleNodes: [],
    expandedKeys: new Set<string>(),
  };
}

test('store engine orders lanes and keeps the board idle while a lane preload fails', async () => {
  const Harness = defineComponent({
    name: 'KanbanEngineHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: store(),
        autoBootstrap: false,
        laneOrder: ['done', 'todo'],
        flatLaneLabel: 'Everything',
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  engine.controller.vm.result = groupResult([
    { key: 'todo', label: 'Todo' },
    { key: 'done', label: 'Done' },
  ]);
  const originalPreload = engine.controller.preloadLane.bind(engine.controller);
  engine.controller.preloadLane = async () => {
    throw new Error('lane preload failed');
  };
  const logged: string[] = [];
  const originalError = console.error;
  console.error = (...args: unknown[]) => {
    logged.push(args.map((item) => String(item)).join(' '));
  };
  try {
    await engine.syncLanesFromController();
  } finally {
    console.error = originalError;
    engine.controller.preloadLane = originalPreload;
  }
  expect(logged.some((line) => line.includes('lane preload failed'))).toBe(true);
  await engine.syncLanesFromController();
  expect(engine.lanes.map((lane: { key: string }) => lane.key)).toEqual(['done', 'todo']);
  expect(engine.boardBusy).toBe(false);

  const TokenHarness = defineComponent({
    name: 'KanbanEngineTokenHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: store(),
        autoBootstrap: false,
        laneOrder: ['Done', 'Todo'],
        laneOrderToken: (lane) => String(lane.label || ''),
      });
    },
    render() {
      return h('div');
    },
  });
  const tokenMounted = mountApp(TokenHarness as any);
  await flushPromises();
  tokenMounted.root.controller.vm.result = groupResult([
    { key: 'todo', label: 'Todo' },
    { key: 'done', label: 'Done' },
  ]);
  await tokenMounted.root.syncLanesFromController();
  expect(tokenMounted.root.lanes.map((lane: { label: string }) => lane.label)).toEqual(['Done', 'Todo']);
  tokenMounted.unmount();

  engine.controller.vm.result = { kind: 'search', rows: [{ Id: '1', Title: 'Flat' }] };
  await engine.syncLanesFromController();
  expect(engine.lanes[0].label).toBe('Everything');
  expect(engine.lanes[0].cards.length).toBe(1);

  mounted.unmount();
});

test('load-more patches the target lane without rematerializing siblings', async () => {
  const Harness = defineComponent({
    name: 'KanbanEngineLoadMoreHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: store(),
        autoBootstrap: false,
        mapRowToCard: (row: any, _index, laneKey) => ({
          id: String(row.payload?.Id ?? row.key ?? ''),
          title: String(row.payload?.Title ?? row.key ?? ''),
          laneKey,
        }),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  engine.controller.vm.result = {
    ...groupResult([
      { key: 'active', label: 'Active' },
      { key: 'revoked', label: 'Revoked' },
    ]),
    rows: [
      {
        kind: 'group',
        key: 'active',
        depth: 0,
        label: 'Active',
        count: 1,
        __condition: { Stage: 'active' },
      },
      {
        kind: 'group',
        key: 'revoked',
        depth: 0,
        label: 'Revoked',
        count: 3,
        __condition: { Stage: 'revoked' },
      },
    ],
    groupRecords: new Map([
      [
        'active',
        [{ kind: 'record', key: 'a1', payload: { Id: 'a1', Title: 'Active 1' } }],
      ],
      [
        'revoked',
        [{ kind: 'record', key: 'r1', payload: { Id: 'r1', Title: 'Revoked 1' } }],
      ],
    ]),
  };
  engine.controller.laneRecords.value = {
    active: [{ kind: 'record', key: 'a1', payload: { Id: 'a1', Title: 'Active 1' } }],
    revoked: [{ kind: 'record', key: 'r1', payload: { Id: 'r1', Title: 'Revoked 1' } }],
  };
  engine.controller.preloadLane = async () => undefined;
  await engine.syncLanesFromController();

  const siblingBefore = engine.lanes.find((lane: { key: string }) => lane.key === 'active');
  const targetBefore = engine.lanes.find((lane: { key: string }) => lane.key === 'revoked');
  expect(siblingBefore?.cards.length).toBe(1);
  expect(targetBefore?.cards.length).toBe(1);

  engine.controller.loadMoreLane = async (key: string) => {
    const next = [
      ...(engine.controller.laneRecords.value[key] || []),
      { kind: 'record', key: 'r2', payload: { Id: 'r2', Title: 'Revoked 2' } },
    ];
    engine.controller.laneRecords.value = {
      ...engine.controller.laneRecords.value,
      [key]: next,
    };
  };

  await engine.onLaneLoadMore({ laneKey: 'revoked' });
  await flushPromises();

  const siblingAfter = engine.lanes.find((lane: { key: string }) => lane.key === 'active');
  const targetAfter = engine.lanes.find((lane: { key: string }) => lane.key === 'revoked');
  expect(siblingAfter).toBe(siblingBefore);
  expect(targetAfter).not.toBe(targetBefore);
  expect(targetAfter?.cards.map((card: { id: string }) => card.id)).toEqual(['r1', 'r2']);
  expect(engine.loadMorePending).toBe(false);
  expect(engine.loadMoreLaneKey).toBeNull();
  expect(engine.boardBusy).toBe(false);

  mounted.unmount();
});

test('deferFirstFrame bootstrap skips empty apply; refresh re-applies after latch', async () => {
  let searchCalls = 0;
  const s = store();
  s.Search = async () => {
    searchCalls += 1;
    return [];
  };
  s.fieldsMetadata = { Title: { type: 'varchar' } };

  const Harness = defineComponent({
    name: 'KanbanEngineDeferHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: true,
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  await engine.bootstrap();
  await flushPromises();
  expect(searchCalls).toBe(0);

  // While Search still owns the first frame, refresh must not synthesize empty.
  await engine.refresh();
  await flushPromises();
  expect(searchCalls).toBe(0);

  await engine.applySearch({ keyword: '', appliedFilters: [], appliedGroups: [] });
  await flushPromises();
  expect(searchCalls).toBeGreaterThanOrEqual(1);
  const afterFirst = searchCalls;

  await engine.bootstrap();
  await flushPromises();
  expect(searchCalls).toBe(afterFirst);

  await engine.refresh();
  await flushPromises();
  expect(searchCalls).toBeGreaterThan(afterFirst);

  mounted.unmount();
});

test('refresh reapplies store queryState when not deferring and no search landed', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  s.state.queryState = { keyword: 'from-store', appliedFilters: [], appliedGroups: [] };

  const Harness = defineComponent({
    name: 'KanbanEngineRefreshQueryStateHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: false,
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  let applies = 0;
  let lastKeyword: string | undefined;
  engine.controller.apply = (async (payload: any) => {
    applies += 1;
    lastKeyword = payload?.keyword;
  }) as any;
  engine.controller.setKeywordFields = async () => undefined;

  // First refresh bootstraps (empty apply) and must not double-apply.
  await engine.refresh();
  await flushPromises();
  expect(applies).toBe(1);

  // Already bootstrapped with no lastSearchQuery: re-apply store queryState.
  await engine.refresh();
  await flushPromises();
  expect(applies).toBe(2);
  expect(lastKeyword).toBe('from-store');

  mounted.unmount();
});

test('refresh re-runs search that landed during first bootstrap', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  let engine: ReturnType<typeof useChoyKanbanStoreEngine> | null = null;
  let applies = 0;
  const keywords: string[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineRefreshDuringBootstrapHarness',
    setup() {
      engine = useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: true,
        beforeBootstrap: async () => {
          await engine!.applySearch({
            keyword: 'early',
            filters: [],
            appliedFilters: [],
            appliedGroups: [],
          });
        },
      });
      return engine;
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  engine!.controller.apply = (async (payload: any) => {
    applies += 1;
    keywords.push(String(payload?.keyword ?? ''));
  }) as any;
  engine!.controller.setKeywordFields = async () => undefined;

  await engine!.refresh();
  await flushPromises();
  // beforeBootstrap applySearch + post-bootstrap refresh re-run.
  expect(applies).toBe(2);
  expect(keywords).toEqual(['early', 'early']);

  mounted.unmount();
});

test('refreshFromQueryState rethrows apply failures', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  const errors: unknown[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineQueryStateErrHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: false,
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  let applies = 0;
  engine.controller.apply = (async () => {
    applies += 1;
    if (applies > 1) throw new Error('queryState boom');
  }) as any;
  engine.controller.setKeywordFields = async () => undefined;

  await engine.bootstrap();
  await flushPromises();

  let threw: unknown = null;
  await engine.refresh().then(
    () => undefined,
    (e: unknown) => {
      threw = e;
    },
  );
  expect(String(threw)).toContain('queryState boom');
  expect(errors.some((e) => String(e).includes('queryState boom'))).toBe(true);

  mounted.unmount();
});

test('refreshFromQueryState reports lane-sync failures via onSearchError', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  s.state.queryState = { pagination: { limit: 20, offset: 0 }, appliedGroups: [] };
  const errors: unknown[] = [];
  let explode = false;

  const Harness = defineComponent({
    name: 'KanbanEngineQueryStateSyncErrHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: false,
        flatLaneLabel: () => {
          if (explode) throw new Error('queryState sync boom');
          return 'All';
        },
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;
  engine.controller.setKeywordFields = async () => undefined;
  await engine.bootstrap();
  await flushPromises();

  // No-op apply so the lanes watcher does not race a second sync.
  engine.controller.apply = async () => undefined;
  explode = true;
  await engine.refresh();
  await flushPromises();
  expect(errors.some((e) => String(e).includes('queryState sync boom'))).toBe(true);

  mounted.unmount();
});

test('refreshFromQueryState recovers when a newer refresh finishes first', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };

  const Harness = defineComponent({
    name: 'KanbanEngineQueryStateRecoverHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: false,
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  let applies = 0;
  engine.controller.apply = (async () => {
    applies += 1;
  }) as any;
  engine.controller.setKeywordFields = async () => undefined;
  await engine.bootstrap();
  await flushPromises();
  const afterBootstrap = applies;

  engine.controller.apply = (async () => {
    applies += 1;
    if (applies === afterBootstrap + 1) {
      await new Promise((r) => setTimeout(r, 30));
    }
  }) as any;

  const first = engine.refresh();
  await new Promise((r) => setTimeout(r, 5));
  const second = engine.refresh();
  await Promise.all([first, second]);
  // first apply + second apply + stale recover apply
  expect(applies).toBe(afterBootstrap + 3);

  mounted.unmount();
});

test('refreshFromQueryState recover path rethrows failures', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  const errors: unknown[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineQueryStateRecoverErrHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: false,
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  let applies = 0;
  engine.controller.apply = (async () => {
    applies += 1;
  }) as any;
  engine.controller.setKeywordFields = async () => undefined;
  await engine.bootstrap();
  await flushPromises();
  const afterBootstrap = applies;

  engine.controller.apply = (async () => {
    applies += 1;
    if (applies === afterBootstrap + 1) {
      await new Promise((r) => setTimeout(r, 30));
      return;
    }
    if (applies === afterBootstrap + 2) return;
    throw new Error('recover qs boom');
  }) as any;

  const first = engine.refresh();
  await new Promise((r) => setTimeout(r, 5));
  const second = engine.refresh();

  let threw: unknown = null;
  await Promise.all([
    first.then(
      () => undefined,
      (e: unknown) => {
        threw = e;
      },
    ),
    second,
  ]);
  expect(String(threw)).toContain('recover qs boom');
  expect(errors.some((e) => String(e).includes('recover qs boom'))).toBe(true);

  mounted.unmount();
});

test('refreshFromQueryState skips sync when superseded while still in flight', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };

  const Harness = defineComponent({
    name: 'KanbanEngineQueryStateStaleHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: false,
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  let applies = 0;
  engine.controller.apply = (async () => {
    applies += 1;
  }) as any;
  engine.controller.setKeywordFields = async () => undefined;
  await engine.bootstrap();
  await flushPromises();
  const afterBootstrap = applies;

  engine.controller.apply = (async () => {
    applies += 1;
    await new Promise((r) => setTimeout(r, 25));
  }) as any;

  const first = engine.refresh();
  await new Promise((r) => setTimeout(r, 5));
  const second = engine.refresh();
  await Promise.all([first, second]);
  // Both applies run; the first is superseded while the second is still in flight
  // (no recover), so only two post-bootstrap applies.
  expect(applies).toBe(afterBootstrap + 2);

  mounted.unmount();
});

test('applySearch reports apply failures without rejecting (emit-safe)', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  const errors: unknown[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineSearchErrHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: true,
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;
  await engine.bootstrap();
  await flushPromises();

  engine.controller.apply = async () => {
    throw new Error('search boom');
  };
  await engine.applySearch({ keyword: 'x', filters: [], appliedFilters: [], appliedGroups: [] });
  expect(errors.some((e) => String(e).includes('search boom'))).toBe(true);

  mounted.unmount();
});

test('refresh rethrows apply failures after onSearchError', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  const errors: unknown[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineRefreshErrHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: true,
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;
  await engine.bootstrap();
  await flushPromises();

  let applyCalls = 0;
  engine.controller.apply = async () => {
    applyCalls += 1;
    if (applyCalls === 1) return;
    throw new Error('refresh boom');
  };
  await engine.applySearch({ keyword: 'ok', filters: [], appliedFilters: [], appliedGroups: [] });
  await flushPromises();

  let threw: unknown = null;
  await engine.refresh().then(
    () => undefined,
    (e: unknown) => {
      threw = e;
    },
  );
  expect(String(threw)).toContain('refresh boom');
  expect(errors.some((e) => String(e).includes('refresh boom'))).toBe(true);

  mounted.unmount();
});

test('stale applySearch recover reports failures without rejecting', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  const errors: unknown[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineRecoverErrHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: true,
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;
  await engine.bootstrap();
  await flushPromises();

  let calls = 0;
  engine.controller.apply = async () => {
    calls += 1;
    if (calls === 1) {
      await new Promise((r) => setTimeout(r, 30));
      return;
    }
    throw new Error('recover boom');
  };

  const first = engine.applySearch({ keyword: 'a', filters: [], appliedFilters: [], appliedGroups: [] });
  await new Promise((r) => setTimeout(r, 5));
  const second = engine.applySearch({ keyword: 'b', filters: [], appliedFilters: [], appliedGroups: [] });
  await Promise.all([first, second]);
  expect(errors.some((e) => String(e).includes('recover boom'))).toBe(true);

  mounted.unmount();
});

test('refresh recover path rethrows when the recovered apply fails', async () => {
  const s = store();
  s.fieldsMetadata = { Title: { type: 'varchar' } };
  const errors: unknown[] = [];

  const Harness = defineComponent({
    name: 'KanbanEngineRefreshRecoverHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: s,
        autoBootstrap: false,
        deferFirstFrame: true,
        onSearchError: (e) => errors.push(e),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;
  await engine.bootstrap();
  await flushPromises();

  engine.controller.apply = async () => undefined;
  await engine.applySearch({ keyword: 'seed', filters: [], appliedFilters: [], appliedGroups: [] });
  await flushPromises();

  let calls = 0;
  engine.controller.apply = async () => {
    calls += 1;
    if (calls === 1) {
      await new Promise((r) => setTimeout(r, 30));
      return;
    }
    throw new Error('recover refresh boom');
  };

  const refreshP = engine.refresh();
  await new Promise((r) => setTimeout(r, 5));
  await engine.applySearch({ keyword: 'late', filters: [], appliedFilters: [], appliedGroups: [] });

  let threw: unknown = null;
  await refreshP.then(
    () => undefined,
    (e: unknown) => {
      threw = e;
    },
  );
  expect(String(threw)).toContain('recover refresh boom');
  expect(errors.some((e) => String(e).includes('recover refresh boom'))).toBe(true);

  mounted.unmount();
});
