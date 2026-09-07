// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import {
  __setImportHubClientForTest,
  describeImportFields,
  parseHeaders,
  previewImport,
  runImport,
  runImportAsync,
} from './client';
import {
  DescribeImportFieldsRequestSchema,
  ImportPolicy,
  ImportRunRequestSchema,
  ParseHeadersRequestSchema,
} from './pb/import_pb';
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

function installHub() {
  const describeImportFieldsFn = makeFn();
  const parseHeadersFn = makeFn();
  const preview = makeFn();
  const run = makeFn();
  const runAsync = makeFn();
  __setImportHubClientForTest({
    describeImportFields: describeImportFieldsFn.fn as any,
    parseHeaders: parseHeadersFn.fn as any,
    preview: preview.fn as any,
    run: run.fn as any,
    runAsync: runAsync.fn as any,
  });
  return { describeImportFieldsFn, parseHeadersFn, preview, run, runAsync };
}

test('core/web import client: calls ImportHub describeImportFields with model', async () => {
  const hub = installHub();
  hub.describeImportFieldsFn.resolve({ fields: [], defaultFields: ['Name'] });
  const resp = await describeImportFields('partner.Partner');
  expect(resp.defaultFields).toEqual(['Name']);
  expect(hub.describeImportFieldsFn.calls[0]?.args).toEqual([
    create(DescribeImportFieldsRequestSchema, { model: 'partner.Partner' }),
    undefined,
  ]);
  __setImportHubClientForTest(null);
});

test('core/web import client: passes abort signal to describeImportFields', async () => {
  const hub = installHub();
  hub.describeImportFieldsFn.resolve({ fields: [] });
  const controller = new AbortController();
  await describeImportFields('base.Country', controller.signal);
  expect(hub.describeImportFieldsFn.calls[0]?.args).toEqual([
    create(DescribeImportFieldsRequestSchema, { model: 'base.Country' }),
    { signal: controller.signal },
  ]);
  __setImportHubClientForTest(null);
});

test('core/web import client: calls ImportHub parseHeaders with source ref', async () => {
  const hub = installHub();
  hub.parseHeadersFn.resolve({ headers: ['Name', 'Code'] });
  const resp = await parseHeaders('src-1');
  expect(resp.headers).toEqual(['Name', 'Code']);
  expect(hub.parseHeadersFn.calls[0]?.args).toEqual([
    create(ParseHeadersRequestSchema, { sourceRef: 'src-1' }),
    undefined,
  ]);
  __setImportHubClientForTest(null);
});

test('core/web import client: passes abort signal to parseHeaders', async () => {
  const hub = installHub();
  hub.parseHeadersFn.resolve({ headers: [] });
  const controller = new AbortController();
  await parseHeaders('src-2', controller.signal);
  expect(hub.parseHeadersFn.calls[0]?.args).toEqual([
    create(ParseHeadersRequestSchema, { sourceRef: 'src-2' }),
    { signal: controller.signal },
  ]);
  __setImportHubClientForTest(null);
});

test('core/web import client: calls previewImport and runImport with atomic policy', async () => {
  const hub = installHub();
  hub.preview.resolve({ report: { stats: { ok: 1 } } });
  hub.run.resolve({ report: { stats: { ok: 2 } } });
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
  expect(hub.preview.calls[0]?.args).toEqual([
    create(ImportRunRequestSchema, {
      targetModel: fullInput.targetModel,
      sourceRef: fullInput.sourceRef,
      columnMapping: fullInput.columnMapping,
      companyId: fullInput.companyId,
      policy: ImportPolicy.ATOMIC,
    }),
    undefined,
  ]);
  expect(hub.run.calls[0]?.args).toEqual([
    create(ImportRunRequestSchema, {
      targetModel: minimalInput.targetModel,
      sourceRef: minimalInput.sourceRef,
      columnMapping: {},
      companyId: '',
      policy: ImportPolicy.ATOMIC,
    }),
    undefined,
  ]);
  __setImportHubClientForTest(null);
});

test('core/web import client: calls runImportAsync with async request wrapper', async () => {
  const hub = installHub();
  hub.runAsync.resolve({ dataTransferJobId: 'ij-1', taskJobId: 'tj-1' });
  const input = { targetModel: 'base.Country', sourceRef: 'src-async' };
  const resp = await runImportAsync(input);
  expect(resp.dataTransferJobId).toBe('ij-1');
  expect(hub.runAsync.calls.length).toBe(1);
  __setImportHubClientForTest(null);
});
