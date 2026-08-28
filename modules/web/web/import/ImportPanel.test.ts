// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { config, flushPromises, mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

const { parseHeaders, previewImport, runImport, uploadImportCsv } = vi.hoisted(() => ({
  parseHeaders: vi.fn(),
  previewImport: vi.fn(),
  runImport: vi.fn(),
  uploadImportCsv: vi.fn(),
}));

vi.mock('@/core/web/import', () => ({
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

async function mountPanel(open = true) {
  const { default: ImportPanel } = await import('./ImportPanel.vue');
  return mount(ImportPanel, {
    props: { model: 'partner.Partner', companyId: 'cmp-1', modelValue: open },
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
}

describe('ImportPanel', () => {
  beforeEach(() => {
    parseHeaders.mockReset();
    previewImport.mockReset();
    runImport.mockReset();
    uploadImportCsv.mockReset();
    uploadImportCsv.mockResolvedValue('att-src-1');
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

    expect(uploadImportCsv).toHaveBeenCalled();
    expect(parseHeaders).toHaveBeenCalledWith('att-src-1', expect.any(AbortSignal));
    expect(previewImport).toHaveBeenCalled();
    expect((wrapper.vm as any).step).toBe(1);

    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect(runImport).toHaveBeenCalled();
    expect((wrapper.vm as any).importDone).toBe(true);
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
    expect(wrapper.find('.import-panel-table').exists()).toBe(true);
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
    expect(wrapper.find('.import-panel-hint').exists()).toBe(false);

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
});
