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

export type ImportHubClient = {
  describeImportFields(
    req: ReturnType<typeof create<typeof DescribeImportFieldsRequestSchema>>,
    options?: ImportCallOptions,
  ): Promise<DescribeImportFieldsResponse>;
  parseHeaders(req: ReturnType<typeof create<typeof ParseHeadersRequestSchema>>, options?: ImportCallOptions): Promise<ParseHeadersResponse>;
  preview(req: ReturnType<typeof create<typeof ImportRunRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunResponse>;
  run(req: ReturnType<typeof create<typeof ImportRunRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunResponse>;
  runAsync(req: ReturnType<typeof create<typeof ImportRunAsyncRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunAsyncResponse>;
};

export type ImportClientDeps = {
  /** Injected hub (tests / alternate transports). Default: CreateWebClient(ImportHub). */
  hub?: ImportHubClient;
};

export type ImportClient = {
  describeImportFields(model: string, signal?: AbortSignal): Promise<DescribeImportFieldsResponse>;
  parseHeaders(sourceRef: string, signal?: AbortSignal): Promise<ParseHeadersResponse>;
  previewImport(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunResponse>;
  runImport(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunResponse>;
  runImportAsync(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunAsyncResponse>;
};

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

/** Composable ImportHub API. Pass `hub` in tests instead of mocking CreateWebClient. */
export function createImportClient(deps: ImportClientDeps = {}): ImportClient {
  const defaultHub = CreateWebClient(ImportHub);
  const hub = (): ImportHubClient => deps.hub ?? (defaultHub() as unknown as ImportHubClient);

  return {
    describeImportFields(model, signal) {
      return hub().describeImportFields(create(DescribeImportFieldsRequestSchema, { model }), callOptions(signal));
    },
    parseHeaders(sourceRef, signal) {
      return hub().parseHeaders(create(ParseHeadersRequestSchema, { sourceRef }), callOptions(signal));
    },
    previewImport(input, signal) {
      return hub().preview(toRunRequest(input, ImportPolicy.ATOMIC), callOptions(signal));
    },
    runImport(input, signal) {
      return hub().run(toRunRequest(input, ImportPolicy.ATOMIC), callOptions(signal));
    },
    runImportAsync(input, signal) {
      return hub().runAsync(
        create(ImportRunAsyncRequestSchema, { run: toRunRequest(input, ImportPolicy.ATOMIC) }),
        callOptions(signal),
      );
    },
  };
}

const defaultImportClient = createImportClient();

export const describeImportFields = defaultImportClient.describeImportFields;
export const parseHeaders = defaultImportClient.parseHeaders;
export const previewImport = defaultImportClient.previewImport;
export const runImport = defaultImportClient.runImport;
export const runImportAsync = defaultImportClient.runImportAsync;

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
