// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';

import { CreateWebClient } from '@/core/web/rpc/client_factory';
import { ImportHub } from '@/core/web/import/pb/import_pb';
import { replaceStoreFactory } from '@/core/web/stores/registry';
import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import ImportPanel from './ImportPanel.vue';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';
import type { ChoyFormDialogViewApi } from '@/web/web/composables/useChoyFormDialogShell';

type MappingSetup = {
  mappingRows: Array<{ header: string; fieldPath: string }>;
  mappingFormStore: UiModelStore | null;
  mappingFormViewRef: ChoyFormDialogViewApi | { value: ChoyFormDialogViewApi | null } | null;
  mappingPropByHeader: Record<string, string> | { value: Record<string, string> };
  catalogFields: Array<{ path?: string; label?: string; children?: unknown[] }>;
  selectedFile: File | null | { value: File | null };
  sourceRef: string | { value: string };
  previewReport: unknown;
  sanitizeHeaderFieldName: (header: string, used: Set<string>) => string;
  rebuildMappingFormStore: () => void;
  syncMappingRowsFromStore: () => void;
  mappingFormValues: () => Record<string, unknown>;
  mappingPropForHeader: (header: string) => string;
  onMappingFormChange: () => void;
  onFileSelected: (file: { raw?: File | null; name?: string }) => void;
  onFileRemoved: () => void;
  uploadAndPreview: () => Promise<void>;
  commitImport: () => Promise<void>;
  onFooterCancel: () => Promise<void>;
  modelValue: boolean;
};

function unwrapRef<T>(v: T | { value: T } | null | undefined): T | undefined {
  if (typeof v !== 'object' || Object(v) !== v) return v as T | undefined;
  if ('value' in (v as object)) return (v as { value: T }).value;
  return v as T;
}

function setMappingFormViewRef(s: MappingSetup, api: ChoyFormDialogViewApi | null): void {
  const raw = s.mappingFormViewRef;
  if (raw && typeof raw === 'object' && 'value' in raw) {
    (raw as { value: ChoyFormDialogViewApi | null }).value = api;
    return;
  }
  (s as { mappingFormViewRef: ChoyFormDialogViewApi | null }).mappingFormViewRef = api;
}

function setSelectedFile(s: MappingSetup, file: File | null): void {
  const raw = s.selectedFile as File | null | { value: File | null };
  if (raw && typeof raw === 'object' && 'value' in raw) {
    (raw as { value: File | null }).value = file;
    return;
  }
  (s as { selectedFile: File | null }).selectedFile = file;
}

function ensureFilePolyfill(): void {
  const g = globalThis as any;
  if (typeof g.Blob !== 'function') {
    g.Blob = class Blob {
      parts: unknown[];
      type: string;
      size: number;
      constructor(parts: unknown[] = [], opts?: { type?: string }) {
        this.parts = parts;
        this.type = opts?.type || '';
        this.size = parts.reduce((n: number, p) => n + String(p ?? '').length, 0);
      }
      async arrayBuffer(): Promise<ArrayBuffer> {
        const text = this.parts.map(p => String(p ?? '')).join('');
        const bytes = new Uint8Array(text.length);
        for (let i = 0; i < text.length; i++) bytes[i] = text.charCodeAt(i) & 0xff;
        return bytes.buffer;
      }
    };
  }
  if (typeof g.File !== 'function') {
    g.File = class File extends g.Blob {
      name: string;
      constructor(parts: unknown[], name: string, opts?: { type?: string }) {
        super(parts, opts);
        this.name = name || '';
      }
    };
  }
}

function stubImportHub() {
  const hub = CreateWebClient(ImportHub)() as any;
  const originals = {
    describeImportFields: hub.describeImportFields,
    parseHeaders: hub.parseHeaders,
    preview: hub.preview,
    run: hub.run,
  };
  hub.describeImportFields = async () => ({
    fields: [{ path: 'Name', label: 'Name' }, { path: 'Email', label: 'Email' }],
    defaultFields: ['Name'],
  });
  hub.parseHeaders = async () => ({ headers: ['Full Name', 'Email'] });
  hub.preview = async () => ({
    report: { stats: { ok: 1, error: 0, total: 1 }, messages: [] },
  });
  hub.run = async () => ({
    report: { stats: { ok: 1, error: 0, total: 1 }, messages: [] },
  });
  return () => {
    hub.describeImportFields = originals.describeImportFields;
    hub.parseHeaders = originals.parseHeaders;
    hub.preview = originals.preview;
    hub.run = originals.run;
  };
}

