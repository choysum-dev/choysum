// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for relation typeahead NameSearch / NameCreate.
 * Pure visibility + quick-create helpers: nameCreateVisibility.test.ts, nameCreateQuickCreate.test.ts.
 */

import { computed, h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { ElButton, ElDialog, ElMessage, ElSelectV2, ElTag } from 'element-plus';

import type { UseField } from '@/web/web/composables/useField';
import { useAuthStore } from '@/auth/web/stores/auth';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OManyToOneField from './OManyToOneField.vue';
import OManyToOneRefField from './OManyToOneRefField.vue';
import OManyToManyTagsField from './OManyToManyTagsField.vue';
import OManyToManyRefTagsField from './OManyToManyRefTagsField.vue';

function makeM2OBinding(relationStore: any): UseField {
  const value = ref<any>(null);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'PartnerId',
    meta: { type: 'ManyToOne', relationModel: 'demo.Partner' } as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    relationStore,
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
  } as any;
}

function makeM2MBinding(relationStore: any): UseField {
  const items = ref<any[]>([]);
  const record = ref({ Id: '1' });
  return {
    env: {
      isForm: true,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'TagIds',
    meta: { type: 'ManyToMany', relationModel: 'demo.Tag' } as any,
    fieldRef: () => items as any,
    fieldRefOf: () => items as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    relationStore,
    store: undefined,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: (row: any) => {
        items.value = [...items.value, row];
      },
      clearItems: () => {
        items.value = [];
      },
    }),
    asView: () => ({ fieldValue: () => items }) as any,
  } as any;
}

const epSfcs = [ElSelectV2, ElDialog, ElButton, ElTag];
const origElMessageError = ElMessage.error;

function installFieldBaseStub() {
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
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
      renderMode: { type: String, default: undefined },
      showInlineError: { type: Boolean, default: undefined },
    },
    setup(p: any, { slots }: any) {
      const fieldValue = () => (p.binding as UseField).fieldRef();
      return () =>
        h('div', { class: 'ob' }, [
          slots.edit?.({ fieldValue, record: { Id: '1' } }),
          slots.display?.({ fieldValue, record: { Id: '1' } }),
        ]);
    },
  });
}

function installEpStubs() {
  stubSfc(ElSelectV2 as any, {
    name: 'ElSelectV2',
    inheritAttrs: false,
    props: {
      remoteMethod: { type: Function, default: undefined },
      modelValue: { type: [String, Number, Object, Array, null] as any, default: undefined },
      options: { type: Array, default: undefined },
      loading: { type: Boolean, default: false },
    },
    setup(props: any, { slots }: any) {
      return () =>
        h('div', { class: 'select-stub' }, [
          h('button', {
            type: 'button',
            'data-test': 'trigger-remote',
            onClick: () => props.remoteMethod?.('  alice  '),
          }),
          h('button', {
            type: 'button',
            'data-test': 'trigger-remote-null',
            onClick: () => props.remoteMethod?.(null),
          }),
          h('button', {
            type: 'button',
            'data-test': 'trigger-remote-empty',
            onClick: () => props.remoteMethod?.(''),
          }),
          slots.footer?.(),
        ]);
    },
  });
  stubSfc(ElDialog as any, {
    name: 'ElDialog',
    props: ['modelValue', 'title'],
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'el-dialog' }, [slots.default?.(), slots.footer?.()]);
    },
  });
  stubSfc(ElButton as any, {
    name: 'ElButton',
    inheritAttrs: false,
    emits: ['click'],
    setup(_: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: 'button',
            class: 'el-btn',
            ...attrs,
            onClick: (e: any) => emit('click', e),
          },
          slots.default?.()
        );
    },
  });
  stubSfc(ElTag as any, {
    name: 'ElTag',
    setup(_: any, { slots }: any) {
      return () => h('span', { class: 'el-tag' }, slots.default?.());
    },
  });
}

function seedCreatePermission() {
  const auth = useAuthStore();
  auth.identity = {
    metadata: {
      activeCompanyId: 'c1',
      enabledCompanyIds: ['c1'],
    },
  } as any;
  auth.permissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: [], actions: ['partner.action.partner_create'] } },
    },
  } as any;
}

async function clickRemote(m: ReturnType<typeof mountApp>, sel = '[data-test="trigger-remote"]') {
  m.click(sel);
  await flushPromises();
  await nextTick();
}

function findCreate(m: ReturnType<typeof mountApp>, testId: string) {
  return (m.q(`[data-testid="${testId}"]`) as HTMLElement | null) ?? null;
}

function fieldsContain(fields: unknown, ...needed: string[]) {
  expect(Array.isArray(fields)).toBe(true);
  const list = fields as unknown[];
  for (const f of needed) {
    expect(list.includes(f)).toBe(true);
  }
}

