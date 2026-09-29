// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';

import type { PageIoMenuItem } from '@/web/web/composables/recordIoTypes';
import { providePageContext } from '@/web/web/composables/usePageContext';
import { RecordExportShell } from '@/web/web/export';
import { RecordImportShell } from '@/web/web/import';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
  type MountAppResult,
} from '@/web/web/__tests__/mountApp';
import ChoyPageIoMenu from '@/web/web/components/layout/ChoyPageIoMenu.vue';
import DropdownMenu from '../vendor/ui/dropdown-menu/DropdownMenu.vue';
import DropdownMenuContent from '../vendor/ui/dropdown-menu/DropdownMenuContent.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuTrigger from '../vendor/ui/dropdown-menu/DropdownMenuTrigger.vue';

describe('ChoyPageIoMenu', () => {
  const menuSfcs = [DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger];

  function installMenuStubs() {
    stubSfc(DropdownMenu as any, {
      name: 'DropdownMenu',
      setup(_: any, { slots }: any) {
        return () => h('div', { 'data-testid': 'dropdown' }, slots.default?.());
      },
    });
    stubSfc(DropdownMenuTrigger as any, {
      name: 'DropdownMenuTrigger',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'dropdown-trigger' }, slots.default?.());
      },
    });
    stubSfc(DropdownMenuContent as any, {
      name: 'DropdownMenuContent',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'dropdown-content' }, slots.default?.());
      },
    });
    stubSfc(DropdownMenuItem as any, {
      name: 'DropdownMenuItem',
      props: {
        disabled: { type: Boolean, default: false },
      },
      inheritAttrs: false,
      emits: ['select'],
      setup(props: any, { slots, emit, attrs }: any) {
        return () =>
          h(
            'button',
            {
              type: 'button',
              disabled: props.disabled || undefined,
              'data-testid': attrs['data-testid'],
              onClick: () => {
                if (props.disabled) return;
                emit('select');
              },
            },
            slots.default?.()
          );
      },
    });
  }

  beforeEach(() => {
    installMenuStubs();
    stubSfc(RecordImportShell as any, {
      name: 'RecordImportShell',
      props: {
        modelValue: { type: Boolean, default: false },
        open: { type: Boolean, default: false },
        model: { type: String, default: '' },
        config: { type: Object, default: undefined },
        companyId: { type: String, default: '' },
      },
      emits: ['update:open', 'update:modelValue', 'imported'],
      setup(props: any, { emit }: any) {
        return () =>
          h(
            'div',
            {
              'data-testid': 'import-shell-stub',
              'data-model': props.model || '',
              'data-company-id': props.companyId || '',
              'data-open': String(props.open ?? props.modelValue ?? false),
              'data-hint': props.config?.import?.uploadHint || '',
            },
            [
              h(
                'button',
                {
                  type: 'button',
                  'data-testid': 'emit-imported',
                  onClick: () => emit('imported'),
                },
                'import'
              ),
            ]
          );
      },
    });
    stubSfc(RecordExportShell as any, {
      name: 'RecordExportShell',
      props: {
        modelValue: { type: Boolean, default: false },
        open: { type: Boolean, default: false },
        model: { type: String, default: '' },
        store: { type: Object, default: undefined },
        listRef: { type: Object, default: undefined },
        companyId: { type: String, default: '' },
      },
      emits: ['update:open', 'update:modelValue'],
      setup(props: any) {
        return () =>
          h('div', {
            'data-testid': 'export-shell-stub',
            'data-model': props.model || '',
            'data-open': String(props.open ?? props.modelValue ?? false),
            'data-list-id': props.listRef?.selectedItems?.value?.[0]?.Id || '',
          });
      },
    });
  });

  afterEach(() => {
    for (const Comp of menuSfcs) restoreSfc(Comp as any);
    restoreSfc(RecordImportShell as any);
    restoreSfc(RecordExportShell as any);
  });

  function mountMenu(props: Record<string, unknown> = {}, extra: Record<string, unknown> = {}) {
    return mountApp(ChoyPageIoMenu as any, {
      props,
      stubs: { Setting: true },
      ...extra,
    });
  }

  /** Invoke product onCommand for keys that may not have a visible item button. */
  function emitCommand(mounted: MountAppResult, cmd: string) {
    const ss = mounted.setupState();
    expect(ss).toBeTruthy();
    expect(typeof ss.onCommand).toBe('function');
    ss.onCommand(cmd);
  }

  test('renders visible items and ignores hidden ones', async () => {
    const onImport = fnRecorder();
    const onExport = fnRecorder();
    const mounted = mountMenu({
      items: [
        { key: 'import', label: 'Import', onClick: onImport },
        { key: 'export', label: 'Export', hidden: true, onClick: onExport },
      ] satisfies PageIoMenuItem[],
    });
    await flushPromises();
    expect(mounted.q('[data-testid=page-io-menu-trigger]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-export]')).toBeFalsy();
    expect(mounted.q('[data-testid=import-shell-stub]')).toBeFalsy();
    mounted.unmount();
  });

  test('hides the dropdown when every item is hidden', async () => {
    const mounted = mountMenu({
      items: [{ key: 'import', label: 'Import', hidden: true, onClick: () => undefined }],
    });
    await flushPromises();
    expect(mounted.q('[data-testid=dropdown]')).toBeFalsy();
    mounted.unmount();
  });

  test('treats a missing items prop as an empty list', async () => {
    const mounted = mountMenu({});
    await flushPromises();
    expect(mounted.q('[data-testid=dropdown]')).toBeFalsy();
    mounted.unmount();
  });

  test('invokes onClick for the commanded item and ignores unknown keys', async () => {
    const onImport = fnRecorder();
    const mounted = mountMenu({
      items: [{ key: 'import', label: 'Import', onClick: onImport }],
    });
    await flushPromises();
    mounted.click('[data-testid=page-io-menu-import]');
    expect(onImport.calls.length).toBe(1);
    emitCommand(mounted, 'missing');
    expect(onImport.calls.length).toBe(1);
    mounted.unmount();
  });

  test('does not invoke onClick for hidden items when commanded', async () => {
    const onImport = fnRecorder();
    const onExport = fnRecorder();
    const mounted = mountMenu({
      items: [
        { key: 'import', label: 'Import', onClick: onImport },
        { key: 'export', label: 'Export', hidden: true, onClick: onExport },
      ],
    });
    await flushPromises();
    emitCommand(mounted, 'export');
    expect(onExport.calls.length).toBe(0);
    expect(onImport.calls.length).toBe(0);
    mounted.unmount();
  });

  test('does not invoke onClick for disabled items when commanded', async () => {
    const onImport = fnRecorder();
    const mounted = mountMenu({
      items: [{ key: 'import', label: 'Import', disabled: true, onClick: onImport }],
    });
    await flushPromises();
    emitCommand(mounted, 'import');
    expect(onImport.calls.length).toBe(0);
    mounted.unmount();
  });

  test('derives menu and panels from action-import/export flags', async () => {
    const refresh = fnRecorder();
    const onImported = fnRecorder();
    const store = { storeId: 's1', fullModelName: 'partner.Partner', state: { result: { total: 2 } } };
    const mounted = mountMenu(
      {
        actionImport: true,
        actionExport: true,
        actionImportUploadHint: 'hint',
        store,
        actionListRef: { refresh, selectedItems: { value: [{ Id: '1' }] } },
      },
      { on: { onImported } }
    );
    await flushPromises();
    expect(mounted.q('[data-testid=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-export]')).toBeTruthy();
    expect(mounted.q('[data-testid=import-shell-stub]')?.getAttribute('data-model')).toBe(
      'partner.Partner'
    );
    expect(mounted.q('[data-testid=export-shell-stub]')?.getAttribute('data-model')).toBe(
      'partner.Partner'
    );
    expect(mounted.q('[data-testid=import-shell-stub]')?.getAttribute('data-hint')).toBe('hint');

    mounted.click('[data-testid=page-io-menu-import]');
    await flushPromises();
    expect(mounted.q('[data-testid=import-shell-stub]')?.getAttribute('data-open')).toBe('true');

    mounted.click('[data-testid=emit-imported]');
    await flushPromises();
    expect(refresh.calls.length).toBe(1);
    expect(onImported.calls.length).toBe(1);
    mounted.unmount();
  });

  test('skips panels when store is missing', async () => {
    const mounted = mountMenu({
      actionImport: true,
      actionExport: true,
    });
    await flushPromises();
    expect(mounted.q('[data-testid=import-shell-stub]')).toBeFalsy();
    expect(mounted.q('[data-testid=export-shell-stub]')).toBeFalsy();
    // Title-row entries still render from action flags so the trigger stays reachable.
    expect(mounted.q('[data-testid=page-io-menu-trigger]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-export]')).toBeTruthy();
    mounted.unmount();
  });

  test('warns when import/export is clicked without a resolvable model/store', async () => {
    const { ChoyMessage } = await import('@/web/web/composables/useChoyMessage');
    const origWarn = ChoyMessage.warning;
    const warnings: string[] = [];
    ChoyMessage.warning = ((msg: string) => {
      warnings.push(String(msg));
    }) as typeof ChoyMessage.warning;
    try {
      const mounted = mountMenu({
        actionImport: true,
        actionExport: true,
      });
      await flushPromises();
      mounted.click('[data-testid=page-io-menu-import]');
      mounted.click('[data-testid=page-io-menu-export]');
      await flushPromises();
      expect(warnings.length).toBe(2);
      expect(warnings[0]).toMatch(/model\/store/i);
    } finally {
      ChoyMessage.warning = origWarn;
    }
  });

  test('skips panels when store lacks fullModelName', async () => {
    const mounted = mountMenu({
      actionImport: true,
      actionExport: true,
      store: { storeId: 's1', state: { result: { total: 1 } } },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=import-shell-stub]')).toBeFalsy();
    expect(mounted.q('[data-testid=export-shell-stub]')).toBeFalsy();
    expect(mounted.q('[data-testid=page-io-menu-trigger]')).toBeTruthy();
    mounted.unmount();
  });

  test('resolves list ref from the page-registered action target', async () => {
    const refresh = fnRecorder();
    const pageStore = {
      storeId: 'from-page',
      fullModelName: 'partner.Partner',
      state: { result: { total: 1 } },
    };
    const Host = defineComponent({
      setup(_, { slots }) {
        const ctx = providePageContext({ store: pageStore });
        ctx.registerActionTarget({
          refresh,
          selectedItems: { value: [{ Id: 'a' }] },
        });
        return () => slots.default?.();
      },
    });
    const mounted = mountApp(Host as any, {
      slots: {
        default: () =>
          h(ChoyPageIoMenu as any, {
            actionImport: true,
            actionExport: true,
          }),
      },
      stubs: { Setting: true },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=export-shell-stub]')?.getAttribute('data-list-id')).toBe('a');
    mounted.click('[data-testid=emit-imported]');
    await flushPromises();
    expect(refresh.calls.length).toBe(1);
    mounted.unmount();
  });

  test('opens export from the derived menu command', async () => {
    const store = { storeId: 's1', fullModelName: 'partner.Partner', state: { result: { total: 1 } } };
    const mounted = mountMenu({
      actionExport: true,
      store,
    });
    await flushPromises();
    expect(mounted.q('[data-testid=page-io-menu-export]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-import]')).toBeFalsy();
    mounted.click('[data-testid=page-io-menu-export]');
    await flushPromises();
    expect(mounted.q('[data-testid=export-shell-stub]')?.getAttribute('data-open')).toBe('true');
    mounted.unmount();
  });

  test('derives import-only menu without export panels', async () => {
    const store = { storeId: 's1', fullModelName: 'partner.Partner', state: { result: { total: 1 } } };
    const mounted = mountMenu({
      actionImport: true,
      store,
    });
    await flushPromises();
    expect(mounted.q('[data-testid=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-testid=page-io-menu-export]')).toBeFalsy();
    expect(mounted.q('[data-testid=export-shell-stub]')).toBeFalsy();
    mounted.unmount();
  });

  test('emits imported without refresh when no list ref is available', async () => {
    const store = { storeId: 's1', fullModelName: 'partner.Partner', state: { result: { total: 1 } } };
    const onImported = fnRecorder();
    const mounted = mountMenu(
      {
        actionImport: true,
        store,
      },
      { on: { onImported } }
    );
    await flushPromises();
    mounted.click('[data-testid=emit-imported]');
    await flushPromises();
    expect(onImported.calls.length).toBe(1);
    mounted.unmount();
  });
});
