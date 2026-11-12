<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :orientation="orientation"
    :hide-label="hideLabel"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    :inline-error-space="inlineErrorSpace"
    :label-for-control="false"
  >
    <template #edit="{ inputId, labelId, describedBy, invalid }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-m2m-kanban-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope
        v-else
        :view-mode="binding.env.viewMode"
        :kind="'Kanban'"
        :field-prefix="String(prop)"
      >
        <div
          class="flex w-full min-w-0 flex-col gap-2"
          :id="inputId || undefined"
          tabindex="-1"
          :aria-labelledby="labelId || undefined"
          :aria-describedby="describedBy || undefined"
          :aria-invalid="invalid ? 'true' : undefined"
          data-testid="choy-m2m-kanban"
        >
          <ChoyO2MKanbanVirtualBoard
            :items="kanbanItems"
            :min-card-width="minCardWidth"
            :gap="gap"
            :max-height="maxHeight"
            :virtualize="boardVirtualize"
            :item-key="kanbanItemKey"
          >
            <template #card="{ item, index }">
              <div
                class="choy-m2m-kanban-card cursor-pointer rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40"
                data-testid="choy-m2m-kanban-card"
                role="button"
                tabindex="0"
                :aria-label="resolveCardTitle(item)"
                @click="emitCardClick(index, item)"
                @keydown.enter.prevent="emitCardClick(index, item)"
              >
                <slot name="card" :item="item" :index="index">
                  <div class="text-sm font-medium leading-snug text-foreground">
                    {{ resolveCardTitle(item) }}
                  </div>
                  <div
                    v-if="resolveCardSubtitle(item)"
                    class="mt-1 truncate text-xs text-muted-foreground"
                  >
                    {{ resolveCardSubtitle(item) }}
                  </div>
                </slot>
              </div>
            </template>
            <template #empty>
              <div
                class="col-span-full flex min-h-[72px] items-center justify-center rounded-md border border-dashed border-border text-sm text-muted-foreground"
              >
                <slot name="empty">{{ effectiveEmptyText }}</slot>
              </div>
            </template>
          </ChoyO2MKanbanVirtualBoard>
        </div>
      </ChoyViewScope>
    </template>

    <template #display="{ inputId, labelId }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-m2m-kanban-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope v-else view-mode="display" :kind="'Kanban'" :field-prefix="String(prop)">
        <div
          class="flex w-full min-w-0 flex-col gap-2"
          :id="inputId || undefined"
          :aria-labelledby="labelId || undefined"
          data-testid="choy-m2m-kanban"
        >
          <ChoyO2MKanbanVirtualBoard
            :items="kanbanItems"
            :min-card-width="minCardWidth"
            :gap="gap"
            :max-height="maxHeight"
            :virtualize="boardVirtualize"
            :item-key="kanbanItemKey"
          >
            <template #card="{ item, index }">
              <div
                class="choy-m2m-kanban-card cursor-pointer rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/30"
                data-testid="choy-m2m-kanban-card"
                role="button"
                tabindex="0"
                :aria-label="resolveCardTitle(item)"
                @click="emitCardClick(index, item)"
                @keydown.enter.prevent="emitCardClick(index, item)"
              >
                <slot name="card" :item="item" :index="index">
                  <div class="text-sm font-medium leading-snug text-foreground">
                    {{ resolveCardTitle(item) }}
                  </div>
                  <div
                    v-if="resolveCardSubtitle(item)"
                    class="mt-1 truncate text-xs text-muted-foreground"
                  >
                    {{ resolveCardSubtitle(item) }}
                  </div>
                </slot>
              </div>
            </template>
            <template #empty>
              <div
                class="col-span-full flex min-h-[72px] items-center justify-center rounded-md border border-dashed border-border text-sm text-muted-foreground"
              >
                <slot name="empty">{{ effectiveEmptyText }}</slot>
              </div>
            </template>
          </ChoyO2MKanbanVirtualBoard>
        </div>
      </ChoyViewScope>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import ChoyO2MKanbanVirtualBoard from './ChoyO2MKanbanVirtualBoard.vue';
import type { O2MKanbanVirtualizeMode } from './o2mKanbanVirtualHelpers';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';
import { formatRelationCountSummary } from './relationCountSummary';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyManyToManyKanbanField' });

defineOptions({ name: 'ChoyManyToManyKanbanField', inheritAttrs: false });

type IsAny<TA> = 0 extends 1 & TA ? true : false;

defineSlots<{
  card(props: { item: any; index: number }): unknown;
  empty(): unknown;
}>();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;
    orientation?: 'horizontal' | 'vertical';
    hideLabel?: boolean;
    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
    /** collapse | reserve | overlay; forwarded to ChoyFieldBase. */
    inlineErrorSpace?: 'collapse' | 'reserve' | 'overlay';
    cardTitleField?: string;
    cardSubtitleField?: string;
    emptyText?: string;
    minCardWidth?: number;
    gap?: number;
    maxHeight?: number;
    boardVirtualize?: O2MKanbanVirtualizeMode;
    /** Kept for Form API parity with O2M kanban; display-first ignores edit chrome. */
    editable?: boolean;
    removable?: boolean;
    showToolbarAdd?: boolean;
    previewInDisplay?: boolean;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    orientation: 'vertical',
    hideLabel: false,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    renderMode: 'auto',
    showInlineError: false,
    cardTitleField: 'Name',
    cardSubtitleField: '',
    emptyText: '',
    minCardWidth: 220,
    gap: 10,
    maxHeight: 480,
    boardVirtualize: 'auto',
    editable: false,
    removable: false,
    showToolbarAdd: false,
    previewInDisplay: false,
  },
);

const emit = defineEmits<{
  (e: 'card-click', payload: { index: number; item: any; mode: 'display' }): void;
}>();

const effectiveEmptyText = computed(() => props.emptyText || _t('No data'));

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

function resolveReturnFieldPaths(): string[] {
  const prefix = String(binding.prop || props.prop || '').trim();
  if (!prefix) return [];
  const out = new Set<string>([`${prefix}.DisplayName`]);
  const title = String(props.cardTitleField || '').trim();
  const subtitle = String(props.cardSubtitleField || '').trim();
  if (title) out.add(`${prefix}.${title}`);
  if (subtitle) out.add(`${prefix}.${subtitle}`);
  return [...out];
}
binding.registerFields(resolveReturnFieldPaths());

const { getItems } = binding.asMutableArray<any>();

const itemCount = computed(() => getItems()?.length ?? 0);
const kanbanItems = computed(() => {
  const items = getItems() || [];
  void items.length;
  return items;
});
const showCellSummary = computed(() => binding.env?.isForm === false);
const cellSummaryText = computed(() => formatRelationCountSummary(itemCount.value, _t));

function kanbanItemKey(item: unknown, index: number): string {
  if (item && typeof item === 'object') {
    const row = item as Record<string, unknown>;
    const id = row.Id ?? row.__rowKey;
    if (id != null && String(id).trim()) return String(id);
  }
  return String(index);
}

function resolveCardTitle(item: any): string {
  const field = String(props.cardTitleField || '').trim();
  const value = field ? item?.[field] : undefined;
  if (value != null && String(value).trim()) return String(value).trim();
  return String(item?.DisplayName || item?.Id || _t('Untitled'));
}

function resolveCardSubtitle(item: any): string {
  const field = String(props.cardSubtitleField || '').trim();
  const value = field ? item?.[field] : undefined;
  return value != null ? String(value) : '';
}

function emitCardClick(index: number, item: any) {
  emit('card-click', { index, item, mode: 'display' });
}
</script>
