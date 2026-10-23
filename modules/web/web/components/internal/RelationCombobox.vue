<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ComboboxRoot
    v-model="modelValue"
    v-model:open="open"
    data-anchor="choy.internal.relation-combobox"
    :disabled="disabled"
    :ignore-filter="true"
    :class="cn('choy-relation-combobox relative w-full', props.class)"
  >
    <ComboboxAnchor class="relative w-full">
      <ComboboxInput
        v-model="query"
        :id="id || undefined"
        :name="name || undefined"
        :disabled="disabled"
        :aria-invalid="props['aria-invalid']"
        :aria-required="props['aria-required']"
        :aria-describedby="props['aria-describedby']"
        :display-value="() => selected?.label ?? ''"
        :placeholder="selected?.label || effectivePlaceholder"
        :class="
          cn(
            'flex h-control w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm font-normal text-foreground shadow-xs',
            'placeholder:text-muted-foreground',
            controlFocusRing,
            showClear && 'pr-8',
            disabled && 'cursor-not-allowed opacity-50',
            'aria-invalid:border-destructive aria-invalid:ring-destructive/20',
          )
        "
        @focus="onInputFocus"
        @input="onQueryInput"
      />
      <button
        v-if="showClear"
        type="button"
        class="absolute top-1/2 right-1.5 inline-flex -translate-y-1/2 text-foreground/50 hover:text-foreground"
        :aria-label="clearLabel"
        data-testid="choy-relation-clear"
        tabindex="-1"
        @mousedown.prevent
        @click.stop="onClear"
      >
        <X class="size-3.5" aria-hidden="true" />
      </button>
    </ComboboxAnchor>
    <ComboboxPortal>
      <ComboboxContent
        position="popper"
        :class="
          cn(
            'z-50 w-[var(--reka-combobox-trigger-width)] origin-(--reka-combobox-content-transform-origin) overflow-hidden rounded-md border border-border bg-popover p-0 text-popover-foreground shadow-md',
            // Fade only (no zoom) so reopen does not flash scale.
            'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
          )
        "
      >
        <ComboboxViewport>
          <div ref="listParent" class="max-h-56 overflow-auto p-1">
            <div
              v-if="searchError"
              class="px-2 py-1.5 text-sm text-destructive"
              role="alert"
            >
              {{ searchError }}
            </div>
            <div
              v-if="loading && !displayOptions.length"
              class="px-2 py-1.5 text-sm text-muted-foreground"
            >
              {{ searchingLabel }}
            </div>
            <ComboboxEmpty
              v-else-if="!loading && !searchError && !displayOptions.length"
              class="px-2 py-1.5 text-sm text-muted-foreground"
            >
              {{ noMatchesLabel }}
            </ComboboxEmpty>
            <!-- Short NameSearch pages: paint immediately (virtualizer needs a measured parent). -->
            <template v-else-if="displayOptions.length && !useVirtualList">
              <ComboboxItem
                v-for="opt in displayOptions"
                :key="opt.id"
                :value="opt.id"
                class="flex w-full cursor-default select-none items-center rounded-sm py-1.5 pe-2 ps-2 text-sm outline-none data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
              >
                <span class="min-w-0 flex-1 truncate">{{ opt.label }}</span>
              </ComboboxItem>
            </template>
            <div
              v-else-if="displayOptions.length"
              :style="{ height: `${totalSize}px`, position: 'relative', width: '100%' }"
            >
              <template
                v-for="virtualRow in virtualRows"
                :key="displayOptions[virtualRow.index]?.id ?? String(virtualRow.key)"
              >
                <ComboboxItem
                  v-if="displayOptions[virtualRow.index]"
                  :value="displayOptions[virtualRow.index]!.id"
                  class="absolute left-0 flex w-full cursor-default select-none items-center rounded-sm py-1.5 pe-2 ps-2 text-sm outline-none data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
                  :style="{
                    transform: `translateY(${virtualRow.start}px)`,
                    height: `${virtualRow.size}px`,
                  }"
                >
                  <span class="min-w-0 flex-1 truncate">{{
                    displayOptions[virtualRow.index]?.label
                  }}</span>
                </ComboboxItem>
              </template>
            </div>
          </div>
        </ComboboxViewport>
        <div v-if="$slots.footer" class="border-t border-border">
          <slot name="footer" />
        </div>
        <button
          v-if="searchMore"
          type="button"
          class="block w-full cursor-pointer select-none border-t border-border px-3 py-2 text-left text-sm text-primary transition-colors hover:bg-muted focus:outline-none focus-visible:bg-muted"
          data-testid="choy-relation-search-more"
          @click="onSearchMore"
        >
          {{ searchMoreLabel }}
        </button>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>

