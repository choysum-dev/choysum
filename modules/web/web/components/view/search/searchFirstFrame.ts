// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  computed,
  getCurrentScope,
  onScopeDispose,
  provide,
  ref,
  toValue,
  watch,
  type ComputedRef,
  type InjectionKey,
  type MaybeRefOrGetter,
  type Ref,
} from 'vue';

/**
 * Search-owned first-frame protocol for collection views (List / Kanban / Chart).
 *
 * Why a storeId registry (not only provide/inject): product pages often put
 * ChoySearchView in a parent-owned `#search` slot. That vnode's inject parent is
 * the page, not ChoyKanbanView/ChoyChartView — so a provide on the collection
 * view never reaches Search. Host + Reporter both join the same gate by storeId.
 *
 * Contract:
 * - Host calls useSearchFirstFrameHost when it may mount Search that initialEmits.
 * - Reporter (ChoySearchView) calls useSearchFirstFrameReporter and markEmitted
 *   on the single mount-time query-update (or markNone when initialEmit is false).
 * - Bootstrap uses gate.shouldDefer instead of inventing per-view defer rules.
 * - Multiple hosts share one gate; expectFirstFrame is vote-counted so a host
 *   without Search cannot clear another host's pending first-frame wait.
 */

export type SearchFirstFramePhase = 'none' | 'pending' | 'emitted';

export type SearchFirstFrameGate = {
  readonly phase: Ref<SearchFirstFramePhase>;
  /** Skip empty bootstrap apply; Search first-frame owns the first load. */
  readonly shouldDefer: ComputedRef<boolean>;
  markEmitted: () => void;
  markNone: () => void;
  /** Host updates whether a first-frame emit is expected. */
  setExpectFirstFrame: (expect: boolean) => void;
};

export const SearchFirstFrameKey: InjectionKey<SearchFirstFrameGate> = Symbol.for(
  'choysum.searchFirstFrame',
);

type RegistryEntry = {
  gate: SearchFirstFrameGate;
  hosts: number;
  reporters: number;
  /** Hosts currently voting expectFirstFrame=true. */
  expectCount: number;
};

const registry = new Map<string, RegistryEntry>();

/** Stable keys for stores without storeId so host + reporter still join. */
let fallbackKeys = new WeakMap<object, string>();
let fallbackSeq = 0;

function createGate(expectFirstFrame: boolean): SearchFirstFrameGate {
  const phase = ref<SearchFirstFramePhase>(expectFirstFrame ? 'pending' : 'none');
  return {
    phase,
    shouldDefer: computed(() => phase.value === 'pending' || phase.value === 'emitted'),
    markEmitted() {
      if (phase.value === 'pending') phase.value = 'emitted';
    },
    markNone() {
      phase.value = 'none';
    },
    setExpectFirstFrame(expect: boolean) {
      if (expect) {
        if (phase.value === 'none') phase.value = 'pending';
        return;
      }
      if (phase.value === 'pending') phase.value = 'none';
    },
  };
}

function storeKey(store: { storeId?: string } | null | undefined): string {
  const id = String(store?.storeId ?? '').trim();
  if (id) return id;
  if (store && typeof store === 'object') {
    let k = fallbackKeys.get(store);
    if (!k) {
      k = `__anon_${++fallbackSeq}`;
      fallbackKeys.set(store, k);
    }
    return k;
  }
  return '';
}

function acquire(
  key: string,
  role: 'host' | 'reporter',
  expectFirstFrame: boolean,
): SearchFirstFrameGate {
  let entry = registry.get(key);
  if (!entry) {
    entry = {
      gate: createGate(expectFirstFrame),
      hosts: 0,
      reporters: 0,
      expectCount: role === 'host' && expectFirstFrame ? 1 : 0,
    };
    registry.set(key, entry);
  } else if (role === 'host') {
    // Only raise the shared expectation. A non-expecting host must not clear
    // pending while another host still needs Search's first frame.
    if (expectFirstFrame) {
      entry.expectCount += 1;
      entry.gate.setExpectFirstFrame(true);
    }
  }
  if (role === 'host') entry.hosts += 1;
  else entry.reporters += 1;
  return entry.gate;
}

