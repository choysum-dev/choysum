<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
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
    /** Show a "Search more…" footer action (host opens a dialog in PR5). */
    searchMore?: boolean;
  }>(),
  {
    placeholder: 'Search…',
    disabled: false,
    clearable: true,
    pageSize: 20,
    estimateSize: 32,
    searchMore: true,
  },
);

const modelValue = defineModel<string | null>({ default: null });
const open = ref(false);
const query = ref('');
const loading = ref(false);
const options = ref<RelationOption[]>([]);
const listParent = ref<HTMLElement | null>(null);

const emit = defineEmits<{
  'search-more': [query: string];
  select: [option: RelationOption | null];
}>();

const selected = computed(() => findRelationOption(options.value, modelValue.value));

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

async function refreshOptions(rawQuery: string): Promise<void> {
  loading.value = true;
  try {
    options.value = await runRelationNameSearch(props.search, rawQuery, props.pageSize);
  } finally {
    loading.value = false;
  }
}

let searchSeq = 0;
watch(
  () => [open.value, query.value] as const,
  async ([isOpen, q]) => {
    if (!isOpen) {
      return;
    }
    const seq = ++searchSeq;
    await refreshOptions(q);
    if (seq !== searchSeq) {
      return;
    }
  },
);

watch(modelValue, (id) => {
  emit('select', findRelationOption(options.value, id));
});

function onClear(): void {
  modelValue.value = null;
  query.value = '';
  emit('select', null);
}

function onSearchMore(): void {
  emit('search-more', normalizeRelationQuery(query.value));
  open.value = false;
}

/** Remote search owns filtering; keep every option visible. */
function alwaysMatch(): boolean {
  return true;
}
</script>

<template>
  <ComboboxRoot
    v-model="modelValue"
    v-model:open="open"
    data-anchor="choy.internal.relation-combobox"
    :disabled="disabled"
    :filter-function="alwaysMatch"
    :class="cn('choy-relation-combobox relative w-full', props.class)"
  >
    <ComboboxAnchor class="flex w-full gap-1">
      <ComboboxInput
        v-model="query"
        :disabled="disabled"
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
            <div v-if="loading" class="px-3 py-2 text-sm text-foreground/60">Searching…</div>
            <ComboboxEmpty v-else-if="!displayOptions.length" class="px-3 py-2 text-sm text-foreground/60">
              No matches
            </ComboboxEmpty>
            <div
              v-else
              :style="{ height: `${totalSize}px`, position: 'relative', width: '100%' }"
            >
              <ComboboxItem
                v-for="virtualRow in virtualRows"
                :key="displayOptions[virtualRow.index]?.id ?? String(virtualRow.key)"
                :value="displayOptions[virtualRow.index]?.id ?? ''"
                class="absolute left-0 flex w-full cursor-default items-center px-2 text-sm outline-none data-[highlighted]:bg-muted"
                :style="{
                  transform: `translateY(${virtualRow.start}px)`,
                  height: `${virtualRow.size}px`,
                }"
              >
                {{ displayOptions[virtualRow.index]?.label }}
              </ComboboxItem>
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
