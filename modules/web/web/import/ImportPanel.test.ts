// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config, flushPromises, mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

const { describeImportFields, parseHeaders, previewImport, runImport, uploadImportCsv } = vi.hoisted(() => ({
  describeImportFields: vi.fn(),
  parseHeaders: vi.fn(),
  previewImport: vi.fn(),
  runImport: vi.fn(),
  uploadImportCsv: vi.fn(),
}));

vi.mock('@/core/web/import', () => ({
  describeImportFields,
  parseHeaders,
  previewImport,
  runImport,
}));

vi.mock('@/core/web/import/upload_csv', () => ({
  uploadImportCsv,
}));

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

async function mountPanel(open = true, props: Record<string, unknown> = {}) {
  const { default: ImportPanel } = await import('./ImportPanel.vue');
  return mount(ImportPanel, {
    props: { model: 'partner.Partner', companyId: 'cmp-1', modelValue: open, ...props },
    global: {
      plugins: [i18n],
      stubs: {
        ElDialog: {
          name: 'ElDialog',
          props: ['modelValue', 'title'],
          emits: ['update:modelValue', 'close', 'open', 'closed'],
          template:
            '<div class="dialog-stub"><div data-test="dialog-title">{{ title }}</div><button data-test="dialog-close" @click="$emit(\'update:modelValue\', false)">x</button><slot /><slot name="footer" /></div>',
          mounted() {
            this.$emit('open');
          },
        },
        ElAlert: {
          template: '<div class="el-alert-stub"><slot name="title" /></div>',
        },
        ElButton: {
          emits: ['click'],
          template: '<button @click="$emit(\'click\')"><slot /></button>',
        },
        ElSelect: {
          props: ['modelValue'],
          emits: ['update:modelValue', 'change'],
          template:
            '<select data-test="import-field-select" :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value); $emit(\'change\', $event.target.value)"><slot /></select>',
        },
        ElOption: {
          props: ['label', 'value'],
          template: '<option :value="value">{{ label }}</option>',
        },
        ElUpload: { template: '<div class="upload-stub"><slot /></div>' },
        ElSteps: { template: '<div class="steps-stub"><slot /></div>' },
        ElStep: { template: '<div class="step-stub" />' },
        ElTable: { template: '<div class="table-stub"><slot /></div>' },
        ElTableColumn: { template: '<div />' },
        ElResult: { template: '<div class="el-result-stub" />' },
      },
    },
  });
}

