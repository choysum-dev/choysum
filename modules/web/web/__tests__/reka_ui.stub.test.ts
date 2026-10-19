// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Covers FE host stub branches for `reka-ui` (aliased in unit host bundle).
 */
import { defineComponent, h, nextTick, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  CollapsibleContent,
  CollapsibleRoot,
  CollapsibleTrigger,
  DialogRoot,
  Primitive,
  createContext,
  useFilter,
  useForwardProps,
  useForwardPropsEmits,
  useId,
} from 'reka-ui';

describe('reka-ui FE stub', () => {
  test('createContext provide/inject and fallible miss', async () => {
    const [useCtx, provideCtx] = createContext('stub-ctx');
    const seen = ref<string | null>(null);
    const Host = defineComponent({
      setup() {
        provideCtx({ label: 'ok' });
        return () =>
          h(
            defineComponent({
              setup() {
                seen.value = (useCtx() as { label: string }).label;
                expect(useCtx(true)).toBeTruthy();
                return () => h('div', { 'data-testid': 'ctx' }, seen.value || '');
              },
            }),
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.text()).toContain('ok');
    mounted.unmount();

    const [useMissing] = createContext('missing');
    const Miss = defineComponent({
      setup() {
        expect(useMissing(true)).toBeUndefined();
        let threw = false;
        try {
          useMissing(false);
        } catch {
          threw = true;
        }
        expect(threw).toBe(true);
        return () => h('div');
      },
    });
    const missMounted = mountApp(Miss as any);
    await flushPromises();
    missMounted.unmount();
  });

  test('helpers and DialogRoot open/closed slot API', async () => {
    expect(useForwardPropsEmits({ a: 1 })).toEqual({ a: 1 });
    expect(useForwardProps({ b: 2 })).toEqual({ b: 2 });
    expect(String(useId())).toContain('reka-stub-');
    expect(useFilter().contains('Hello', 'he')).toBe(true);
    expect(useFilter().contains('Hello', 'z')).toBe(false);
    expect(useFilter().contains('Hello', '')).toBe(true);

    const open = ref(true);
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            DialogRoot,
            {
              open: open.value,
              'onUpdate:open': (v: boolean) => {
                open.value = v;
              },
            },
            {
              default: ({ close }: { close: () => void }) =>
                h('button', { 'data-testid': 'close', onClick: close }, 'x'),
            },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-reka-stub=DialogRoot]')?.getAttribute('data-state')).toBe('open');
    mounted.q('[data-testid=close]')!.dispatchEvent(new Event('click'));
    await flushPromises();
    await nextTick();
    open.value = false;
    const closed = mountApp(
      defineComponent({
        setup: () => () => h(DialogRoot, { open: false }, { default: () => h('span', 'c') }),
      }) as any,
    );
    await flushPromises();
    expect(closed.q('[data-reka-stub=DialogRoot]')?.getAttribute('data-state')).toBe('closed');
    closed.unmount();
    mounted.unmount();

    const undefinedOpen = mountApp(
      defineComponent({
        setup: () => () => h(DialogRoot, {}, { default: () => h('span', 'u') }),
      }) as any,
    );
    await flushPromises();
    expect(undefinedOpen.q('[data-reka-stub=DialogRoot]')?.getAttribute('data-state')).toBe('open');
    undefinedOpen.unmount();
  });

  test('CollapsibleRoot/Trigger/Content and Primitive asChild branches', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            CollapsibleRoot,
            { defaultOpen: true },
            {
              default: () => [
                h(CollapsibleTrigger, {}, { default: () => 'toggle' }),
                h(CollapsibleContent, {}, { default: () => h('span', { 'data-testid': 'body' }, 'body') }),
              ],
            },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-reka-stub=CollapsibleRoot]')?.getAttribute('data-state')).toBe('open');
    expect(mounted.q('[data-testid=body]')).not.toBeNull();
    mounted.q('[data-reka-stub=CollapsibleTrigger]')!.dispatchEvent(new Event('click'));
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-reka-stub=CollapsibleRoot]')?.getAttribute('data-state')).toBe('closed');
    expect(mounted.q('[data-testid=body]')).toBeNull();
    mounted.unmount();

    const controlled = mountApp(
      defineComponent({
        setup: () => () =>
          h(CollapsibleRoot, { open: false }, {
            default: () => [
              h(CollapsibleTrigger, { asChild: true }, { default: () => h('span', 'child') }),
              h(CollapsibleContent, {}, { default: () => 'hidden' }),
            ],
          }),
      }) as any,
    );
    await flushPromises();
    expect(controlled.q('[data-reka-stub=CollapsibleRoot]')?.getAttribute('data-state')).toBe(
      'closed',
    );
    controlled.unmount();

    const prim = mountApp(
      defineComponent({
        setup: () => () =>
          h('div', [
            h(Primitive, { as: 'section' }, { default: () => 'sec' }),
            h(Primitive, { asChild: true }, { default: () => h('em', 'child') }),
            h(Primitive, { asChild: true }),
          ]),
      }) as any,
    );
    await flushPromises();
    expect(prim.text()).toContain('sec');
    expect(prim.text()).toContain('child');
    prim.unmount();
  });
});
