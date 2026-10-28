<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :help="help"
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
    <!-- Reuse the slot-provided fieldValue for both form and row rendering. -->
    <template #edit="{ fieldValue, inputId, describedBy, invalid}">
      <ChoyVarcharCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :maxlength="effectiveMaxLength ?? undefined"
        :show-word-limit="showWordLimit && effectiveMaxLength != null"
        :input-type="inputType"
        :force-trim-commit="shouldTrimTrailing"
        v-bind="$attrs"
        :id="inputId"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined"
      />
    </template>
    <template #display="{ fieldValue }">
      <div
        v-if="isEmptyDisplay(fieldValue().value) && showFormEmptyPlaceholder"
        class="choy-varchar-display choy-varchar-display--empty text-sm text-muted-foreground"
      >
        {{ emptyDisplayText }}
      </div>
      <div
        v-else-if="copyable && !isEmptyDisplay(fieldValue().value) && showFormEmptyPlaceholder"
        class="choy-varchar-display choy-varchar-display--copyable inline-flex max-w-full items-center gap-1"
      >
        <span
          class="choy-field-display-text inline-block min-w-0 max-w-full truncate whitespace-nowrap text-foreground"
          :title="toDisplayText(fieldValue().value) || undefined"
          >{{ toDisplayText(fieldValue().value) }}</span
        >
        <ChoyButton
          type="button"
          size="sm"
          variant="ghost"
          class="choy-varchar-copy-btn h-6 w-6 shrink-0 p-0 text-muted-foreground hover:text-foreground"
          :aria-label="copyAriaLabel"
          :title="copyAriaLabel"
          @click="copyDisplayValue(fieldValue().value)"
        >
          <Copy class="size-3.5" />
        </ChoyButton>
      </div>
      <span
        v-else
        class="choy-field-display-text inline-block max-w-full truncate whitespace-nowrap text-foreground"
        :title="toDisplayText(fieldValue().value) || undefined"
        >{{ toDisplayText(fieldValue().value) }}</span
      >
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { computed, defineComponent, h } from 'vue';
import { Copy } from 'lucide-vue-next';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Non-numeric varchar aggregates are narrowed to count_distinct.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import Input from '../vendor/ui/input/Input.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyVarcharField' });

defineOptions({ name: 'ChoyVarcharField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type ViewType = string | null;
type VarcharWidget = 'text' | 'password' | 'email' | 'url';

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    /** Explicit help tip; preferred over FieldsGet / meta.help when set. */
    help?: string;
    rules?: RuleItem[];

    maxLength?: number;
    nullable?: boolean;
    trimOnBlur?: 'none' | 'both';
    /** Strip trailing spaces (char fixed-length padding). Defaults on when meta.type is char. */
    trimTrailingSpaces?: boolean;
    placeholder?: string;
    showWordLimit?: boolean;
    widget?: VarcharWidget;
    /**
     * HTML input type alias for `widget` (`password` | `email` | `url` | `text`).
     * Prefer `widget` in new call sites; `type` is kept so auth forms and attrs keep working.
     */
    type?: VarcharWidget;
    /** Form display: show a copy button next to non-empty values. */
    copyable?: boolean;
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
    // Only count_distinct is supported for varchar aggregates.
    agg?: NarrowAggProp<NonNumericAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    nullable: true,
    trimOnBlur: 'none',
    placeholder: '',
    showWordLimit: true,
    widget: 'text',
    copyable: false,

    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,

    formItemProps: () => ({}),
    vColumnProps: () => ({}),

    bufferStrategy: 'idle',
    bufferIdleDelay: 360,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;
const metaSize = computed<number | undefined>(() => binding.meta?.size as any);
const effectiveMaxLength = computed(() => props.maxLength ?? metaSize.value ?? undefined);

const shouldTrimTrailing = computed(() => {
  if (props.trimTrailingSpaces === true) return true;
  if (props.trimTrailingSpaces === false) return false;
  return (binding.meta as { type?: string } | undefined)?.type === 'char';
});

const inputType = computed(() => {
  const widget = props.widget ?? 'text';
  const typeAlias = props.type;
  // Non-text type= (auth forms) must win over the widget default 'text'.
  // Explicit widget=password|email|url still wins when set.
  if (
    (typeAlias === 'password' || typeAlias === 'email' || typeAlias === 'url') &&
    widget === 'text'
  ) {
    return typeAlias;
  }
  if (widget === 'password' || widget === 'email' || widget === 'url') return widget;
  return 'text';
});

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

const copyAriaLabel = computed(() => _t('Copy'));

function normalizeString(raw: unknown): ViewType {
  if (raw == null) return null;
  let s = String(raw);
  if (shouldTrimTrailing.value) s = s.trimEnd();
  return s;
}

const toView = (raw: any): ViewType => normalizeString(raw);
const fromView = (v: ViewType) => v as unknown as V;
const toDisplayText = (v: ViewType) => {
  if (v == null) return '';
  return shouldTrimTrailing.value ? String(v).trimEnd() : String(v);
};

function isEmptyDisplay(v: ViewType): boolean {
  const text = toDisplayText(v);
  return text === '';
}

async function copyDisplayValue(v: ViewType): Promise<void> {
  const text = toDisplayText(v);
  if (!text) return;
  try {
    const clipboard = globalThis.navigator?.clipboard;
    if (!clipboard?.writeText) {
      throw new Error('clipboard unavailable');
    }
    await clipboard.writeText(text);
    ChoyMessage.success(_t('Copied'));
  } catch {
    ChoyMessage.error(_t('Copy failed'));
  }
}

function strLen(s: string) {
  return Array.from(s).length;
}
function cutToLen(s: string, max?: number) {
  if (!max || max < 0) return s;
  const arr = Array.from(s);
  return arr.slice(0, max).join('');
}

/**
 * Delay buffered-commit creation until the cell component has access to fieldValue.
 */
const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: string | null) => {
    if (v == null) return null;
    let s = String(v);
    if (shouldTrimTrailing.value) s = s.trimEnd();
    if (props.trimOnBlur === 'both') s = s.trim();
    s = cutToLen(s, effectiveMaxLength.value);
    if (s === '' && props.nullable) return null;
    return s;
  },
  equals: (a: string | null, b: string | null) => {
    if (a !== b) return false;
    // View values can already be trimEnd'd while the store still holds char pads.
    if (shouldTrimTrailing.value) {
      const raw = binding.fieldRef?.()?.value;
      if (typeof raw === 'string' && raw !== raw.trimEnd()) return false;
    }
    return true;
  },
}));

