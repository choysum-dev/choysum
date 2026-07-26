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
    <template #edit="{ fieldValue, record }">
      <OMonetaryCell :field-value="fieldValue" :options="makeBufferOptions(() => resolveScaleFrom(record().value))" :placeholder="placeholder" v-bind="$attrs" />
    </template>

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
import Decimal from '@/core/utils/decimal';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore, formatFixedDecimalString, formatCurrencyFromConfig } from '@/web/web/stores/i18nStore';
import {
  asMonetaryDecimal,
  clampMonetaryValue,
  currencyFieldPaths,
  formatMonetaryDisplayText,
  isIntermediateMonetaryInput,
  parseStrictMonetary,
  quantizeMonetaryForCompare,
  resolveAggregateDisplayValue,
  resolveCurrencyValue,
  resolveMonetaryScaleFromRecord,
  validateMonetaryValue,
} from './omonetary_helpers';

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
    agg?: AggProp;
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

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const currencyFieldName = computed<string | undefined>(() => {
  const raw = (binding.meta as any)?.currencyField as string | undefined;
  const trimmed = typeof raw === 'string' ? raw.trim() : '';
  return trimmed || undefined;
});

function registerSiblingCurrencyField() {
  for (const path of currencyFieldPaths(String(binding.prop), currencyFieldName.value)) {
    binding.registerFields(path);
  }
}
onMounted(() => registerSiblingCurrencyField());
watch([currencyFieldName, () => String(binding.prop)], () => registerSiblingCurrencyField());

const metaPrecision = computed<number | undefined>(() => (binding.meta as any)?.precision);
const effectivePrecision = computed<number>(() => metaPrecision.value ?? props.precision ?? 38);
const effectiveRound = computed<Decimal.Rounding>(() => props.roundingMode);

function resolveScaleFrom(obj: any): number {
  return resolveMonetaryScaleFromRecord(obj, currencyFieldName.value, String(binding.prop), props.scale ?? 6);
}

function resolveDisplayValue(raw: any, obj: any): any {
  return resolveAggregateDisplayValue(raw, obj, { bindingProp: String(binding.prop || ''), agg: props.agg });
}

const minD = computed<Decimal | null>(() => (props.min === undefined ? null : asMonetaryDecimal(props.min)));
const maxD = computed<Decimal | null>(() => (props.max === undefined ? null : asMonetaryDecimal(props.max)));

const toView = (raw: any): ViewType => {
  if (raw == null) return null;
  const d = asMonetaryDecimal(raw);
  return d ? d.toString() : null;
};
const fromView = (v: ViewType) => {
  if (v == null || v === '') return null as unknown as V;
  const d = asMonetaryDecimal(v);
  return (d ?? null) as unknown as V;
};

function toDisplayText(v: any, getScale: () => number, record?: any) {
  const scale = getScale();
  try {
    const i18n = useI18nStore();
    return formatMonetaryDisplayText(v, {
      scale,
      roundingMode: effectiveRound.value,
      currency: resolveCurrencyValue(record, currencyFieldName.value, String(binding.prop)),
      formatters: {
        formatCurrencyFromConfig,
        formatFixedDecimalString,
        numberFormat: i18n.currentLocale?.numberFormat,
      },
    });
  } catch {
    return formatMonetaryDisplayText(v, {
      scale,
      roundingMode: effectiveRound.value,
      currency: resolveCurrencyValue(record, currencyFieldName.value, String(binding.prop)),
    });
  }
}

function makeBufferOptions(getScale: () => number) {
  return {
    strategy: props.bufferStrategy!,
    idleDelay: props.bufferIdleDelay,
    commitOnBlur: props.commitOnBlur,
    normalize: (v: Decimal | null) =>
      v == null
        ? null
        : clampMonetaryValue(v, getScale(), effectiveRound.value, {
            precision: effectivePrecision.value,
            min: minD.value,
            max: maxD.value,
          }),
    equals: (a: Decimal | null, b: Decimal | null) => {
      const s = getScale();
      const qa = quantizeMonetaryForCompare(a, s, effectiveRound.value);
      const qb = quantizeMonetaryForCompare(b, s, effectiveRound.value);
      if (qa == null && qb == null) return true;
      if (qa == null || qb == null) return false;
      return qa.eq(qb);
    },
    getScale,
  };
}

const OMonetaryCell = defineComponent({
  name: 'OMonetaryCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
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
        if (isIntermediateMonetaryInput(editingRaw.value)) return;
        editingRaw.value = toView(nv);
      },
      { immediate: true }
    );

    const buffer = useBufferedCommit<Decimal | null>(
      () => {
        const raw = modelRef.value;
        if (raw == null || raw === '') return null;
        if (raw instanceof Decimal) return raw;
        return asMonetaryDecimal(raw);
      },
      v => {
        modelRef.value = v as any;
      },
      p.options as any
    );

    function currentScale(): number {
      try {
        const n = (p.options as any).getScale();
        // getScale already applies props.scale as fallback; reject out-of-range results.
        if (typeof n === 'number' && Number.isInteger(n) && n >= 0 && n <= 18) return n;
      } catch {}
      return 6;
    }

    function bounds() {
      return { precision: effectivePrecision.value, min: minD.value, max: maxD.value };
    }

    function onInput(raw: string) {
      if (raw === '' && props.nullable) {
        editingRaw.value = null;
        buffer.setEditing(null);
        return;
      }
      if (raw === '' && !props.nullable) {
        return;
      }
      if (!/^[-]?\d*(\.\d*)?$/.test(raw)) return;
      editingRaw.value = raw;
      if (isIntermediateMonetaryInput(raw)) return;
      const d = parseStrictMonetary(raw, currentScale(), bounds());
      if (!d) return;
      buffer.setEditing(clampMonetaryValue(d, currentScale(), effectiveRound.value, bounds()));
    }

    function onBlur() {
      const raw = editingRaw.value;
      const scale = currentScale();
      if (raw == null) {
        buffer.setEditing(null);
        buffer.onBlur();
        return;
      }
      if (isIntermediateMonetaryInput(raw)) {
        const d = parseStrictMonetary(raw.replace(/\.$/, ''), scale, bounds());
        if (!d) {
          buffer.setEditing(null);
          buffer.onBlur();
          return;
        }
        buffer.setEditing(clampMonetaryValue(d, scale, effectiveRound.value, bounds()));
        buffer.onBlur();
        return;
      }
      const d = parseStrictMonetary(raw, scale, bounds());
      if (d) buffer.setEditing(clampMonetaryValue(d, scale, effectiveRound.value, bounds()));
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

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    const rec = binding.recordRef().value;
    const scale = resolveScaleFrom(rec);
    const msg = validateMonetaryValue(
      value,
      scale,
      { precision: effectivePrecision.value, min: minD.value, max: maxD.value },
      (msg, a) => {
        if (a !== undefined) return _t(msg as any, a as any);
        return _t(msg as any);
      }
    );
    if (msg) return cb(new Error(msg));
    cb();
  },
  trigger: 'blur',
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...props.rules, internalRule]);
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
