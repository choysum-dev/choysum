// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw, nextTick, reactive, ref, Comment, Fragment, Text } from 'vue';
import { createRouter, createMemoryHistory } from 'vue-router';
import { createPinia, setActivePinia } from 'pinia';
import { ElIcon } from 'element-plus';

import { flushPromises, fnRecorder, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import OStatInfo from './OStatInfo.vue';
import OButtonBox from './OButtonBox.vue';
import { resolveStatDisplayValue, slotHasContent } from './ostatinfo_helpers';

describe('ostatinfo_helpers', () => {
  test('prefers explicit value over relation length', () => {
    expect(resolveStatDisplayValue({ value: 3, relationValue: [1, 2] })).toBe(3);
    expect(resolveStatDisplayValue({ value: 0, relationValue: [1] })).toBe(0);
  });

  test('uses relation array length when value is absent', () => {
    expect(resolveStatDisplayValue({ relationValue: ['a', 'b'] })).toBe(2);
    expect(resolveStatDisplayValue({ relationValue: [] })).toBe(0);
  });

  test('falls back to em dash (not 0) when unloaded', () => {
    expect(resolveStatDisplayValue({})).toBe('—');
    expect(resolveStatDisplayValue({ relationValue: null })).toBe('—');
    expect(resolveStatDisplayValue({ value: null })).toBe('—');
    expect(resolveStatDisplayValue({ emptyValue: 0 })).toBe(0);
  });

  test('detects empty vs meaningful slot trees', () => {
    expect(slotHasContent(null)).toBe(false);
    expect(slotHasContent(undefined)).toBe(false);
    expect(slotHasContent([])).toBe(false);
    expect(slotHasContent([h(Comment, 'x')])).toBe(false);
    expect(slotHasContent([h(Text, '   ')])).toBe(false);
    expect(slotHasContent([h(Text)])).toBe(false);
    expect(slotHasContent([h(Text, 'hi')])).toBe(true);
    expect(slotHasContent([h('div')])).toBe(true);
    expect(slotHasContent([h(Fragment, [h(Comment), h('span')])])).toBe(true);
    expect(slotHasContent([null, undefined, 42, 'plain'])).toBe(false);
    expect(slotHasContent([{ type: Fragment, children: 'x' }])).toBe(false);
  });
});

describe('OStatInfo', () => {
  const push = fnRecorder(async () => undefined as any);

  function makeRouter() {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'home', component: { template: '<div />' } },
        { path: '/users', name: 'users', component: { template: '<div />' } },
      ],
    });
    router.push = push as any;
    return router;
  }

  beforeEach(() => {
    setActivePinia(createPinia());
    push.mockReset();
  });

  afterEach(() => {
    restoreSfc(ElIcon as any);
  });

  test('renders value and label; emits click', async () => {
    const emitted: any[][] = [];
    const { unmount, q, click } = mountApp(OStatInfo as any, {
      props: { value: 5, label: 'Users' },
      on: { onClick: (...args: any[]) => emitted.push(args) },
      plugins: [makeRouter()],
    });
    expect(q('.o-stat-info__value')?.textContent).toBe('5');
    expect(q('.o-stat-info__label')?.textContent).toBe('Users');
    expect(q('.o-stat-info__icon')).toBeFalsy();
    click('.o-stat-info');
    expect(emitted.length).toBe(1);
    expect(push.calls.length).toBe(0);
    unmount();
  });

  test('renders icon when icon prop is set', () => {
    stubSfc(ElIcon as any, {
      name: 'ElIcon',
      setup(_: any, { slots, attrs }: any) {
        return () =>
          h(
            'i',
            { ...attrs, class: ['el-icon-stub', attrs.class] },
            slots.default?.()
          );
      },
    });
    const IconStub = markRaw(
      defineComponent({
        name: 'IconStub',
        setup() {
          return () => h('span', { class: 'icon-stub' });
        },
      })
    );
    const { unmount, q } = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'Users', icon: IconStub },
      plugins: [makeRouter()],
    });
    expect(q('.o-stat-info__icon')).toBeTruthy();
    expect(q('.icon-stub')).toBeTruthy();
    unmount();
  });

  test('uses relation length via store+prop when value omitted', async () => {
    const draft = reactive({ Users: [{ Id: '1' }, { Id: '2' }] });
    const { unmount, q } = mountApp(OStatInfo as any, {
      props: {
        store: { storeId: 's', fieldsMetadata: { Users: { type: 'onetomany' } } },
        prop: 'Users',
        label: 'Users',
      },
      provide: {
        'form-root': {
          draft,
          getField: (p: string) => (draft as any)[p],
          setField: (p: string, v: any) => {
            (draft as any)[p] = v;
          },
        },
      },
      plugins: [makeRouter()],
    });
    await flushPromises();
    expect(q('.o-stat-info__value')?.textContent).toBe('2');
    unmount();
  });

  test('skips useField when only store or only prop is set', () => {
    const a = mountApp(OStatInfo as any, {
      props: { store: { storeId: 's' }, label: 'Users', value: 1 },
      plugins: [makeRouter()],
    });
    expect(a.q('.o-stat-info__value')?.textContent).toBe('1');
    a.unmount();

    const b = mountApp(OStatInfo as any, {
      props: { prop: 'Users', label: 'Users', value: 1 },
      plugins: [makeRouter()],
    });
    expect(b.q('.o-stat-info__value')?.textContent).toBe('1');
    b.unmount();
  });

  test('shows em dash when relation is not an array', async () => {
    const draft = reactive({ Users: undefined as any });
    const { unmount, q } = mountApp(OStatInfo as any, {
      props: {
        store: { storeId: 's', fieldsMetadata: { Users: { type: 'onetomany' } } },
        prop: 'Users',
        label: 'Users',
      },
      provide: {
        'form-root': {
          draft,
          getField: (p: string) => (draft as any)[p],
          setField: (p: string, v: any) => {
            (draft as any)[p] = v;
          },
        },
      },
      plugins: [makeRouter()],
    });
    expect(q('.o-stat-info__value')?.textContent).toBe('—');
    unmount();
  });

  test('router.push(to) after emit on click', async () => {
    const emitted: any[][] = [];
    const { unmount, click } = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'Go', to: { name: 'users' } },
      on: { onClick: (...args: any[]) => emitted.push(args) },
      plugins: [makeRouter()],
    });
    click('.o-stat-info');
    expect(emitted.length).toBe(1);
    expect(push.calls[0]?.[0]).toEqual({ name: 'users' });
    unmount();
  });

  test('does not render when visible=false; ignores click when disabled', async () => {
    const hidden = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'X', visible: false },
      plugins: [makeRouter()],
    });
    expect(hidden.q('.o-stat-info')).toBeFalsy();
    hidden.unmount();

    const emitted: any[][] = [];
    const disabled = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'X', disabled: true, to: { name: 'users' } },
      on: { onClick: (...args: any[]) => emitted.push(args) },
      plugins: [makeRouter()],
    });
    const btn = disabled.q('button') as HTMLButtonElement;
    btn.disabled = false;
    btn.click();
    expect(emitted.length).toBe(0);
    expect(push.calls.length).toBe(0);
    disabled.unmount();
  });
});

