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
import ChoyFieldBase from './ChoyFieldBase.vue';
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
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
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

function mountStatusbar(
  props: Record<string, unknown>,
  binding: UseField,
  opts?: { slot?: 'edit' | 'display' | 'both'; provide?: Record<string, unknown> }
) {
  lastBaseProps.current = null;
  installFieldBaseStub(opts?.slot ?? 'edit');
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

function stageMap(m: { qa: (s: string) => Element[] }) {
  return Object.fromEntries(
    optionValues(m).map((o) => [o.value, o.el.getAttribute('data-stage') || '']),
  );
}

function edgeMap(m: { qa: (s: string) => Element[] }) {
  return Object.fromEntries(
    optionValues(m).map((o) => [o.value, o.el.getAttribute('data-edge') || '']),
  );
}

describe('ChoyStatusbarField mount wiring', () => {
  afterEach(() => {
    restoreSfc(ChoyFieldBase as any);
  });

  test('display mode is non-clickable and lists ribbon options', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, isEditMode: false });
    const m = mountStatusbar({}, binding);
    await flushPromises();
    const root = statusbarRoot(m);
    expect(root?.getAttribute('role')).toBe('radiogroup');
    expect(root?.getAttribute('aria-disabled')).toBe('true');
    expect(root?.getAttribute('data-clickable')).toBeFalsy();
    expect(JSON.parse(JSON.stringify(optionValues(m).map((o) => o.value)))).toEqual([
      'draft',
      'confirmed',
      'done',
    ]);
    expect(optionValues(m).map((o) => o.text)).toEqual(['Draft', 'Confirmed', 'Done']);
    expect(optionValues(m).every((o) => o.disabled)).toBe(false);
    const opts = optionValues(m);
    expect(opts[0]?.el.getAttribute('data-selected')).toBe('true');
    expect(opts[1]?.el.getAttribute('data-selected')).toBeFalsy();
    m.unmount();
  });

  test('edit mode is clickable by default', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft', isEditMode: true });
    const m = mountStatusbar({}, binding);
    await flushPromises();
    const root = statusbarRoot(m);
    expect(root?.getAttribute('data-clickable')).toBe('true');
    expect(root?.getAttribute('aria-disabled')).toBeFalsy();
    expect(optionValues(m).every((o) => o.disabled)).toBe(false);
    m.click('[data-value=done]');
    await flushPromises();
    expect(value.value).toBe('done');
    m.unmount();
  });

  test('clickable false stays non-interactive even in edit mode', async () => {
    const store = makeStore();
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft', isEditMode: true });
    const m = mountStatusbar({ clickable: false }, binding);
    await flushPromises();
    expect(statusbarRoot(m)?.getAttribute('data-clickable')).toBeFalsy();
    expect(statusbarRoot(m)?.getAttribute('aria-disabled')).toBe('true');
    m.click('[data-value=done]');
    await flushPromises();
    expect(value.value).toBe('draft');
    m.unmount();
  });

  test('marks selected when clickable true', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'confirmed', isEditMode: false });
    const m = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    const root = statusbarRoot(m);
    expect(root?.getAttribute('role')).toBe('radiogroup');
    expect(root?.getAttribute('data-clickable')).toBe('true');
    const opts = optionValues(m);
    expect(opts.find((o) => o.value === 'confirmed')?.el.getAttribute('data-selected')).toBe('true');
    expect(opts.find((o) => o.value === 'draft')?.el.getAttribute('data-selected')).toBeFalsy();
    m.unmount();
  });

  test('assigns past/current/future stages and first/middle/last edges', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'confirmed', isEditMode: false });
    const m = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    expect(stageMap(m)).toEqual({
      draft: 'past',
      confirmed: 'current',
      done: 'future',
    });
    expect(edgeMap(m)).toEqual({
      draft: 'first',
      confirmed: 'middle',
      done: 'last',
    });
    m.unmount();
  });

  test('all stages are future when current value is null', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: null, isEditMode: false });
    const m = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    expect(stageMap(m)).toEqual({
      draft: 'future',
      confirmed: 'future',
      done: 'future',
    });
    m.unmount();
  });

  test('fallback current value is current with prior options past', async () => {
    const store = makeStore();
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'unknown', isEditMode: false });
    const m = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    // unknown is appended as current fallback by resolveStatusbarOptions
    expect(stageMap(m)).toEqual({
      draft: 'past',
      confirmed: 'past',
      done: 'past',
      unknown: 'current',
    });
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
    restoreSfc(ChoyFieldBase as any);

    const readonlyMount = mountStatusbar({ clickable: true, readonly: true }, binding);
    await flushPromises();
    expect(statusbarRoot(readonlyMount)?.getAttribute('aria-disabled')).toBe('true');
    readonlyMount.unmount();
    restoreSfc(ChoyFieldBase as any);

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
    restoreSfc(ChoyFieldBase as any);

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
    expect(m.q('.ob')?.getAttribute('data-form-item-class') || '').toContain('choy-statusbar-form-item');
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
