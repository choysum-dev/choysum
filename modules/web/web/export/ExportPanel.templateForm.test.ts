// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/core/web/stores/registry';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';
import ExportPanel from './ExportPanel.vue';

type TemplateFormSetup = {
  resetTemplateSaveForm: () => void;
  templateSaveFormStore: UiModelStore | null | { value: UiModelStore | null };
  templateSaveFormInitialValues: Record<string, unknown> | { value: Record<string, unknown> };
  canSaveTemplate: boolean | { value: boolean };
  selectedFieldPaths: string[] | { value: string[] };
  selectedTemplateId: string | { value: string };
  saveCurrentTemplate: () => Promise<void>;
  onFooterCancel: () => Promise<void>;
  busy: boolean | { value: boolean };
};

function unwrapRef<T>(v: T | { value: T } | undefined | null): T | undefined {
  // Object(v) === v excludes null (typeof null === 'object').
  if (typeof v !== 'object' || Object(v) !== v) return v as T | undefined;
  if ('value' in v) return (v as { value: T }).value;
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
    const s = asSetup(m);
    s.resetTemplateSaveForm();
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
});