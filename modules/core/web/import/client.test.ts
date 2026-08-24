// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { ImportPolicy, ImportRunRequestSchema, ParseHeadersRequestSchema } from './pb/import_pb';

const parseHeaders = vi.fn();
const preview = vi.fn();
const run = vi.fn();
const createWebClient = vi.fn(() => () => ({
  parseHeaders,
  preview,
  run,
}));

vi.mock('../rpc/client_factory', () => ({
  CreateWebClient: createWebClient,
}));

describe('core/web import client', () => {
  beforeEach(() => {
    parseHeaders.mockReset();
    preview.mockReset();
    run.mockReset();
    createWebClient.mockClear();
  });

  it('calls ImportHub parseHeaders with source ref', async () => {
    parseHeaders.mockResolvedValue({ headers: ['Name', 'Code'] });
    const { parseHeaders: parseHeadersFn } = await import('./client');
    const resp = await parseHeadersFn('src-1');
    expect(resp.headers).toEqual(['Name', 'Code']);
    expect(parseHeaders).toHaveBeenCalledWith(create(ParseHeadersRequestSchema, { sourceRef: 'src-1' }), undefined);
  });

  it('passes abort signal to parseHeaders', async () => {
    parseHeaders.mockResolvedValue({ headers: [] });
    const { parseHeaders: parseHeadersFn } = await import('./client');
    const controller = new AbortController();
    await parseHeadersFn('src-2', controller.signal);
    expect(parseHeaders).toHaveBeenCalledWith(create(ParseHeadersRequestSchema, { sourceRef: 'src-2' }), { signal: controller.signal });
  });

  it('calls previewImport and runImport with atomic policy', async () => {
    preview.mockResolvedValue({ report: { stats: { ok: 1 } } });
    run.mockResolvedValue({ report: { stats: { ok: 2 } } });
    const { previewImport, runImport } = await import('./client');
    const fullInput = {
      targetModel: 'partner.Partner',
      sourceRef: 'src-3',
      companyId: 'cmp-1',
      columnMapping: { Name: 'Name' },
    };
    const minimalInput = {
      targetModel: 'partner.Partner',
      sourceRef: 'src-4',
    };
    await previewImport(fullInput);
    await runImport(minimalInput);
    expect(preview).toHaveBeenCalledWith(
      create(ImportRunRequestSchema, {
        targetModel: fullInput.targetModel,
        sourceRef: fullInput.sourceRef,
        columnMapping: fullInput.columnMapping,
        companyId: fullInput.companyId,
        policy: ImportPolicy.ATOMIC,
      }),
      undefined,
    );
    expect(run).toHaveBeenCalledWith(
      create(ImportRunRequestSchema, {
        targetModel: minimalInput.targetModel,
        sourceRef: minimalInput.sourceRef,
        columnMapping: {},
        companyId: '',
        policy: ImportPolicy.ATOMIC,
      }),
      undefined,
    );
  });
});
