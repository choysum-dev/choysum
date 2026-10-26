<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="formItemProps"
    :orientation="orientation"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, describedBy, invalid}">
      <ChoyTextCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :rows="rows"
        :autosize="autosize"
        :maxlength="maxLength ?? undefined"
        :show-word-limit="showWordLimit && !!maxLength"
        v-bind="$attrs"
        :id="inputId"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined"
      />
    </template>

    <template #display="{ fieldValue }">
      <div
        v-if="isEmptyDisplay(fieldValue().value) && showFormEmptyPlaceholder"
        class="choy-textfield-display choy-textfield-display--empty text-sm text-muted-foreground"
      >
        {{ emptyDisplayText }}
      </div>
      <div v-else class="choy-textfield-display whitespace-pre-wrap wrap-break-word text-foreground">{{ toDisplayText(fieldValue().value) }}</div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { computed, defineComponent, h } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Narrow aggregation types to count_distinct only.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyTextField' });

defineOptions({ name: 'ChoyTextField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    maxLength?: number; // Optional length limit for text input.
    nullable?: boolean; // Empty strings map to null by default.
    trimOnBlur?: 'none' | 'both';
    placeholder?: string;

    rows?: number;
    autosize?: boolean | { minRows?: number; maxRows?: number };
    showWordLimit?: boolean;
    /** Form display placeholder when empty (defaults to blank). */
    emptyText?: string;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    formItemProps?: Partial<FormItemProps>;
    orientation?: 'horizontal' | 'vertical';
    vColumnProps?: {
      width?: number | string;
      minWidth?: number;
      align?: 'left' | 'center' | 'right';
      fixed?: 'left' | 'right';
      sortable?: boolean;
    };
    bufferStrategy?: CommitStrategy;
    bufferIdleDelay?: number;
    commitOnBlur?: boolean;
    // Only count_distinct is supported for text fields.
    agg?: NarrowAggProp<NonNumericAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    nullable: true,
    trimOnBlur: 'none',
    placeholder: '',
    rows: 4,
    autosize: () => ({ minRows: 3, maxRows: 10 }),
    showWordLimit: true,

    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,

    formItemProps: () => ({}),
    vColumnProps: () => ({}),

    bufferStrategy: 'blur',
    bufferIdleDelay: 400,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const showFormEmptyPlaceholder = computed(() => {
  const mode = props.renderMode;
  if (mode === 'table' || mode === 'inline') return false;
  if (mode === 'form') return true;
  return binding.env?.isForm !== false;
});

const emptyDisplayText = computed(() => {
  if (props.emptyText != null) return String(props.emptyText);
  return '';
});

const toView = (raw: any): ViewType => (raw == null ? null : String(raw));
const fromView = (v: ViewType) => v as unknown as V;
const toDisplayText = (v: ViewType) => (v == null ? '' : String(v));
function isEmptyDisplay(v: ViewType): boolean {
  return toDisplayText(v) === '';
}

function strLen(s: string) {
  return Array.from(s).length;
}
function cutToLen(s: string, max?: number) {
  if (!max || max < 0) return s;
  const arr = Array.from(s);
  return arr.slice(0, max).join('');
}

// Use the configurable buffer strategy from props.
const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: string | null) => {
    if (v == null) return null;
    let s = String(v);
    if (props.trimOnBlur === 'both') s = s.trim();
    s = cutToLen(s, props.maxLength);
    if (s === '' && props.nullable) return null;
    return s;
  },
  equals: (a: string | null, b: string | null) => a === b,
}));

type AutosizeOpts = boolean | { minRows?: number; maxRows?: number };

function resolveAutosizeBounds(autosize: AutosizeOpts | undefined, rows: number): {
  enabled: boolean;
  minRows: number;
  maxRows: number;
} {
  const baseRows = rows > 0 ? rows : 3;
  if (autosize === false || autosize == null) {
    return { enabled: false, minRows: baseRows, maxRows: baseRows };
  }
  if (autosize === true) {
    return { enabled: true, minRows: baseRows, maxRows: Math.max(baseRows, 20) };
  }
  const minRows = autosize.minRows != null && autosize.minRows > 0 ? autosize.minRows : baseRows;
  const maxRows =
    autosize.maxRows != null && autosize.maxRows > 0 ? Math.max(minRows, autosize.maxRows) : Math.max(minRows, 20);
  return { enabled: true, minRows, maxRows };
}