describe('ImportPanel', () => {
  beforeEach(() => {
    describeImportFields.mockReset();
    parseHeaders.mockReset();
    previewImport.mockReset();
    runImport.mockReset();
    uploadImportCsv.mockReset();
    uploadImportCsv.mockResolvedValue('att-src-1');
    describeImportFields.mockResolvedValue({
      fields: [
        { path: 'Name', label: 'Name', children: [] },
        { path: 'Code', label: 'Code', children: [] },
        {
          path: 'CompanyId',
          label: 'Company',
          children: [{ path: 'CompanyId/Code', label: 'Company / Code', children: [] }],
        },
      ],
      defaultFields: ['Name', 'Code', 'CompanyId/Code'],
    });
    parseHeaders.mockResolvedValue({ headers: ['Name', 'Code'] });
    previewImport.mockResolvedValue({
      report: {
        stats: { total: 1, ok: 1, error: 0 },
        messages: [{ row: 2, field: 'Name', code: 'ok', text: 'looks good' }],
      },
    });
    runImport.mockResolvedValue({ report: { stats: { total: 1, ok: 1, error: 0 }, messages: [] } });
  });

  it('runs upload, preview, and import flow', async () => {
    const wrapper = await mountPanel();
    const file = new File(['Name,Code\nA,1\n'], 'partners.csv', { type: 'text/csv' });
    (wrapper.vm as any).onFileSelected({ raw: file });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();

    expect(describeImportFields).toHaveBeenCalledWith('partner.Partner', expect.any(AbortSignal));
    expect(uploadImportCsv).toHaveBeenCalled();
    expect(parseHeaders).toHaveBeenCalledWith('att-src-1', expect.any(AbortSignal));
    expect(previewImport).toHaveBeenCalledWith(
      expect.objectContaining({
        columnMapping: { Name: 'Name', Code: 'Code' },
      }),
      expect.any(AbortSignal),
    );
    expect((wrapper.vm as any).mappingRows).toEqual([
      { header: 'Name', fieldPath: 'Name' },
      { header: 'Code', fieldPath: 'Code' },
    ]);
    expect((wrapper.vm as any).step).toBe(1);

    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect(runImport).toHaveBeenCalled();
    expect((wrapper.vm as any).importDone).toBe(true);
  });

  it('loads import field catalog on open', async () => {
    const wrapper = await mountPanel();
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    expect(describeImportFields).toHaveBeenCalledWith('partner.Partner', expect.any(AbortSignal));
    expect((wrapper.vm as any).catalogDefaults).toEqual(['Name', 'Code', 'CompanyId/Code']);
    expect((wrapper.vm as any).catalogOptions.map((o: { path: string }) => o.path)).toContain('CompanyId/Code');
  });

  it('shows error when import report has errors', async () => {
    runImport.mockResolvedValue({ report: { stats: { error: 1 }, messages: [{ text: 'duplicate code' }] } });
    const wrapper = await mountPanel();
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(false);
    expect((wrapper.vm as any).importError).toBe('duplicate code');
  });

  it('uses fallback import error when report has no message text', async () => {
    runImport.mockResolvedValue({ report: { stats: { error: 2 }, messages: [{ text: '' }] } });
    const wrapper = await mountPanel();
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('Import failed with 2 error(s).');
  });

  it('ignores stale preview results after session reset', async () => {
    let resolveUpload: (value: string) => void = () => {};
    uploadImportCsv.mockImplementation(
      () =>
        new Promise<string>(resolve => {
          resolveUpload = resolve;
        }),
    );
    const wrapper = await mountPanel();
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    (wrapper.vm as any).onFileSelected({ raw: file });
    const pending = (wrapper.vm as any).uploadAndPreview();
    (wrapper.vm as any).resetState();
    resolveUpload('att-stale');
    parseHeaders.mockResolvedValue({ headers: ['Name'] });
    previewImport.mockResolvedValue({ report: { stats: { ok: 1, error: 0 } } });
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
    expect((wrapper.vm as any).sourceRef).toBe('');
  });

  it('shows preview warning when preview report has errors', async () => {
    previewImport.mockResolvedValue({
      report: { stats: { total: 1, ok: 0, error: 1 }, messages: [{ row: 2, text: 'bad row' }] },
    });
    const wrapper = await mountPanel();
    const file = new File(['Name\n'], 'partners.csv', { type: 'text/csv' });
    (wrapper.vm as any).onFileSelected({ raw: file });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(1);
    expect((wrapper.vm as any).previewAlertType).toBe('warning');
    expect((wrapper.vm as any).previewSummary).toContain('0 ok');
  });

  it('handles upload and preview errors', async () => {
    uploadImportCsv.mockRejectedValue(new Error('upload failed'));
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(2);
    expect((wrapper.vm as any).importError).toBe('upload failed');
  });

  it('handles non-error preview failures', async () => {
    uploadImportCsv.mockRejectedValue('plain failure');
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('plain failure');
  });

  it('ignores abort errors during preview', async () => {
    uploadImportCsv.mockRejectedValue(new DOMException('aborted', 'AbortError'));
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
    expect((wrapper.vm as any).importError).toBe('');
  });

  it('blocks close while busy and clears file on remove', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).busy = true;
    const done = vi.fn();
    (wrapper.vm as any).handleBeforeClose(done);
    expect(done).not.toHaveBeenCalled();
    (wrapper.vm as any).busy = false;
    (wrapper.vm as any).handleBeforeClose(done);
    expect(done).toHaveBeenCalled();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    (wrapper.vm as any).onFileRemoved();
    expect((wrapper.vm as any).selectedFile).toBeNull();
  });

  it('no-ops upload and import without required state', async () => {
    const wrapper = await mountPanel();
    await (wrapper.vm as any).uploadAndPreview();
    await (wrapper.vm as any).commitImport();
    expect(uploadImportCsv).not.toHaveBeenCalled();
    expect(runImport).not.toHaveBeenCalled();
  });

  it('closes dialog on finish and resets when hidden', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).step = 2;
    (wrapper.vm as any).importDone = true;
    (wrapper.vm as any).finish();
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false]);
    await wrapper.setProps({ modelValue: false });
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
  });

  it('handles import runtime errors and stale commit results', async () => {
    runImport.mockRejectedValue(new Error('run exploded'));
    const wrapper = await mountPanel();
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('run exploded');

    runImport.mockResolvedValue({ report: { stats: { ok: 1, error: 0 } } });
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    const pending = (wrapper.vm as any).commitImport();
    (wrapper.vm as any).resetState();
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(false);
  });

  it('renders upload, preview, and success template states', async () => {
    const wrapper = await mountPanel();
    expect((wrapper.vm as any).step).toBe(0);
    expect(wrapper.find('.import-panel-section').exists()).toBe(true);
    const file = new File(['Name,Code\nA,1\n'], 'partners.csv', { type: 'text/csv' });
    (wrapper.vm as any).onFileSelected({ raw: file });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(1);
    expect((wrapper.vm as any).headers).toEqual(['Name', 'Code']);
    expect(wrapper.find('[data-test="import-mapping-table"]').exists()).toBe(true);
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(true);
    expect((wrapper.vm as any).step).toBe(2);
    await (wrapper.vm as any).finish();
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false]);
  });

  it('renders import error alert and disables import when preview has errors', async () => {
    uploadImportCsv.mockRejectedValueOnce(new Error('preview failed'));
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(2);
    expect((wrapper.vm as any).importError).toBe('preview failed');

    previewImport.mockResolvedValue({
      report: { stats: { total: 1, ok: 0, error: 1 }, messages: [{ row: 2, text: 'bad row' }] },
    });
    uploadImportCsv.mockResolvedValue('att-src-2');
    const wrapper2 = await mountPanel();
    (wrapper2.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper2.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper2.vm as any).canImport).toBe(false);
  });

  it('handles missing upload raw file and stale intermediate preview steps', async () => {
    const wrapper = await mountPanel(false);
    (wrapper.vm as any).onFileSelected({});
    expect((wrapper.vm as any).selectedFile).toBeNull();

    let resolveUpload: (value: string) => void = () => {};
    uploadImportCsv.mockImplementation(
      () =>
        new Promise<string>(resolve => {
          resolveUpload = resolve;
        }),
    );
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    (wrapper.vm as any).onFileSelected({ raw: file });
    const pending = (wrapper.vm as any).uploadAndPreview();
    resolveUpload('att-mid');
    let resolveHeaders: (value: { headers: string[] }) => void = () => {};
    parseHeaders.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveHeaders = resolve;
        }),
    );
    await Promise.resolve();
    (wrapper.vm as any).resetState();
    resolveHeaders({ headers: ['Name'] });
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
  });

  it('ignores stale preview after headers resolve and keeps busy when session ends early', async () => {
    let resolveUpload: (value: string) => void = () => {};
    uploadImportCsv.mockImplementation(
      () =>
        new Promise<string>(resolve => {
          resolveUpload = resolve;
        }),
    );
    let resolveHeaders: (value: { headers: string[] }) => void = () => {};
    parseHeaders.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveHeaders = resolve;
        }),
    );
    const wrapper = await mountPanel();
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    const pending = (wrapper.vm as any).uploadAndPreview();
    resolveUpload('att-headers');
    await flushPromises();
    (wrapper.vm as any).resetState();
    resolveHeaders({ headers: ['Name'] });
    previewImport.mockResolvedValue({ report: { stats: { ok: 1, error: 0 } } });
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
    expect((wrapper.vm as any).busy).toBe(false);
  });

  it('supports cancel visibility and success preview summary branches', async () => {
    previewImport.mockResolvedValue({ report: { stats: { total: 2, ok: 2, error: 0 }, messages: [] } });
    const wrapper = await mountPanel();
    (wrapper.vm as any).visible = false;
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
    await wrapper.setProps({ modelValue: true });
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).previewAlertType).toBe('success');
    expect((wrapper.vm as any).previewSummary).toContain('2 ok');
    (wrapper.vm as any).previewReport = null;
    expect((wrapper.vm as any).previewSummary).toBe('');
  });

  it('renders dialog bindings and preview alert title', async () => {
    const { default: ImportPanel } = await import('./ImportPanel.vue');
    const wrapper = mount(ImportPanel, {
      props: { companyId: 'cmp-1', modelValue: true },
      attachTo: document.body,
      global: {
        plugins: [i18n],
        stubs: {
          ElDialog: false,
          ElAlert: false,
        },
      },
    });
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(1);
    expect((wrapper.vm as any).previewSummary).toContain('Preview:');
    wrapper.unmount();
  });

  it('handles commitImport string errors and omits company id prop', async () => {
    runImport.mockRejectedValue('commit failed');
    const { default: ImportPanel } = await import('./ImportPanel.vue');
    const wrapper = mount(ImportPanel, {
      props: { modelValue: true },
      global: {
        plugins: [i18n],
        stubs: {
          ElDialog: {
            template: '<div class="dialog-stub"><slot /><slot name="footer" /></div>',
          },
        },
      },
    });
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('commit failed');
    expect(runImport).toHaveBeenCalledWith(expect.objectContaining({ companyId: '' }));
  });

  it('covers nullable preview fields and explicit stats branches', async () => {
    parseHeaders.mockResolvedValue({ headers: null });
    previewImport.mockResolvedValue({});
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).headers).toEqual([]);
    expect((wrapper.vm as any).previewReport).toBeNull();
    expect((wrapper.vm as any).previewMessages).toEqual([]);
    expect((wrapper.vm as any).step).toBe(1);
    expect(wrapper.find('.import-panel-hint').exists()).toBe(true);

    (wrapper.vm as any).previewReport = { stats: { total: 4, ok: 3, error: 1 } };
    expect((wrapper.vm as any).previewSummary).toBe('Preview: 3 ok, 1 errors, 4 total');
    expect((wrapper.vm as any).previewAlertType).toBe('warning');
  });

  it('uses nullish fallbacks for missing preview and import stats fields', async () => {
    previewImport.mockResolvedValue({ report: { stats: { ok: 2, total: 2 }, messages: [] } });
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).previewAlertType).toBe('success');
    expect((wrapper.vm as any).previewSummary).toBe('Preview: 2 ok, 0 errors, 2 total');

    runImport.mockResolvedValue({ report: { stats: { ok: 1 } } });
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(true);
  });

  it('passes empty company id to preview when prop is omitted', async () => {
    const { default: ImportPanel } = await import('./ImportPanel.vue');
    const wrapper = mount(ImportPanel, {
      props: { modelValue: true },
      global: {
        plugins: [i18n],
        stubs: {
          ElDialog: {
            props: ['modelValue'],
            emits: ['update:modelValue', 'close'],
            template: '<div class="dialog-stub"><slot /><slot name="footer" /></div>',
          },
          ElAlert: {
            template: '<div class="el-alert-stub"><slot name="title" /></div>',
          },
        },
      },
    });
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect(previewImport).toHaveBeenCalledWith(
      expect.objectContaining({ companyId: '' }),
      expect.any(AbortSignal),
    );
  });

  it('passes provided company id to previewImport', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect(previewImport).toHaveBeenCalledWith(
      expect.objectContaining({ companyId: 'cmp-1', sourceRef: 'att-src-1' }),
      expect.any(AbortSignal),
    );
  });

  it('ignores stale preview response after headers resolve', async () => {
    let resolvePreview: (value: { report: { stats: { ok: number; error: number } } }) => void = () => {};
    parseHeaders.mockResolvedValue({ headers: ['Name'] });
    previewImport.mockImplementation(
      () =>
        new Promise(resolve => {
          resolvePreview = resolve;
        }),
    );
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    const pending = (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    (wrapper.vm as any).resetState();
    resolvePreview({ report: { stats: { ok: 1, error: 0 } } });
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
    expect((wrapper.vm as any).previewReport).toBeNull();
  });

  it('ignores stale session errors during upload and commit catch paths', async () => {
    uploadImportCsv.mockRejectedValue(new Error('late upload failure'));
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    const pendingUpload = (wrapper.vm as any).uploadAndPreview();
    (wrapper.vm as any).resetState();
    await pendingUpload;
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('');

    let rejectCommit: (reason: string) => void = () => {};
    runImport.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectCommit = reject;
        }),
    );
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    const pendingCommit = (wrapper.vm as any).commitImport();
    (wrapper.vm as any).resetState();
    rejectCommit('late commit failure');
    await pendingCommit;
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('');
  });

  it('keeps state when visibility stays open', async () => {
    const wrapper = await mountPanel(false);
    (wrapper.vm as any).step = 1;
    (wrapper.vm as any).headers = ['Name'];
    await wrapper.setProps({ modelValue: true });
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(1);
    expect((wrapper.vm as any).headers).toEqual(['Name']);
  });

  it('renders done footer button after successful import', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).step = 2;
    (wrapper.vm as any).importDone = true;
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(true);
    await (wrapper.vm as any).finish();
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false]);
  });

  it('invalidates preview when mapping changes and requires re-preview', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).onFileSelected({ raw: new File(['Name\n'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).canImport).toBe(true);
    (wrapper.vm as any).mappingRows = [{ header: 'Name', fieldPath: 'Code' }];
    (wrapper.vm as any).onMappingChange();
    expect((wrapper.vm as any).previewReport).toBeNull();
    expect((wrapper.vm as any).canImport).toBe(false);
    expect((wrapper.vm as any).resolvedMapping).toEqual({ Name: 'Code' });
  });

  it('auto-matches only leaf catalog paths and prefers columnMapping prop', async () => {
    parseHeaders.mockResolvedValue({ headers: ['Name', 'CompanyId', 'CompanyId/Code', 'Extra'] });
    const { default: ImportPanel } = await import('./ImportPanel.vue');
    const wrapper = mount(ImportPanel, {
      props: {
        model: 'partner.Partner',
        modelValue: true,
        columnMapping: { Extra: 'Name' },
      },
      global: {
        plugins: [i18n],
        stubs: {
          ElDialog: {
            props: ['modelValue'],
            emits: ['update:modelValue', 'close'],
            template: '<div class="dialog-stub"><slot /><slot name="footer" /></div>',
          },
          ElAlert: { template: '<div class="el-alert-stub"><slot name="title" /></div>' },
        },
      },
    });
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).catalogOptions.map((o: { path: string }) => o.path)).toEqual([
      'Name',
      'Code',
      'CompanyId/Code',
    ]);
    expect((wrapper.vm as any).mappingRows).toEqual([
      { header: 'Name', fieldPath: 'Name' },
      { header: 'CompanyId', fieldPath: '' },
      { header: 'CompanyId/Code', fieldPath: 'CompanyId/Code' },
      { header: 'Extra', fieldPath: 'Name' },
    ]);
  });

  it('covers catalog option and flatten edge cases for blank nodes', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).catalogFields = [
      { path: '  ', label: 'Blank' },
      { path: 'Leaf', label: '', children: [] },
      {
        path: 'Parent',
        label: 'Parent',
        children: [
          { path: '  ', label: 'Skip' },
          { path: 'Parent/Child', label: '', children: [] },
        ],
      },
    ];
    expect((wrapper.vm as any).catalogOptions).toEqual([
      { path: 'Leaf', label: 'Leaf (Leaf)' },
      { path: 'Parent/Child', label: 'Parent/Child (Parent/Child)' },
    ]);
    expect((wrapper.vm as any).flattenPaths((wrapper.vm as any).catalogFields)).toEqual(['Leaf', 'Parent/Child']);
  });

  it('handles catalog load errors, abort, empty model, and null field lists', async () => {
    const wrapper = await mountPanel();
    await flushPromises();

    describeImportFields.mockRejectedValueOnce(new Error('catalog down'));
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    expect((wrapper.vm as any).catalogError).toBe('catalog down');

    describeImportFields.mockRejectedValueOnce('plain catalog fail');
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    expect((wrapper.vm as any).catalogError).toBe('plain catalog fail');

    describeImportFields.mockRejectedValueOnce(new DOMException('aborted', 'AbortError'));
    (wrapper.vm as any).catalogError = 'keep';
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    expect((wrapper.vm as any).catalogError).toBe('');

    describeImportFields.mockResolvedValueOnce({ fields: null, defaultFields: null });
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    expect((wrapper.vm as any).catalogFields).toEqual([]);
    expect((wrapper.vm as any).catalogDefaults).toEqual([]);
    expect((wrapper.vm as any).defaultFieldsHint).toBe('');

    await wrapper.setProps({ model: '   ' });
    const callsBefore = describeImportFields.mock.calls.length;
    await (wrapper.vm as any).loadCatalog();
    expect(describeImportFields.mock.calls.length).toBe(callsBefore);
  });

  it('skips catalog reload when fields or catalog error already present', async () => {
    const wrapper = await mountPanel();
    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    describeImportFields.mockClear();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect(describeImportFields).not.toHaveBeenCalled();

    (wrapper.vm as any).catalogFields = [];
    (wrapper.vm as any).catalogError = 'stale';
    describeImportFields.mockClear();
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect(describeImportFields).not.toHaveBeenCalled();
  });

  it('shows default fields hint and keeps re-preview available after mapping edit', async () => {
    const wrapper = await mountPanel();
    await flushPromises();
    expect(wrapper.find('[data-test="dialog-title"]').text().length).toBeGreaterThan(0);
    expect(wrapper.find('[data-test="import-default-fields"]').exists()).toBe(true);
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    (wrapper.vm as any).onMappingChange();
    await flushPromises();
    expect((wrapper.vm as any).canImport).toBe(false);
    expect((wrapper.vm as any).step).toBe(1);
    expect((wrapper.vm as any).sourceRef).toBeTruthy();
  });

  it('ignores stale session right after upload resolves and in preview catch', async () => {
    let resolveCatalog: (value: { fields: unknown[]; defaultFields: string[] }) => void = () => {};
    describeImportFields.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveCatalog = resolve;
        }),
    );
    const wrapper = await mountPanel(false);
    (wrapper.vm as any).catalogFields = [];
    (wrapper.vm as any).catalogError = '';
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    const pendingCatalog = (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    (wrapper.vm as any).resetState();
    resolveCatalog({ fields: [], defaultFields: [] });
    await pendingCatalog;
    await flushPromises();
    expect((wrapper.vm as any).sourceRef).toBe('');

    describeImportFields.mockResolvedValue({
      fields: [
        { path: 'Name', label: 'Name', children: [] },
        { path: 'Code', label: 'Code', children: [] },
      ],
      defaultFields: ['Name', 'Code'],
    });
    let resolveUpload: (value: string) => void = () => {};
    uploadImportCsv.mockImplementation(
      () =>
        new Promise<string>(resolve => {
          resolveUpload = resolve;
        }),
    );
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    const pendingUpload = (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    resolveUpload('att-after-upload');
    (wrapper.vm as any).resetState();
    await pendingUpload;
    await flushPromises();
    expect((wrapper.vm as any).sourceRef).toBe('');

    let rejectUpload: (reason: Error) => void = () => {};
    uploadImportCsv.mockImplementation(
      () =>
        new Promise<string>((_resolve, reject) => {
          rejectUpload = reject;
        }),
    );
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    const pendingFail = (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    (wrapper.vm as any).resetState();
    rejectUpload(new Error('late fail'));
    await pendingFail;
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('');
  });

  it('covers nullish catalog/mapping branches and cancel path', async () => {
    const wrapper = await mountPanel();
    await flushPromises();
    (wrapper.vm as any).catalogFields = null;
    expect((wrapper.vm as any).catalogOptions).toEqual([]);
    expect((wrapper.vm as any).flattenPaths(null)).toEqual([]);
    (wrapper.vm as any).catalogFields = [
      { path: null, label: null, children: null },
      { path: 'Name', label: null, children: undefined },
      {
        path: 'CompanyId',
        label: null,
        children: [{ path: null, label: null }, { path: 'CompanyId/Code', label: null }],
      },
    ];
    expect((wrapper.vm as any).catalogOptions.map((o: { path: string }) => o.path)).toEqual(['Name', 'CompanyId/Code']);
    (wrapper.vm as any).mappingRows = [
      { header: null, fieldPath: 'Name' },
      { header: 'Code', fieldPath: null },
      { header: 'Name', fieldPath: 'Name' },
    ];
    expect((wrapper.vm as any).resolvedMapping).toEqual({ Name: 'Name' });

    await (wrapper.vm as any).loadCatalog();
    await flushPromises();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    const select = wrapper.find('[data-test="import-field-select"]');
    expect(select.exists()).toBe(true);
    await select.setValue('Code');
    await flushPromises();
    expect((wrapper.vm as any).previewReport).toBeNull();

    const cancel = wrapper.findAll('button').find(b => (b.text() || '').includes('Cancel'));
    expect(cancel).toBeTruthy();
    await cancel!.trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).visible).toBe(false);
  });

  it('uses custom upload hint when provided', async () => {
    const { default: ImportPanel } = await import('./ImportPanel.vue');
    const wrapper = mount(ImportPanel, {
      props: { model: 'partner.Partner', modelValue: true, uploadHint: 'Custom hint' },
      global: {
        plugins: [i18n],
        stubs: {
          ElDialog: {
            props: ['modelValue'],
            emits: ['update:modelValue', 'close'],
            template: '<div class="dialog-stub"><slot /><slot name="footer" /></div>',
          },
        },
      },
    });
    expect((wrapper.vm as any).resolvedUploadHint).toBe('Custom hint');
  });
});
