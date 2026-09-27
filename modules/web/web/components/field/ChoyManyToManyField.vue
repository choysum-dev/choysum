<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    data-anchor="choy.many-to-many-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId }">
      <div class="choy-many-to-many-field flex flex-col gap-2">
        <!-- tags -->
        <template v-if="widget === 'tags'">
          <div class="flex flex-wrap gap-1.5">
            <Badge
              v-for="opt in selectedOptions"
              :key="opt.id"
              variant="secondary"
              class="gap-1"
              @click="emit('tag-click', opt.id)"
            >
              {{ opt.label }}
              <button
                v-if="!readonly && !disabled"
                type="button"
                class="ml-1 text-foreground/60 hover:text-foreground"
                :aria-label="`Remove ${opt.label}`"
                @click.stop="removeId(opt.id)"
              >
                ×
              </button>
            </Badge>
            <span
              v-if="selectedOptions.length === 0"
              class="text-xs text-foreground/50"
            >
              No tags
            </span>
          </div>
          <RelationCombobox
            v-if="!readonly && !disabled && search"
            :id="controlId"
            v-model="pickId"
            :search="search"
            :search-key="searchKey"
            :page-size="pageSize"
            :placeholder="placeholder"
            :clearable="true"
            @select="onPickSelect"
          />
        </template>

        <!-- list -->
        <template v-else-if="widget === 'list'">
          <RelationCombobox
            v-if="!readonly && !disabled && search"
            v-model="pickId"
            :search="search"
            :search-key="searchKey"
            :page-size="pageSize"
            :placeholder="placeholder"
            :clearable="true"
            @select="onPickSelect"
          />
          <DataTable
            :columns="columns"
            :data="selectedOptions"
            :row-id="(row) => row.id"
            :height="height"
            :enable-row-selection="false"
            :enable-sorting="true"
          />
          <div v-if="!readonly && !disabled" class="flex flex-wrap gap-1">
            <ChoyButton
              v-for="opt in selectedOptions"
              :key="opt.id"
              size="sm"
              variant="ghost"
              @click="removeId(opt.id)"
            >
              Remove {{ opt.label }}
            </ChoyButton>
          </div>
        </template>

        <!-- tree -->
        <template v-else>
          <div class="max-h-64 overflow-auto rounded-md border border-border p-2">
            <label
              v-for="node in flatTree"
              :key="node.id"
              class="flex items-center gap-2 py-1 text-sm"
              :style="{ paddingLeft: `${node.depth}rem` }"
            >
              <Checkbox
                :model-value="selectedSet.has(node.id)"
                :disabled="disabled || readonly"
                @update:model-value="toggleTreeId(node.id, $event === true)"
              />
              <span>{{ node.label }}</span>
            </label>
            <div
              v-if="flatTree.length === 0"
              class="px-2 py-4 text-center text-xs text-foreground/50"
            >
              No nodes
            </div>
          </div>
        </template>

        <!-- fallback when tags without search still need a combobox slot -->
        <RelationCombobox
          v-if="widget === 'tags' && !search && !readonly && !disabled"
          v-model="pickId"
          :search="noopSearch"
          :placeholder="placeholder"
          :disabled="true"
        />
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { ColumnDef } from '@tanstack/vue-table';
import DataTable from '../internal/DataTable.vue';
import RelationCombobox from '../internal/RelationCombobox.vue';
import type {
  RelationNameSearchFn,
  RelationOption,
} from '../internal/relationComboboxHelpers';
import type { ClassValue } from '../../lib/utils';
import Badge from '../vendor/ui/badge/Badge.vue';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

export type ChoyManyToManyWidget = 'tags' | 'list' | 'tree';

export type ChoyManyToManyTreeNode = {
  id: string;
  label: string;
  children?: ChoyManyToManyTreeNode[];
};

/**
 * Many-to-many field with widget: tags | list | tree.
 * Model is an id list; host supplies search / options / tree nodes.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      widget?: ChoyManyToManyWidget;
      /** tags / list: resolve labels for selected ids */
      options?: RelationOption[];
      search?: RelationNameSearchFn;
      searchKey?: string;
      pageSize?: number;
      /** list widget columns when showing selected rows as a table */
      columns?: ColumnDef<RelationOption, unknown>[];
      height?: number;
      /** tree widget */
      treeNodes?: ChoyManyToManyTreeNode[];
      placeholder?: string;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    widget: 'tags',
    options: () => [],
    pageSize: 20,
    columns: () => [
      { accessorKey: 'label', header: 'Name', size: 200 },
      { accessorKey: 'id', header: 'Id', size: 120 },
    ],
    height: 200,
    treeNodes: () => [],
    placeholder: 'Search…',
  },
);

const model = defineModel<string[]>({ default: () => [] });

const emit = defineEmits<{
  'tag-add': [id: string];
  'tag-remove': [id: string];
  'tag-click': [id: string];
}>();

const pickId = ref<string | null>(null);

const selectedOptions = computed(() => {
  const byId = new Map((props.options ?? []).map(o => [o.id, o]));
  return (model.value ?? []).map(id => byId.get(id) ?? { id, label: id });
});

const selectedSet = computed(() => new Set(model.value ?? []));

function addId(id: string | null): void {
  if (!id || props.readonly || props.disabled) return;
  if ((model.value ?? []).includes(id)) return;
  model.value = [...(model.value ?? []), id];
  emit('tag-add', id);
}

function removeId(id: string): void {
  if (props.readonly || props.disabled) return;
  model.value = (model.value ?? []).filter(x => x !== id);
  emit('tag-remove', id);
}

function onPickSelect(option: RelationOption | null): void {
  if (!option) return;
  addId(option.id);
  pickId.value = null;
}

function toggleTreeId(id: string, checked: boolean): void {
  if (checked) addId(id);
  else removeId(id);
}

function flattenTree(
  nodes: ChoyManyToManyTreeNode[],
  depth = 0,
): Array<{ id: string; label: string; depth: number }> {
  const out: Array<{ id: string; label: string; depth: number }> = [];
  for (const n of nodes) {
    out.push({ id: n.id, label: n.label, depth });
    if (n.children?.length) out.push(...flattenTree(n.children, depth + 1));
  }
  return out;
}

const flatTree = computed(() => flattenTree(props.treeNodes ?? []));

const noopSearch: RelationNameSearchFn = async () => [];
</script>
