// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, provide, type InjectionKey } from 'vue';
import {
  ElButton,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElIcon,
} from 'element-plus';

import type { PageIoMenuItem } from '@/web/web/composables/recordIoTypes';
import { provideOPageContext } from '@/web/web/composables/usePageContext';
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
import OPageIoMenu from './OPageIoMenu.vue';

const DropdownCommandKey: InjectionKey<(cmd: string) => void> = Symbol('dropdown-command');

describe('OPageIoMenu', () => {
  function installEpStubs() {
    stubSfc(ElDropdown as any, {
      name: 'ElDropdown',
      emits: ['command'],
      setup(_: any, { slots, emit }: any) {
        provide(DropdownCommandKey, (cmd: string) => emit('command', cmd));
        return () =>
          h('div', { 'data-test': 'dropdown' }, [
            slots.default?.(),
            slots.dropdown?.(),
            h('button', {
              type: 'button',
              'data-test': 'dropdown-emit-command',
              onClick: (event: Event) => {
                const target = event.currentTarget as HTMLElement | null;
                const cmd = target?.getAttribute('data-command');
                if (cmd != null) emit('command', cmd);
              },
            }),
          ]);
      },
    });
    stubSfc(ElDropdownMenu as any, {
      setup(_: any, { slots }: any) {
        return () => h('div', {}, slots.default?.());
      },
    });
    stubSfc(ElDropdownItem as any, {
      props: {
        command: { type: [String, Number, Object], default: undefined },
        disabled: { type: Boolean, default: false },
      },
      inheritAttrs: false,
      setup(props: any, { slots, attrs }: any) {
        const fire = inject(DropdownCommandKey, null);
        return () =>
          h(
            'button',
            {
              type: 'button',
              disabled: props.disabled || undefined,
              'data-test': attrs['data-test'] || `page-io-menu-${props.command}`,
              onClick: () => {
                // Always forward command so product onCommand can guard disabled/hidden.
                fire?.(String(props.command ?? ''));
              },
            },
            slots.default?.()
          );
      },
    });
    stubSfc(ElButton as any, {
      inheritAttrs: false,
      setup(_: any, { slots, attrs }: any) {
        return () => h('button', { type: 'button', ...attrs }, slots.default?.());
      },
    });
    stubSfc(ElIcon as any, {
      setup(_: any, { slots }: any) {
        return () => h('span', {}, slots.default?.());
      },
    });
  }

  beforeEach(() => {
    installEpStubs();
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
              'data-test': 'import-shell-stub',
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
                  'data-test': 'emit-imported',
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
            'data-test': 'export-shell-stub',
            'data-model': props.model || '',
            'data-open': String(props.open ?? props.modelValue ?? false),
            'data-list-id': props.listRef?.selectedItems?.value?.[0]?.Id || '',
          });
      },
    });
  });

  afterEach(() => {
    restoreSfc(ElDropdown as any);
    restoreSfc(ElDropdownMenu as any);
    restoreSfc(ElDropdownItem as any);
    restoreSfc(ElButton as any);
    restoreSfc(ElIcon as any);
    restoreSfc(RecordImportShell as any);
    restoreSfc(RecordExportShell as any);
  });

  function mountMenu(props: Record<string, unknown> = {}, extra: Record<string, unknown> = {}) {
    return mountApp(OPageIoMenu as any, {
      props,
      stubs: { Setting: true },
      ...extra,
    });
  }

  /** Fire `@command` for keys with no visible item button (unknown / hidden). */
  function emitCommand(mounted: MountAppResult, cmd: string) {
    const btn = mounted.q('[data-test=dropdown-emit-command]') as HTMLElement | null;
    expect(btn).toBeTruthy();
    btn!.setAttribute('data-command', cmd);
    mounted.click('[data-test=dropdown-emit-command]');
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
    expect(mounted.q('[data-test=page-io-menu-trigger]')).toBeTruthy();
    expect(mounted.q('[data-test=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-test=page-io-menu-export]')).toBeFalsy();
    expect(mounted.q('[data-test=import-shell-stub]')).toBeFalsy();
    mounted.unmount();
  });

  test('hides the dropdown when every item is hidden', async () => {
    const mounted = mountMenu({
      items: [{ key: 'import', label: 'Import', hidden: true, onClick: () => undefined }],
    });
    await flushPromises();
    expect(mounted.q('[data-test=dropdown]')).toBeFalsy();
    mounted.unmount();
  });

  test('treats a missing items prop as an empty list', async () => {
    const mounted = mountMenu({});
    await flushPromises();
    expect(mounted.q('[data-test=dropdown]')).toBeFalsy();
    mounted.unmount();
  });

  test('invokes onClick for the commanded item and ignores unknown keys', async () => {
    const onImport = fnRecorder();
    const mounted = mountMenu({
      items: [{ key: 'import', label: 'Import', onClick: onImport }],
    });
    await flushPromises();
    mounted.click('[data-test=page-io-menu-import]');
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
    // Fire via dropdown command path (disabled HTML buttons may swallow DOM clicks).
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
    expect(mounted.q('[data-test=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-test=page-io-menu-export]')).toBeTruthy();
    expect(mounted.q('[data-test=import-shell-stub]')?.getAttribute('data-model')).toBe(
      'partner.Partner'
    );
    expect(mounted.q('[data-test=export-shell-stub]')?.getAttribute('data-model')).toBe(
      'partner.Partner'
    );
    expect(mounted.q('[data-test=import-shell-stub]')?.getAttribute('data-hint')).toBe('hint');

    mounted.click('[data-test=page-io-menu-import]');
    await flushPromises();
    expect(mounted.q('[data-test=import-shell-stub]')?.getAttribute('data-open')).toBe('true');

    mounted.click('[data-test=emit-imported]');
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
    expect(mounted.q('[data-test=import-shell-stub]')).toBeFalsy();
    expect(mounted.q('[data-test=export-shell-stub]')).toBeFalsy();
    expect(mounted.q('[data-test=page-io-menu-import]')).toBeFalsy();
    expect(mounted.q('[data-test=page-io-menu-export]')).toBeFalsy();
    mounted.unmount();
  });

  test('skips panels when store lacks fullModelName', async () => {
    const mounted = mountMenu({
      actionImport: true,
      actionExport: true,
      store: { storeId: 's1', state: { result: { total: 1 } } },
    });
    await flushPromises();
    expect(mounted.q('[data-test=import-shell-stub]')).toBeFalsy();
    expect(mounted.q('[data-test=export-shell-stub]')).toBeFalsy();
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
        const ctx = provideOPageContext({ store: pageStore });
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
          h(OPageIoMenu as any, {
            actionImport: true,
            actionExport: true,
          }),
      },
      stubs: { Setting: true },
    });
    await flushPromises();
    expect(mounted.q('[data-test=export-shell-stub]')?.getAttribute('data-list-id')).toBe('a');
    mounted.click('[data-test=emit-imported]');
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
    expect(mounted.q('[data-test=page-io-menu-export]')).toBeTruthy();
    expect(mounted.q('[data-test=page-io-menu-import]')).toBeFalsy();
    mounted.click('[data-test=page-io-menu-export]');
    await flushPromises();
    expect(mounted.q('[data-test=export-shell-stub]')?.getAttribute('data-open')).toBe('true');
    mounted.unmount();
  });

  test('derives import-only menu without export panels', async () => {
    const store = { storeId: 's1', fullModelName: 'partner.Partner', state: { result: { total: 1 } } };
    const mounted = mountMenu({
      actionImport: true,
      store,
    });
    await flushPromises();
    expect(mounted.q('[data-test=page-io-menu-import]')).toBeTruthy();
    expect(mounted.q('[data-test=page-io-menu-export]')).toBeFalsy();
    expect(mounted.q('[data-test=export-shell-stub]')).toBeFalsy();
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
    mounted.click('[data-test=emit-imported]');
    await flushPromises();
    expect(onImported.calls.length).toBe(1);
    mounted.unmount();
  });
});
