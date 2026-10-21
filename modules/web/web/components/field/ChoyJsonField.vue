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
    <template #edit="{ fieldValue, inputId, describedBy, invalid }">
      <ChoyJsonCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="effectivePlaceholder"
        :nullable="nullable"
        :allow-array="allowArray"
        :id="inputId"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined"
      />
    </template>

    <template #display="{ fieldValue }">
      <pre
        v-if="isTableLike"
        class="choy-json-display whitespace-pre-wrap wrap-break-word px-2 py-1 font-mono leading-snug"
        >{{ displayString(fieldValue().value) }}</pre
      >
      <div
        v-else-if="normalizeIncoming(fieldValue().value) == null"
        class="choy-json-display choy-json-display--empty min-h-[1.4em] rounded-md border border-dashed border-border px-3 py-2 text-sm text-muted-foreground"
      >
        {{ _t('No content') }}
      </div>
      <div
        v-else
        class="choy-json-pretty-wrap overflow-hidden rounded-md border border-border bg-muted/20"
      >
        <VueJsonPretty
          class="choy-json-pretty w-full wrap-break-word px-3 py-2 text-[13px] leading-snug"
          :data="normalizeIncoming(fieldValue().value)!"
          :deep="prettyDeep"
          :show-length="true"
          :show-line="true"
          :collapsed-on-click-brackets="true"
        />
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, Record<string, any> | any[] | null | undefined>, V = FieldPathType<T, P>">
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RuleItem } from 'async-validator';
import { ref, watch, computed, defineComponent, h, onBeforeUnmount, nextTick } from 'vue';
import VueJsonPretty from 'vue-json-pretty';
import 'vue-json-pretty/lib/styles.css';
import { Braces } from 'lucide-vue-next';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';
import { mountJsonCodeMirror, type JsonCodeMirrorHandle } from './jsonCodeMirror';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyJsonField' });

// Keep lint tip msgids extractable (runtime translate() passes them dynamically).
void [
  _t('Invalid JSON'),
  _t('Unexpected end of JSON'),
  _t('Unexpected character after JSON (line %s, column %s)'),
  _t('Unexpected character after JSON (position %s)'),
  _t('Invalid JSON near line %s, column %s'),
  _t('Invalid JSON at position %s'),
];

defineOptions({ name: 'ChoyJsonField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type JsonVal = Record<string, any> | any[] | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    nullable?: boolean;
    allowArray?: boolean;
    placeholder?: string;
    /** @deprecated Ignored; CodeMirror sizes via CSS min-height. */
    autosize?: boolean | { minRows?: number; maxRows?: number };
    /** Collapse depth for form readonly vue-json-pretty tree. */
    prettyDeep?: number;

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
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    nullable: true,
    allowArray: false,
    placeholder: '',
    prettyDeep: 3,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'blur',
    bufferIdleDelay: 500,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  },
);

const effectivePlaceholder = computed(() => props.placeholder || _t('Enter a JSON object'));

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const isTableLike = computed(() => {
  const mode = props.renderMode;
  if (mode === 'table' || mode === 'inline') return true;
  if (mode === 'form') return false;
  return binding.env?.isForm === false;
});

const toView = (raw: any): JsonVal => {
  if (raw == null) return null;
  if (typeof raw === 'object') return raw as any;
  try {
    const v = JSON.parse(String(raw));
    return typeof v === 'object' ? v : null;
  } catch {
    return null;
  }
};
const fromView = (v: JsonVal) => v as unknown as V;

function stableStringify(v: any): string {
  try {
    if (v == null) return '';
    if (Array.isArray(v)) {
      return JSON.stringify(v, null, 2);
    }
    return JSON.stringify(v, Object.keys(v || {}).sort(), 2);
  } catch {
    return '';
  }
}

function normalizeIncoming(v: any): JsonVal {
  if (v == null) return null;
  if (typeof v === 'object') return v as any;
  if (typeof v === 'string') {
    try {
      const parsed = JSON.parse(v);
      if (parsed === null) return null;
      if (typeof parsed === 'object') return parsed as any;
      return null;
    } catch {
      return null;
    }
  }
  return null;
}

function displayString(v: any): string {
  if (v == null) return '';
  const norm = normalizeIncoming(v);
  return norm == null ? '' : stableStringify(norm);
}

function tryParse(raw: string): { ok: boolean; val?: any; err?: string } {
  try {
    const v = JSON.parse(raw);
    if (v === null) return { ok: true, val: null };
    const isArr = Array.isArray(v);
    const isObj = typeof v === 'object' && !isArr;
    if (isArr && !props.allowArray) return { ok: false, err: _t('Arrays are not allowed') };
    if (!isArr && !isObj) return { ok: false, err: _t('Must be an object') };
    return { ok: true, val: v };
  } catch {
    return { ok: false, err: _t('JSON parse failed') };
  }
}

function jsonEquals(a: any, b: any) {
  if (a === b) return true;
  try {
    return JSON.stringify(a) === JSON.stringify(b);
  } catch {
    return false;
  }
}

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (typeof value === 'string') {
      try {
        value = JSON.parse(value);
      } catch {
        /* ignore */
      }
    }
    if (value == null) {
      if (props.nullable) return cb();
      return cb(new Error(_t('Cannot be empty')));
    }
    if (typeof value !== 'object') return cb(new Error(_t('Must be an object')));
    if (Array.isArray(value) && !props.allowArray) return cb(new Error(_t('Arrays are not allowed')));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: JsonVal) => v,
  equals: jsonEquals,
}));