function mountPanel() {
  return mountApp(ImportPanel as any, {
    props: {
      model: 'partner.Partner',
      companyId: 'c1',
      modelValue: true,
    },
  });
}

function asSetup(m: ReturnType<typeof mountPanel>): MappingSetup {
  return m.setupState() as MappingSetup;
}

describe('ImportPanel mapping FormView store', () => {
  test('sanitizeHeaderFieldName handles empty, digits, and collisions', () => {
    const m = mountPanel();
    const s = asSetup(m);
    const used = new Set<string>();
    expect(s.sanitizeHeaderFieldName('  Name!  ', used)).toBe('Name');
    expect(s.sanitizeHeaderFieldName('', used)).toBe('column');
    expect(s.sanitizeHeaderFieldName('123', used)).toBe('c_123');
    expect(s.sanitizeHeaderFieldName('Name', used)).toBe('Name_2');
    m.unmount();
  });

  test('rebuildMappingFormStore builds selection fields and sync round-trips', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    s.catalogFields.splice(0, s.catalogFields.length, { path: 'Name', label: 'Name' }, { path: 'Email', label: 'Email' });
    s.mappingRows.splice(
      0,
      s.mappingRows.length,
      { header: 'Full Name', fieldPath: '' },
      { header: 'Email', fieldPath: 'Email' },
    );
    s.rebuildMappingFormStore();
    expect(s.mappingFormStore).toBeTruthy();
    expect(s.mappingPropForHeader('Full Name')).toBe('Full_Name');
    expect(s.mappingPropForHeader('Email')).toBe('Email');
    expect(s.mappingFormStore!.getField('Email')).toBe('Email');

    s.mappingFormStore!.setField('Full_Name', 'Name');
    s.syncMappingRowsFromStore();
    expect(s.mappingRows[0]!.fieldPath).toBe('Name');

    s.mappingRows.splice(0, s.mappingRows.length);
    s.rebuildMappingFormStore();
    expect(s.mappingFormStore).toBeNull();
    m.unmount();
  });

  test('syncMappingRowsFromStore no-ops without store', () => {
    const m = mountPanel();
    const s = asSetup(m);
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'A', fieldPath: 'x' });
    // Ensure store cleared.
    s.mappingRows.splice(0, s.mappingRows.length);
    s.rebuildMappingFormStore();
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'A', fieldPath: 'x' });
    s.syncMappingRowsFromStore();
    expect(s.mappingRows[0]!.fieldPath).toBe('x');
    m.unmount();
  });

  test('onFooterCancel closes via defineModel', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m) as MappingSetup & { visible?: boolean };
    await s.onFooterCancel();
    // setupState unwraps defineModel; host may expose `visible` as false after close.
    expect(s.visible === false || s.modelValue === false).toBe(true);
    m.unmount();
  });

  test('mappingFormValues mirrors FormView draft into store; onMappingFormChange clears preview', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    s.catalogFields.splice(0, s.catalogFields.length, { path: 'Name', label: 'Name' });
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'Full Name', fieldPath: '' });
    s.rebuildMappingFormStore();
    expect(s.mappingFormStore).toBeTruthy();

    setMappingFormViewRef(s, {
      getFormData: () => ({ Full_Name: 'Name' }),
    });
    const values = s.mappingFormValues();
    expect(values.Full_Name).toBe('Name');
    expect(s.mappingFormStore!.getField('Full_Name')).toBe('Name');

    (s as { previewReport: unknown }).previewReport = { stats: { ok: 1, error: 0, total: 1 } };
    s.onMappingFormChange();
    expect(unwrapRef((s as { previewReport: unknown }).previewReport as any)).toBeNull();

    // Row with no prop mapping is skipped (continue in syncMappingRowsFromStore).
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'Orphan', fieldPath: 'x' });
    const propMap = s.mappingPropByHeader as Record<string, string> | { value: Record<string, string> };
    if (propMap && typeof propMap === 'object' && 'value' in propMap) {
      (propMap as { value: Record<string, string> }).value = {};
    } else {
      (s as { mappingPropByHeader: Record<string, string> }).mappingPropByHeader = {};
    }
    s.syncMappingRowsFromStore();
    expect(s.mappingRows[0]!.fieldPath).toBe('x');

    m.unmount();
  });

  test('file select clears derived mapping state; store/catalog watchers sync selection', async () => {
    ensureFilePolyfill();
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    s.catalogFields.splice(0, s.catalogFields.length, { path: 'Name', label: 'Name' });
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'Full Name', fieldPath: 'Name' });
    s.rebuildMappingFormStore();
    expect(s.mappingFormStore).toBeTruthy();
    await nextTick();
    await flushPromises();

    // Same-store field edit triggers mapping watcher → onMappingChange.
    const previewRaw = (s as { previewReport: unknown }).previewReport;
    if (previewRaw && typeof previewRaw === 'object' && 'value' in (previewRaw as object)) {
      (previewRaw as { value: unknown }).value = { stats: { ok: 1, error: 0, total: 1 } };
    } else {
      (s as { previewReport: unknown }).previewReport = { stats: { ok: 1, error: 0, total: 1 } };
    }
    s.mappingFormStore!.setField('Full_Name', 'Other');
    await nextTick();
    await flushPromises();
    expect(unwrapRef((s as { previewReport: unknown | { value: unknown } }).previewReport as any)).toBeNull();

    // catalogOptions change refreshes selection items on the store.
    s.catalogFields.splice(0, s.catalogFields.length, { path: 'Email', label: 'Email' });
    await nextTick();
    await flushPromises();
    expect(s.mappingFormStore).toBeTruthy();

    const file = new (globalThis as any).File(['a,b\n'], 'rows.csv', { type: 'text/csv' });
    s.onFileSelected({ raw: file, name: 'rows.csv' });
    expect(unwrapRef(s.selectedFile as any)).toBeTruthy();
    expect(s.mappingFormStore).toBeNull();
    expect(s.mappingRows.length).toBe(0);

    s.mappingRows.splice(0, s.mappingRows.length, { header: 'A', fieldPath: '' });
    s.rebuildMappingFormStore();
    s.onFileRemoved();
    expect(unwrapRef(s.selectedFile as any) == null).toBe(true);
    expect(s.mappingFormStore).toBeNull();
    m.unmount();
  });

  test('uploadAndPreview and commitImport sync mapping from store', async () => {
    ensureFilePolyfill();
    const restoreHub = stubImportHub();
    const prepare = fnRecorder(async () => ({
      uploadId: 'up-1',
      uploadTarget: { method: 'PUT', url: 'https://example.com/u', headers: {} },
    }));
    const finalize = fnRecorder(async () => ({ AttachmentContentId: 'src-1' }));
    const restoreAttach = replaceStoreFactory('document.AttachmentContent', () => ({
      PrepareUpload: prepare,
      FinalizeUpload: finalize,
    }));
    const origFetch = globalThis.fetch;
    globalThis.fetch = fnRecorder(async () => ({ ok: true, status: 200 })) as any;

    try {
      const m = mountPanel();
      await flushPromises();
      const s = asSetup(m);
      const file = new (globalThis as any).File(['Name,Email\nA,a@x\n'], 'rows.csv', { type: 'text/csv' });
      setSelectedFile(s, file);
      await s.uploadAndPreview();
      await flushPromises();
      expect(unwrapRef(s.sourceRef as any)).toBe('src-1');
      expect(s.mappingFormStore).toBeTruthy();
      expect(s.mappingRows.length).toBe(2);

      s.mappingFormStore!.setField(s.mappingPropForHeader('Full Name'), 'Name');
      s.mappingFormStore!.setField(s.mappingPropForHeader('Email'), 'Email');
      await s.commitImport();
      await flushPromises();
      expect((s as { importDone?: boolean }).importDone === true || unwrapRef((s as any).importDone) === true).toBe(
        true,
      );
      m.unmount();
    } finally {
      globalThis.fetch = origFetch;
      restoreAttach();
      restoreHub();
    }
  });
});
