// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import { ValidationPipelineError, type ConstraintMode } from '../../metadata';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import type { ObjectRecord } from '../../../../utils/types';

export function wrapRepositoryValidationError(meta: ModelMetadata, error: ValidationPipelineError, mode: ConstraintMode): ChoysumError {
  const primaryIssue = error.issues.find(issue => issue.severity === 'error') || error.issues[0];
  const message = primaryIssue?.message || error.message || 'validation failed';
  const wrapped = ChoysumError.wrap(
    error,
    {
      domain: 'core.repository',
      code: 'validation_failed',
      message,
    },
    true
  ).withGrpcCode(GrpcCode.InvalidArgument);

  const metadata: Record<string, string> = {
    mode,
    issueCount: String(error.issues.length),
  };

  const fields = Array.from(new Set(error.issues.map(issue => String(issue.field || '').trim()).filter(Boolean)));

  const fieldIssues: Record<string, Array<Record<string, string>>> = {};
  const fieldIssueSummary: Record<string, Record<string, string>> = {};
  const globalIssues: Array<Record<string, string>> = [];
  const issueList: Array<Record<string, string>> = [];
  for (const issue of error.issues) {
    const field = String(issue.field || '').trim();

    const item: Record<string, string> = {
      scope: String(issue.scope || ''),
      code: String(issue.code || ''),
      message: String(issue.message || ''),
      severity: String(issue.severity || ''),
    };
    if (issue.method) item.method = String(issue.method);

    const listItem: Record<string, string> = { ...item };
    if (field) listItem.field = field;
    issueList.push(listItem);

    if (!field) {
      globalIssues.push(item);
      continue;
    }

    if (!fieldIssues[field]) fieldIssues[field] = [];
    fieldIssues[field].push(item);

    if (!fieldIssueSummary[field]) {
      fieldIssueSummary[field] = {
        firstCode: item.code,
        issueCount: '0',
      };
    }
    const summary = fieldIssueSummary[field];
    const count = Number(summary.issueCount);
    summary.issueCount = String(count + 1);
    if (!summary.kernelCode && item.scope === 'kernel' && item.code) {
      summary.firstKernelCode = item.code;
      summary.kernelCode = item.code;
    }
  }

  const modelName = meta.fullModelName || meta.modelName || meta.name;
  const primaryKernelIssue = error.issues.find(issue => issue.scope === 'kernel' && issue.code);
  if (modelName) metadata.model = modelName;
  if (primaryIssue?.scope) metadata.scope = primaryIssue.scope;
  if (primaryIssue?.field) metadata.field = primaryIssue.field;
  if (primaryIssue?.method) metadata.method = primaryIssue.method;
  if (primaryIssue?.code) metadata.issueCode = primaryIssue.code;
  if (primaryIssue?.scope === 'sql') {
    if (primaryIssue?.code) metadata.sqlCode = primaryIssue.code;
    const sqlMeta = (primaryIssue?.meta || {}) as ObjectRecord;
    const sqlConstraint = String(sqlMeta.sqlConstraint || '').trim();
    const sqlField = String(sqlMeta.sqlField || '').trim();
    const sqlColumns = Array.isArray(sqlMeta.sqlColumns) ? (sqlMeta.sqlColumns as unknown[]).map(v => String(v || '').trim()).filter(Boolean) : [];
    if (sqlConstraint) metadata.sqlConstraint = sqlConstraint;
    if (sqlField) metadata.sqlField = sqlField;
    if (sqlColumns.length > 0) metadata.sqlColumns = sqlColumns.join(',');
  }
  if (primaryKernelIssue?.code) metadata.kernelCode = primaryKernelIssue.code;
  if (fields.length) {
    metadata.fields = fields.join(',');
    metadata.fieldCount = String(fields.length);
    metadata.fieldIssues = JSON.stringify(fieldIssues);
    metadata.fieldIssueSummary = JSON.stringify(fieldIssueSummary);
  }
  if (globalIssues.length) {
    metadata.globalIssueCount = String(globalIssues.length);
    metadata.globalIssues = JSON.stringify(globalIssues);
  }
  metadata.issues = JSON.stringify(issueList);

  return wrapped.withMetadata(metadata);
}
