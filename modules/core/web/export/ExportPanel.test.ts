// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { config, flushPromises, mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';

config.global.renderStubDefaultSlot = true;

const { describeExportFields, previewExport, runExport, downloadExportCsvBytes } = vi.hoisted(() => ({
  describeExportFields: vi.fn(),
  previewExport: vi.fn(),
  runExport: vi.fn(),
  downloadExportCsvBytes: vi.fn(),
}));

vi.mock('./client', () => ({
  describeExportFields,
  previewExport,
  runExport,
  ExportMode: { DATA: 1 },
}));

vi.mock('./download_csv', () => ({
  downloadExportCsvBytes,
  suggestExportFileName: (model: string) => `${model.split('.').pop() || 'export'}.csv`,
}));

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: { en: {} },
});

async function mountPanel(props: Record<string, unknown> = {}, open = true) {
  const { default: ExportPanel } = await import('./ExportPanel.vue');
  return mount(ExportPanel, {
    props: {
      model: 'partner.Partner',
      companyId: 'cmp-1',
      defaultFields: ['Name', 'Code'],
      filteredCount: 5,
      modelValue: open,
      ...props,
    },
    global: {
      plugins: [i18n],
      stubs: {
        ElDialog: {
          props: ['modelValue'],
          emits: ['update:modelValue', 'open', 'closed'],
          template: '<div class="dialog-stub"><slot /><slot name="footer" /></div>',
        },
        ElCollapse: { template: '<div><slot /></div>' },
        ElCollapseItem: { template: '<div><slot /></div>' },
        ElTree: {
          props: ['data', 'defaultCheckedKeys'],
          methods: {
            getCheckedKeys: () => ['Name'],
          },
          template: '<div class="tree-stub" />',
        },
        ElAlert: { template: '<div class="el-alert-stub"><slot name="title" /></div>' },
        ElResult: { template: '<div class="el-result-stub" />' },
        ElButton: {
          emits: ['click'],
          template: '<button @click="$emit(\'click\')"><slot /></button>',
        },
      },
    },
  });
}

