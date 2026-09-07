// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { CreateWebClient } from '../rpc/client_factory';
import {
  DescribeFieldsRequestSchema,
  ExportHub,
  ExportMode,
  ExportRunRequestSchema,
  type DescribeFieldsResponse,
  type ExportFieldNode,
  type ExportReport,
  type ExportRunResponse,
} from './pb/export_pb';

type ExportCallOptions = { signal?: AbortSignal };

export type ExportRunInput = {
  model: string;
  mode?: ExportMode;
  fields?: string[];
  domain?: string;
  ids?: string[];
  companyId?: string;
};

export type ExportTerminologyRunInput = {
  application: string;
  module: string;
  lang: string;
};

export type ExportHubClient = {
  describeFields(req: ReturnType<typeof create<typeof DescribeFieldsRequestSchema>>, options?: ExportCallOptions): Promise<DescribeFieldsResponse>;
  preview(req: ReturnType<typeof create<typeof ExportRunRequestSchema>>, options?: ExportCallOptions): Promise<ExportRunResponse>;
  run(req: ReturnType<typeof create<typeof ExportRunRequestSchema>>, options?: ExportCallOptions): Promise<ExportRunResponse>;
};

export type ExportClientDeps = {
  /** Injected hub (tests / alternate transports). Default: CreateWebClient(ExportHub). */
  hub?: ExportHubClient;
};

export type ExportClient = {
  describeExportFields(model: string, signal?: AbortSignal): Promise<DescribeFieldsResponse>;
  previewExport(input: ExportRunInput, signal?: AbortSignal): Promise<ExportRunResponse>;
  runExport(input: ExportRunInput, signal?: AbortSignal): Promise<ExportRunResponse>;
  runTerminologyExport(input: ExportTerminologyRunInput, signal?: AbortSignal): Promise<ExportRunResponse>;
};

function callOptions(signal?: AbortSignal): ExportCallOptions | undefined {
  if (signal == null) {
    return undefined;
  }
  return { signal };
}

function toRunRequest(input: ExportRunInput) {
  return create(ExportRunRequestSchema, {
    model: input.model,
    mode: input.mode ?? ExportMode.DATA,
    fields: input.fields ?? [],
    domain: input.domain ?? '',
    ids: input.ids ?? [],
    companyId: input.companyId ?? '',
  });
}

function toTerminologyRunRequest(input: ExportTerminologyRunInput) {
  return create(ExportRunRequestSchema, {
    profile: 'terminology',
    application: input.application,
    module_: input.module,
    lang: input.lang,
  });
}

/** Composable ExportHub API. Pass `hub` in tests instead of mocking CreateWebClient. */
export function createExportClient(deps: ExportClientDeps = {}): ExportClient {
  const defaultHub = CreateWebClient(ExportHub);
  const hub = (): ExportHubClient => deps.hub ?? (defaultHub() as unknown as ExportHubClient);

  return {
    describeExportFields(model, signal) {
      return hub().describeFields(create(DescribeFieldsRequestSchema, { model }), callOptions(signal));
    },
    previewExport(input, signal) {
      return hub().preview(toRunRequest(input), callOptions(signal));
    },
    runExport(input, signal) {
      return hub().run(toRunRequest(input), callOptions(signal));
    },
    runTerminologyExport(input, signal) {
      return hub().run(toTerminologyRunRequest(input), callOptions(signal));
    },
  };
}

const defaultExportClient = createExportClient();

export const describeExportFields = defaultExportClient.describeExportFields;
export const previewExport = defaultExportClient.previewExport;
export const runExport = defaultExportClient.runExport;
export const runTerminologyExport = defaultExportClient.runTerminologyExport;

export { ExportHub, ExportMode, type DescribeFieldsResponse, type ExportFieldNode, type ExportReport, type ExportRunResponse };