<script setup lang="ts">
import { computed, nextTick, onWatcherCleanup, ref, watch } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { X } from 'lucide-vue-next';
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxViewport
} from 'reka-ui';
import { cn, controlFocusRing, type ClassValue } from '../../lib/utils';
import { createTranslate } from '@/web/web/i18n';
import {
  findRelationOption,
  normalizeRelationPageSize,
  normalizeRelationQuery,
  resolveRelationSearchMemo,
  runRelationNameSearch,
  upsertRelationOption,
  type RelationNameSearchFn,
  type RelationOption,
  type RelationSearchMemo,
} from './relationComboboxHelpers';

/**
 * L3 relation typeahead (Reka Combobox + virtual list + NameSearch).
 * Model is the selected option id (string | null). Not a public Choy* export.
 */
const { _t } = createTranslate('web', { scope: 'web/components/internal/RelationCombobox' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    clearable?: boolean;
    pageSize?: number;
    estimateSize?: number;
    /** Debounce for remote NameSearch keystrokes (ms). */
    debounce?: number;
    search: RelationNameSearchFn;
    /**
     * Stable identity for the search target (relation/model). Prefer this over
     * watching `search` by function identity so inline arrows do not clear options.
     */
    searchKey?: string;
    /** Optional seed so an initial modelValue keeps its label before search. */
    selectedOption?: RelationOption | null;
    /** Show a Search more footer action (host opens a dialog). */
    searchMore?: boolean;
    id?: string;
    name?: string;
    'aria-invalid'?: boolean | 'true' | 'false';
    'aria-required'?: boolean | 'true' | 'false';
    'aria-describedby'?: string;
  }>(),
  {
    placeholder: undefined,
    disabled: false,
    clearable: true,
    pageSize: 20,
    estimateSize: 32,
    debounce: 150,
    selectedOption: null,
    searchMore: true,
  },
);

defineSlots<{
  footer?: () => unknown;
}>();

const modelValue = defineModel<string | null>({ default: null });
const open = ref(false);
const query = ref('');
/**
 * Odoo AutoComplete useInput: NameSearch uses the input only after the user edits.
 * Focus/open with an unchanged selection still shows the label in the box, but RPC
 * uses "" so the dropdown is a full page (not filtered by the display name).
 */
const userEdited = ref(false);
const rpcQuery = computed(() => (userEdited.value ? query.value : ''));
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

const effectivePlaceholder = computed(() =>
  props.placeholder != null && String(props.placeholder).length > 0
    ? String(props.placeholder)
    : _t('Search…'),
);
const clearLabel = computed(() => _t('Clear'));
const searchingLabel = computed(() => _t('Searching…'));
const noMatchesLabel = computed(() => _t('No matches'));
const searchMoreLabel = computed(() => _t('Search more…'));
const searchFailedLabel = computed(() => _t('Search failed. Please retry.'));
const showClear = computed(
  () => Boolean(props.clearable && modelValue.value && !props.disabled),
);

function clearSearchError(): void {
  if (searchError.value === null) {
    return;
  }
  searchError.value = null;
  emit('search-error', null);
}

const selected = computed(() => {
  const id = String(modelValue.value ?? '').trim();
  if (!id) {
    return null;
  }
  return (
    findRelationOption(options.value, id) ??
    (String(pinnedSelected.value?.id ?? '').trim() === id ? pinnedSelected.value : null) ??
    (String(props.selectedOption?.id ?? '').trim() === id ? props.selectedOption : null)
  );
});

const displayOptions = computed(() =>
  upsertRelationOption(options.value, selected.value),
);

/** Virtualize only past a typical NameSearch page; short lists skip measure-on-open flicker. */
const useVirtualList = computed(() => displayOptions.value.length > 48);

const virtualizer = useVirtualizer({
  get count() {
    return open.value && useVirtualList.value ? displayOptions.value.length : 0;
  },
  getScrollElement: () => (open.value ? (listParent.value as Element | null) : null),
  estimateSize: () =>
    Number.isFinite(props.estimateSize) && props.estimateSize > 0 ? props.estimateSize : 32,
  // Prepending the selected option shifts indices; keep measurements bound to the option.
  getItemKey: (index) => displayOptions.value[index]?.id ?? index,
  overscan: 6,
});

const virtualRows = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());

/**
 * Prefer searchKey so inline `search` arrows do not thrash watchers: a fresh function
 * identity (created by every parent render) must not clear `options` / `pinnedSelected`
 * or restart the debounce. Without a searchKey the newest `props.search` is still read
 * at call time; pass searchKey when switching search targets.
 */
const searchIdentity = computed(() => props.searchKey ?? '');

