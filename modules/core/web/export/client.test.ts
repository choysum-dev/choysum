// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { ExportMode, ExportRunRequestSchema, DescribeFieldsRequestSchema } from './pb/export_pb';

const describeFields = vi.fn();
const preview = vi.fn();
const run = vi.fn();
const createWebClient = vi.fn(() => () => ({
  describeFields,
  preview,
  run,
}));

vi.mock('../rpc/client_factory', () => ({
  CreateWebClient: createWebClient,
}));

describe('core/web export client', () => {
  beforeEach(() => {
    describeFields.mockReset();
    preview.mockReset();
    run.mockReset();
    createWebClient.mockClear();
  });

  it('calls ExportHub describeFields with model', async () => {
    describeFields.mockResolvedValue({ fields: [], defaultFields: ['Name'] });
    const { describeExportFields } = await import('./client');
    const resp = await describeExportFields('partner.Partner');
    expect(resp.defaultFields).toEqual(['Name']);
    expect(describeFields).toHaveBeenCalledWith(create(DescribeFieldsRequestSchema, { model: 'partner.Partner' }), undefined);
  });

  it('calls previewExport and runExport with ids and domain', async () => {
    preview.mockResolvedValue({ report: { stats: { ok: 1 } } });
    run.mockResolvedValue({ report: { stats: { ok: 2 } }, csvData: new Uint8Array([1, 2]) });
    const { previewExport, runExport } = await import('./client');
    const input = {
      model: 'partner.Partner',
      companyId: 'cmp-1',
      ids: ['p1'],
      fields: ['Name', 'Code'],
    };
    await previewExport(input);
    await runExport(input);
    expect(preview).toHaveBeenCalledWith(
      create(ExportRunRequestSchema, {
        model: 'partner.Partner',
        mode: ExportMode.DATA,
        fields: ['Name', 'Code'],
        domain: '',
        ids: ['p1'],
        companyId: 'cmp-1',
      }),
      undefined
    );
    expect(run).toHaveBeenCalled();
  });
});
