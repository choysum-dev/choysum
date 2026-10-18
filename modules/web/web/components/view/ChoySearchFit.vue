<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  Wide action bars keep the field in the center column. Narrow bars show an
  icon there; a click opens or closes the field on the next row.
-->
<template>
  <div ref="rootRef" class="flex min-w-0 w-full items-center" :class="wide ? undefined : 'justify-end'">
    <button
      v-show="!wide"
      type="button"
      class="relative inline-flex size-control shrink-0 items-center justify-center rounded-md border-0 text-foreground outline-none hover:bg-accent hover:text-accent-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
      :class="opened ? 'bg-accent text-accent-foreground' : undefined"
      data-testid="choy-search-fit-toggle"
      :data-search-active="active ? 'true' : 'false'"
      :aria-expanded="opened ? 'true' : 'false'"
      :aria-label="label"
      @click="toggleField"
    >
      <SearchIcon class="size-4" aria-hidden="true" />
      <span
        v-if="active"
        class="pointer-events-none absolute end-1 top-1 size-1.5 rounded-full bg-primary"
        data-testid="choy-search-fit-dot"
      />
    </button>
    <Teleport :to="teleportTarget" :disabled="wide || !teleportTarget">
      <div v-show="wide || opened" class="min-w-0 w-full" :class="wide ? undefined : 'pt-2'" data-testid="choy-search-fit-field">
        <slot />
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Search as SearchIcon } from 'lucide-vue-next';
import { createTranslate } from '@/web/web/i18n';
import { searchFieldFits } from '@/web/web/components/view/searchFit';

defineOptions({ name: 'ChoySearchFit' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ChoySearchFit' });

const rootRef = ref<HTMLElement | null>(null);
const rowEl = ref<HTMLElement | null>(null);
const wide = ref(true);
const opened = ref(false);
const active = ref(false);
let disconnect: (() => void) | null = null;
let mo: { disconnect: () => void } | null = null;

const label = computed(() => (opened.value ? _t('Hide search') : _t('Show search')));
/** Teleport host; cast through unknown — QuickJS DOM typings diverge from lib.dom Element. */
const teleportTarget = computed(() => (rowEl.value ? (rowEl.value as unknown as Element) : null));

function syncActive() {
  const node =
    rowEl.value?.querySelector('[data-anchor="choy.search"]') ||
    rootRef.value?.querySelector('[data-anchor="choy.search"]');
  active.value = node?.getAttribute('data-search-active') === 'true';
}

function applyWidth(width: number) {
  if (width <= 0) return;
  const next = searchFieldFits(width);
  if (wide.value && !next) opened.value = false;
  wide.value = next;
}

function barWidth(): number {
  const root = rootRef.value;
  const bar = root?.closest('[data-slot="view-action-bar"]') as HTMLElement | null;
  return (bar || root)?.clientWidth ?? 0;
}

function focusInput() {
  const input = (rowEl.value?.querySelector('input') ||
    rootRef.value?.querySelector('input') ||
    rowEl.value?.querySelector('textarea') ||
    rootRef.value?.querySelector('textarea')) as HTMLElement | null;
  if (!input) return;
  // Focus the field without opening its filter menu; a later user focus still opens it.
  input.setAttribute('data-search-focus-quiet', '');
  input.focus();
  queueMicrotask(() => input.removeAttribute('data-search-focus-quiet'));
}

function toggleField() {
  opened.value = !opened.value;
  if (opened.value) nextTick(() => focusInput());
}

watch(wide, () => {
  void nextTick(() => syncActive());
});

onMounted(() => {
  const root = rootRef.value;
  const bar = root?.closest('[data-slot="view-action-bar"]') as HTMLElement | null;
  const parent = bar?.parentElement;
  if (bar && parent) {
    const row = document.createElement('div');
    row.setAttribute('data-slot', 'search-fit-row');
    row.className = 'min-w-0';
    parent.insertBefore(row, bar.nextSibling);
    rowEl.value = row;
  }
  applyWidth(barWidth());
  syncActive();
  const Observer = (globalThis as { ResizeObserver?: typeof ResizeObserver }).ResizeObserver;
  const observed = (bar || root) as unknown as Element | null;
  if (typeof Observer === 'function' && observed) {
    const observer = new Observer((entries) => {
      applyWidth(entries[0]?.contentRect?.width ?? 0);
    });
    observer.observe(observed);
    disconnect = () => observer.disconnect();
  }
  const MutationObs = (globalThis as { MutationObserver?: typeof MutationObserver }).MutationObserver;
  if (typeof MutationObs === 'function') {
    const observer = new MutationObs(() => syncActive());
    const opts: MutationObserverInit = {
      attributes: true,
      subtree: true,
      childList: true,
      attributeFilter: ['data-search-active'],
    };
    if (root) observer.observe(root as unknown as Node, opts);
    if (rowEl.value) observer.observe(rowEl.value as unknown as Node, opts);
    mo = observer;
  }
});

onBeforeUnmount(() => {
  disconnect?.();
  mo?.disconnect();
  rowEl.value?.remove();
  rowEl.value = null;
});
</script>
