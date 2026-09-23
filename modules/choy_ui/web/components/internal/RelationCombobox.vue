<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, onWatcherCleanup, ref, watch } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxViewport,
} from 'reka-ui';
import { cn, type ClassValue } from '../../lib/utils';
import Button from '../vendor/ui/button/Button.vue';
import {
  findRelationOption,
  normalizeRelationQuery,
  runRelationNameSearch,
  upsertRelationOption,
  type RelationNameSearchFn,
  type RelationOption,
} from './relationComboboxHelpers';

/**
 * L3 relation typeahead (Reka Combobox + virtual list + NameSearch).
 * Model is the selected option id (string | null). Not a public Choy* export.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    clearable?: boolean;
    pageSize?: number;
    estimateSize?: number;
    search: RelationNameSearchFn;
    /** Optional seed so an initial modelValue keeps its label before search. */
    selectedOption?: RelationOption | null;
    /** Show a "Search more…" footer action (host opens a dialog in PR5). */
    searchMore?: boolean;
  }>(),
  {
    placeholder: 'Search…',
    disabled: false,
    clearable: true,
    pageSize: 20,
    estimateSize: 32,
    selectedOption: null,
    searchMore: true,
  },
);

const modelValue = defineModel<string | null>({ default: null });
const open = ref(false);
const query = ref('');
const loading = ref(false);
const searchError = ref<string | null>(null);
const options = ref<RelationOption[]>([]);
/** Survives remote pages that omit the current selection. */
const pinnedSelected = ref<RelationOption | null>(props.selectedOption ?? null);
const listParent = ref<HTMLElement | null>(null);

const emit = defineEmits<{
  'search-more': [query: string];
  'search-error': [message: string | null];
  select: [option: RelationOption | null];
}>();

function clearSearchError(): void {
  if (searchError.value === null) {
    return;
  }
  searchError.value = null;
  emit('search-error', null);
}

const selected = computed(() => {
  if (!modelValue.value) {
    return null;
  }
  return (
    findRelationOption(options.value, modelValue.value) ??
    (pinnedSelected.value?.id === modelValue.value ? pinnedSelected.value : null) ??
    (props.selectedOption?.id === modelValue.value ? props.selectedOption : null)
  );
});

const displayOptions = computed(() =>
  upsertRelationOption(options.value, selected.value),
);

const virtualizer = useVirtualizer({
  get count() {
    return displayOptions.value.length;
  },
  getScrollElement: () => listParent.value as Element | null,
  estimateSize: () => props.estimateSize,
  overscan: 6,
});

const virtualRows = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());

let searchSeq = 0;
watch(
  [() => open.value, () => query.value, () => props.search, () => props.pageSize],
  async ([isOpen, q]) => {
    if (!isOpen) {
      return;
    }
    const seq = ++searchSeq;
    loading.value = true;
    clearSearchError();
    // Debounce keystrokes so only the newest keyword reaches remote NameSearch.
    await new Promise<void>((resolve) => {
      const timer = setTimeout(resolve, 150);
      onWatcherCleanup(() => {
        clearTimeout(timer);
        // Abort this run on re-query, close, or scope teardown.
        searchSeq += 1;
        resolve();
      });
    });
    if (seq !== searchSeq || !open.value) {
      if (seq === searchSeq) {
        loading.value = false;
      }
      return;
    }
    try {
      const results = await runRelationNameSearch(props.search, q, props.pageSize);
      if (seq !== searchSeq) {
        return;
      }
      options.value = results;
      clearSearchError();
      if (modelValue.value) {
        const found = findRelationOption(results, modelValue.value);
        if (found) {
          pinnedSelected.value = found;
        }
      }
    } catch (error) {
      if (seq === searchSeq) {
        options.value = [];
        const message = error instanceof Error ? error.message : String(error);
        searchError.value = 'Search failed. Please retry.';
        emit('search-error', message || searchError.value);
      }
    } finally {
      if (seq === searchSeq) {
        loading.value = false;
      }
    }
  },
);

watch(
  () => props.selectedOption,
  (option) => {
    if (option?.id) {
      pinnedSelected.value = option;
    }
  },
);

