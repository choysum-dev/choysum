<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="formItemProps"
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
    <!-- Form and inline editing: normalize and compare using the current record/row scale -->
    <template #edit="{ fieldValue, record }">
      <ODecimalCell :field-value="fieldValue" :options="makeBufferOptions(() => resolveScaleFrom(record().value))" :placeholder="placeholder" v-bind="$attrs" />
    </template>

    <!-- Display: format with dynamic scale (rounding plus fixed decimal places) -->
    <template #display="{ fieldValue, record }">
      <span class="o-field-display-text">{{
        toDisplayText(
          resolveDisplayValue(fieldValue().value, record().value),
          () => resolveScaleFrom(record().value),
          record().value
        )
      }}</span>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, Decimal | null>, V = FieldPathType<T, P>">
import { ref, computed, watch, defineComponent, h, onMounted } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElInput, type FormItemProps } from 'element-plus';
import { useField } from '@/web/web/composables/useField';
import type { UseField, AggProp } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import Decimal, { isDecimal } from '@/core/utils/decimal';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore, formatFixedDecimalString, formatCurrencyFromConfig } from '@/web/web/stores/i18nStore';

const { _t } = createTranslate('web', { scope: 'web/components/field/OMonetaryField' });


defineOptions({ name: 'OMonetaryField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    precision?: number;
    scale?: number;
    min?: string | number | Decimal;
    max?: string | number | Decimal;
    roundingMode?: Decimal.Rounding;
    nullable?: boolean;
    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    placeholder?: string;
    formItemProps?: Partial<FormItemProps>;
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
    // Support aggregates, used to read metrics in grouped mode
    agg?: AggProp;
    // Added: render mode and inline error support
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    roundingMode: Decimal.ROUND_HALF_UP,
    nullable: true,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    placeholder: '',
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'idle',
    bufferIdleDelay: 320,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

// Binding, passing agg through to useField for aggregate registration
const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

/* ================== Currency sibling registration ================== */
const currencyFieldName = computed<string | undefined>(() => {
  const raw = (binding.meta as any)?.currencyField as string | undefined;
  const trimmed = typeof raw === 'string' ? raw.trim() : '';
  return trimmed || undefined;
});

function registerSiblingCurrencyField() {
  const s = currencyFieldName.value;
  if (!s) return;
  const segs = String(binding.prop).split('.').filter(Boolean);
  if (!segs.length) return;
  segs[segs.length - 1] = s;
  binding.registerFields(segs.join('.'));
  // Prefetch common currency display fields when the relation is expanded on the record.
  binding.registerFields(`${segs.join('.')}.DecimalDigits`);
  binding.registerFields(`${segs.join('.')}.Symbol`);
  binding.registerFields(`${segs.join('.')}.Code`);
}
onMounted(() => registerSiblingCurrencyField());
watch([currencyFieldName, () => String(binding.prop)], () => registerSiblingCurrencyField());

/* ================== Precision / rounding config (field-level) ================== */
const metaPrecision = computed<number | undefined>(() => (binding.meta as any)?.precision);
const effectivePrecision = computed<number>(() => metaPrecision.value ?? props.precision ?? 38);

// P0 monetary always HALF_UP (D7); props.roundingMode remains an escape hatch for tests.
const effectiveRound = computed<Decimal.Rounding>(() => props.roundingMode ?? Decimal.ROUND_HALF_UP);

/* ================== Helper: resolve currency digits / symbol ================== */
function getByPath(obj: any, path: string) {
  return String(path)
    .split('.')
    .filter(Boolean)
    .reduce((a, k) => (a == null ? a : a[k]), obj);
}

function resolveCurrencyValue(obj: any): any {
  const s = currencyFieldName.value;
  if (!s || !obj) return undefined;
  const segs = String(binding.prop).split('.').filter(Boolean);
  if (segs.length) {
    segs[segs.length - 1] = s;
    const viaFullPath = getByPath(obj, segs.join('.'));
    if (viaFullPath != null) return viaFullPath;
  }
  return obj?.[s];
}

function readCurrencyDigits(currency: any): number | undefined {
  if (currency == null) return undefined;
  if (typeof currency === 'object') {
    const n = Number(currency.DecimalDigits ?? currency.decimalDigits);
    if (Number.isInteger(n) && n >= 0 && n <= 18) return n;
  }
  return undefined;
}

function readCurrencyCode(currency: any): string | undefined {
  if (!currency || typeof currency !== 'object') return undefined;
  const code = currency.Code ?? currency.code;
  return typeof code === 'string' && code.trim() ? code.trim() : undefined;
}

function readCurrencySymbol(currency: any): string | undefined {
  if (!currency || typeof currency !== 'object') return undefined;
  const symbol = currency.Symbol ?? currency.symbol;
  return typeof symbol === 'string' && symbol.trim() ? symbol.trim() : undefined;
}

/** Digits from Currency.DecimalDigits; when missing, do not invent digits for write — display falls back to 2. */
function resolveScaleFrom(obj: any): number {
  try {
    const digits = readCurrencyDigits(resolveCurrencyValue(obj));
    if (digits != null) return digits;
  } catch {}
  return props.scale ?? 2;
}

/* ================== Aggregate value resolution (display-mode fallback) ================== */
function leafOf(path: string): string {
  const segs = String(path).split('.').filter(Boolean);
  return segs.length ? segs[segs.length - 1]! : String(path);
}

function resolveDisplayValue(raw: any, obj: any): any {
  if (raw != null && raw !== '') return raw;
  if (!obj) return raw;

  const path = String(binding.prop || '');
  const leaf = leafOf(path);
  const metrics = obj?.metrics && typeof obj.metrics === 'object' ? (obj.metrics as Record<string, any>) : null;

  // Aggregate declaration from field props, used to build preferred candidate keys
  const agg = props.agg as AggProp | undefined;
  const suffixOf = (fn: string) => (fn === 'count' ? '__count' : `__${fn}`);

  const candidates: string[] = [];
  if (agg) {
    if (typeof agg === 'string') {
      const suf = suffixOf(agg);
      if (suf === '__count') {
        candidates.push('__count');
      } else {
        candidates.push(`${path}${suf}`, `${leaf}${suf}`);
      }
    } else if (agg.agg) {
      if (agg.alias && String(agg.alias).trim()) candidates.push(String(agg.alias).trim());
      const suf = suffixOf(agg.agg);
      if (suf === '__count') {
        candidates.push('__count');
      } else {
        candidates.push(`${path}${suf}`, `${leaf}${suf}`);
      }
    }
  }

  // Try candidate keys in order: metrics first, then top-level mirrors
  for (const k of candidates) {
    if (!k) continue;
    if (metrics && metrics[k] != null) return metrics[k];
    if (obj[k] != null) return obj[k];
  }

  // Without agg or when candidates miss, try matching a unique key in metrics by path/leaf prefix to avoid false positives
  const AGG_SUFFIX = ['__sum', '__avg', '__min', '__max', '__count', '__count_distinct'];
  // Pure aggregate suffixes: ignore __count so count does not skew unique matching
  const PURE_AGG_SUFFIX = ['__sum', '__avg', '__min', '__max', '__count_distinct'];
  if (metrics) {
    const mkeys = Object.keys(metrics);
    // Prefer strict matching: exclude __count and only consider unique metrics for this field
    const strictByPath = mkeys.filter(k => k.startsWith(path + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
    if (strictByPath.length === 1 && metrics[strictByPath[0]] != null) return metrics[strictByPath[0]];
    const strictByLeaf = mkeys.filter(k => k.startsWith(leaf + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
    if (strictByLeaf.length === 1 && metrics[strictByLeaf[0]] != null) return metrics[strictByLeaf[0]];
    // Fallback: preserve the legacy logic that includes __count for backward compatibility
    const byPath = mkeys.filter(k => k === '__count' || k.startsWith(path + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
    const byLeaf = mkeys.filter(k => k === '__count' || k.startsWith(leaf + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
    const uniq = byPath.length === 1 ? byPath[0] : byLeaf.length === 1 ? byLeaf[0] : null;
    if (uniq && metrics[uniq] != null) return metrics[uniq];
  }

  // Retry a unique match at the top level (obj is non-null after the early return above)
  const tkeys = Object.keys(obj);
  // Prefer strict top-level matching without __count
  const tStrictByPath = tkeys.filter(k => k.startsWith(path + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
  if (tStrictByPath.length === 1 && obj[tStrictByPath[0]] != null) return obj[tStrictByPath[0]];
  const tStrictByLeaf = tkeys.filter(k => k.startsWith(leaf + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
  if (tStrictByLeaf.length === 1 && obj[tStrictByLeaf[0]] != null) return obj[tStrictByLeaf[0]];
  // Fallback to the legacy logic that allows __count
  const tByPath = tkeys.filter(k => k === '__count' || k.startsWith(path + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
  const tByLeaf = tkeys.filter(k => k === '__count' || k.startsWith(leaf + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
  const tUniq = tByPath.length === 1 ? tByPath[0] : tByLeaf.length === 1 ? tByLeaf[0] : null;
  if (tUniq && obj[tUniq] != null) return obj[tUniq];

  return raw;
}

/* ================== Numeric helpers ================== */
function asDecimal(v: unknown): Decimal | null {
  try {
    if (v == null || v === '') return null;
    if (isDecimal(v)) return v as Decimal;
    return new Decimal(v as any);
  } catch {
    return null;
  }
}
const minD = computed<Decimal | null>(() => (props.min === undefined ? null : asDecimal(props.min)));
const maxD = computed<Decimal | null>(() => (props.max === undefined ? null : asDecimal(props.max)));

/* ================== View mapping ================== */
const toView = (raw: any): ViewType => {
  if (raw == null) return null;
  const d = asDecimal(raw);
  return d ? d.toString() : null;
};
const fromView = (v: ViewType) => {
  if (v == null || v === '') return null as unknown as V;
  const d = asDecimal(v);
  return (d ?? null) as unknown as V;
};

// Display: currency symbol/code + quantized amount (formatCurrency when code is known).
function toDisplayText(v: any, getScale?: () => number, record?: any) {
  if (v == null || v === '') return '';
  const d = asDecimal(v);
  if (!d) return '';
  const scale = typeof getScale === 'function' ? getScale() : resolveScaleFrom(record);
  try {
    const q = d.toDecimalPlaces(scale, effectiveRound.value);
    const currency = resolveCurrencyValue(record);
    const code = readCurrencyCode(currency);
    const symbol = readCurrencySymbol(currency);
    try {
      const i18n = useI18nStore();
      const numberFormat = { ...(i18n.currentLocale?.numberFormat || {}), decimalDigits: scale };
      if (code) {
        return formatCurrencyFromConfig(Number(q.toString()), numberFormat, code);
      }
      const fixed = q.toFixed(scale);
      const formatted = formatFixedDecimalString(fixed, numberFormat);
      return symbol ? `${symbol} ${formatted}` : formatted;
    } catch {
      const fixed = q.toFixed(scale);
      return symbol ? `${symbol} ${fixed}` : fixed;
    }
  } catch {
    return d.toString();
  }
}

/* ================== Buffer options (inject getScale from context) ================== */
function makeBufferOptions(getScale: () => number) {
  return {
    strategy: props.bufferStrategy!,
    idleDelay: props.bufferIdleDelay,
    commitOnBlur: props.commitOnBlur,
    normalize: (v: Decimal | null) => (v == null ? null : clampValue(v, getScale())),
    equals: (a: Decimal | null, b: Decimal | null) => {
      const s = getScale();
      const qa = quantizeForCompare(a, s);
      const qb = quantizeForCompare(b, s);
      if (qa == null && qb == null) return true;
      if (qa == null || qb == null) return false;
      return qa.eq(qb);
    },
    getScale,
  };
}

/* ================== Editable cell (uses dynamic scale) ================== */
const ODecimalCell = defineComponent({
  name: 'OMonetaryCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true }, // Must include getScale()
    placeholder: String,
  },
  setup(p, { attrs }) {
    const modelRef = computed<any>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v;
      },
    });

    const editingRaw = ref<string | null>(null);
    watch(
      () => modelRef.value,
      nv => {
        if (isIntermediateInput(editingRaw.value)) return;
        editingRaw.value = toView(nv);
      },
      { immediate: true }
    );

    const buffer = useBufferedCommit<Decimal | null>(
      () => {
        const raw = modelRef.value;
        if (raw == null || raw === '') return null;
        if (raw instanceof Decimal) return raw;
        return asDecimal(raw);
      },
      v => {
        modelRef.value = v as any;
      },
      p.options as any
    );

    function currentScale(): number {
      const fn = (p.options as any)?.getScale as (() => number) | undefined;
      try {
        const n = fn ? fn() : undefined;
        if (typeof n === 'number' && Number.isInteger(n) && n >= 0 && n <= 18) return n;
      } catch {}
      return 6;
    }

    function onInput(raw: string) {
      if (raw === '' && /* nullable comes from parent props */ true) {
        editingRaw.value = null;
        buffer.setEditing(null);
        return;
      }
      if (!/^[-]?\d*(\.\d*)?$/.test(raw)) return;
      editingRaw.value = raw;
      if (isIntermediateInput(raw)) return;
      const d = parseStrict(raw, currentScale());
      if (!d) return;
      buffer.setEditing(clampValue(d, currentScale()));
    }

    function onBlur() {
      const raw = editingRaw.value;
      const scale = currentScale();
      if (raw == null) {
        buffer.setEditing(null);
        buffer.onBlur();
        return;
      }
      if (isIntermediateInput(raw)) {
        const d = parseStrict(raw.replace(/\.$/, ''), scale);
        if (!d) {
          buffer.setEditing(null);
          buffer.onBlur();
          return;
        }
        buffer.setEditing(clampValue(d, scale));
        buffer.onBlur();
        return;
      }
      const d = parseStrict(raw, scale);
      if (d) buffer.setEditing(clampValue(d, scale));
      buffer.onBlur();
    }

    return () =>
      h(ElInput, {
        ...attrs,
        class: 'o-input o-monetary-input',
        modelValue: editingRaw.value,
        placeholder: p.placeholder,
        inputmode: 'decimal',
        'onUpdate:modelValue': (val: any) => onInput(val),
        onBlur,
      });
  },
});

/* ================== Validation / quantization helpers ================== */
function isIntermediateInput(s: string | null): boolean {
  if (s == null) return false;
  if (s === '-' || s === '.' || s === '-.' || /^-?\d+\.$/.test(s)) return true;
  return false;
}

function parseStrict(s: string, scale: number): Decimal | null {
  const t = s.trim();
  if (!/^[-]?\d*(\.\d*)?$/.test(t)) return null;
  try {
    const d = new Decimal(t);
    if (!d.isFinite()) return null;
    const places = d.decimalPlaces();
    if (places != null && places > scale) return null;
    const digits = d.abs().sd(true);
    if (digits != null && digits > effectivePrecision.value) return null;
    if (minD.value && d.lessThan(minD.value)) return null;
    if (maxD.value && d.greaterThan(maxD.value)) return null;
    return d;
  } catch {
    return null;
  }
}

function clampValue(d: Decimal, scale: number): Decimal {
  let v = d.toDecimalPlaces(scale, effectiveRound.value);
  const digits = v.abs().sd(true) ?? 0;
  if (digits > effectivePrecision.value) {
    const shift = digits - effectivePrecision.value;
    v = v.div(new Decimal(10).pow(shift)).toDecimalPlaces(scale, effectiveRound.value);
  }
  if (minD.value) v = Decimal.max(v, minD.value);
  if (maxD.value) v = Decimal.min(v, maxD.value);
  return v;
}

// Used only for equals comparisons; do not clamp to min/max
function quantizeForCompare(x: any, scale: number): Decimal | null {
  const d = asDecimal(x);
  if (!d) return null;
  return d.toDecimalPlaces(scale, effectiveRound.value);
}

function isValidValue(value: any, scale: number): string | null {
  if (value == null || value === '') return null;
  const d = asDecimal(value);
  if (!d || !d.isFinite()) return _t('Must be a valid number');
  if ((d.decimalPlaces() ?? 0) > scale) return _t('Decimal places must not exceed %s', scale);
  const digits = d.abs().sd(true) ?? 0;
  if (digits > effectivePrecision.value) return _t('Total digits must not exceed %s', effectivePrecision.value);
  if (minD.value && d.lessThan(minD.value)) return _t('Must not be less than %s', minD.value.toString());
  if (maxD.value && d.greaterThan(maxD.value)) return _t('Must not be greater than %s', maxD.value.toString());
  return null;
}

/* ================== Form rule (validate against dynamic scale from the current record) ================== */
const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    // Form rules cannot access a row object, so derive scale from the root record here
    const rec = binding.recordRef().value;
    const scale = resolveScaleFrom(rec);
    const msg = isValidValue(value, scale);
    if (msg) return cb(new Error(msg));
    cb();
  },
  trigger: 'blur',
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style scoped lang="scss">
.o-field-display-text {
  line-height: var(--el-component-size-base, 32px);
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 11px;
  text-align: right;
}
.o-monetary-input {
  width: 100%;
  :deep(.el-input__inner) {
    text-align: right;
  }
  :deep(.el-input__wrapper input) {
    text-align: right;
  }
}
</style>
