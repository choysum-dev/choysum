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

type ExportHubClient = {
  describeFields(req: ReturnType<typeof create<typeof DescribeFieldsRequestSchema>>, options?: ExportCallOptions): Promise<DescribeFieldsResponse>;
  preview(req: ReturnType<typeof create<typeof ExportRunRequestSchema>>, options?: ExportCallOptions): Promise<ExportRunResponse>;
  run(req: ReturnType<typeof create<typeof ExportRunRequestSchema>>, options?: ExportCallOptions): Promise<ExportRunResponse>;
};

const exportHubClient = CreateWebClient(ExportHub);

function exportHub(): ExportHubClient {
  return exportHubClient() as unknown as ExportHubClient;
}

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

export function describeExportFields(model: string, signal?: AbortSignal): Promise<DescribeFieldsResponse> {
  return exportHub().describeFields(create(DescribeFieldsRequestSchema, { model }), callOptions(signal));
}

export function previewExport(input: ExportRunInput, signal?: AbortSignal): Promise<ExportRunResponse> {
  return exportHub().preview(toRunRequest(input), callOptions(signal));
}

export function runExport(input: ExportRunInput, signal?: AbortSignal): Promise<ExportRunResponse> {
  return exportHub().run(toRunRequest(input), callOptions(signal));
}

export function runTerminologyExport(input: ExportTerminologyRunInput, signal?: AbortSignal): Promise<ExportRunResponse> {
  return exportHub().run(toTerminologyRunRequest(input), callOptions(signal));
}

export { ExportHub, ExportMode, type DescribeFieldsResponse, type ExportFieldNode, type ExportReport, type ExportRunResponse };
