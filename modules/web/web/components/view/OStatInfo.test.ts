// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Comment, Fragment, Text, defineComponent, h, markRaw, nextTick, provide, ref } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
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
  const push = fnRecorder(async (to: unknown) => to);

  beforeEach(() => {
    push.mockReset();
    push.mockImplementation(async (to: unknown) => to);
  });

  function withRouter(extra?: { provide?: Record<string | symbol, unknown>; on?: Record<string, (...args: any[]) => void> }) {
    const { plugins } = buildPageMountGlobal({ router: { push } });
    return { plugins, ...extra };
  }

  test('renders value and label; emits click', async () => {
    const onClick = fnRecorder();
    const { unmount, q, setupState } = mountApp(OStatInfo as any, {
      props: { value: 5, label: 'Users' },
      ...withRouter({ on: { onClick } }),
    });
    expect(q('.o-stat-info__value')?.textContent).toBe('5');
    expect(q('.o-stat-info__label')?.textContent).toBe('Users');
    expect(q('.o-stat-info__icon')).toBeFalsy();
    setupState().onClick(new Event('click'));
    expect(onClick.calls.length).toBe(1);
    expect(push.calls.length).toBe(0);
    unmount();
  });

  test('renders icon when icon prop is set', () => {
    const IconStub = markRaw(
      defineComponent({
        name: 'IconStub',
        setup: () => () => h('span', { class: 'icon-stub' }),
      })
    );
    const { unmount, q } = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'Users', icon: IconStub },
      ...withRouter(),
      stubs: {
        ElIcon: {
          name: 'ElIcon',
          setup(_p: any, { slots }: any) {
            return () => h('i', { class: 'el-icon-stub' }, slots.default?.());
          },
        },
      },
    });
    expect(q('.o-stat-info__icon')).toBeTruthy();
    expect(q('.icon-stub')).toBeTruthy();
    unmount();
  });

  test('uses relation length via store+prop when value omitted', async () => {
    const Host = defineComponent({
      setup() {
        provide('form-root', {
          draft: { Users: [{ Id: '1' }, { Id: '2' }] },
        });
        return () =>
          h(OStatInfo as any, {
            store: { storeId: 's' },
            prop: 'Users',
            label: 'Users',
          });
      },
    });
    const { plugins } = buildPageMountGlobal({ router: { push } });
    const { unmount, q } = mountApp(Host, { plugins });
    await flushPromises();
    expect(q('.o-stat-info__value')?.textContent).toBe('2');
    unmount();
  });

  test('shows explicit value when only store or only prop is set', () => {
    const a = mountApp(OStatInfo as any, {
      props: { store: { storeId: 's' }, label: 'Users', value: 1 },
      ...withRouter(),
    });
    expect(a.q('.o-stat-info__value')?.textContent).toBe('1');
    a.unmount();

    const b = mountApp(OStatInfo as any, {
      props: { prop: 'Users', label: 'Users', value: 1 },
      ...withRouter(),
    });
    expect(b.q('.o-stat-info__value')?.textContent).toBe('1');
    b.unmount();
  });

  test('shows em dash when relation is not an array', async () => {
    const Host = defineComponent({
      setup() {
        provide('form-root', { draft: { Users: undefined } });
        return () =>
          h(OStatInfo as any, {
            store: { storeId: 's' },
            prop: 'Users',
            label: 'Users',
          });
      },
    });
    const { plugins } = buildPageMountGlobal({ router: { push } });
    const { unmount, q } = mountApp(Host, { plugins });
    await flushPromises();
    expect(q('.o-stat-info__value')?.textContent).toBe('—');
    unmount();
  });

  test('router.push(to) after emit on click', async () => {
    const onClick = fnRecorder();
    const { unmount, setupState } = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'Go', to: { name: 'users' } },
      ...withRouter({ on: { onClick } }),
    });
    setupState().onClick(new Event('click'));
    expect(onClick.calls.length).toBe(1);
    expect(push.calls[0]?.[0]).toEqual({ name: 'users' });
    unmount();
  });

  test('does not render when visible=false; ignores click when disabled', async () => {
    const hidden = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'X', visible: false },
      ...withRouter(),
    });
    expect(hidden.q('.o-stat-info')).toBeFalsy();
    hidden.unmount();

    const onClick = fnRecorder();
    const disabled = mountApp(OStatInfo as any, {
      props: { value: 1, label: 'X', disabled: true, to: { name: 'users' } },
      ...withRouter({ on: { onClick } }),
    });
    disabled.setupState().onClick(new Event('click'));
    expect(onClick.calls.length).toBe(0);
    expect(push.calls.length).toBe(0);
    disabled.unmount();
  });
});

describe('OButtonBox', () => {
  test('does not render root when default slot is empty', () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h('div', { class: 'host' }, [
            h(OButtonBox as any),
            h(OButtonBox as any, null, { default: () => h(OStatInfo as any, { value: 1, label: 'A' }) }),
          ]);
      },
    });
    const { plugins } = buildPageMountGlobal();
    const { unmount, q, qa } = mountApp(Host, { plugins });
    expect(qa('.o-button-box').length).toBe(1);
    expect(q('.o-stat-info__value')?.textContent).toBe('1');
    unmount();
  });

  test('remounts shell when slot children appear later', async () => {
    const show = ref(false);
    const Host = defineComponent({
      setup() {
        return () =>
          h('div', { class: 'host' }, [
            h(OButtonBox as any, null, {
              default: () => (show.value ? h(OStatInfo as any, { value: 1, label: 'A' }) : null),
            }),
          ]);
      },
    });
    const { plugins } = buildPageMountGlobal();
    const { unmount, q } = mountApp(Host, { plugins });
    expect(q('.o-button-box')).toBeFalsy();
    show.value = true;
    await nextTick();
    expect(q('.o-button-box')).toBeTruthy();
    expect(q('.o-stat-info__value')?.textContent).toBe('1');
    unmount();
  });
});
