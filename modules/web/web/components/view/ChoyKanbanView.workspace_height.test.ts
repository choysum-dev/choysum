// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Workspace height: board/body use flex-1 fill (no outer shell scroll); ActionBar
 * matches List chrome (no bottom border); page title is hosted in view chrome.
 */
import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import ChoyKanbanView from './ChoyKanbanView.vue';

function makeStore() {
  return {
    storeId: 'web.kanban.workspace.height',
    modelName: 'demo.Widget',
    fullModelName: 'demo.Widget',
    fieldsMetadata: {},
    state: { queryState: {} },
    meta: { fields: {} },
  } as any;
}

describe('ChoyKanbanView workspace height', () => {
  test('body and board keep flex fill classes without body padding', async () => {
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showHeader: false,
      },
    });
    await flushPromises();

    const root = mounted.q('[data-anchor="choy.kanban-view"]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root!.className).toContain('gap-4');

    const body = mounted.q('.choy-kanban-view__body') as HTMLElement | null;
    expect(body).not.toBeNull();
    expect(body!.className).toContain('min-h-0');
    expect(body!.className).toContain('flex-1');
    expect(body!.className).toContain('overflow-hidden');
    expect(body!.className.split(/\s+/)).not.toContain('p-1');

    const board = mounted.q('.choy-kanban-view__board') as HTMLElement | null;
    expect(board).not.toBeNull();
    expect(board!.className).toContain('min-h-0');
    expect(board!.className).toContain('flex-1');

    mounted.unmount();
  });

  test('ActionBar drops the bottom border like ListView', async () => {
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showHeader: true,
        showActions: true,
      },
    });
    await flushPromises();

    const bar = mounted.q('[data-slot="view-action-bar"]') as HTMLElement | null;
    expect(bar).not.toBeNull();
    const cls = bar!.className || '';
    expect(cls).toContain('border-b-0');
    expect(cls).toContain('[border-bottom-width:0px]');

    mounted.unmount();
  });

  test('hosts title overflow menu for Refresh like ListView', async () => {
    const mounted = mountApp(ChoyKanbanView as any, {
      props: {
        store: makeStore(),
        autoBootstrap: false,
        showHeader: true,
        showActions: true,
      },
    });
    await flushPromises();

    // Overflow items live in a portal; the ⋮ trigger is the stable List-parity signal.
    expect(mounted.q('[data-testid="page-io-menu-trigger"]')).not.toBeNull();

    mounted.unmount();
  });

  test('hosts page title in ActionBar when mounted under ChoyPage', async () => {
    const store = makeStore();
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyPage as any,
            {
              store,
              title: 'Token board',
              padding: true,
            },
            {
              default: () =>
                h(ChoyKanbanView as any, {
                  store,
                  autoBootstrap: false,
                  showHeader: true,
                  showActions: true,
                }),
            },
          );
      },
    });

    const mounted = mountApp(Host);
    await flushPromises();

    // Page band title is claimed away; ActionBar hosts it.
    const pageRoot = mounted.q('[data-anchor="choy.page"]') as HTMLElement | null;
    expect(pageRoot?.className || '').toContain('choy-inset-pad-no-top');

    const titles = mounted.qa('[data-slot=page-title]');
    expect(titles.length).toBe(1);
    expect(titles[0]?.textContent?.trim()).toBe('Token board');
    expect(titles[0]?.closest('[data-anchor="choy.kanban.view-chrome"]')).not.toBeNull();

    mounted.unmount();
  });
});
