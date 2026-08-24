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

describe('PartnerImportWizard', () => {
  beforeEach(() => {
    parseHeaders.mockReset();
    previewImport.mockReset();
    runImport.mockReset();
    uploadImportCsv.mockReset();
    uploadImportCsv.mockResolvedValue('att-src-1');
    parseHeaders.mockResolvedValue({ headers: ['Name', 'Code'] });
    previewImport.mockResolvedValue({ report: { stats: { total: 1, ok: 1, error: 0 }, messages: [] } });
    runImport.mockResolvedValue({ report: { stats: { total: 1, ok: 1, error: 0 }, messages: [] } });
  });

  it('runs upload, preview, and import flow', async () => {
    const { default: PartnerImportWizard } = await import('./PartnerImportWizard.vue');
    const wrapper = mount(PartnerImportWizard, {
      props: { companyId: 'cmp-1', modelValue: true },
      global: { plugins: [i18n], stubs: { ElDialog: false } },
    });

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
    const { default: PartnerImportWizard } = await import('./PartnerImportWizard.vue');
    const wrapper = mount(PartnerImportWizard, {
      props: { companyId: 'cmp-1', modelValue: true },
      global: { plugins: [i18n], stubs: { ElDialog: false } },
    });
    (wrapper.vm as any).sourceRef = 'att-src-1';
    (wrapper.vm as any).previewReport = { stats: { error: 0 } };
    await (wrapper.vm as any).commitImport();
    await flushPromises();
    expect((wrapper.vm as any).importDone).toBe(false);
    expect((wrapper.vm as any).importError).toBe('duplicate code');
  });

  it('ignores stale preview results after session reset', async () => {
    let resolveUpload: (value: string) => void = () => {};
    uploadImportCsv.mockImplementation(
      () =>
        new Promise<string>(resolve => {
          resolveUpload = resolve;
        }),
    );
    const { default: PartnerImportWizard } = await import('./PartnerImportWizard.vue');
    const wrapper = mount(PartnerImportWizard, {
      props: { companyId: 'cmp-1', modelValue: true },
      global: { plugins: [i18n], stubs: { ElDialog: false } },
    });
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
});
