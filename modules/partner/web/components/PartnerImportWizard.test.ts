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

async function mountWizard(open = true) {
  const { default: PartnerImportWizard } = await import('./PartnerImportWizard.vue');
  return mount(PartnerImportWizard, {
    props: { companyId: 'cmp-1', modelValue: open },
    global: { plugins: [i18n], stubs: { ElDialog: false } },
  });
}

describe('PartnerImportWizard', () => {
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
    const wrapper = await mountWizard();
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
    const wrapper = await mountWizard();
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(false);
    expect((wrapper.vm as any).importError).toBe('duplicate code');
  });

  it('uses fallback import error when report has no message text', async () => {
    runImport.mockResolvedValue({ report: { stats: { error: 2 }, messages: [{ text: '' }] } });
    const wrapper = await mountWizard();
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
    const wrapper = await mountWizard();
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
    const wrapper = await mountWizard();
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
    const wrapper = await mountWizard();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(2);
    expect((wrapper.vm as any).importError).toBe('upload failed');
  });

  it('handles non-error preview failures', async () => {
    uploadImportCsv.mockRejectedValue('plain failure');
    const wrapper = await mountWizard();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).importError).toBe('plain failure');
  });

  it('ignores abort errors during preview', async () => {
    uploadImportCsv.mockRejectedValue(new DOMException('aborted', 'AbortError'));
    const wrapper = await mountWizard();
    (wrapper.vm as any).onFileSelected({ raw: new File(['x'], 'partners.csv', { type: 'text/csv' }) });
    await (wrapper.vm as any).uploadAndPreview();
    await flushPromises();
    expect((wrapper.vm as any).step).toBe(0);
    expect((wrapper.vm as any).importError).toBe('');
  });

  it('blocks close while busy and clears file on remove', async () => {
    const wrapper = await mountWizard();
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
    const wrapper = await mountWizard();
    await (wrapper.vm as any).uploadAndPreview();
    await (wrapper.vm as any).commitImport();
    expect(uploadImportCsv).not.toHaveBeenCalled();
    expect(runImport).not.toHaveBeenCalled();
  });

  it('closes dialog on finish and resets when hidden', async () => {
    const wrapper = await mountWizard();
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
    const wrapper = await mountWizard();
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
});
