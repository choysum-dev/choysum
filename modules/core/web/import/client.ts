// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { CreateWebClient } from '../rpc/client_factory';
import {
  ImportHub,
  ImportPolicy,
  ImportRunRequestSchema,
  ParseHeadersRequestSchema,
  type ImportReport,
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
  parseHeaders(req: ReturnType<typeof create<typeof ParseHeadersRequestSchema>>, options?: ImportCallOptions): Promise<ParseHeadersResponse>;
  preview(req: ReturnType<typeof create<typeof ImportRunRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunResponse>;
  run(req: ReturnType<typeof create<typeof ImportRunRequestSchema>>, options?: ImportCallOptions): Promise<ImportRunResponse>;
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

export function parseHeaders(sourceRef: string, signal?: AbortSignal): Promise<ParseHeadersResponse> {
  return importHub().parseHeaders(create(ParseHeadersRequestSchema, { sourceRef }), callOptions(signal));
}

export function previewImport(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunResponse> {
  return importHub().preview(toRunRequest(input, ImportPolicy.ATOMIC), callOptions(signal));
}

export function runImport(input: ImportRunInput, signal?: AbortSignal): Promise<ImportRunResponse> {
  return importHub().run(toRunRequest(input, ImportPolicy.ATOMIC), callOptions(signal));
}

export { ImportHub, ImportPolicy, type ImportReport, type ImportRunResponse, type ParseHeadersResponse };
