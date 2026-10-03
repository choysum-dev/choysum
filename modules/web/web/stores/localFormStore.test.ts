// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h } from 'vue';
import * as VueRouter from 'vue-router';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import { listRegisteredModelNames } from '@/web/web/stores/registry';
import { createLocalFormStore } from '@/web/web/stores/localFormStore';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

function loginFields() {
  return [
    { name: 'Username', label: 'Username', type: 'varchar' as const },
    { name: 'RememberMe', label: 'Remember me', type: 'boolean' as const },
  ];
}

test('createLocalFormStore reads and writes declared fields only', () => {
  const store = createLocalFormStore({
    fields: loginFields(),
    initialValues: { Username: 'admin', Extra: 'drop-me' },
  });

  expect(store.getValues()).toEqual({ Username: 'admin', RememberMe: false });
  expect(store.getField('Extra')).toBeUndefined();
  expect(Object.prototype.hasOwnProperty.call(store.getValues(), 'Extra')).toBe(false);

  store.setField('Username', 'alice');
  store.setField('RememberMe', true);
  store.setField('Extra', 'nope');
  expect(store.getField('Username')).toBe('alice');
  expect(store.getField('RememberMe')).toBe(true);
  expect(store.getValues().Extra).toBeUndefined();
});

test('createLocalFormStore treats undefined initialValues as type defaults', () => {
  const store = createLocalFormStore({
    fields: loginFields(),
    initialValues: { Username: undefined, RememberMe: undefined },
  });
  expect(store.getField('Username')).toBe('');
  expect(store.getField('RememberMe')).toBe(false);
});

test('createLocalFormStore does not treat inherited Object keys as fields', () => {
  const initialValues = Object.create(null) as Record<string, unknown>;
  initialValues['__proto__'] = 'own';
  const store = createLocalFormStore({
    fields: [{ name: '__proto__', label: 'Proto', type: 'varchar' }],
    initialValues,
  });
  expect(store.getField('__proto__')).toBe('own');
  expect(store.getFieldMeta('__proto__')?.string).toBe('Proto');
  expect(store.getFieldMeta('toString')).toBeUndefined();
  expect(store.getFieldMeta('constructor')).toBeUndefined();
  expect(store.getField('toString')).toBeUndefined();
});

test('createLocalFormStore metadata is FieldsGet-shaped without RPC methods succeeding', async () => {
  const store = createLocalFormStore({ fields: loginFields() });
  const usernameMeta = store.getFieldMeta('Username');
  expect(usernameMeta?.type).toBe('varchar');
  expect(usernameMeta?.string).toBe('Username');
  expect(usernameMeta?.typeAnnotation).toBe('string');
  expect(store.getFieldMeta('RememberMe')?.type).toBe('boolean');
  const slice = await store.ensureFieldsGet(['Username']);
  expect(slice.Username?.string).toBe('Username');
  expect(slice.RememberMe).toBeUndefined();
  expect(typeof (store as any).DefaultGet).not.toBe('function');
  expect(() => store.Browse('1')).toThrow(/does not support Browse/);
  expect(() => store.Create({})).toThrow(/does not support Create/);
  expect(() => store.UpdateById('1', {})).toThrow(/does not support UpdateById/);
});

test('createLocalFormStore is not registered as a model factory', () => {
  const before = listRegisteredModelNames();
  const store = createLocalFormStore({ fields: loginFields(), storeId: 'local-form:unit' });
  expect(store.storeId).toBe('local-form:unit');
  expect(listRegisteredModelNames()).toEqual(before);
});