describe('OButtonBox', () => {
  test('does not render root when default slot is empty', () => {
    const empty = mountApp(OButtonBox as any, { slots: {} });
    expect(empty.q('.o-button-box')).toBeFalsy();
    empty.unmount();

    const withChild = mountApp(OButtonBox as any, {
      slots: { default: () => h(OStatInfo as any, { value: 1, label: 'A' }) },
      plugins: [
        createRouter({
          history: createMemoryHistory(),
          routes: [{ path: '/', component: { template: '<div />' } }],
        }),
      ],
    });
    expect(withChild.q('.o-button-box')).toBeTruthy();
    expect(withChild.q('.o-stat-info__value')?.textContent).toBe('1');
    withChild.unmount();
  });

  test('remounts shell when slot children appear later', async () => {
    const show = ref(false);
    const Host = defineComponent({
      components: { OButtonBox, OStatInfo },
      setup() {
        return { show };
      },
      template: `
        <OButtonBox>
          <OStatInfo v-if="show" :value="1" label="A" />
        </OButtonBox>
      `,
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    });
    const { unmount, q } = mountApp(Host as any, { plugins: [router] });
    expect(q('.o-button-box')).toBeFalsy();
    show.value = true;
    await nextTick();
    expect(q('.o-button-box')).toBeTruthy();
    expect(q('.o-stat-info__value')?.textContent).toBe('1');
    unmount();
  });
});

// OFormView #button-box slot source-order contracts (filesystem source scans) deferred to QJS knife.
