// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for OStatusbarField. Pure option/gate helpers live in ostatusbar_helpers.test.ts.
 */

import { computed, h, nextTick, ref } from 'vue';
import { ElSegmented } from 'element-plus';

import type { UseField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OStatusbarField from './OStatusbarField.vue';

function makeBinding(opts: {
  prop?: string;
  isEditMode?: boolean;
  meta?: WebFieldMetadata;
  store?: any;
  value?: string | null;
}): { binding: UseField; value: ReturnType<typeof ref<string | null>> } {
  const value = ref<string | null>(opts.value !== undefined ? opts.value : 'draft');
  const record = ref({ Id: '1', State: value.value });
  const binding = {
    env: {
      isForm: true,
      isEditMode: opts.isEditMode !== false,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop || 'State',
    meta: opts.meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => ({ ...record.value, State: value.value })) as any,
    registerFields: () => {},
    store: opts.store,
    asView: () => ({ fieldValue: () => value }) as any,
  } as any;
  return { binding, value };
}

const staticMeta: WebFieldMetadata = {
  id: '1',
  type: 'selection',
  typeAnnotation: 'string',
  string: 'State',
  selection: [
    { value: 'draft', label: 'Draft' },
    { value: 'confirmed', label: 'Confirmed' },
    { value: 'done', label: 'Done' },
  ],
};

function makeStore(meta: WebFieldMetadata = staticMeta) {
  const FieldsGet = fnRecorder(async () => ({ State: meta }));
  const helpers = createFieldsGetHelpers(
    { fieldsMetadata: { State: meta }, FieldsGet },
    { getLang: () => 'en_US' }
  );
  return { fieldsMetadata: { State: meta }, FieldsGet, ...helpers };
}

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function installFieldBaseStub(slot: 'edit' | 'display' | 'both' = 'edit') {
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
    props: ['binding', 'readonly', 'renderMode', 'rules', 'formItemProps', 'toView', 'fromView', 'label'],
    setup(p: any, { slots }: any) {
      lastBaseProps.current = p;
      if (typeof p.toView === 'function') {
        p.toView('done');
        p.toView(null);
      }
      if (typeof p.fromView === 'function') {
        p.fromView('done');
        p.fromView(null);
      }
      const fieldValue = () => (p.binding as UseField).fieldRef();
      const record = () => ({ Id: '1', State: (fieldValue() as any).value });
      return () => {
        const children: any[] = [];
        if (slot === 'edit' || slot === 'both') children.push(slots.edit?.({ fieldValue, record }));
        if (slot === 'display' || slot === 'both') children.push(slots.display?.({ fieldValue, record }));
        return h('div', { class: 'ob', 'data-form-item-class': String((p.formItemProps as any)?.class || '') }, children);
      };
    },
  });
}

function installSegmentedStub() {
  stubSfc(ElSegmented as any, {
    name: 'ElSegmented',
    props: ['modelValue', 'options', 'disabled'],
    emits: ['update:modelValue'],
    inheritAttrs: false,
    setup(p: any, { emit, attrs }: any) {
      // QJS/Vue may expose the listener as onUpdate:modelValue or onUpdate:model-value.
      const fire = (v: unknown) => {
        emit('update:modelValue', v);
        const handler =
          attrs['onUpdate:modelValue'] ??
          attrs.onUpdateModelValue ??
          attrs['onUpdate:model-value'];
        if (typeof handler === 'function') handler(v);
      };
      return () =>
        h(
          'div',
          {
            class: 'o-statusbar',
            'data-disabled': String(!!p.disabled),
            'data-model': String(p.modelValue ?? ''),
            'data-options': JSON.stringify(p.options || []),
          },
          [
            ...((p.options as any[]) || []).map((opt: any) =>
              h(
                'button',
                {
                  type: 'button',
                  class: 'seg-opt',
                  'data-value': String(opt.value),
                  disabled: !!(p.disabled || opt.disabled) || undefined,
                  onClick: () => {
                    if (p.disabled || opt.disabled) return;
                    fire(opt.value);
                  },
                },
                opt.label
              )
            ),
            h(
              'button',
              {
                type: 'button',
                class: 'seg-empty',
                onClick: () => fire(null),
              },
              'empty'
            ),
          ]
        );
    },
  });
}

function mountStatusbar(
  props: Record<string, unknown>,
  binding: UseField,
  opts?: { slot?: 'edit' | 'display' | 'both'; provide?: Record<string, unknown> }
) {
  lastBaseProps.current = null;
  installFieldBaseStub(opts?.slot ?? 'edit');
  installSegmentedStub();
  return mountApp(OStatusbarField as any, {
    props: {
      binding,
      renderMode: 'inline',
      ...props,
    },
    provide: opts?.provide,
  });
}