// A different search target (e.g. a reused field pointing at another relation)
// must not keep stale options or a stale pinned label.
watch([() => props.search, () => props.pageSize], () => {
  options.value = [];
  pinnedSelected.value = props.selectedOption ?? null;
  clearSearchError();
  // While open, the search watcher already re-queries with the new target;
  // bumping searchSeq here would cancel that fresh request and leave the list empty.
  if (!open.value) {
    searchSeq += 1;
    loading.value = false;
  }
});

watch(modelValue, (id) => {
  if (!id) {
    pinnedSelected.value = null;
    emit('select', null);
    return;
  }
  const found =
    findRelationOption(options.value, id) ??
    (pinnedSelected.value?.id === id ? pinnedSelected.value : null) ??
    (props.selectedOption?.id === id ? props.selectedOption : null);
  if (!found) {
    // Id is set but its label is not loaded yet; not the same as clearing.
    return;
  }
  pinnedSelected.value = found;
  emit('select', found);
});

watch(open, (isOpen) => {
  if (!isOpen) {
    // Drop the typeahead keyword so the trigger shows the selected label again.
    query.value = '';
    searchSeq += 1;
    loading.value = false;
    // Avoid flashing prior keyword results until the next search resolves.
    options.value = [];
    clearSearchError();
  }
});

function onClear(): void {
  modelValue.value = null;
  query.value = '';
}

function onSearchMore(): void {
  emit('search-more', normalizeRelationQuery(query.value));
  open.value = false;
}
</script>

<template>
  <ComboboxRoot
    v-model="modelValue"
    v-model:open="open"
    data-anchor="choy.internal.relation-combobox"
    :disabled="disabled"
    :ignore-filter="true"
    :class="cn('choy-relation-combobox relative w-full', props.class)"
  >
    <ComboboxAnchor class="flex w-full gap-1">
      <ComboboxInput
        v-model="query"
        :disabled="disabled"
        :display-value="() => selected?.label ?? ''"
        :placeholder="selected?.label || placeholder"
        :class="
          cn(
            'flex h-9 w-full rounded-md border border-border bg-background px-3 py-1 text-sm text-foreground shadow-sm',
            'placeholder:text-foreground/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            disabled && 'opacity-50',
          )
        "
      />
      <Button
        v-if="clearable && modelValue && !disabled"
        type="button"
        variant="ghost"
        size="sm"
        @click="onClear"
      >
        Clear
      </Button>
    </ComboboxAnchor>
    <ComboboxPortal>
      <ComboboxContent
        position="popper"
        class="z-50 w-[var(--reka-combobox-trigger-width)] overflow-hidden rounded-md border border-border bg-background text-foreground shadow-md"
      >
        <ComboboxViewport>
          <div ref="listParent" class="max-h-56 overflow-auto">
            <div
              v-if="loading && !displayOptions.length"
              class="px-3 py-2 text-sm text-foreground/60"
            >
              Searching…
            </div>
            <div
              v-else-if="searchError"
              class="px-3 py-2 text-sm text-danger"
              role="alert"
            >
              {{ searchError }}
            </div>
            <ComboboxEmpty v-else-if="!displayOptions.length" class="px-3 py-2 text-sm text-foreground/60">
              No matches
            </ComboboxEmpty>
            <div
              v-else
              :style="{ height: `${totalSize}px`, position: 'relative', width: '100%' }"
            >
              <template
                v-for="virtualRow in virtualRows"
                :key="displayOptions[virtualRow.index]?.id ?? String(virtualRow.key)"
              >
                <ComboboxItem
                  v-if="displayOptions[virtualRow.index]"
                  :value="displayOptions[virtualRow.index]!.id"
                  class="absolute left-0 flex w-full cursor-default items-center px-2 text-sm outline-none data-[highlighted]:bg-muted"
                  :style="{
                    transform: `translateY(${virtualRow.start}px)`,
                    height: `${virtualRow.size}px`,
                  }"
                >
                  {{ displayOptions[virtualRow.index]?.label }}
                </ComboboxItem>
              </template>
            </div>
          </div>
        </ComboboxViewport>
        <button
          v-if="searchMore"
          type="button"
          class="w-full border-t border-border px-3 py-2 text-left text-sm text-primary hover:bg-muted"
          data-testid="choy-relation-search-more"
          @click="onSearchMore"
        >
          Search more…
        </button>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>
