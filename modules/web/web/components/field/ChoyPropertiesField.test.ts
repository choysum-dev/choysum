// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, reactive, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
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
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
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
    restoreSfc(FieldBase as any);
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
      expect(mounted.q('[data-testid="o-properties-form"]')).toBeTruthy();
      expect(mounted.q('[data-name="code"]')).toBeTruthy();
      expect(mounted.q('[data-name="html_x"]')).toBeFalsy();

      setControlValue(mounted.el, '#o-properties-Properties-code', 'B9');
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
      expect(noRpcWrap.q('[data-testid="o-properties-empty"]')).toBeTruthy();
    } finally {
      noRpcWrap.unmount();
    }

    const noProp = makeBinding({
      prop: '',
      ResolveProperties: fnRecorder(async () => [{ name: 'x', type: 'char' }]),
    });
    const noPropWrap = await mountField({ binding: noProp, renderMode: 'form' });
    try {
      expect(noPropWrap.q('[data-testid="o-properties-empty"]')).toBeTruthy();
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
      expect(failWrap.q('[data-testid="o-properties-empty"]')).toBeTruthy();
    } finally {
      failWrap.unmount();
    }

    const nonArray = makeBinding({
      ResolveProperties: fnRecorder(async () => ({ not: 'array' }) as any),
    });
    const nonArrayWrap = await mountField({ binding: nonArray, renderMode: 'form' });
    try {
      expect(nonArrayWrap.q('[data-testid="o-properties-empty"]')).toBeTruthy();
    } finally {
      nonArrayWrap.unmount();
    }

    const noStore = makeBinding({ store: undefined, map: {} });
    const noStoreWrap = await mountField({ binding: noStore, renderMode: 'form' });
    try {
      expect(noStoreWrap.q('[data-testid="o-properties-empty"]')).toBeTruthy();
    } finally {
      noStoreWrap.unmount();
    }
  });

  test('honors item.readonly disabled controls', async () => {
    const ResolveProperties = fnRecorder(async () => [
      { name: 'code', type: 'char', string: 'Code', value: 'A', readonly: true },
      { name: 'qty', type: 'integer', value: 3, readonly: false },
    ] as ResolvedPropertyItem[]);
    const binding = makeBinding({ map: { code: 'A', qty: 3 }, ResolveProperties });
    const mounted = await mountField({ binding, renderMode: 'form' });
    try {
      expect(mounted.q('[data-name="code"]')?.getAttribute('data-readonly')).toBe('1');
      expect(mounted.q('[data-name="qty"]')?.getAttribute('data-readonly')).toBe('0');
      const code = mounted.q('#o-properties-Properties-code') as HTMLInputElement | null;
      expect(code).toBeTruthy();
      expect(code!.getAttribute('disabled') != null).toBe(true);
    } finally {
      mounted.unmount();
    }
  });
});