function updateHostExpectVote(key: string, from: boolean, to: boolean): void {
  if (from === to) return;
  const entry = registry.get(key);
  if (!entry) return;
  if (to) {
    entry.expectCount += 1;
    entry.gate.setExpectFirstFrame(true);
    return;
  }
  entry.expectCount = Math.max(0, entry.expectCount - 1);
  if (entry.expectCount === 0) {
    entry.gate.setExpectFirstFrame(false);
  }
}

function release(key: string, role: 'host' | 'reporter', hostWasExpecting?: boolean): void {
  const entry = registry.get(key);
  if (!entry) return;
  if (role === 'host') {
    entry.hosts = Math.max(0, entry.hosts - 1);
    if (hostWasExpecting) {
      entry.expectCount = Math.max(0, entry.expectCount - 1);
      if (entry.expectCount === 0 && entry.gate.phase.value === 'pending') {
        entry.gate.setExpectFirstFrame(false);
      }
    }
  } else {
    entry.reporters = Math.max(0, entry.reporters - 1);
  }
  if (entry.hosts === 0 && entry.reporters === 0) {
    registry.delete(key);
  }
}

/**
 * Collection view host: join (and provide) the first-frame gate for this store.
 */
export function useSearchFirstFrameHost(
  store: { storeId?: string } | null | undefined,
  opts: { expectFirstFrame: MaybeRefOrGetter<boolean> },
): SearchFirstFrameGate {
  const key = storeKey(store);
  const expectNow = () => !!toValue(opts.expectFirstFrame);
  // Only join the registry when a scope can release the entry (avoids leaks).
  const hasScope = !!getCurrentScope();
  // Ref so dispose always sees the latest vote (after watch flips), never a
  // stale initial true that would double-decrement expectCount.
  const expecting = ref(expectNow());
  const gate = key && hasScope ? acquire(key, 'host', expecting.value) : createGate(expecting.value);

  if (key && hasScope) {
    onScopeDispose(() => release(key, 'host', expecting.value));
  }

  if (hasScope) {
    watch(
      () => !!toValue(opts.expectFirstFrame),
      (v) => {
        if (key) {
          updateHostExpectVote(key, expecting.value, v);
        } else {
          gate.setExpectFirstFrame(v);
        }
        expecting.value = v;
      },
    );
  }

  provide(SearchFirstFrameKey, gate);
  return gate;
}

/**
 * ChoySearchView: attach to the host gate (by storeId) and report first-frame emit.
 */
export function useSearchFirstFrameReporter(
  store: { storeId?: string } | null | undefined,
  opts: { initialEmit: MaybeRefOrGetter<boolean> },
): { markEmitted: () => void; markNone: () => void } {
  const key = storeKey(store);
  const initialEmit = () => !!toValue(opts.initialEmit);
  const hasScope = !!getCurrentScope();
  const gate = key && hasScope ? acquire(key, 'reporter', initialEmit()) : createGate(initialEmit());

  if (key && hasScope) {
    onScopeDispose(() => release(key, 'reporter'));
  }

  if (!initialEmit()) {
    gate.markNone();
  } else {
    gate.setExpectFirstFrame(true);
  }

  return {
    markEmitted: () => gate.markEmitted(),
    markNone: () => gate.markNone(),
  };
}

/** Test / diagnostics: clear the registry between cases. */
export function resetSearchFirstFrameRegistryForTests(): void {
  registry.clear();
  fallbackKeys = new WeakMap<object, string>();
  fallbackSeq = 0;
}

/** Test helper: peek gate without acquiring. */
export function peekSearchFirstFrameGateForTests(storeId: string): SearchFirstFrameGate | null {
  return registry.get(String(storeId || '').trim())?.gate ?? null;
}
