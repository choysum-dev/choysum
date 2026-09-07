// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { createExportClient, type ExportHubClient } from './client';
import { ExportMode, ExportRunRequestSchema, DescribeFieldsRequestSchema } from './pb/export_pb';
import { ensureAbortController } from '../testing/qjs_polyfills';

ensureAbortController();

type Call = { args: unknown[] };

function makeFn() {
  const calls: Call[] = [];
  let impl: ((...args: any[]) => any) | undefined;
  const fn = (...args: any[]) => {
    calls.push({ args });
    return impl ? impl(...args) : undefined;
  };
  return {
    fn,
    calls,
    resolve(value: unknown) {
      impl = async () => value;
    },
  };
}

function installClient() {
  const describeFields = makeFn();
  const preview = makeFn();
  const run = makeFn();
  const hub: ExportHubClient = {
    describeFields: describeFields.fn as any,
    preview: preview.fn as any,
    run: run.fn as any,
  };
  const client = createExportClient({ hub });
  return { client, describeFields, preview, run };
}

test('core/web export client: calls ExportHub describeFields with model', async () => {
  const { client, describeFields } = installClient();
  describeFields.resolve({ fields: [], defaultFields: ['Name'] });
  const resp = await client.describeExportFields('partner.Partner');
  expect(resp.defaultFields).toEqual(['Name']);
  expect(describeFields.calls[0]?.args).toEqual([
    create(DescribeFieldsRequestSchema, { model: 'partner.Partner' }),
    undefined,
  ]);
});

test('core/web export client: calls previewExport and runExport with ids and domain', async () => {
  const { client, preview, run } = installClient();
  preview.resolve({ report: { stats: { ok: 1 } } });
  run.resolve({ report: { stats: { ok: 2 } }, csvData: new Uint8Array([1, 2]) });
  const input = {
    model: 'partner.Partner',
    companyId: 'cmp-1',
    ids: ['p1'],
    fields: ['Name', 'Code'],
  };
  await client.previewExport(input);
  await client.runExport(input);
  expect(preview.calls[0]?.args).toEqual([
    create(ExportRunRequestSchema, {
      model: 'partner.Partner',
      mode: ExportMode.DATA,
      fields: ['Name', 'Code'],
      domain: '',
      ids: ['p1'],
      companyId: 'cmp-1',
    }),
    undefined,
  ]);
  expect(run.calls.length).toBe(1);
});

test('core/web export client: passes abort signal to export hub calls', async () => {
  const { client, describeFields, preview, run } = installClient();
  describeFields.resolve({ fields: [] });
  preview.resolve({ report: {} });
  run.resolve({ report: {} });
  const signal = new AbortController().signal;
  await client.describeExportFields('base.Country', signal);
  await client.previewExport({ model: 'base.Country' }, signal);
  await client.runExport({ model: 'base.Country' }, signal);
  expect(describeFields.calls[0]?.args).toEqual([
    create(DescribeFieldsRequestSchema, { model: 'base.Country' }),
    { signal },
  ]);
  expect(preview.calls[0]?.args[1]).toEqual({ signal });
  expect(run.calls[0]?.args[1]).toEqual({ signal });
});

test('core/web export client: calls runTerminologyExport with terminology profile fields', async () => {
  const { client, run } = installClient();
  run.resolve({ report: { stats: { ok: 1 } }, poData: new Uint8Array([112, 111]) });
  await client.runTerminologyExport({ application: 'auth', module: 'base', lang: 'zh_CN' });
  expect(run.calls[0]?.args).toEqual([
    create(ExportRunRequestSchema, {
      profile: 'terminology',
      application: 'auth',
      module_: 'base',
      lang: 'zh_CN',
    }),
    undefined,
  ]);
});