test('createLocalFormStore covers helpers, context, and remaining RPC stubs', async () => {
  const empty = createLocalFormStore({ fields: undefined as any });
  expect(empty.fieldNames).toEqual([]);
  expect(empty.getValues()).toEqual({});
  expect(empty.storeId.startsWith('local-form:')).toBe(true);

  const blankId = createLocalFormStore({ fields: loginFields(), storeId: '   ' });
  expect(blankId.storeId.startsWith('local-form:')).toBe(true);

  const store = createLocalFormStore({
    fields: [{ name: 'Username', label: 'Username', type: 'varchar', help: 'login id' }],
  });
  expect(store.getFieldMeta('Username')?.help).toBe('login id');
  expect(store.getFieldMeta('')).toBeUndefined();
  expect(store.getFieldMeta('   ')).toBeUndefined();
  expect(store.getField('')).toBeUndefined();
  expect(store.getFieldsGetTranslatedString('Username')).toBeUndefined();
  expect(store.getFieldsGetTranslatedHelp('Username')).toBeUndefined();
  store.clearFieldsGetCache();
  store.destroy();

  const all = await store.ensureFieldsGet();
  expect(all.Username?.string).toBe('Username');
  expect(await store.FieldsGet([])).toEqual(all);
  const slice = await store.ensureFieldsGet(['', 'Nope', 'Username']);
  expect(slice.Username?.string).toBe('Username');
  expect(Object.keys(slice)).toEqual(['Username']);

  store.setContext(null as any);
  expect(store.getContext()).toEqual({});
  store.setContext({ lang: 'zh_CN' });
  expect(store.getContext()).toEqual({ lang: 'zh_CN' });
  const seen = await store.withContext({ company: '1' }, async () => store.getContext());
  expect(seen).toEqual({ lang: 'zh_CN', company: '1' });
  expect(store.getContext()).toEqual({ lang: 'zh_CN' });
  await store
    .withContext({ company: '2' }, async () => {
      throw new Error('boom');
    })
    .then(
      () => {
        throw new Error('expected withContext to reject');
      },
      err => {
        expect((err as Error).message).toBe('boom');
      }
    );
  expect(store.getContext()).toEqual({ lang: 'zh_CN' });

  expect(await store.Onchange({} as any, [] as any)).toEqual({ value: undefined, messages: [] });

  const stubs = [
    'BrowseMany',
    'CreateMany',
    'Update',
    'Copy',
    'NameSearch',
    'NameCreate',
    'Count',
    'Search',
    'ReadGroup',
    'ReadGroupCount',
    'Delete',
    'DeleteById',
    'GetFieldTranslations',
    'UpdateFieldTranslations',
    'GetFieldCompanyValues',
    'UpdateFieldCompanyValues',
    'ResolveProperties',
  ] as const;
  for (const method of stubs) {
    expect(() => (store as any)[method]()).toThrow(new RegExp(`does not support ${method}`));
  }
});

test('createLocalFormStore rejects duplicate or empty field names', () => {
  expect(() =>
    createLocalFormStore({
      fields: [
        { name: 'Username', label: 'A', type: 'varchar' },
        { name: 'Username', label: 'B', type: 'varchar' },
      ],
    })
  ).toThrow(/duplicate field/);
  expect(() => createLocalFormStore({ fields: [{ name: '  ', label: 'X', type: 'varchar' }] })).toThrow(
    /non-empty/
  );
  expect(() =>
    createLocalFormStore({ fields: [{ name: 'Age', label: 'Age', type: 'number' as any }] })
  ).toThrow(/unsupported field type/);
});

describe('createLocalFormStore + ChoyFormView', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  test('embedded create FormView seeds draft and FieldBase shows metadata label', async () => {
    const store = createLocalFormStore({
      fields: loginFields(),
      initialValues: { Username: 'admin' },
    });
    const onActionError = fnRecorder();
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });

    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyFormView as any,
            {
              store,
              viewMode: 'create',
              embedded: true,
              showHeader: false,
              showActions: false,
              showMessages: false,
              resolveRecordIdFromRoute: false,
              initialValues: { Username: 'admin' },
              onActionError,
            },
            {
              default: () => h(ChoyVarcharField as any, { store, prop: 'Username' }),
            }
          );
      },
    });

    const wrapper = mountApp(Host as any, { plugins: [createPinia(), router] });
    await flushPromises();

    expect(onActionError.calls.length).toBe(0);
    expect(wrapper.text()).toContain('Username');
    const input = wrapper.q('input.choy-input') as HTMLInputElement | null;
    expect(input).toBeTruthy();
    expect(input!.value).toBe('admin');

    wrapper.unmount();
  });
});
