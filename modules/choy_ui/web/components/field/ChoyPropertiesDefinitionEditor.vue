<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ref, watch } from 'vue';
import type { PropertyItemDefinition } from '@/core/service/orm/model/properties_types';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import Input from '../vendor/ui/input/Input.vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyButton from '../layout/ChoyButton.vue';
import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
  type DefinitionEditorDraftItem,
} from './propertiesDefinitionHelpers';

/**
 * Edit a PropertyDefinition item list as drafts. Emits `saved` with validated items.
 * Host performs Create/Update RPC (none in isolation).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    items?: PropertyItemDefinition[];
    disabled?: boolean;
  }>(),
  {
    items: () => [],
    disabled: false,
  },
);

const emit = defineEmits<{
  saved: [items: PropertyItemDefinition[]];
}>();

const drafts = ref<DefinitionEditorDraftItem[]>(definitionItemsToDrafts(props.items));
const error = ref('');

watch(
  () => props.items,
  (next, prev) => {
    // A parent re-render can pass a brand-new array with identical content;
    // only re-seed when the definition content actually changed.
    if (JSON.stringify(next ?? []) === JSON.stringify(prev ?? [])) return;
    drafts.value = definitionItemsToDrafts(next);
    error.value = '';
  },
);

function addRow(): void {
  drafts.value = [...drafts.value, emptyDraftItem()];
}

function removeRow(index: number): void {
  drafts.value = drafts.value.filter((_, i) => i !== index);
}

function onSave(): void {
  try {
    const items = draftsToDefinitionItems(drafts.value);
    error.value = '';
    emit('saved', items);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}
</script>

<template>
  <div
    data-anchor="choy.properties-definition-editor"
    :class="['choy-properties-definition-editor flex flex-col gap-3', props.class]"
  >
    <div
      v-for="(draft, index) in drafts"
      :key="index"
      class="grid gap-2 rounded-md border border-border p-3 sm:grid-cols-2"
    >
      <Input v-model="draft.name" placeholder="name" :disabled="disabled" />
      <Select v-model="draft.type" :disabled="disabled">
        <SelectTrigger class="w-full" placeholder="type" />
        <SelectContent>
          <SelectItem
            v-for="t in PROPERTY_DEFINITION_V1_TYPE_OPTIONS"
            :key="t"
            :value="t"
          >
            {{ t }}
          </SelectItem>
        </SelectContent>
      </Select>
      <Input v-model="draft.string" placeholder="label" :disabled="disabled" />
      <Input v-model="draft.default" placeholder="default" :disabled="disabled" />
      <label class="flex items-center gap-2 text-sm sm:col-span-2">
        <Checkbox
          :model-value="draft.readonly"
          :disabled="disabled"
          @update:model-value="draft.readonly = $event === true"
        />
        Readonly
      </label>
      <Textarea
        v-if="draft.type === 'selection'"
        v-model="draft.selectionText"
        class="font-mono text-xs sm:col-span-2"
        :rows="3"
        placeholder='[["a","A"],["b","B"]]'
        :disabled="disabled"
      />
      <div class="sm:col-span-2">
        <ChoyButton size="sm" variant="outline" :disabled="disabled" @click="removeRow(index)">
          Remove
        </ChoyButton>
      </div>
    </div>
    <p v-if="error" class="text-sm text-danger">{{ error }}</p>
    <div class="flex flex-wrap gap-2">
      <ChoyButton size="sm" variant="outline" :disabled="disabled" @click="addRow">Add property</ChoyButton>
      <ChoyButton size="sm" :disabled="disabled" @click="onSave">Save definition</ChoyButton>
    </div>
  </div>
</template>
