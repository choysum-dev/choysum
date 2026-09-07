// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { createImportClient, type ImportHubClient } from './client';
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

function installClient() {
  const describeImportFieldsFn = makeFn();
  const parseHeadersFn = makeFn();
  const preview = makeFn();
  const run = makeFn();
  const runAsync = makeFn();
  const hub: ImportHubClient = {
    describeImportFields: describeImportFieldsFn.fn as any,
    parseHeaders: parseHeadersFn.fn as any,
    preview: preview.fn as any,
    run: run.fn as any,
    runAsync: runAsync.fn as any,
  };
  const client = createImportClient({ hub });
  return { client, describeImportFieldsFn, parseHeadersFn, preview, run, runAsync };
}

test('core/web import client: calls ImportHub describeImportFields with model', async () => {
  const { client, describeImportFieldsFn } = installClient();
  describeImportFieldsFn.resolve({ fields: [], defaultFields: ['Name'] });
  const resp = await client.describeImportFields('partner.Partner');
  expect(resp.defaultFields).toEqual(['Name']);
  expect(describeImportFieldsFn.calls[0]?.args).toEqual([
    create(DescribeImportFieldsRequestSchema, { model: 'partner.Partner' }),
    undefined,
  ]);
});

test('core/web import client: passes abort signal to describeImportFields', async () => {
  const { client, describeImportFieldsFn } = installClient();
  describeImportFieldsFn.resolve({ fields: [] });
  const controller = new AbortController();
  await client.describeImportFields('base.Country', controller.signal);
  expect(describeImportFieldsFn.calls[0]?.args).toEqual([
    create(DescribeImportFieldsRequestSchema, { model: 'base.Country' }),
    { signal: controller.signal },
  ]);
});

test('core/web import client: calls ImportHub parseHeaders with source ref', async () => {
  const { client, parseHeadersFn } = installClient();
  parseHeadersFn.resolve({ headers: ['Name', 'Code'] });
  const resp = await client.parseHeaders('src-1');
  expect(resp.headers).toEqual(['Name', 'Code']);
  expect(parseHeadersFn.calls[0]?.args).toEqual([
    create(ParseHeadersRequestSchema, { sourceRef: 'src-1' }),
    undefined,
  ]);
});

test('core/web import client: passes abort signal to parseHeaders', async () => {
  const { client, parseHeadersFn } = installClient();
  parseHeadersFn.resolve({ headers: [] });
  const controller = new AbortController();
  await client.parseHeaders('src-2', controller.signal);
  expect(parseHeadersFn.calls[0]?.args).toEqual([
    create(ParseHeadersRequestSchema, { sourceRef: 'src-2' }),
    { signal: controller.signal },
  ]);
});

test('core/web import client: calls previewImport and runImport with atomic policy', async () => {
  const { client, preview, run } = installClient();
  preview.resolve({ report: { stats: { ok: 1 } } });
  run.resolve({ report: { stats: { ok: 2 } } });
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
  await client.previewImport(fullInput);
  await client.runImport(minimalInput);
  expect(preview.calls[0]?.args).toEqual([
    create(ImportRunRequestSchema, {
      targetModel: fullInput.targetModel,
      sourceRef: fullInput.sourceRef,
      columnMapping: fullInput.columnMapping,
      companyId: fullInput.companyId,
      policy: ImportPolicy.ATOMIC,
    }),
    undefined,
  ]);
  expect(run.calls[0]?.args).toEqual([
    create(ImportRunRequestSchema, {
      targetModel: minimalInput.targetModel,
      sourceRef: minimalInput.sourceRef,
      columnMapping: {},
      companyId: '',
      policy: ImportPolicy.ATOMIC,
    }),
    undefined,
  ]);
});

test('core/web import client: calls runImportAsync with async request wrapper', async () => {
  const { client, runAsync } = installClient();
  runAsync.resolve({ dataTransferJobId: 'ij-1', taskJobId: 'tj-1' });
  const input = { targetModel: 'base.Country', sourceRef: 'src-async' };
  const resp = await client.runImportAsync(input);
  expect(resp.dataTransferJobId).toBe('ij-1');
  expect(runAsync.calls.length).toBe(1);
});
