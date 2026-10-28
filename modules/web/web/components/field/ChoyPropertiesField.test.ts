// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, reactive, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import PropertiesField from './ChoyPropertiesField.vue';

function makeBinding(opts: {
  map?: Record<string, unknown> | null;
  record?: Record<string, unknown>;
  isForm?: boolean;
  prop?: string;
  store?: any;
  ResolveProperties?: ((...args: any[]) => Promise<any>) | false;
}): UseField & { __value: any; __store: any } {
  const value = ref(opts.map === undefined ? {} : opts.map);
  const recordRef = ref(opts.record ?? { Id: '1', Properties: value.value });
  let store: any;
  if (Object.prototype.hasOwnProperty.call(opts, 'store')) {
    store = opts.store;
  } else if (opts.ResolveProperties === false) {
    store = {};
  } else {
    store = {
      ResolveProperties:
        opts.ResolveProperties ?? fnRecorder(async () => [] as ResolvedPropertyItem[]),
    };
  }
  return {
    env: {
      isForm: opts.isForm !== false,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop === undefined ? 'Properties' : opts.prop,
    meta: reactive({ type: 'properties' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: () => undefined,
    store: store as any,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
    __store: store,
  } as any;
}

function installFieldBaseStub() {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    inheritAttrs: false,
    props: {
      binding: { type: Object, required: true },
      toView: { type: Function, default: undefined },
      fromView: { type: Function, default: undefined },
      rules: { type: Array, default: undefined },
      label: { type: String, default: undefined },
      formItemProps: { type: Object, default: undefined },
      vColumnProps: { type: Object, default: undefined },
      required: { type: [Boolean, Function, Object], default: undefined },
      readonly: { type: [Boolean, Function, Object], default: undefined },
      visible: { type: [Boolean, Function, Object], default: undefined },
      cellVisible: { type: [Boolean, Function, Object], default: undefined },
      renderMode: { type: String, default: 'auto' },
      showInlineError: { type: Boolean, default: false },
    },
    setup(props: any, { slots }: any) {
      const fieldValue = () => {
        const raw = (props.binding as UseField).fieldRef();
        const toView = props.toView as ((v: any) => any) | undefined;
        const fromView = props.fromView as ((v: any) => any) | undefined;
        return computed({
          get: () => (toView ? toView(raw.value) : raw.value),
          set: v => {
            raw.value = fromView ? fromView(v) : v;
          },
        });
      };
      return () =>
        h('div', { class: 'ob' }, [slots.edit?.({ fieldValue }), slots.display?.({ fieldValue })]);
    },
  } as any);
}

function setControlValue(root: Element, selector: string, value: string) {
  const el = root.querySelector(selector) as HTMLInputElement | HTMLTextAreaElement | null;
  expect(el).toBeTruthy();
  el!.value = value;
  el!.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('PropertiesField', () => {
  beforeEach(() => {
    installFieldBaseStub();
  });

  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
  });

  async function mountField(props: Record<string, unknown>) {
    const mounted = mountApp(PropertiesField as any, { props });
    await flushPromises();
    return mounted;
  }

  test('resolves on mount and write replaces the full properties map', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'active', type: 'boolean', value: true },
      { name: 'code', type: 'char', value: 'A1' },
      { name: 'qty', type: 'integer', value: 2 },
      { name: 'amount', type: 'float', value: 1.5 },
      { name: 'day', type: 'date', value: '2024-01-02T00:00:00Z' },
      { name: 'when', type: 'datetime', value: '2024-06-30T16:00:00.000Z' },
      { name: 'html_x', type: 'html', string: 'Bad' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { active: true, code: 'A1', qty: 2, orphan: 'keep-until-replace' },
      ResolveProperties,
    });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      expect(ResolveProperties.calls.length).toBeGreaterThan(0);
      expect(mounted.q('[data-testid="choy-properties-form"]')).toBeTruthy();
      expect(mounted.q('[data-name="code"]')).toBeTruthy();
      expect(mounted.q('[data-name="html_x"]')).toBeFalsy();

      setControlValue(mounted.el, '#choy-properties-Properties-code', 'B9');
      await nextTick();
      expect(binding.__value.value).toEqual({
        active: true,
        code: 'B9',
        qty: 2,
        amount: 1.5,
        day: '2024-01-02T00:00:00Z',
        when: '2024-06-30T16:00:00.000Z',
      });
      expect(Object.prototype.hasOwnProperty.call(binding.__value.value, 'orphan')).toBe(false);
    } finally {
      mounted.unmount();
    }
  });

  test('clears when ResolveProperties is missing, fails, or returns non-arrays', async () => {
    const noRpc = makeBinding({ ResolveProperties: false, map: {} });
    const noRpcWrap = await mountField({ binding: noRpc, renderMode: 'form' });
    try {
      const empty = noRpcWrap.q('[data-testid="choy-properties-empty"]');
      expect(empty).toBeTruthy();
      expect((empty!.textContent || '').trim()).toMatch(/No properties defined/);
    } finally {
      noRpcWrap.unmount();
    }

    const noProp = makeBinding({
      prop: '',
      ResolveProperties: fnRecorder(async () => [{ name: 'x', type: 'char' }]),
    });
    const noPropWrap = await mountField({ binding: noProp, renderMode: 'form' });
    try {
      expect(noPropWrap.q('[data-testid="choy-properties-empty"]')).toBeTruthy();
    } finally {
      noPropWrap.unmount();
    }

    const failing = makeBinding({
      ResolveProperties: fnRecorder(async () => {
        throw new Error('rpc down');
      }),
    });
    const failWrap = await mountField({ binding: failing, renderMode: 'form' });
    try {
      expect(failWrap.q('[data-testid="choy-properties-empty"]')).toBeTruthy();
    } finally {
      failWrap.unmount();
    }

    const nonArray = makeBinding({
      ResolveProperties: fnRecorder(async () => ({ not: 'array' }) as any),
    });
    const nonArrayWrap = await mountField({ binding: nonArray, renderMode: 'form' });
    try {
      expect(nonArrayWrap.q('[data-testid="choy-properties-empty"]')).toBeTruthy();
    } finally {
      nonArrayWrap.unmount();
    }

    const noStore = makeBinding({ store: undefined, map: {} });
    const noStoreWrap = await mountField({ binding: noStore, renderMode: 'form' });
    try {
      expect(noStoreWrap.q('[data-testid="choy-properties-empty"]')).toBeTruthy();
    } finally {
      noStoreWrap.unmount();
    }
  });

  test('renders readonly items as plain text (no disabled inputs)', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'code', type: 'char', string: 'Code', value: 'A', readonly: true },
      { name: 'flag', type: 'boolean', value: true, readonly: true },
      { name: 'qty', type: 'integer', value: 3, readonly: false },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({ map: { code: 'A', flag: true, qty: 3 }, ResolveProperties });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      expect(mounted.q('[data-name="code"]')?.getAttribute('data-readonly')).toBe('1');
      expect(mounted.q('[data-name="qty"]')?.getAttribute('data-readonly')).toBe('0');
      expect(mounted.q('#choy-properties-Properties-code')).toBeNull();
      const readonlyVals = Array.from(
        mounted.el.querySelectorAll('[data-testid="choy-properties-readonly-value"]')
      );
      expect(readonlyVals.length).toBe(2);
      expect(readonlyVals[0]?.textContent).toBe('A');
      expect(readonlyVals[1]?.textContent).toBe('Yes');
      expect(readonlyVals[0]?.className || '').toMatch(/muted/);
      const qty = mounted.q('#choy-properties-Properties-qty') as HTMLInputElement | null;
      expect(qty).toBeTruthy();
      expect(qty!.getAttribute('disabled')).toBeNull();
      expect(qty!.getAttribute('data-slot')).toBe('input');
    } finally {
      mounted.unmount();
    }
  });

  test('table renderMode shows compact value summary', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'code', type: 'char', string: 'Code', value: 'A1' },
      { name: 'qty', type: 'integer', string: 'Qty', value: 2 },
      { name: 'note', type: 'text', string: 'Note', value: 'x' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { code: 'A1', qty: 2, note: 'x' },
      ResolveProperties,
      isForm: false,
    });
    const mounted = await mountField({ binding, renderMode: 'table' });
    try {
      const summary = mounted.q('[data-testid="choy-properties-summary"]');
      expect(summary).toBeTruthy();
      const text = summary!.textContent || '';
      expect(text).toMatch(/Code:\s*A1/);
      expect(text).toMatch(/Qty:\s*2/);
      expect(text).toMatch(/\+1 more/);
      expect(mounted.q('[data-testid="choy-properties-form"]')).toBeNull();
      expect(mounted.q('#choy-properties-Properties-code')).toBeNull();
    } finally {
      mounted.unmount();
    }
  });

  test('display formats date/datetime and selection labels', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'day', type: 'date', string: 'Day', value: '2024-01-02T00:00:00Z' },
      { name: 'when', type: 'datetime', string: 'When', value: '2024-06-30T16:00:00.000Z' },
      {
        name: 'prio',
        type: 'selection',
        string: 'Priority',
        value: 'high',
        selection: [
          ['low', 'Low'],
          ['high', 'High'],
        ],
      },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: {
        day: '2024-01-02T00:00:00Z',
        when: '2024-06-30T16:00:00.000Z',
        prio: 'high',
      },
      ResolveProperties,
    });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      const displayRoot = mounted.el.querySelector('.choy-properties-form--display') as HTMLElement | null;
      expect(displayRoot).toBeTruthy();
      const valueOf = (name: string) => {
        const row = Array.from(displayRoot!.querySelectorAll('[data-name]')).find(
          el => el.getAttribute('data-name') === name,
        ) as HTMLElement | undefined;
        expect(row).toBeTruthy();
        return (row!.querySelector('.choy-properties-item__value') as HTMLElement | null)?.textContent || '';
      };
      expect(valueOf('day')).toBe('2024-01-02');
      expect(valueOf('when')).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
      expect(valueOf('prio')).toBe('High');
    } finally {
      mounted.unmount();
    }
  });

  test('shows form-help tip only for items with help', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'prio', type: 'selection', string: 'Priority', help: 'Record urgency', value: 'high', selection: [['high', 'High']] },
      { name: 'code', type: 'char', string: 'Code', value: 'A1' },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({
      map: { prio: 'high', code: 'A1' },
      ResolveProperties,
    });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      const prioRow = mounted.el.querySelector('[data-name="prio"]') as HTMLElement | null;
      const codeRow = mounted.el.querySelector('[data-name="code"]') as HTMLElement | null;
      expect(prioRow).toBeTruthy();
      expect(codeRow).toBeTruthy();
      const prioHelp = prioRow!.querySelector('[data-slot="form-help"]') as HTMLElement | null;
      expect(prioHelp).toBeTruthy();
      expect(prioHelp!.getAttribute('aria-label') || '').toContain('Record urgency');
      expect(codeRow!.querySelector('[data-slot="form-help"]')).toBeFalsy();
    } finally {
      mounted.unmount();
    }
  });

  test('integer write rejects decimal keystrokes; float keeps intermediate dot', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'qty', type: 'integer', value: 2 },
      { name: 'score', type: 'float', value: 1 },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({ map: { qty: 2, score: 1 }, ResolveProperties });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      const qty = mounted.q('#choy-properties-Properties-qty') as HTMLInputElement | null;
      expect(qty?.getAttribute('type') || 'text').toBe('text');
      setControlValue(mounted.el, '#choy-properties-Properties-qty', '3.5');
      await nextTick();
      expect(binding.__value.value.qty).toBe(2);
      expect(qty!.value).toBe('2');
      setControlValue(mounted.el, '#choy-properties-Properties-qty', '4');
      await nextTick();
      expect(binding.__value.value.qty).toBe(4);

      setControlValue(mounted.el, '#choy-properties-Properties-score', '3.');
      await nextTick();
      const score = mounted.q('#choy-properties-Properties-score') as HTMLInputElement | null;
      expect(score?.value).toBe('3.');
      expect(binding.__value.value.score).toBe(1);
      setControlValue(mounted.el, '#choy-properties-Properties-score', '3.5');
      await nextTick();
      expect(binding.__value.value.score).toBe(3.5);
    } finally {
      mounted.unmount();
    }
  });

  test('numeric inputs resync DOM when letters are typed', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'qty', type: 'integer', value: 4 },
      { name: 'score', type: 'float', value: 1.5 },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({ map: { qty: 4, score: 1.5 }, ResolveProperties });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      const qty = mounted.q('#choy-properties-Properties-qty') as HTMLInputElement | null;
      const score = mounted.q('#choy-properties-Properties-score') as HTMLInputElement | null;
      expect(qty).toBeTruthy();
      expect(score).toBeTruthy();
      setControlValue(mounted.el, '#choy-properties-Properties-qty', 'a111');
      await nextTick();
      expect(qty!.value).toBe('4');
      expect(binding.__value.value.qty).toBe(4);
      setControlValue(mounted.el, '#choy-properties-Properties-score', 'd333.22');
      await nextTick();
      expect(score!.value).toBe('1.5');
      expect(binding.__value.value.score).toBe(1.5);
    } finally {
      mounted.unmount();
    }
  });
});