/**
 * Internal cell component with per-cell buffering backed by ChoyFieldBase fieldValue.
 */
const ChoyVarcharCell = defineComponent({
  name: 'ChoyVarcharCell',
  inheritAttrs: false,
  props: {
    fieldValue: { type: Function, required: true }, // () => WritableComputedRef<ViewType>
    options: { type: Object, required: true },
    placeholder: String,
    maxlength: [Number, String],
    showWordLimit: Boolean,
    inputType: { type: String, default: 'text' },
    forceTrimCommit: Boolean,
  },
  setup(p, { attrs }) {
    // Adapt useBufferedCommit to the slot-provided getter and setter.
    const modelRef = computed<ViewType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v as any;
      },
    });

    const buffer = useBufferedCommit<ViewType>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );

    function onBlur() {
      // Mark dirty so normalize + equals can flush char pads even when the view looks unchanged.
      if (p.forceTrimCommit) {
        buffer.setEditing(buffer.editingValue.value);
      }
      buffer.onBlur();
    }

    return () => {
      const text = buffer.editingValue.value ?? '';
      const max =
        p.maxlength == null || p.maxlength === '' ? undefined : Number(p.maxlength);
      const showLimit = !!p.showWordLimit && max != null && Number.isFinite(max) && max > 0;
      const rest = { ...(attrs as Record<string, unknown>) };
      delete rest.type;
      const input = h(Input, {
        ...rest,
        type: p.inputType || 'text',
        class: showLimit ? 'choy-varchar-input w-full pe-14' : 'choy-varchar-input w-full',
        placeholder: p.placeholder,
        maxlength: p.maxlength,
        modelValue: text,
        'onUpdate:modelValue': (v: string) => buffer.setEditing(v ?? ''),
        onBlur,
      });
      if (!showLimit) return input;
      return h('div', { class: 'choy-varchar-input-wrap relative w-full' }, [
        input,
        h(
          'span',
          {
            class:
              'choy-varchar-word-limit pointer-events-none absolute end-2 top-1/2 -translate-y-1/2 text-xs tabular-nums text-muted-foreground',
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
      return cb(new Error(_t('Value is required')));
    }
    if (typeof value !== 'string') return cb(new Error(_t('Value must be a string')));
    const normalized = shouldTrimTrailing.value ? value.trimEnd() : value;
    const n = strLen(normalized);
    const N = effectiveMaxLength.value;
    if (N != null && n > N) return cb(new Error(_t('Length must not exceed %s', N)));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>
