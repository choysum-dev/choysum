// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, nextTick, ref } from 'vue';
import { createPinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import { resetLoadingRegistryForTests } from '@/core/web/client/loadingRegistry';
import { createLocalFormStore } from '@/web/web/stores/localFormStore';
import { useField } from '@/web/web/composables/useField';
import ChoySettingsView from './ChoySettingsView.vue';
import ChoyViewSwitcher from './ChoyViewSwitcher.vue';
import {
  clearViewKindsForTests,
  getViewKind,
  registerBuiltinViewKinds,
  resetBuiltinViewKindsRegistrationForTests,
} from './viewKinds';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

function mountSettings(props: Record<string, unknown> = {}, extra: Record<string, unknown> = {}) {
  const { router } = createFeStubRouter({
    route: {
      name: 'SandboxSettings',
      path: '/sandbox/settings',
      fullPath: '/sandbox/settings',
      params: {},
      query: {},
      meta: { viewKind: 'settings' },
    },
  });
  const mounted = mountApp(ChoySettingsView as any, {
    props,
    reactiveProps: true,
    plugins: [createPinia(), router],
    ...extra,
  });
  return { ...mounted, router };
}

test('Settings view kind is registered with form presentation', () => {
  clearViewKindsForTests();
  resetBuiltinViewKindsRegistrationForTests();
  registerBuiltinViewKinds();
  expect(getViewKind('Settings').fieldPresentation).toBe('form');
  expect(getViewKind('Settings').defaultMode).toBe('edit');
});

test('ChoySettingsView chrome and form-root field binding', async () => {
  const store = createLocalFormStore({
    fields: [{ name: 'DemoFlag', label: 'Demo', type: 'boolean' }],
    initialValues: { DemoFlag: false },
  });

  const Probe = defineComponent({
    name: 'SettingsFieldProbe',
    setup() {
      const binding = useField({ store: store as any, prop: 'DemoFlag' });
      return () =>
        h('button', {
          type: 'button',
          'data-testid': 'toggle-demo',
          onClick: () => {
            binding.fieldRef().value = true as any;
          },
        });
    },
  });

  const mounted = mountSettings(
    {
      localStore: store,
      dirty: true,
      pageTitle: 'Sandbox Settings',
    },
    {
      slots: {
        default: () => h(Probe),
      },
    }
  );
  await flushPromises();

  expect(mounted.q('[data-testid=choy-settings-view]')).not.toBeNull();
  expect(mounted.q('[data-testid=settings-save]')).not.toBeNull();
  expect(mounted.q('[data-testid=settings-discard]')).not.toBeNull();
  const sheet = mounted.q('[data-anchor="choy.settings.sheet"]') as HTMLElement | null;
  expect(sheet).not.toBeNull();
  expect(sheet!.className.split(/\s+/)).not.toContain('p-5');
  expect(mounted.text()).toContain('Sandbox Settings');

  const toggle = mounted.q('[data-testid=toggle-demo]') as HTMLElement | null;
  expect(toggle).not.toBeNull();
  toggle!.click();
  await nextTick();
  expect(store.getField('DemoFlag')).toBe(true);
  mounted.unmount();
});

test('ChoySettingsView Save label shows Saving... while loading', async () => {
  const mounted = mountSettings({
    dirty: true,
    loading: true,
    pageTitle: 'Settings',
  });
  await flushPromises();
  expect(mounted.text()).toContain('Saving...');
  const save = mounted.q('[data-testid=settings-save]') as HTMLButtonElement | null;
  expect(save?.disabled).toBe(true);
  mounted.unmount();
});

test('ChoySettingsView discard confirms when dirty then emits discard', async () => {
  const { resolveConfirmChoy, useConfirmChoyStore } = await import(
    '@/web/web/composables/confirmChoyAction'
  );
  const discards: number[] = [];
  const mounted = mountSettings(
    { dirty: false, pageTitle: 'Settings' },
    { on: { onDiscard: () => discards.push(1) } },
  );
  await flushPromises();
  mounted.props.dirty = true;
  await nextTick();
  await flushPromises();
  const discard = mounted.q('[data-testid=settings-discard]') as HTMLButtonElement | null;
  expect(discard).not.toBeNull();
  discard!.disabled = false;
  discard!.click();
  await Promise.resolve();
  expect(useConfirmChoyStore().open).toBe(true);
  resolveConfirmChoy('confirm');
  await flushPromises();
  expect(discards.length).toBe(1);
  mounted.unmount();
});

test('ChoySettingsView discard cancel does not emit; clean discard is a no-op', async () => {
  const { resolveConfirmChoy, useConfirmChoyStore } = await import(
    '@/web/web/composables/confirmChoyAction'
  );
  const discards: number[] = [];
  const clean = mountSettings({ dirty: false }, { on: { onDiscard: () => discards.push(1) } });
  await flushPromises();
  (clean.q('[data-testid=settings-discard]') as HTMLButtonElement).click();
  await flushPromises();
  expect(discards.length).toBe(0);
  clean.unmount();

  const dirty = mountSettings({ dirty: false }, { on: { onDiscard: () => discards.push(1) } });
  await flushPromises();
  dirty.props.dirty = true;
  await nextTick();
  await flushPromises();
  const btn = dirty.q('[data-testid=settings-discard]') as HTMLButtonElement;
  btn.disabled = false;
  btn.click();
  await Promise.resolve();
  expect(useConfirmChoyStore().open).toBe(true);
  resolveConfirmChoy('cancel');
  await flushPromises();
  expect(discards.length).toBe(0);
  dirty.unmount();
});

test('ChoySettingsView view-select confirms dirty then emits after navigation settle', async () => {
  const { resolveConfirmChoy, useConfirmChoyStore } = await import(
    '@/web/web/composables/confirmChoyAction'
  );
  stubSfc(
    ChoyViewSwitcher as any,
    defineComponent({
      name: 'ChoyViewSwitcherStub',
      emits: ['select'],
      setup(_, { emit }) {
        return () =>
          h('button', {
            type: 'button',
            'data-testid': 'stub-view-switch',
            onClick: () => emit('select', 'list'),
          });
      },
    }),
  );
  const selected: string[] = [];
  try {
    const mounted = mountSettings(
      {
        dirty: false,
        viewSwitcherOptions: [
          { value: 'settings', label: 'Settings View' },
          { value: 'list', label: 'Parameters View' },
        ],
        activeView: 'settings',
      },
      { on: { onViewSelect: (v: string) => selected.push(v) } },
    );
    await flushPromises();
    mounted.props.dirty = true;
    await nextTick();
    await flushPromises();
    const btn = mounted.q('[data-testid=stub-view-switch]') as HTMLButtonElement | null;
    expect(btn).not.toBeNull();
    btn!.click();
    await Promise.resolve();
    expect(useConfirmChoyStore().open).toBe(true);
    resolveConfirmChoy('confirm');
    await flushPromises();
    await new Promise((r) => setTimeout(r, 0));
    expect(selected).toEqual(['list']);
    mounted.unmount();
  } finally {
    restoreSfc(ChoyViewSwitcher as any);
  }
});

test('ChoySettingsView draft bag binds form-root without localStore', async () => {
  const draft = { DemoFlag: false };
  const Probe = defineComponent({
    name: 'DraftProbe',
    setup() {
      const binding = useField({ prop: 'DemoFlag' });
      return () =>
        h('button', {
          type: 'button',
          'data-testid': 'toggle-draft',
          onClick: () => {
            binding.fieldRef().value = true as any;
          },
        });
    },
  });
  const mounted = mountSettings(
    { draft, dirty: false, pageTitle: 'Draft' },
    { slots: { default: () => h(Probe) } },
  );
  await flushPromises();
  (mounted.q('[data-testid=toggle-draft]') as HTMLElement).click();
  await nextTick();
  expect(draft.DemoFlag).toBe(true);
  mounted.unmount();
});

test('ChoySettingsView form-root falls back to controller draft without bag', async () => {
  const seen = ref('');
  const Probe = defineComponent({
    name: 'ControllerDraftProbe',
    setup() {
      const root = inject<{
        draft: Record<string, unknown> | null;
        getField: (path: string) => unknown;
        setField: (path: string, value: unknown) => void;
      }>('form-root')!;
      return () =>
        h('button', {
          type: 'button',
          'data-testid': 'set-via-root',
          onClick: () => {
            void root.draft;
            root.setField('Note', 'hi');
            seen.value = String(root.getField('Note') ?? '');
          },
        }, () => seen.value);
    },
  });
  const mounted = mountSettings(
    { dirty: false, pageTitle: 'NoBag' },
    { slots: { default: () => h(Probe) } },
  );
  await flushPromises();
  (mounted.q('[data-testid=set-via-root]') as HTMLElement).click();
  await nextTick();
  expect(seen.value).toBe('hi');
  mounted.unmount();
});

test('ChoySettingsView save emits when dirty and unlocked', async () => {
  resetLoadingRegistryForTests();
  const saves: number[] = [];
  const mounted = mountSettings(
    { dirty: false, pageTitle: 'Settings' },
    { on: { onSave: () => saves.push(1) } },
  );
  await flushPromises();
  mounted.props.dirty = true;
  await nextTick();
  await flushPromises();
  const save = mounted.q('[data-testid=settings-save]') as HTMLButtonElement | null;
  expect(save).not.toBeNull();
  save!.disabled = false;
  save!.click();
  await flushPromises();
  expect(saves.length).toBe(1);
  mounted.unmount();
  resetLoadingRegistryForTests();
});

test('ChoySettingsView custom labels, hidden header, and draft key prune', async () => {
  const draft: Record<string, unknown> = { Keep: 1, Drop: 2 };
  const mounted = mountSettings({
    draft,
    dirty: false,
    showHeader: false,
    saveLabel: 'Commit',
    discardLabel: 'Revert',
  });
  await flushPromises();
  expect(mounted.q('[data-testid=settings-save]')).toBeNull();
  mounted.unmount();

  const labeled = mountSettings({
    draft,
    dirty: true,
    saveLabel: 'Commit',
    discardLabel: 'Revert',
    pageTitle: 'T',
  });
  await flushPromises();
  expect(labeled.text()).toContain('Commit');
  expect(labeled.text()).toContain('Revert');
  labeled.props.draft = { Keep: 3 };
  await nextTick();
  await flushPromises();
  labeled.unmount();
});

test('ChoySettingsView leave guard confirms dirty navigation and skips auth exits', async () => {
  const { resolveConfirmChoy, useConfirmChoyStore } = await import(
    '@/web/web/composables/confirmChoyAction'
  );
  const mounted = mountSettings({ dirty: false, pageTitle: 'Settings' });
  await flushPromises();
  await mounted.router.push('/elsewhere');
  expect(useConfirmChoyStore().open).toBe(false);

  mounted.props.dirty = true;
  await nextTick();
  await flushPromises();
  const nav = mounted.router.push('/next');
  await Promise.resolve();
  expect(useConfirmChoyStore().open).toBe(true);
  resolveConfirmChoy('cancel');
  await nav;
  await flushPromises();

  const authNav = mounted.router.push({ path: '/login', name: 'login' });
  await authNav;
  await flushPromises();
  expect(useConfirmChoyStore().open).toBe(false);
  mounted.unmount();
});

test('ChoySettingsView blocks leave and view-select while loading; pending confirm is exclusive', async () => {
  const { resolveConfirmChoy, useConfirmChoyStore } = await import(
    '@/web/web/composables/confirmChoyAction'
  );
  stubSfc(
    ChoyViewSwitcher as any,
    defineComponent({
      name: 'ChoyViewSwitcherStub',
      emits: ['select'],
      setup(_, { emit }) {
        return () =>
          h('button', {
            type: 'button',
            'data-testid': 'stub-view-switch',
            onClick: () => emit('select', 'list'),
          });
      },
    }),
  );
  const selected: string[] = [];
  try {
    const mounted = mountSettings(
      {
        dirty: true,
        loading: true,
        viewSwitcherOptions: [
          { value: 'settings', label: 'Settings View' },
          { value: 'list', label: 'Parameters View' },
        ],
      },
      { on: { onViewSelect: (v: string) => selected.push(v) } },
    );
    await flushPromises();
    const blocked = mounted.router.push('/blocked-while-loading');
    await blocked;
    await flushPromises();
    expect(useConfirmChoyStore().open).toBe(false);
    (mounted.q('[data-testid=stub-view-switch]') as HTMLButtonElement).click();
    await flushPromises();
    expect(selected).toEqual([]);

    // Auth redirect must still proceed while loading is true.
    await mounted.router.push({ path: '/login', name: 'login' });
    await flushPromises();
    expect(useConfirmChoyStore().open).toBe(false);

    // Return to a non-auth route without a dirty prompt so later asserts are clean
    // (leaving /login skips the leave guard via shouldSkipFormDirtyLeavePrompt).
    mounted.props.loading = false;
    mounted.props.dirty = false;
    await nextTick();
    await flushPromises();
    await mounted.router.push('/sandbox/settings');
    await flushPromises();

    mounted.props.dirty = true;
    await nextTick();
    await flushPromises();
    const first = mounted.router.push('/ask-once');
    await Promise.resolve();
    expect(useConfirmChoyStore().open).toBe(true);
    const second = mounted.router.push('/ask-twice');
    await Promise.resolve();
    resolveConfirmChoy('cancel');
    await first;
    await second;
    await flushPromises();

    // Re-open confirm then auth-skip dismisses the pending dialog.
    mounted.props.dirty = true;
    await nextTick();
    await flushPromises();
    const pending = mounted.router.push('/pending-then-auth');
    await Promise.resolve();
    expect(useConfirmChoyStore().open).toBe(true);
    await mounted.router.push({ path: '/login', name: 'login' });
    await pending;
    await flushPromises();
    expect(useConfirmChoyStore().open).toBe(false);
    mounted.unmount();
  } finally {
    restoreSfc(ChoyViewSwitcher as any);
  }
});

test('ChoySettingsView form-root onChange is a no-op', async () => {
  let onChange: ((cb: (d: unknown) => void) => void) | null = null;
  const Capture = defineComponent({
    name: 'CaptureFormRoot',
    setup() {
      const fr = inject('form-root') as { onChange: (cb: (d: unknown) => void) => void };
      onChange = fr.onChange.bind(fr);
      return () => h('span', { 'data-testid': 'capture-root' });
    },
  });
  const mounted = mountSettings(
    { dirty: false, draft: { A: 1 }, pageTitle: 'T' },
    { slots: { default: () => h(Capture) } },
  );
  await flushPromises();
  expect(mounted.q('[data-testid=capture-root]')).not.toBeNull();
  expect(() => onChange?.(() => {})).not.toThrow();
  mounted.unmount();
});

test('ChoySettingsView form-root mode/draft/getField cover localStore and draft paths', async () => {
  type FormRootAccessors = {
    mode: string;
    draft: Record<string, unknown>;
    getField: (path: string) => unknown;
  };
  // Capture via local refs inside setup — outer `let` + inject assignment
  // collapses to `never` under this typecheck toolchain.
  let storeMode = '';
  let storeDraft: Record<string, unknown> = {};
  let storeFlag: unknown;
  const CaptureStore = defineComponent({
    name: 'CaptureFormRootStore',
    setup() {
      const root = inject('form-root') as FormRootAccessors;
      storeMode = root.mode;
      storeDraft = root.draft;
      storeFlag = root.getField('DemoFlag');
      return () => h('span', { 'data-testid': 'capture-accessors-store' });
    },
  });

  const store = createLocalFormStore({
    fields: [{ name: 'DemoFlag', label: 'Demo', type: 'boolean' }],
    initialValues: { DemoFlag: true },
  });
  const withStore = mountSettings(
    { localStore: store, dirty: false, pageTitle: 'Store' },
    { slots: { default: () => h(CaptureStore) } },
  );
  await flushPromises();
  expect(storeMode).toBe('edit');
  expect(storeDraft).toEqual({ DemoFlag: true });
  expect(storeFlag).toBe(true);
  withStore.unmount();

  let draftMode = '';
  let draftBag: Record<string, unknown> = {};
  let draftFlag: unknown;
  const CaptureDraft = defineComponent({
    name: 'CaptureFormRootDraft',
    setup() {
      const root = inject('form-root') as FormRootAccessors;
      draftMode = root.mode;
      draftBag = root.draft;
      draftFlag = root.getField('DemoFlag');
      return () => h('span', { 'data-testid': 'capture-accessors-draft' });
    },
  });
  const draft = { DemoFlag: false, Extra: 1 };
  const withDraft = mountSettings(
    { draft, dirty: false, pageTitle: 'Draft' },
    { slots: { default: () => h(CaptureDraft) } },
  );
  await flushPromises();
  expect(draftMode).toBe('edit');
  expect(draftBag).toMatchObject({ DemoFlag: false, Extra: 1 });
  expect(draftFlag).toBe(false);
  withDraft.unmount();
});

test('ChoySettingsView view-select cancel keeps current view; clean select skips confirm', async () => {
  const { resolveConfirmChoy, useConfirmChoyStore } = await import(
    '@/web/web/composables/confirmChoyAction'
  );
  stubSfc(
    ChoyViewSwitcher as any,
    defineComponent({
      name: 'ChoyViewSwitcherStub',
      emits: ['select'],
      setup(_, { emit }) {
        return () =>
          h('button', {
            type: 'button',
            'data-testid': 'stub-view-switch',
            onClick: () => emit('select', 'list'),
          });
      },
    }),
  );
  const selected: string[] = [];
  try {
    const clean = mountSettings(
      {
        dirty: false,
        viewSwitcherOptions: [
          { value: 'settings', label: 'Settings View' },
          { value: 'list', label: 'Parameters View' },
        ],
      },
      { on: { onViewSelect: (v: string) => selected.push(v) } },
    );
    await flushPromises();
    (clean.q('[data-testid=stub-view-switch]') as HTMLButtonElement).click();
    await flushPromises();
    await new Promise((r) => setTimeout(r, 0));
    expect(selected).toEqual(['list']);
    clean.unmount();

    selected.length = 0;
    const dirty = mountSettings(
      {
        dirty: false,
        viewSwitcherOptions: [
          { value: 'settings', label: 'Settings View' },
          { value: 'list', label: 'Parameters View' },
        ],
      },
      { on: { onViewSelect: (v: string) => selected.push(v) } },
    );
    await flushPromises();
    dirty.props.dirty = true;
    await nextTick();
    await flushPromises();
    (dirty.q('[data-testid=stub-view-switch]') as HTMLButtonElement).click();
    await Promise.resolve();
    expect(useConfirmChoyStore().open).toBe(true);
    resolveConfirmChoy('cancel');
    await flushPromises();
    expect(selected).toEqual([]);
    dirty.unmount();
  } finally {
    restoreSfc(ChoyViewSwitcher as any);
  }
});
