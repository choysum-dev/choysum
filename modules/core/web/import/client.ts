// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { CreateWebClient } from '../rpc/client_factory';
import {
  DescribeImportFieldsRequestSchema,
  ImportHub,
  ImportPolicy,
  ImportRunAsyncRequestSchema,
  ImportRunRequestSchema,
  ParseHeadersRequestSchema,
  type DescribeImportFieldsResponse,
  type ImportFieldNode,
  type ImportReport,
  type ImportRunAsyncResponse,
  type ImportRunResponse,
  type ParseHeadersResponse,
} from './pb/import_pb';

type ImportCallOptions = { signal?: AbortSignal };

export type ImportRunInput = {
  targetModel: string;
  sourceRef: string;
  columnMapping?: Record<string, string>;
  companyId?: string;
};

type ImportHubClient = {
  describeImportFields(
    req: ReturnType<typeof create<typeof DescribeImportFieldsRequestSchema>>,
    options?: ImportCallOptions,
  ): Promise<DescribeImportFieldsResponse>;
  parseHeaders(req: ReturnType<typeof create<typeof ParseHeadersRequestSchema>>, options?: ImportCallOptions): Promise<ParseHeadersResponse>;
  preview(req: ReturnType<typeof create<typeof ImportRunRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunResponse>;
  run(req: ReturnType<typeof create<typeof ImportRunRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunResponse>;
  runAsync(req: ReturnType<typeof create<typeof ImportRunAsyncRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunAsyncResponse>;
};

const importHubClient = CreateWebClient(ImportHub);

function importHub(): ImportHubClient {
  return importHubClient() as unknown as ImportHubClient;
}

function callOptions(signal?: AbortSignal): ImportCallOptions | undefined {
  if (signal == null) {
    return undefined;
  }
  return { signal };
}

function toRunRequest(input: ImportRunInput, dryRunPolicy: ImportPolicy) {
  return create(ImportRunRequestSchema, {
    targetModel: input.targetModel,
    sourceRef: input.sourceRef,
    columnMapping: input.columnMapping ?? {},
    companyId: input.companyId ?? '',
    policy: dryRunPolicy,
  });
}

export function describeImportFields(model: string, signal?: AbortSignal): Promise<DescribeImportFieldsResponse> {
  return importHub().describeImportFields(create(DescribeImportFieldsRequestSchema, { model }), callOptions(signal));
}

export function parseHeaders(sourceRef: string, signal?: AbortSignal): Promise<ParseHeadersResponse> {
  return importHub().parseHeaders(create(ParseHeadersRequestSchema, { sourceRef }), callOptions(signal));
}

export function previewImport(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunResponse> {
  return importHub().preview(toRunRequest(input, ImportPolicy.ATOMIC), callOptions(signal));
}

export function runImport(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunResponse> {
  return importHub().run(toRunRequest(input, ImportPolicy.ATOMIC), callOptions(signal));
}

export function runImportAsync(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunAsyncResponse> {
  return importHub().runAsync(
    create(ImportRunAsyncRequestSchema, { run: toRunRequest(input, ImportPolicy.ATOMIC) }),
    callOptions(signal)
  );
}

export {
  ImportHub,
  ImportPolicy,
  type DescribeImportFieldsResponse,
  type ImportFieldNode,
  type ImportReport,
  type ImportRunAsyncResponse,
  type ImportRunResponse,
  type ParseHeadersResponse,
};
