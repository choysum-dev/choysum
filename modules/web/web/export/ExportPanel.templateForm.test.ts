// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/core/web/stores/registry';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';
import ExportPanel from './ExportPanel.vue';
import { ensureAbortController } from '@/core/web/testing/qjs_polyfills';
import { CreateWebClient } from '@/core/web/rpc/client_factory';
import { ExportHub } from '@/core/web/export/client';

type TemplateFormSetup = {
  resetTemplateSaveForm: () => void;
  templateSaveFormStore: UiModelStore | null | { value: UiModelStore | null };
  templateSaveFormInitialValues: Record<string, unknown> | { value: Record<string, unknown> };
  canSaveTemplate: boolean | { value: boolean };
  selectedFieldPaths: string[] | { value: string[] };
  selectedTemplateId: string | { value: string };
  saveCurrentTemplate: () => Promise<boolean>;
  onFooterCancel: () => Promise<void>;
  busy: boolean | { value: boolean };
};

function unwrapRef<T>(v: T | { value: T } | undefined | null): T | undefined {
  // Object(v) === v excludes null (typeof null === 'object').
  if (typeof v !== 'object' || Object(v) !== v) return v as T | undefined;
  const obj = v as object;
  if ('value' in obj) return (obj as { value: T }).value;
  return v as T;
}

function freshSetup(m: ReturnType<typeof mountPanel>): TemplateFormSetup {
  return m.setupState() as TemplateFormSetup;
}

function templateStore(m: ReturnType<typeof mountPanel>): UiModelStore | null {
  return unwrapRef(freshSetup(m).templateSaveFormStore) ?? null;
}

function canSave(m: ReturnType<typeof mountPanel>): boolean {
  return !!unwrapRef(freshSetup(m).canSaveTemplate);
}

function mountPanel(extra: Record<string, unknown> = {}) {
  return mountApp(ExportPanel as any, {
    props: {
      model: 'partner.Partner',
      companyId: 'c1',
      modelValue: true,
      defaultFields: ['Name', 'Email'],
      ...extra,
    },
  });
}

function asSetup(m: ReturnType<typeof mountPanel>): TemplateFormSetup {
  return m.setupState() as TemplateFormSetup;
}

ensureAbortController();