describe('relation typeahead NameSearch / NameCreate', () => {
  let pinia: ReturnType<typeof createPinia>;
  let msgError: ReturnType<typeof fnRecorder>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    seedCreatePermission();
    msgError = fnRecorder();
    (ElMessage as any).error = msgError;
    installFieldBaseStub();
    installEpStubs();
  });

  afterEach(() => {
    (ElMessage as any).error = origElMessageError;
    restoreSfc(OFieldBase as any);
    for (const c of epSfcs) restoreSfc(c as any);
  });

  function mountField(Comp: any, props: Record<string, unknown>) {
    return mountApp(Comp, {
      props: { renderMode: 'form', ...props },
      plugins: [pinia],
    });
  }

  test('OManyToOneField NameSearch trims keyword and uses pageSize fields', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 'p1', DisplayName: 'Alice' }]);
    const Search = fnRecorder();
    const m = mountField(OManyToOneField, {
      binding: makeM2OBinding({ NameSearch, Search, fullModelName: 'partner.Partner' }),
      pageSize: 15,
    });

    await clickRemote(m);
    expect(NameSearch.calls.length).toBe(1);
    expect(NameSearch.calls[0]).toEqual([
      'alice',
      [],
      { fields: ['Id', 'DisplayName'], limit: 15 },
    ]);
    expect(Search.calls.length).toBe(0);

    NameSearch.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-null"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');

    NameSearch.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-empty"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');
    m.unmount();
  });

  test('OManyToOneField remote search is a no-op without relationStore', async () => {
    const missing = mountField(OManyToOneField, {
      binding: makeM2OBinding(undefined),
    });
    await clickRemote(missing);
    expect(missing.q('[data-test="trigger-remote"]')).toBeTruthy();
    expect(missing.q('.select-stub')).toBeTruthy();
    missing.unmount();
  });

  test('OManyToOneRefField NameSearch mirrors M2O wiring', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 'p1', DisplayName: 'Alice' }]);
    const Search = fnRecorder();
    const m = mountField(OManyToOneRefField, {
      binding: makeM2OBinding({ NameSearch, Search, fullModelName: 'partner.Partner' }),
      pageSize: 12,
    });

    await clickRemote(m);
    expect(NameSearch.calls.length).toBe(1);
    expect(NameSearch.calls[0]).toEqual([
      'alice',
      [],
      { fields: ['Id', 'DisplayName'], limit: 12 },
    ]);
    expect(Search.calls.length).toBe(0);

    NameSearch.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-null"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');
    m.unmount();
  });

  test('OManyToOneRefField remote search is a no-op without relationStore', async () => {
    const warn = console.warn;
    console.warn = () => {};
    try {
      const missing = mountField(OManyToOneRefField, {
        binding: makeM2OBinding(undefined),
      });
      await clickRemote(missing);
      expect(missing.q('[data-test="trigger-remote"]')).toBeTruthy();
      expect(missing.q('.select-stub')).toBeTruthy();
      missing.unmount();
    } finally {
      console.warn = warn;
    }
  });

  test('OManyToManyTagsField NameSearch uses conditions and label fields', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 't1', DisplayName: 'Alice' }]);
    const Search = fnRecorder();
    const m = mountField(OManyToManyTagsField, {
      binding: makeM2MBinding({ NameSearch, Search, fullModelName: 'partner.Partner' }),
      suggestLimit: 8,
    });

    await clickRemote(m);
    expect(NameSearch.calls.length).toBe(1);
    const [keyword, condition, options] = NameSearch.calls[0]!;
    expect(keyword).toBe('alice');
    expect(condition).toEqual([]);
    expect((options as any).limit).toBe(8);
    fieldsContain((options as any).fields, 'Id', 'DisplayName');
    expect(Search.calls.length).toBe(0);

    NameSearch.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-empty"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');
    expect((NameSearch.calls[0]?.[2] as any)?.limit).toBe(8);

    NameSearch.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-null"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');
    m.unmount();
  });

  test('OManyToManyRefTagsField NameSearch uses hydration fields', async () => {
    const NameSearch = fnRecorder(async () => [{ Id: 't1', DisplayName: 'Alice' }]);
    const Search = fnRecorder(async () => []);
    const m = mountField(OManyToManyRefTagsField, {
      binding: makeM2MBinding({ NameSearch, Search, fullModelName: 'partner.Partner' }),
      suggestLimit: 9,
    });

    await clickRemote(m);
    expect(NameSearch.calls.length).toBe(1);
    const [keyword, condition, options] = NameSearch.calls[0]!;
    expect(keyword).toBe('alice');
    expect(condition).toEqual([]);
    expect((options as any).limit).toBe(9);
    fieldsContain((options as any).fields, 'Id', 'DisplayName');
    expect(Search.calls.length).toBe(0);

    NameSearch.mockClear();
    Search.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-empty"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');
    expect(Search.calls.length).toBe(0);

    NameSearch.mockClear();
    await clickRemote(m, '[data-test="trigger-remote-null"]');
    expect(NameSearch.calls[0]?.[0]).toBe('');
    m.unmount();
  });

  test('Create click sets M2O value and appends M2M ids when allowCreate', async () => {
    const createdM2O = { Id: 'new1', DisplayName: 'alice', Name: 'alice' };
    const NameCreateM2O = fnRecorder(async (name: string) => ({
      Id: 'new1',
      DisplayName: name,
      Name: name,
    }));
    const bindingM2O = makeM2OBinding({
      NameSearch: fnRecorder(async () => []),
      NameCreate: NameCreateM2O,
      fullModelName: 'partner.Partner',
    });
    const m2o = mountField(OManyToOneField, {
      binding: bindingM2O,
      allowCreate: true,
      nameField: 'Code',
    });
    await clickRemote(m2o);
    const createM2O = findCreate(m2o, 'o-m2o-name-create');
    expect(createM2O).toBeTruthy();
    expect((createM2O!.textContent || '').includes('alice')).toBe(true);
    createM2O!.click();
    await flushPromises();
    expect(NameCreateM2O.calls[0]).toEqual(['alice', undefined, { nameField: 'Code' }]);
    expect(bindingM2O.fieldRef().value).toEqual(createdM2O);

    NameCreateM2O.mockImplementation(async () => {
      throw new Error('denied');
    });
    await clickRemote(m2o);
    findCreate(m2o, 'o-m2o-name-create')!.click();
    await flushPromises();
    expect(msgError.calls.some(c => c[0] === 'denied')).toBe(true);
    m2o.unmount();

    const NameCreateRef = fnRecorder(async (name: string) => ({
      Id: 'r1',
      DisplayName: name,
      Name: name,
    }));
    const bindingRef = makeM2OBinding({
      NameSearch: fnRecorder(async () => []),
      NameCreate: NameCreateRef,
      fullModelName: 'partner.Partner',
    });
    const m2oRef = mountField(OManyToOneRefField, {
      binding: bindingRef,
      allowCreate: true,
    });
    await clickRemote(m2oRef);
    findCreate(m2oRef, 'o-m2o-name-create')!.click();
    await flushPromises();
    expect(NameCreateRef.calls[0]).toEqual(['alice', undefined, undefined]);
    expect(bindingRef.fieldRef().value).toEqual({ Id: 'r1', DisplayName: 'alice', Name: 'alice' });
    m2oRef.unmount();

    const NameCreateTags = fnRecorder(async (name: string) => ({
      Id: 't9',
      DisplayName: name,
      Name: name,
    }));
    const bindingTags = makeM2MBinding({
      NameSearch: fnRecorder(async () => []),
      NameCreate: NameCreateTags,
      fullModelName: 'partner.Partner',
    });
    const tags = mountField(OManyToManyTagsField, {
      binding: bindingTags,
      allowCreate: true,
    });
    await clickRemote(tags);
    findCreate(tags, 'o-m2m-name-create')!.click();
    await flushPromises();
    expect(NameCreateTags.calls[0]).toEqual(['alice', undefined, undefined]);
    expect(bindingTags.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['t9']);

    NameCreateTags.mockImplementation(async () => ({ Id: 't9', DisplayName: 'alice', Name: 'alice' }));
    await clickRemote(tags);
    findCreate(tags, 'o-m2m-name-create')!.click();
    await flushPromises();
    expect(bindingTags.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['t9']);
    tags.unmount();

    const NameCreateRefTags = fnRecorder(async (name: string) => ({
      Id: 'rt1',
      DisplayName: name,
      Name: name,
    }));
    const bindingRefTags = makeM2MBinding({
      NameSearch: fnRecorder(async () => []),
      NameCreate: NameCreateRefTags,
      Search: fnRecorder(async () => []),
      fullModelName: 'partner.Partner',
    });
    const refTags = mountField(OManyToManyRefTagsField, {
      binding: bindingRefTags,
      allowCreate: true,
      nameField: 'Title',
    });
    await clickRemote(refTags);
    findCreate(refTags, 'o-m2m-name-create')!.click();
    await flushPromises();
    expect(NameCreateRefTags.calls[0]).toEqual(['alice', undefined, { nameField: 'Title' }]);
    expect(bindingRefTags.fieldRef().value.map((r: any) => r.Id ?? r)).toEqual(['rt1']);
    refTags.unmount();
  });

  test('Create entry hidden when allowCreate is false or unset', async () => {
    for (const [Comp, testId, makeBinding] of [
      [OManyToOneField, 'o-m2o-name-create', makeM2OBinding],
      [OManyToOneRefField, 'o-m2o-name-create', makeM2OBinding],
      [OManyToManyTagsField, 'o-m2m-name-create', makeM2MBinding],
      [OManyToManyRefTagsField, 'o-m2m-name-create', makeM2MBinding],
    ] as const) {
      for (const allowCreate of [undefined, false] as const) {
        const m = mountField(Comp, {
          binding: makeBinding({
            NameSearch: fnRecorder(async () => []),
            NameCreate: fnRecorder(),
            Search: fnRecorder(async () => []),
            fullModelName: 'partner.Partner',
          }),
          ...(allowCreate === false ? { allowCreate: false } : {}),
        });
        await clickRemote(m);
        expect(findCreate(m, testId)).toBeNull();
        m.unmount();
      }
    }
  });
});
