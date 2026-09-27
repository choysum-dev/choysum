<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    data-anchor="choy.properties-field"
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
    <template #default>
      <div class="choy-properties-field flex flex-col gap-3">
        <div
          v-if="renderable.length === 0"
          class="rounded-md border border-dashed border-border px-3 py-4 text-sm text-foreground/50"
        >
          No properties defined
        </div>
        <div
          v-for="item in renderable"
          :key="item.name"
          class="flex flex-col gap-1"
        >
          <label class="text-sm font-medium text-foreground">
            {{ item.string || item.name }}
          </label>
          <Checkbox
            v-if="item.type === 'boolean'"
            :model-value="asBool(readValue(item.name))"
            :disabled="disabled || readonly || item.readonly === true"
            @update:model-value="onCheckbox(item.name, $event)"
          />
          <Input
            v-else-if="item.type === 'integer' || item.type === 'float'"
            :model-value="asString(readValue(item.name))"
            type="number"
            :disabled="disabled || readonly || item.readonly === true"
            @update:model-value="onNumber(item.name, $event, item.type === 'integer')"
          />
          <Textarea
            v-else-if="item.type === 'text'"
            :model-value="asString(readValue(item.name))"
            :rows="3"
            :disabled="disabled || readonly || item.readonly === true"
            @update:model-value="setValue(item.name, $event)"
          />
          <Select
            v-else-if="item.type === 'selection'"
            :model-value="(readValue(item.name) as string | null) ?? null"
            :disabled="disabled || readonly || item.readonly === true"
            @update:model-value="setValue(item.name, $event)"
          >
            <SelectTrigger class="w-full" placeholder="Select…" />
            <SelectContent>
              <SelectItem
                v-for="opt in normalizeSelectionOptions(item.selection)"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </SelectItem>
            </SelectContent>
          </Select>
          <Input
            v-else
            :model-value="asString(readValue(item.name))"
            :disabled="disabled || readonly || item.readonly === true"
            @update:model-value="setValue(item.name, $event)"
          />
        </div>
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import Input from '../vendor/ui/input/Input.vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';
import {
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  writePropertyValue,
  type PropertiesMap,
} from './propertiesHelpers';

/**
 * Dynamic properties map field. Host injects resolved `items` (no ResolveProperties RPC).
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      items?: ResolvedPropertyItem[];
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    items: () => [],
  },
);

const model = defineModel<PropertiesMap>({ default: () => Object.create(null) });

const renderable = computed(() => filterRenderablePropertyItems(props.items ?? []).renderable);

function readValue(name: string): unknown {
  const map = model.value || Object.create(null);
  if (Object.prototype.hasOwnProperty.call(map, name)) {
    return map[name];
  }
  // Match buildFullPropertiesMap: prefer item.value, then item.default.
  const item = (props.items ?? []).find(i => i?.name === name);
  if (!item) return undefined;
  if (Object.prototype.hasOwnProperty.call(item, 'value')) return item.value;
  if (Object.prototype.hasOwnProperty.call(item, 'default')) return item.default;
  return undefined;
}

function setValue(name: string, value: unknown): void {
  model.value = writePropertyValue(props.items ?? [], model.value, name, value);
}

function asString(v: unknown): string {
  return v == null ? '' : String(v);
}

function asBool(v: unknown): boolean {
  return v === true;
}

function onCheckbox(name: string, value: boolean | 'indeterminate'): void {
  setValue(name, value === true);
}

function onNumber(name: string, raw: string, integer: boolean): void {
  const text = String(raw ?? '').trim();
  if (!text) {
    setValue(name, null);
    return;
  }
  const n = Number(text);
  if (!Number.isFinite(n)) return;
  setValue(name, integer ? Math.trunc(n) : n);
}
</script>