describe('OStatusbarField mount wiring', () => {
  afterEach(() => {
    restoreSfc(OFieldBase as any);
    restoreSfc(ElSegmented as any);
  });

  test('defaults to non-clickable and lists meta options', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });
    const m = mountStatusbar({}, binding);
    await flushPromises();
    expect(m.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('true');
    expect(JSON.parse(m.q('.o-statusbar')?.getAttribute('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'confirmed',
      'done',
    ]);
    m.unmount();
  });

  test('renders both edit and display slots', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });
    const m = mountStatusbar({ clickable: true }, binding, { slot: 'both' });
    await flushPromises();
    expect(m.qa('.o-statusbar').length).toBe(2);
    m.unmount();
  });

  test('clickable writes value; beforeChange false cancels write', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const beforeChange = fnRecorder(() => false);
    const m = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();

    expect(m.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('false');
    m.click('[data-value="done"]');
    await flushPromises();
    await nextTick();
    expect(beforeChange.calls.length).toBe(1);
    expect(beforeChange.calls[0]).toEqual(['done', 'draft']);
    expect(value.value).toBe('draft');

    beforeChange.mockImplementation(() => true);
    m.click('[data-value="done"]');
    await flushPromises();
    expect(value.value).toBe('done');
    m.unmount();
  });

  test('writes immediately when clickable and beforeChange omitted', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const m = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    m.click('[data-value="confirmed"]');
    await flushPromises();
    expect(value.value).toBe('confirmed');
    m.unmount();
  });

  test('skips same-value and empty emits; ignores disabled options via onchange', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const onchange = ref({
      selection: [{ field: 'State', selection: ['draft', 'confirmed', 'done'], disabled: ['done'] }],
    });
    const m = mountStatusbar({ clickable: true }, binding, { provide: { lastOnchangeResult: onchange } });
    await flushPromises();

    m.click('[data-value="draft"]');
    await flushPromises();
    expect(value.value).toBe('draft');

    m.click('.seg-empty');
    await flushPromises();
    expect(value.value).toBe('draft');

    const opts = JSON.parse(m.q('.o-statusbar')?.getAttribute('data-options') || '[]');
    const done = opts.find((o: any) => o.value === 'done');
    expect(done?.disabled).toBe(true);
    expect(m.q('[data-value="done"]')?.hasAttribute('disabled')).toBe(true);
    m.unmount();
  });

  test('disables while async beforeChange is pending', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    let resolveGate!: (ok: boolean) => void;
    const gate = new Promise<boolean>(r => {
      resolveGate = r;
    });
    const beforeChange = fnRecorder(() => gate);
    const m = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();

    m.click('[data-value="done"]');
    await nextTick();
    expect(m.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('true');
    expect(value.value).toBe('draft');

    resolveGate(true);
    await flushPromises();
    expect(value.value).toBe('done');
    expect(m.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('false');
    m.unmount();
  });

  test('respects disabled / readonly boolean / meta isReadonly', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });

    const disabledMount = mountStatusbar({ clickable: true, disabled: true }, binding);
    await flushPromises();
    expect(disabledMount.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('true');
    disabledMount.unmount();
    restoreSfc(OFieldBase as any);
    restoreSfc(ElSegmented as any);

    const readonlyMount = mountStatusbar({ clickable: true, readonly: true }, binding);
    await flushPromises();
    expect(readonlyMount.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('true');
    readonlyMount.unmount();
    restoreSfc(OFieldBase as any);
    restoreSfc(ElSegmented as any);

    const metaReadonly = {
      ...staticMeta,
      isReadonly: true,
    } as WebFieldMetadata;
    const storeRo = makeStore(metaReadonly);
    const { binding: roBinding } = makeBinding({ meta: metaReadonly, store: storeRo });
    const metaMount = mountStatusbar({ clickable: true }, roBinding);
    await flushPromises();
    expect(metaMount.q('.o-statusbar')?.getAttribute('data-disabled')).toBe('true');
    metaMount.unmount();
  });

  test('applies statusbarVisible whitelist and keeps current fallback', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'confirmed' });
    const m = mountStatusbar({ statusbarVisible: ['draft', 'done'] }, binding);
    await flushPromises();
    expect(JSON.parse(m.q('.o-statusbar')?.getAttribute('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'done',
      'confirmed',
    ]);
    m.unmount();
    restoreSfc(OFieldBase as any);
    restoreSfc(ElSegmented as any);

    const { binding: selBinding } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const selMount = mountStatusbar({ selection: ['done', 'draft'] }, selBinding);
    await flushPromises();
    expect(JSON.parse(selMount.q('.o-statusbar')?.getAttribute('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'done',
      'draft',
    ]);
    selMount.unmount();
  });

  test('ensures FieldsGet on mount and merges formItemProps class', async () => {
    const FieldsGet = fnRecorder(async () => ({ State: staticMeta }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet },
      { getLang: () => 'en_US' }
    );
    const ensureCalls: unknown[][] = [];
    const store = {
      fieldsMetadata: { State: staticMeta },
      FieldsGet,
      ensureFieldsGet: async (...args: any[]) => {
        ensureCalls.push(args);
        return helpers.ensureFieldsGet(...(args as [string[], string[]?]));
      },
      getFieldMeta: helpers.getFieldMeta,
    };
    const { binding } = makeBinding({ meta: staticMeta, store });
    const m = mountStatusbar({ formItemProps: { class: 'extra-class' } }, binding);
    await flushPromises();
    expect(ensureCalls.length).toBe(1);
    expect(ensureCalls[0]![0]).toEqual(['State']);
    expect(m.q('.ob')?.getAttribute('data-form-item-class') || '').toContain('o-statusbar-form-item');
    expect(m.q('.ob')?.getAttribute('data-form-item-class') || '').toContain('extra-class');
    m.unmount();
  });

  test('skips ensureFieldsGet when store lacks helper', async () => {
    const { binding } = makeBinding({
      meta: staticMeta,
      store: { getFieldMeta: () => staticMeta },
    });
    const m = mountStatusbar({}, binding);
    await flushPromises();
    expect(m.q('.o-statusbar')).toBeTruthy();
    m.unmount();
  });
});
