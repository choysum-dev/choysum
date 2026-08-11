<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <button
    v-if="visible"
    type="button"
    class="o-stat-info"
    :class="{ 'is-disabled': disabled }"
    :disabled="disabled"
    v-bind="$attrs"
    @click="onClick"
  >
    <el-icon v-if="resolvedIcon" class="o-stat-info__icon">
      <component :is="resolvedIcon" />
    </el-icon>
    <span class="o-stat-info__body">
      <span class="o-stat-info__value">{{ displayValue }}</span>
      <span class="o-stat-info__label">{{ label }}</span>
    </span>
  </button>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>">
import { computed, markRaw, toRaw, type Component } from 'vue';
import { useRouter, type RouteLocationRaw } from 'vue-router';
import { ElIcon } from 'element-plus';
import type { BaseModel, FieldPath } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import { resolveStatDisplayValue } from '@/web/web/components/view/ostatinfo_helpers';

defineOptions({ name: 'OStatInfo', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    /** Explicit value; preferred over relation length (D7). */
    value?: number | string | null;
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    /** Button caption (i18n-ready). */
    label: string;
    icon?: Component | object;
    /** When set, click also `router.push(to)` after emit (D8). */
    to?: RouteLocationRaw;
    disabled?: boolean;
    /** When false, the button is not rendered (D9). */
    visible?: boolean;
    /** Shown when neither `value` nor a relation array is available. Default em dash. */
    emptyValue?: number | string;
  }>(),
  {
    disabled: false,
    visible: true,
    emptyValue: '—',
  }
);

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void;
}>();

const router = useRouter();

// Avoid Vue "component made reactive" warn when a Component is passed as icon prop.
const resolvedIcon = computed(() => {
  const icon = props.icon;
  return icon ? markRaw(toRaw(icon as object)) : null;
});

// Optional relation binding for Array.length; never auto-registers into form fields (D7).
const relationBinding =
  props.store != null && props.prop != null
    ? useField<T, string, unknown>({ store: props.store, prop: String(props.prop), autoRegister: false })
    : null;

const displayValue = computed(() =>
  resolveStatDisplayValue({
    value: props.value,
    relationValue: relationBinding ? relationBinding.fieldRef().value : undefined,
    emptyValue: props.emptyValue,
  })
);

function onClick(event: MouseEvent) {
  if (props.disabled) return;
  emit('click', event);
  if (props.to != null) {
    void router?.push(props.to);
  }
}
</script>

<style scoped>
.o-stat-info {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 72px;
  padding: 6px 10px;
  margin: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-primary);
  font: inherit;
  line-height: 1.25;
  text-align: left;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    border-color 0.15s ease;
}

.o-stat-info:hover:not(.is-disabled):not(:disabled) {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
}

.o-stat-info.is-disabled,
.o-stat-info:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.o-stat-info__icon {
  font-size: 18px;
  color: var(--el-text-color-secondary);
}

.o-stat-info__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.o-stat-info__value {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.o-stat-info__label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
</style>
