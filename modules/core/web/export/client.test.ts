// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import {
  __setExportHubClientForTest,
  describeExportFields,
  previewExport,
  runExport,
  runTerminologyExport,
} from './client';
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
    reset() {
      calls.length = 0;
      impl = undefined;
    },
    resolve(value: unknown) {
      impl = async () => value;
    },
  };
}

function installHub() {
  const describeFields = makeFn();
  const preview = makeFn();
  const run = makeFn();
  __setExportHubClientForTest({
    describeFields: describeFields.fn as any,
    preview: preview.fn as any,
    run: run.fn as any,
  });
  return { describeFields, preview, run };
}

test('core/web export client: calls ExportHub describeFields with model', async () => {
  const hub = installHub();
  hub.describeFields.resolve({ fields: [], defaultFields: ['Name'] });
  const resp = await describeExportFields('partner.Partner');
  expect(resp.defaultFields).toEqual(['Name']);
  expect(hub.describeFields.calls[0]?.args).toEqual([
    create(DescribeFieldsRequestSchema, { model: 'partner.Partner' }),
    undefined,
  ]);
  __setExportHubClientForTest(null);
});

test('core/web export client: calls previewExport and runExport with ids and domain', async () => {
  const hub = installHub();
  hub.preview.resolve({ report: { stats: { ok: 1 } } });
  hub.run.resolve({ report: { stats: { ok: 2 } }, csvData: new Uint8Array([1, 2]) });
  const input = {
    model: 'partner.Partner',
    companyId: 'cmp-1',
    ids: ['p1'],
    fields: ['Name', 'Code'],
  };
  await previewExport(input);
  await runExport(input);
  expect(hub.preview.calls[0]?.args).toEqual([
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
  expect(hub.run.calls.length).toBe(1);
  __setExportHubClientForTest(null);
});

test('core/web export client: passes abort signal to export hub calls', async () => {
  const hub = installHub();
  hub.describeFields.resolve({ fields: [] });
  hub.preview.resolve({ report: {} });
  hub.run.resolve({ report: {} });
  const signal = new AbortController().signal;
  await describeExportFields('base.Country', signal);
  await previewExport({ model: 'base.Country' }, signal);
  await runExport({ model: 'base.Country' }, signal);
  expect(hub.describeFields.calls[0]?.args).toEqual([
    create(DescribeFieldsRequestSchema, { model: 'base.Country' }),
    { signal },
  ]);
  expect(hub.preview.calls[0]?.args[1]).toEqual({ signal });
  expect(hub.run.calls[0]?.args[1]).toEqual({ signal });
  __setExportHubClientForTest(null);
});

test('core/web export client: calls runTerminologyExport with terminology profile fields', async () => {
  const hub = installHub();
  hub.run.resolve({ report: { stats: { ok: 1 } }, poData: new Uint8Array([112, 111]) });
  await runTerminologyExport({ application: 'auth', module: 'base', lang: 'zh_CN' });
  expect(hub.run.calls[0]?.args).toEqual([
    create(ExportRunRequestSchema, {
      profile: 'terminology',
      application: 'auth',
      module_: 'base',
      lang: 'zh_CN',
    }),
    undefined,
  ]);
  __setExportHubClientForTest(null);
});