describe('ExportPanel', () => {
  beforeEach(() => {
    describeExportFields.mockReset();
    previewExport.mockReset();
    runExport.mockReset();
    downloadExportCsvBytes.mockReset();
    describeExportFields.mockResolvedValue({
      fields: [{ path: 'Name', label: 'Name' }, { path: 'Code', label: 'Code' }, { path: '  ', label: 'Blank' }],
      defaultFields: ['Name', 'Code'],
    });
    previewExport.mockResolvedValue({ report: { stats: { total: 1, ok: 1, error: 0 } } });
    runExport.mockResolvedValue({ report: { stats: { ok: 2 } }, csvData: new Uint8Array([1, 2]) });
  });

  it('loads fields on open and runs preview/export', async () => {
    const wrapper = await mountPanel();
    await (wrapper.vm as any).onOpen();
    await flushPromises();

    expect(describeExportFields).toHaveBeenCalledWith('partner.Partner');
    expect((wrapper.vm as any).fieldTree.length).toBeGreaterThan(0);

    await (wrapper.vm as any).runPreview();
    await flushPromises();
    expect(previewExport).toHaveBeenCalled();
    expect((wrapper.vm as any).previewReport?.stats?.ok).toBe(1);

    await (wrapper.vm as any).commitExport();
    await flushPromises();
    expect(runExport).toHaveBeenCalled();
    expect(downloadExportCsvBytes).toHaveBeenCalled();
    expect((wrapper.vm as any).exportDone).toBe(true);
  });

  it('shows export error when report has errors', async () => {
    runExport.mockResolvedValue({ report: { stats: { error: 1 }, messages: [{ text: 'row failed' }] } });
    const wrapper = await mountPanel();
    await (wrapper.vm as any).commitExport();
    await flushPromises();
    expect((wrapper.vm as any).exportDone).toBe(false);
    expect((wrapper.vm as any).exportError).toBe('row failed');
  });

  it('uses artifact subtitle when csv bytes are omitted', async () => {
    runExport.mockResolvedValue({ report: { stats: { ok: 3 }, artifactRef: 'doc-123' } });
    const wrapper = await mountPanel();
    await (wrapper.vm as any).commitExport();
    await flushPromises();
    expect((wrapper.vm as any).exportDone).toBe(true);
    expect((wrapper.vm as any).exportSuccessSubtitle).toContain('doc-123');
    expect(downloadExportCsvBytes).not.toHaveBeenCalled();
  });

  it('ignores stale field load after session reset', async () => {
    let resolveDescribe: (value: unknown) => void = () => {};
    describeExportFields.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveDescribe = resolve;
        }),
    );
    const wrapper = await mountPanel();
    await (wrapper.vm as any).onOpen();
    (wrapper.vm as any).resetState();
    resolveDescribe({ fields: [{ path: 'Stale', label: 'Stale' }], defaultFields: ['Stale'] });
    await flushPromises();
    expect((wrapper.vm as any).fieldTree).toEqual([]);
  });

  it('ignores stale field load errors after session reset', async () => {
    let rejectDescribe: (reason: unknown) => void = () => {};
    describeExportFields.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectDescribe = reject;
        }),
    );
    const wrapper = await mountPanel();
    await (wrapper.vm as any).onOpen();
    (wrapper.vm as any).resetState();
    rejectDescribe(new Error('network down'));
    await flushPromises();
    expect((wrapper.vm as any).exportError).toBe('');
  });

  it('ignores stale preview results after session reset', async () => {
    let resolvePreview: (value: unknown) => void = () => {};
    previewExport.mockImplementation(
      () =>
        new Promise(resolve => {
          resolvePreview = resolve;
        }),
    );
    const wrapper = await mountPanel();
    const pending = (wrapper.vm as any).runPreview();
    (wrapper.vm as any).resetState();
    resolvePreview({ report: { stats: { ok: 9 } } });
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).previewReport).toBeNull();
    expect((wrapper.vm as any).busy).toBe(false);
  });

  it('ignores stale export results after session reset', async () => {
    let resolveRun: (value: unknown) => void = () => {};
    runExport.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveRun = resolve;
        }),
    );
    const wrapper = await mountPanel();
    const pending = (wrapper.vm as any).commitExport();
    (wrapper.vm as any).onOpen();
    resolveRun({ report: { stats: { ok: 9 } }, csvData: new Uint8Array([1]) });
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).exportDone).toBe(false);
    expect(downloadExportCsvBytes).not.toHaveBeenCalled();
    expect((wrapper.vm as any).busy).toBe(false);
  });

  it('builds selected scope summary and passes ids', async () => {
    const wrapper = await mountPanel({ ids: ['p1', 'p2'], filteredCount: 0 });
    expect((wrapper.vm as any).scopeSummary).toContain('2');
    const input = (wrapper.vm as any).buildRunInput();
    expect(input.ids).toEqual(['p1', 'p2']);
    expect(input.domain).toBe('');
  });

  it('uses filtered scope summary when no rows are selected', async () => {
    const wrapper = await mountPanel({ filteredCount: 7 });
    expect((wrapper.vm as any).scopeSummary).toContain('7');
  });

  it('uses unknown filtered scope summary when count is zero', async () => {
    const wrapper = await mountPanel({ filteredCount: 0 });
    expect((wrapper.vm as any).scopeSummary).toContain('matching the current filters');
  });

  it('uses domain when no ids are selected', async () => {
    const wrapper = await mountPanel({ domain: '{"And":[]}' });
    const input = (wrapper.vm as any).buildRunInput();
    expect(input.ids).toEqual([]);
    expect(input.domain).toBe('{"And":[]}');
  });

  it('surfaces field load errors', async () => {
    describeExportFields.mockRejectedValue(new Error('network down'));
    const wrapper = await mountPanel();
    await (wrapper.vm as any).onOpen();
    await flushPromises();
    expect((wrapper.vm as any).exportError).toBe('network down');
  });

  it('handles non-error preview failures', async () => {
    previewExport.mockRejectedValue('plain failure');
    const wrapper = await mountPanel();
    await (wrapper.vm as any).runPreview();
    await flushPromises();
    expect((wrapper.vm as any).exportError).toBe('plain failure');
  });

  it('handles non-error export failures', async () => {
    runExport.mockRejectedValue('plain failure');
    const wrapper = await mountPanel();
    await (wrapper.vm as any).commitExport();
    await flushPromises();
    expect((wrapper.vm as any).exportError).toBe('plain failure');
  });

  it('shows preview warning when preview report has message-only errors', async () => {
    previewExport.mockResolvedValue({
      report: { stats: { total: 1, ok: 0, error: 0 }, messages: [{ type_: 1, text: 'bad row' }] },
    });
    const wrapper = await mountPanel();
    await (wrapper.vm as any).runPreview();
    await flushPromises();
    expect((wrapper.vm as any).previewAlertType).toBe('warning');
    expect((wrapper.vm as any).previewSummary).toContain('errors');
  });

  it('shows preview warning when preview report has errors', async () => {
    previewExport.mockResolvedValue({
      report: { stats: { total: 1, ok: 0, error: 1 }, messages: [{ text: 'bad row' }] },
    });
    const wrapper = await mountPanel();
    await (wrapper.vm as any).runPreview();
    await flushPromises();
    expect((wrapper.vm as any).previewAlertType).toBe('warning');
    expect((wrapper.vm as any).previewSummary).toContain('0 ok');
  });

  it('clears preview when fields change', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).previewReport = { stats: { ok: 1 } };
    (wrapper.vm as any).onFieldCheck();
    expect((wrapper.vm as any).previewReport).toBeNull();
  });

  it('applies default fields from props watch', async () => {
    const wrapper = await mountPanel({ defaultFields: ['CompanyId.Code'] });
    expect((wrapper.vm as any).selectedFieldPaths).toEqual(['CompanyId/Code']);
  });

  it('uses server default fields when props defaults are empty', async () => {
    const wrapper = await mountPanel({ defaultFields: [] });
    await (wrapper.vm as any).onOpen();
    await flushPromises();
    expect((wrapper.vm as any).selectedFieldPaths).toEqual(['Name', 'Code']);
  });

  it('maps nested field nodes and label fallbacks', async () => {
    describeExportFields.mockResolvedValue({
      fields: [{
        path: 'CompanyId',
        label: '',
        children: [{ path: 'CompanyId/Code', label: 'Code' }],
      }],
      defaultFields: [],
    });
    const wrapper = await mountPanel({ defaultFields: [] });
    await (wrapper.vm as any).onOpen();
    await flushPromises();
    expect((wrapper.vm as any).fieldTree[0].label).toBe('CompanyId');
    expect((wrapper.vm as any).fieldTree[0].children?.[0].path).toBe('CompanyId/Code');
  });

  it('ignores stale preview errors after session reset', async () => {
    let rejectPreview: (reason: unknown) => void = () => {};
    previewExport.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectPreview = reject;
        }),
    );
    const wrapper = await mountPanel();
    const pending = (wrapper.vm as any).runPreview();
    (wrapper.vm as any).resetState();
    rejectPreview(new Error('stale preview'));
    await pending;
    await flushPromises();
    expect((wrapper.vm as any).exportError).toBe('');
  });

  it('resets state on closed and clears busy on reopen', async () => {
    const wrapper = await mountPanel();
    (wrapper.vm as any).busy = true;
    (wrapper.vm as any).previewReport = { stats: { ok: 1 } };
    (wrapper.vm as any).resetState();
    expect((wrapper.vm as any).busy).toBe(false);
    expect((wrapper.vm as any).previewReport).toBeNull();
    await (wrapper.vm as any).onOpen();
    expect((wrapper.vm as any).busy).toBe(false);
  });

  it('does not overwrite selected fields when watch runs after selection', async () => {
    const wrapper = await mountPanel({ defaultFields: ['Name'] });
    (wrapper.vm as any).selectedFieldPaths = ['Code'];
    await wrapper.setProps({ defaultFields: ['Name', 'CompanyId.Code'] });
    expect((wrapper.vm as any).selectedFieldPaths).toEqual(['Code']);
  });
});