/** Skip identical NameSearch RPCs (reopen / trim) when searchKey encodes domain. */
let previousSearch: RelationSearchMemo | null = null;
let searchSeq = 0;
watch(
  [() => open.value, rpcQuery, searchIdentity, () => props.pageSize],
  async ([isOpen, q]) => {
    if (!isOpen) {
      return;
    }
    const seq = ++searchSeq;
    clearSearchError();
    const keyword = normalizeRelationQuery(String(q ?? ''));
    const cached = resolveRelationSearchMemo(
      previousSearch,
      keyword,
      searchIdentity.value,
      props.pageSize,
    );
    if (cached) {
      options.value = cached;
      loading.value = false;
      if (modelValue.value) {
        const found = findRelationOption(cached, modelValue.value);
        if (found) {
          pinnedSelected.value = found;
        }
      }
      return;
    }
    // Empty list: show Searching immediately (avoids a No matches flash during debounce).
    // Cached options: keep them on screen and refresh silently after debounce.
    if (options.value.length === 0) {
      loading.value = true;
    }
    const delay = Number.isFinite(props.debounce) ? Math.max(0, props.debounce) : 0;
    await new Promise<void>((resolve) => {
      const timer = setTimeout(resolve, delay);
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
      const results = await runRelationNameSearch(props.search, keyword, props.pageSize);
      if (seq !== searchSeq) {
        return;
      }
      previousSearch = {
        name: keyword,
        searchKey: searchIdentity.value,
        pageSize: normalizeRelationPageSize(props.pageSize),
        records: results,
      };
      options.value = results;
      if (listParent.value) {
        listParent.value.scrollTop = 0;
      }
      clearSearchError();
      if (modelValue.value) {
        const found = findRelationOption(results, modelValue.value);
        if (found) {
          pinnedSelected.value = found;
        }
      }
    } catch (error) {
      if (seq === searchSeq) {
        // Keep prior options so the user can still pick; surface the failure above.
        const message = error instanceof Error ? error.message : String(error);
        searchError.value = searchFailedLabel.value;
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
watch(searchIdentity, () => {
  options.value = [];
  previousSearch = null;
  pinnedSelected.value = props.selectedOption ?? null;
  clearSearchError();
  // While open, the search watcher already re-queries with the new target;
  // bumping searchSeq here would cancel that fresh request and leave the list empty.
  if (!open.value) {
    searchSeq += 1;
    loading.value = false;
  }
});

watch(modelValue, (rawId) => {
  const id = String(rawId ?? '').trim();
  if (!id) {
    pinnedSelected.value = null;
    emit('select', null);
    if (open.value) {
      open.value = false;
    }
    return;
  }
  const found =
    findRelationOption(options.value, id) ??
    (String(pinnedSelected.value?.id ?? '').trim() === id ? pinnedSelected.value : null) ??
    (String(props.selectedOption?.id ?? '').trim() === id ? props.selectedOption : null);
  if (!found) {
    // Id is set but its label is not loaded yet; not the same as clearing.
    return;
  }
  pinnedSelected.value = found;
  emit('select', found);
  // Host NameCreate / external writes update model while the panel is open — close it.
  if (open.value) {
    open.value = false;
  }
});

watch(open, (isOpen) => {
  if (!isOpen) {
    // Pin the current label so the closed trigger stays correct after a pick.
    const current = findRelationOption(options.value, modelValue.value);
    if (current) {
      pinnedSelected.value = current;
    }
    // Drop the typeahead keyword so the trigger shows the selected label again.
    // Keep last options so the next open can paint immediately (avoid empty→Searching flash).
    query.value = '';
    userEdited.value = false;
    searchSeq += 1;
    loading.value = false;
    clearSearchError();
  }
});

function onClear(): void {
  modelValue.value = null;
  query.value = '';
  userEdited.value = false;
}

/** Real keystrokes / paste — not Combobox syncing the selected label into the filter. */
function onQueryInput(): void {
  userEdited.value = true;
}

/**
 * Odoo-style: focus opens the panel; NameSearch uses "" until the user edits
 * (rpcQuery), even if the input still shows the selected label.
 */
function onInputFocus(ev: FocusEvent): void {
  if (props.disabled) {
    return;
  }
  const el = ev.target as HTMLInputElement | null;
  const saved = String(selected.value?.label ?? '').trim();
  const current = String(el?.value ?? query.value ?? '').trim();
  // Same heuristic as Odoo: input already differs from the saved display → search it.
  userEdited.value = Boolean(saved && current && current !== saved);
  if (!open.value) {
    open.value = true;
  }
  // Select-all so the next key replaces the label instead of appending.
  if (el && selected.value && !userEdited.value) {
    void nextTick(() => {
      el.select?.();
    });
  }
}

function onSearchMore(): void {
  emit('search-more', normalizeRelationQuery(rpcQuery.value));
  open.value = false;
}
</script>
