// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for ChoyStatusbarField. Pure option/gate helpers live in statusbarHelpers.test.ts.
 */

import { computed, h, nextTick, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import FieldBase from './FieldBase.vue';
import ChoyStatusbarField from './ChoyStatusbarField.vue';

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
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
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

function installChoyButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      disabled: Boolean,
      variant: String,
      size: String,
      type: String,
      class: [String, Array, Object],
    },
    emits: ['click'],
    inheritAttrs: false,
    setup(props: any, { emit, slots, attrs }: any) {
      return () =>
        h(
          'button',
          {
            ...attrs,
            type: 'button',
            class: ['choy-statusbar-opt', props.class, attrs.class],
            disabled: props.disabled || undefined,
            'data-value': attrs['data-value'],
            'data-disabled': props.disabled ? '1' : '0',
            'aria-pressed': attrs['aria-pressed'],
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.()
        );
    },
  } as any);
}

function mountStatusbar(
  props: Record<string, unknown>,
  binding: UseField,
  opts?: { slot?: 'edit' | 'display' | 'both'; provide?: Record<string, unknown> }
) {
  lastBaseProps.current = null;
  installFieldBaseStub(opts?.slot ?? 'edit');
  installChoyButtonStub();
  return mountApp(ChoyStatusbarField as any, {
    props: {
      binding,
      renderMode: 'inline',
      ...props,
    },
    provide: opts?.provide,
  });
}

function statusbarRoot(m: { q: (s: string) => Element | null }) {
  return m.q('[data-testid=choy-statusbar]');
}

function optionValues(m: { qa: (s: string) => Element[] }) {
  return m.qa('.choy-statusbar-opt').map((btn) => {
    const value = btn.getAttribute('data-value') || '';
    const text = (btn.textContent || '').trim();
    const disabled = btn.hasAttribute('disabled');
    return { value, text, disabled, el: btn };
  });
}

describe('ChoyStatusbarField mount wiring', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);
  });

  test('defaults to non-clickable and lists meta options', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });
    const m = mountStatusbar({}, binding);
    await flushPromises();
    const root = statusbarRoot(m);
    expect(root?.getAttribute('aria-disabled')).toBe('true');
    expect(JSON.parse(JSON.stringify(optionValues(m).map((o) => o.value)))).toEqual([
      'draft',
      'confirmed',
      'done',
    ]);
    expect(optionValues(m).map((o) => o.text)).toEqual(['Draft', 'Confirmed', 'Done']);
    expect(optionValues(m).every((o) => o.disabled)).toBe(true);
    m.unmount();
  });

  test('renders both edit and display slots', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });
    const m = mountStatusbar({ clickable: true }, binding, { slot: 'both' });
    await flushPromises();
    expect(m.qa('[data-testid=choy-statusbar]').length).toBe(2);
    m.unmount();
  });

  test('clickable writes value; beforeChange false cancels write', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const beforeChange = fnRecorder(() => false);
    const m = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();

    expect(statusbarRoot(m)?.getAttribute('aria-disabled')).toBeFalsy();
    m.click('[data-value=done]');
    await flushPromises();
    await nextTick();
    expect(beforeChange.calls.length).toBe(1);
    expect(beforeChange.calls[0]).toEqual(['done', 'draft']);
    expect(value.value).toBe('draft');

    beforeChange.mockImplementation(() => true);
    m.click('[data-value=done]');
    await flushPromises();
    expect(value.value).toBe('done');
    m.unmount();
  });

  test('writes immediately when clickable and beforeChange omitted', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const m = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    m.click('[data-value=confirmed]');
    await flushPromises();
    expect(value.value).toBe('confirmed');
    m.unmount();
  });

  test('skips same-value; ignores disabled options via onchange', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const onchange = ref({
      selection: [{ field: 'State', selection: ['draft', 'confirmed', 'done'], disabled: ['done'] }],
    });
    const m = mountStatusbar({ clickable: true }, binding, { provide: { lastOnchangeResult: onchange } });
    await flushPromises();

    m.click('[data-value=draft]');
    await flushPromises();
    expect(value.value).toBe('draft');

    const done = optionValues(m).find((o) => o.value === 'done')!;
    expect(done.disabled).toBe(true);
    m.unmount();
  });

  test('disables while async beforeChange is pending', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    let resolveGate!: (ok: boolean) => void;
    const gate = new Promise<boolean>((r) => {
      resolveGate = r;
    });
    const beforeChange = fnRecorder(() => gate);
    const m = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();

    m.click('[data-value=done]');
    await nextTick();
    expect(statusbarRoot(m)?.getAttribute('aria-disabled')).toBe('true');
    expect(value.value).toBe('draft');

    resolveGate(true);
    await flushPromises();
    expect(value.value).toBe('done');
    expect(statusbarRoot(m)?.getAttribute('aria-disabled')).toBeFalsy();
    m.unmount();
  });

  test('respects disabled / readonly boolean / meta isReadonly', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store });

    const disabledMount = mountStatusbar({ clickable: true, disabled: true }, binding);
    await flushPromises();
    expect(statusbarRoot(disabledMount)?.getAttribute('aria-disabled')).toBe('true');
    disabledMount.unmount();
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);

    const readonlyMount = mountStatusbar({ clickable: true, readonly: true }, binding);
    await flushPromises();
    expect(statusbarRoot(readonlyMount)?.getAttribute('aria-disabled')).toBe('true');
    readonlyMount.unmount();
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);

    const metaReadonly = {
      ...staticMeta,
      isReadonly: true,
    } as WebFieldMetadata;
    const storeRo = makeStore(metaReadonly);
    const { binding: roBinding } = makeBinding({ meta: metaReadonly, store: storeRo });
    const metaMount = mountStatusbar({ clickable: true }, roBinding);
    await flushPromises();
    expect(statusbarRoot(metaMount)?.getAttribute('aria-disabled')).toBe('true');
    metaMount.unmount();
  });

  test('applies statusbarVisible whitelist and keeps current fallback', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'confirmed' });
    const m = mountStatusbar({ statusbarVisible: ['draft', 'done'] }, binding);
    await flushPromises();
    expect(optionValues(m).map((o) => o.value)).toEqual(['draft', 'done', 'confirmed']);
    m.unmount();
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyButton as any);

    const { binding: selBinding } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const selMount = mountStatusbar({ selection: ['done', 'draft'] }, selBinding);
    await flushPromises();
    expect(optionValues(selMount).map((o) => o.value)).toEqual(['done', 'draft']);
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
    expect(statusbarRoot(m)).toBeTruthy();
    m.unmount();
  });
});
