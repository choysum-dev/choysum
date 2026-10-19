// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Covers FE host stub branches for `vue-sonner` (aliased in unit host bundle).
 */
import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { toast, Toaster } from 'vue-sonner';

describe('vue-sonner FE stub', () => {
  test('toast levels, object titles, dismiss by id and clear-all', async () => {
    (toast as any)._clear?.();

    const defaultId = (toast as any)('plain');
    const objId = (toast as any)({ title: 'FromObject' });
    const emptyObjId = (toast as any)({});
    const nullId = (toast as any)(null);
    const successId = toast.success('ok');

    const entries = () => (toast as any)._entries as Array<{ id: number; level: string; title: string }>;
    expect(entries().length).toBe(5);
    expect(entries().find((e) => e.id === defaultId)?.level).toBe('default');
    expect(entries().find((e) => e.id === objId)?.title).toBe('FromObject');
    expect(entries().find((e) => e.id === emptyObjId)?.title).toBe('[object Object]');
    expect(entries().find((e) => e.id === nullId)?.title).toBe('');
    expect(entries().find((e) => e.id === successId)?.level).toBe('success');

    toast.dismiss(successId);
    expect(entries().some((e) => e.id === successId)).toBe(false);
    // Missing id is a no-op.
    toast.dismiss(999999);
    expect(entries().length).toBe(4);

    toast.dismiss();
    expect(entries().length).toBe(0);
  });

  test('Toaster stub mounts attrs and default slot', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            Toaster,
            { class: 'sonner-host', 'data-testid': 'sonner-toaster' },
            { default: () => h('span', { 'data-testid': 'sonner-slot' }, 'slot') },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-sonner-stub=Toaster]')).not.toBeNull();
    expect(mounted.q('[data-testid=sonner-toaster]')).not.toBeNull();
    expect(mounted.q('[data-testid=sonner-slot]')?.textContent).toBe('slot');
    mounted.unmount();
  });
});