function applyTextareaAutosize(el: HTMLTextAreaElement, minRows: number, maxRows: number): void {
  const style = typeof getComputedStyle === 'function' ? getComputedStyle(el) : null;
  const lineHeight = style ? parseFloat(style.lineHeight) : NaN;
  const paddingTop = style ? parseFloat(style.paddingTop) : 0;
  const paddingBottom = style ? parseFloat(style.paddingBottom) : 0;
  const lh = Number.isFinite(lineHeight) && lineHeight > 0 ? lineHeight : 20;
  const pad =
    (Number.isFinite(paddingTop) ? paddingTop : 0) + (Number.isFinite(paddingBottom) ? paddingBottom : 0);
  const minH = lh * minRows + pad;
  const maxH = lh * maxRows + pad;
  el.style.height = 'auto';
  const next = Math.min(maxH, Math.max(minH, el.scrollHeight));
  el.style.height = `${next}px`;
  el.style.overflowY = el.scrollHeight > maxH + 1 ? 'auto' : 'hidden';
}

const ChoyTextCell = defineComponent({
  name: 'ChoyTextCell',
  inheritAttrs: false,
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    placeholder: String,
    rows: Number,
    autosize: [Boolean, Object],
    maxlength: [Number, String],
    showWordLimit: Boolean,
  },
  setup(p, { attrs }) {
    const modelRef = computed<ViewType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v;
      },
    });
    const buffer = useBufferedCommit<ViewType>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );
    const onBlur = () => buffer.onBlur();
    let taEl: HTMLTextAreaElement | null = null;

    function bindTextareaEl(el: unknown) {
      if (el == null) {
        taEl = null;
        return;
      }
      if ((el as HTMLElement).tagName === 'TEXTAREA') {
        taEl = el as HTMLTextAreaElement;
        return;
      }
      const root = (el as { $el?: HTMLElement }).$el ?? (el as HTMLElement);
      taEl =
        root?.tagName === 'TEXTAREA'
          ? (root as HTMLTextAreaElement)
          : ((root as HTMLElement | null)?.querySelector?.('textarea') as HTMLTextAreaElement | null) ?? null;
    }

    function syncAutosize() {
      if (!taEl) return;
      const bounds = resolveAutosizeBounds(p.autosize as AutosizeOpts | undefined, p.rows ?? 3);
      if (!bounds.enabled) {
        taEl.style.height = '';
        taEl.style.overflowY = '';
        return;
      }
      applyTextareaAutosize(taEl, bounds.minRows, bounds.maxRows);
    }

    return () => {
      const text = buffer.editingValue.value ?? '';
      const max =
        p.maxlength == null || p.maxlength === '' ? undefined : Number(p.maxlength);
      const showLimit = !!p.showWordLimit && max != null && Number.isFinite(max) && max > 0;
      const bounds = resolveAutosizeBounds(p.autosize as AutosizeOpts | undefined, p.rows ?? 3);
      const baseClass = showLimit ? 'choy-textarea w-full pb-6' : 'choy-textarea w-full';
      const textarea = h(Textarea, {
        ...attrs,
        class: bounds.enabled ? `${baseClass} resize-none` : baseClass,
        placeholder: p.placeholder,
        rows: bounds.minRows,
        maxlength: p.maxlength,
        modelValue: text,
        'onUpdate:modelValue': (v: string) => {
          buffer.setEditing(v ?? '');
          queueMicrotask(syncAutosize);
        },
        onBlur,
        onVnodeMounted: (vnode: { el?: unknown }) => {
          bindTextareaEl(vnode.el);
          queueMicrotask(syncAutosize);
        },
        onVnodeUpdated: (vnode: { el?: unknown }) => {
          bindTextareaEl(vnode.el);
          queueMicrotask(syncAutosize);
        },
      });
      if (!showLimit) return textarea;
      return h('div', { class: 'choy-textarea-wrap relative w-full' }, [
        textarea,
        h(
          'span',
          {
            class:
              'choy-textarea-word-limit pointer-events-none absolute bottom-2 end-2 text-xs tabular-nums text-muted-foreground',
            'aria-hidden': 'true',
          },
          `${strLen(text)} / ${max}`,
        ),
      ]);
    };
  },
});

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null) {
      if (props.nullable) return cb();
      return cb(new Error(_t('Cannot be empty')));
    }
    if (typeof value !== 'string') return cb(new Error(_t('Must be a string')));
    if (props.maxLength != null) {
      const n = strLen(value);
      if (n > props.maxLength) return cb(new Error(_t('Length must not exceed %s', props.maxLength)));
    }
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