const ChoyJsonCell = defineComponent({
  name: 'ChoyJsonCell',
  inheritAttrs: false,
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    placeholder: String,
    nullable: Boolean,
    allowArray: Boolean,
  },
  setup(p, { attrs }) {
    const modelRef = computed<JsonVal>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v as any;
      },
    });

    const editingText = ref<string>('');
    const textDirty = ref(false);
    const parseError = ref<string | null>(null);
    const hostEl = ref<HTMLElement | null>(null);
    let cm: JsonCodeMirrorHandle | null = null;
    let mounting = false;

    const buffer = useBufferedCommit<JsonVal>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any,
    );

    function syncFromModel(v: JsonVal) {
      if (textDirty.value) return;
      const norm = normalizeIncoming(v);
      if (v !== norm) modelRef.value = norm;
      const next = norm == null ? '' : stableStringify(norm);
      editingText.value = next;
      cm?.setDoc(next);
    }

    watch(
      () => modelRef.value,
      v => {
        syncFromModel(v);
      },
      { immediate: true },
    );

    function onInput(v: string) {
      editingText.value = v;
      textDirty.value = true;
      parseError.value = null;
    }

    function onBlur() {
      const raw = editingText.value;
      if (raw.trim() === '') {
        if (p.nullable) {
          buffer.setEditing(null);
          buffer.onBlur();
          textDirty.value = false;
          parseError.value = null;
          return;
        }
        parseError.value = _t('Cannot be empty');
        buffer.onBlur();
        return;
      }
      const parsed = tryParse(raw);
      if (!parsed.ok) {
        parseError.value = parsed.err || _t('Parse failed');
        buffer.onBlur();
        return;
      }
      parseError.value = null;
      const canonical = stableStringify(parsed.val);
      editingText.value = canonical;
      textDirty.value = false;
      cm?.setDoc(canonical);
      buffer.setEditing(parsed.val);
      buffer.onBlur();
    }

    function formatEditing(): void {
      const raw = cm ? cm.getDoc() : editingText.value;
      if (raw.trim() === '') {
        if (p.nullable) {
          parseError.value = null;
          editingText.value = '';
          cm?.setDoc('');
          return;
        }
        parseError.value = _t('Cannot be empty');
        return;
      }
      const parsed = tryParse(raw);
      if (!parsed.ok) {
        parseError.value = parsed.err || _t('Parse failed');
        return;
      }
      parseError.value = null;
      const canonical = stableStringify(parsed.val);
      editingText.value = canonical;
      textDirty.value = true;
      cm?.setDoc(canonical);
      buffer.setEditing(parsed.val);
    }

    function ensureMounted() {
      if (cm || mounting || !hostEl.value) return;
      mounting = true;
      const controlId = attrs.id != null && String(attrs.id) !== '' ? String(attrs.id) : '';
      const formDescribedBy = attrs['aria-describedby'] != null ? String(attrs['aria-describedby']) : '';
      const parent = hostEl.value as unknown as HTMLElement;
      cm = mountJsonCodeMirror({
        parent,
        doc: editingText.value ?? '',
        id: controlId || undefined,
        ariaDescribedBy: formDescribedBy || undefined,
        ariaInvalid: attrs['aria-invalid'] === true || attrs['aria-invalid'] === 'true',
        translate: (msgid, ...args) => _t(msgid, ...(args as any[])),
        onUpdate: (text: string) => {
          onInput(text);
        },
        onBlur: () => {
          onBlur();
        },
      });
      mounting = false;
    }

    watch(hostEl, el => {
      if (el) void nextTick(() => ensureMounted());
    });

    onBeforeUnmount(() => {
      cm?.destroy();
      cm = null;
    });

    return () => {
      const controlId = attrs.id != null && String(attrs.id) !== '' ? String(attrs.id) : '';
      const parseErrorId = controlId ? `${controlId}-parse-error` : '';
      const formDescribedBy = attrs['aria-describedby'] != null ? String(attrs['aria-describedby']) : '';
      const describedBy = [formDescribedBy, parseError.value ? parseErrorId : '']
        .map(s => s.trim())
        .filter(Boolean)
        .join(' ');
      const formInvalid = attrs['aria-invalid'] === true || attrs['aria-invalid'] === 'true';
      const formatLabel = _t('Format');
      return h(
        'div',
        {
          class: [
            'choy-json-cell choy-json-field__edit overflow-hidden rounded-md border bg-background',
            parseError.value || formInvalid ? 'border-destructive' : 'border-border',
          ],
        },
        [
          h(
            'div',
            {
              class: 'flex items-center justify-end gap-1 border-b border-border bg-muted/30 px-1.5 py-1',
              role: 'toolbar',
            },
            [
              h(
                'button',
                {
                  type: 'button',
                  class:
                    'choy-json-format-btn inline-flex h-7 items-center gap-1 rounded px-2 text-xs text-muted-foreground hover:bg-muted hover:text-foreground',
                  title: formatLabel,
                  'aria-label': formatLabel,
                  onClick: (e: Event) => {
                    e.preventDefault();
                    formatEditing();
                  },
                },
                [h(Braces, { class: 'size-3.5', 'aria-hidden': 'true' }), formatLabel],
              ),
            ],
          ),
          h('div', {
            class: 'choy-json-cm-host min-h-[8rem] w-full',
            'data-placeholder': p.placeholder || '',
            'aria-describedby': describedBy || undefined,
            ref: ((el: Element | null) => {
              hostEl.value = el instanceof HTMLElement ? el : null;
            }) as any,
          }),
          parseError.value
            ? h(
                'div',
                {
                  id: parseErrorId || undefined,
                  class: 'choy-json-err border-t border-border px-3 py-1.5 text-xs text-destructive',
                  role: 'alert',
                },
                parseError.value,
              )
            : null,
        ],
      );
    };
  },
});
</script>