describe('ExportPanel template save FormView / UiModel', () => {
  let restoreExportTemplate: (() => void) | undefined;
  const templateSearch = fnRecorder(async () => [] as any[]);
  const templateCreate = fnRecorder(async (values: Record<string, unknown>) => ({
    Id: 'tpl-new',
    Name: values.Name,
    Fields: values.Fields,
    ImportCompatible: values.ImportCompatible,
    UserId: values.UserId,
    CreatedUid: 'me',
  }));

  beforeEach(() => {
    ensureAbortController();
    // Empty defaults: tests that set defaultFields via props stay in control of effectiveFields.
    const hub = CreateWebClient(ExportHub)() as any;
    hub.describeFields = async () => ({ fields: [], defaultFields: [] });
    templateSearch.mockReset();
    templateSearch.mockImplementation(async () => []);
    templateCreate.mockReset();
    templateCreate.mockImplementation(async (values: Record<string, unknown>) => ({
      Id: 'tpl-new',
      Name: values.Name,
      Fields: values.Fields,
      ImportCompatible: values.ImportCompatible,
      UserId: values.UserId,
      CreatedUid: 'me',
    }));
    restoreExportTemplate = replaceStoreFactory('web.ExportTemplate', () => ({
      Search: templateSearch,
      Create: templateCreate,
    }));
  });

  afterEach(() => {
    restoreExportTemplate?.();
    restoreExportTemplate = undefined;
  });

  test('resetTemplateSaveForm opens Name and Shared on a fresh UiModel store', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m) as TemplateFormSetup & {
      templateSaveFormKey: number | { value: number };
    };
    const keyBefore = Number(unwrapRef(s.templateSaveFormKey) ?? 0);
    s.resetTemplateSaveForm();
    const after = asSetup(m) as TemplateFormSetup & {
      templateSaveFormKey: number | { value: number };
    };
    expect(Number(unwrapRef(after.templateSaveFormKey) ?? 0)).toBe(keyBefore + 1);
    const store = templateStore(m);
    expect(store).toBeTruthy();
    expect(store!.getField('Name')).toBe('');
    expect(store!.getField('Shared')).toBe(false);
    const initial = unwrapRef(s.templateSaveFormInitialValues);
    expect(initial).toMatchObject({ Name: '', Shared: false });
    const nameMeta = store!.getFieldMeta('Name') as { string?: string; type?: string } | undefined;
    expect(nameMeta?.type).toBe('varchar');
    m.unmount();
  });

  test('canSaveTemplate is false without Name and true when Name is set', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    s.resetTemplateSaveForm();
    expect(canSave(m)).toBe(false);

    const store = templateStore(m)!;
    store.setField('Name', '  ');
    expect(canSave(m)).toBe(false);

    store.setField('Name', 'My columns');
    expect(canSave(m)).toBe(true);

    const paths = unwrapRef(freshSetup(m).selectedFieldPaths) ?? [];
    paths.splice(0, paths.length);
    // effectiveFields falls back to defaultFields when selection is empty.
    expect(canSave(m)).toBe(true);

    const noDefaults = mountPanel({ defaultFields: [] });
    await flushPromises();
    noDefaults.setupState().resetTemplateSaveForm();
    const bare = templateStore(noDefaults)!;
    bare.setField('Name', 'Named only');
    expect(canSave(noDefaults)).toBe(false);
    noDefaults.unmount();
    m.unmount();
  });

  test('saveCurrentTemplate reads Name/Shared from the store and clears fields after save', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    s.resetTemplateSaveForm();
    const store = templateStore(m)!;
    store.setField('Name', 'Cols');
    store.setField('Shared', true);
    expect(store.getValues().Name).toBe('Cols');
    expect(templateStore(m)!.getField('Name')).toBe('Cols');

    await s.saveCurrentTemplate();
    await flushPromises();

    expect(templateCreate.calls.length).toBe(1);
    expect(templateCreate.calls[0]![0]).toMatchObject({
      Name: 'Cols',
      Application: 'partner',
      ModelName: 'Partner',
      Fields: ['Name', 'Email'],
      UserId: null,
    });
    expect(unwrapRef(freshSetup(m).selectedTemplateId)).toBe('tpl-new');
    expect(templateStore(m)!.getField('Name')).toBe('');
    expect(templateStore(m)!.getField('Shared')).toBe(false);
    expect(unwrapRef(freshSetup(m).busy)).toBe(false);
    m.unmount();
  });

  test('onFooterCancel emits update:modelValue false', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    await s.onFooterCancel();
    expect(unwrapRef((freshSetup(m) as { visible?: boolean | { value: boolean } }).visible)).toBe(false);
    m.unmount();
  });


  test('submitTemplateForm returns handled on success and throws when save fails', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m) as TemplateFormSetup & {
      submitTemplateForm: () => Promise<{ handled: boolean; skipSuccessMessage?: boolean }>;
      exportError: string | { value: string };
    };
    s.resetTemplateSaveForm();
    templateStore(m)!.setField('Name', 'Cols');
    templateStore(m)!.setField('Shared', false);

    const ok = await s.submitTemplateForm();
    expect(ok).toEqual({ handled: true, skipSuccessMessage: true });

    templateCreate.mockImplementation(async () => {
      throw new Error('save failed');
    });
    templateStore(m)!.setField('Name', 'Again');
    let thrown: unknown;
    try {
      await s.submitTemplateForm();
    } catch (err) {
      thrown = err;
    }
    expect(thrown instanceof Error && /save failed|Failed to save/.test(thrown.message)).toBe(true);
    expect(String(unwrapRef(s.exportError) || '')).toContain('save failed');

    templateCreate.mockImplementation(async () => null as any);
    templateStore(m)!.setField('Name', '');
    const saved = await s.saveCurrentTemplate();
    expect(saved).toBe(false);
    m.unmount();
  });

  test('preparePanelOpen runs when visible flips true; templates disabled clears store', async () => {
    ensureAbortController();
    let describeCalls = 0;
    const hub = CreateWebClient(ExportHub)() as any;
    const origDescribe = hub.describeFields;
    hub.describeFields = async () => {
      describeCalls += 1;
      return { fields: [{ path: 'Name', label: 'Name' }], defaultFields: ['Name'] };
    };

    const m = mountApp(ExportPanel as any, {
      props: {
        model: 'partner.Partner',
        companyId: 'c1',
        modelValue: false,
        defaultFields: ['Name', 'Email'],
      },
      reactiveProps: true,
    });
    await flushPromises();
    expect(templateStore(m)).toBeNull();
    expect(describeCalls).toBe(0);

    (m.props as { modelValue: boolean }).modelValue = true;
    await flushPromises();
    expect(templateStore(m)).toBeTruthy();
    expect(describeCalls).toBeGreaterThan(0);
    m.unmount();

    const bare = mountPanel({ model: 'Partner', defaultFields: ['Name'], modelValue: false });
    await flushPromises();
    const bs = asSetup(bare) as TemplateFormSetup & { preparePanelOpen: () => void };
    bs.resetTemplateSaveForm();
    expect(templateStore(bare)).toBeTruthy();
    bs.preparePanelOpen();
    await flushPromises();
    expect(templateStore(bare)).toBeNull();
    bare.unmount();

    hub.describeFields = origDescribe;
  });



  test('abort helpers: AbortController path, host fallback, and DOMException AbortError', async () => {
    ensureAbortController();
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m) as TemplateFormSetup & {
      beginActiveRpc: () => AbortSignal;
      beginFieldsRpc: () => AbortSignal;
      isAbortError: (err: unknown) => boolean;
      noopAbortSignal: () => AbortSignal;
    };

    // Success path (FE host polyfill): beginActiveRpc try body.
    const live = s.beginActiveRpc();
    expect(live.aborted).toBe(false);

    // DOMException AbortError branch (guard + instanceof).
    const g = globalThis as any;
    const PrevDOM = g.DOMException;
    class FakeDOMException extends Error {
      constructor(message?: string, name?: string) {
        super(message);
        this.name = name || 'Error';
      }
    }
    g.DOMException = FakeDOMException;
    try {
      expect(s.isAbortError(new FakeDOMException('aborted', 'AbortError'))).toBe(true);
      expect(s.isAbortError(new FakeDOMException('other', 'NotAbort'))).toBe(false);
      const named = new Error('aborted');
      named.name = 'AbortError';
      expect(s.isAbortError(named)).toBe(true);
    } finally {
      if (PrevDOM === undefined) delete g.DOMException;
      else g.DOMException = PrevDOM;
    }

    // Host without usable AbortController: both begin* catch → noopAbortSignal.
    const PrevAC = g.AbortController;
    g.AbortController = class {
      constructor() {
        throw new Error('AbortController unavailable');
      }
    };
    try {
      const active = s.beginActiveRpc();
      expect(active.aborted).toBe(false);
      const fields = s.beginFieldsRpc();
      expect(fields.aborted).toBe(false);
      // Direct noop for the return-object line when begin* is not the only caller.
      expect(s.noopAbortSignal().aborted).toBe(false);
    } finally {
      g.AbortController = PrevAC;
    }
    m.unmount();
  });

  test('templateFormValues prefers FormView draft and syncTemplateStoreFromDraft mirrors it', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m) as TemplateFormSetup & {
      templateFormValues: () => { Name: string; Shared: boolean };
      syncTemplateStoreFromDraft: () => void;
      templateSaveFormViewRef: { getFormData?: () => unknown } | { value: { getFormData?: () => unknown } | null } | null;
    };
    s.resetTemplateSaveForm();
    const store = templateStore(m)!;
    store.setField('Name', 'FromStore');
    // Inject a FormView-like draft API.
    const api = { getFormData: () => ({ Name: 'FromDraft', Shared: true }) };
    const raw = s.templateSaveFormViewRef as any;
    if (raw && typeof raw === 'object' && 'value' in raw) raw.value = api;
    else (s as any).templateSaveFormViewRef = api;
    expect(s.templateFormValues()).toEqual({ Name: 'FromDraft', Shared: true });
    s.syncTemplateStoreFromDraft();
    expect(store.getField('Name')).toBe('FromDraft');
    expect(store.getField('Shared')).toBe(true);
    expect(canSave(m)).toBe(true);
    m.unmount();
  });

});
