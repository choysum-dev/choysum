// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  ImportHub,
  ImportPolicy,
  type ImportRunRequest,
  type ImportRunResponse,
  type ImportReport,
  type ParseHeadersRequest,
  type ParseHeadersResponse,
  type ImportMessage,
  type ImportStats,
  ImportMessageType,
  type DescribeImportFieldsResponse,
  type ImportFieldNode,
} from './pb/import_pb';

export {
  ImportHub,
  ImportPolicy,
  type ImportRunRequest,
  type ImportRunResponse,
  type ImportReport,
  type ParseHeadersRequest,
  type ParseHeadersResponse,
  type ImportMessage,
  type ImportStats,
  ImportMessageType,
  type DescribeImportFieldsResponse,
  type ImportFieldNode,
};

export type { ImportRunInput } from './client';
export type CoreImportHub = typeof ImportHub;

/** V1 web wizard: column mapping keyed by CSV header */
export type ColumnMapping = Record<string, string>;

export { describeImportFields, parseHeaders, previewImport, runImport } from './client';
