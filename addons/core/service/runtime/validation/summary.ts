// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../../error';
import type { ObjectRecord } from '../../../utils/types';

/**
 * Lightweight validation issue payload used by validation summaries.
 */
export interface ValidationIssueLite {
  scope?: string;
  field?: string;
  code?: string;
  message?: string;
  severity?: string;
  method?: string;
}

/**
 * Summary information for issues grouped by field.
 */
export interface ValidationFieldIssueSummary {
  firstCode?: string;
  issueCount?: number;
  firstKernelCode?: string;
  // Backward-compatible alias kept for existing consumers.
  kernelCode?: string;
}

/**
 * Fully resolved validation summary derived from metadata or a ChoysumError.
 */
export interface ResolvedValidationSummary {
  kernelCode?: string;
  sqlCode?: string;
  issues: ValidationIssueLite[];
  fieldIssueSummary: Record<string, ValidationFieldIssueSummary>;
  getFieldFirstCode: (field: string) => string | undefined;
  getFieldKernelCode: (field: string) => string | undefined;
}

type MetadataMap = Record<string, string>;

function asMetadata(input?: MetadataMap | ChoysumError | { metadata?: MetadataMap }): MetadataMap {
  if (!input) return {};
  if (input instanceof ChoysumError) return input.metadata || {};

  const maybeMetadata = (input as { metadata?: MetadataMap }).metadata;
  if (maybeMetadata && typeof maybeMetadata === 'object') {
    return maybeMetadata;
  }

  return input as MetadataMap;
}

function parseJsonArray<T>(raw: string | undefined): T[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(String(raw));
    return Array.isArray(parsed) ? (parsed as T[]) : [];
  } catch {
    return [];
  }
}

function parseFieldIssueSummary(raw: string | undefined): Record<string, ValidationFieldIssueSummary> {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(String(raw)) as Record<string, ObjectRecord>;
    const out: Record<string, ValidationFieldIssueSummary> = {};
    for (const [field, item] of Object.entries(parsed || {})) {
      const firstCode = String(item?.firstCode || '').trim();
      const firstKernelCode = String(item?.firstKernelCode || '').trim();
      const kernelCode = String(item?.kernelCode || '').trim();
      const issueCountRaw = Number(item?.issueCount || 0);
      out[field] = {
        firstCode: firstCode || undefined,
        firstKernelCode: firstKernelCode || undefined,
        kernelCode: kernelCode || undefined,
        issueCount: Number.isFinite(issueCountRaw) ? issueCountRaw : 0,
      };
    }
    return out;
  } catch {
    return {};
  }
}

function buildSummaryFromIssues(issues: ValidationIssueLite[]): Record<string, ValidationFieldIssueSummary> {
  const fieldSummary: Record<string, ValidationFieldIssueSummary> = {};
  for (const issue of issues) {
    const field = String(issue?.field || '').trim();
    if (!field) continue;

    if (!fieldSummary[field]) {
      fieldSummary[field] = {
        firstCode: String(issue?.code || '').trim() || undefined,
        issueCount: 0,
      };
    }

    const row = fieldSummary[field];
    row.issueCount = Number(row.issueCount || 0) + 1;

    const code = String(issue?.code || '').trim();
    if (!row.firstKernelCode && String(issue?.scope || '').trim() === 'kernel' && code) {
      row.firstKernelCode = code;
      row.kernelCode = code;
    }
  }
  return fieldSummary;
}

/**
 * Resolves a normalized validation summary from metadata or a ChoysumError.
 */
export function resolveValidationSummary(input?: MetadataMap | ChoysumError | { metadata?: MetadataMap }): ResolvedValidationSummary {
  const metadata = asMetadata(input);
  const issues = parseJsonArray<ValidationIssueLite>(metadata.issues);

  const parsedSummary = parseFieldIssueSummary(metadata.fieldIssueSummary);
  const fieldIssueSummary = Object.keys(parsedSummary).length > 0 ? parsedSummary : buildSummaryFromIssues(issues);

  for (const row of Object.values(fieldIssueSummary)) {
    if (!row.firstKernelCode && row.kernelCode) {
      row.firstKernelCode = row.kernelCode;
    }
    if (!row.kernelCode && row.firstKernelCode) {
      row.kernelCode = row.firstKernelCode;
    }
  }

  const kernelCodeFromMetadata = String(metadata.kernelCode || '').trim();
  const sqlCodeFromMetadata = String(metadata.sqlCode || '').trim();

  const kernelCode = kernelCodeFromMetadata || String(issues.find(item => String(item?.scope || '').trim() === 'kernel')?.code || '').trim() || undefined;
  const sqlCode = sqlCodeFromMetadata || String(issues.find(item => String(item?.scope || '').trim() === 'sql')?.code || '').trim() || undefined;

  const getFieldFirstCode = (field: string): string | undefined => {
    const key = String(field || '').trim();
    if (!key) return undefined;
    const row = fieldIssueSummary[key];
    const code = String(row?.firstCode || '').trim();
    return code || undefined;
  };

  const getFieldKernelCode = (field: string): string | undefined => {
    const key = String(field || '').trim();
    if (!key) return undefined;
    const row = fieldIssueSummary[key];
    const firstKernelCode = String(row?.firstKernelCode || '').trim();
    if (firstKernelCode) return firstKernelCode;
    const kernelCodeAlias = String(row?.kernelCode || '').trim();
    if (kernelCodeAlias) return kernelCodeAlias;

    const fallback = issues.find(item => String(item?.field || '').trim() === key && String(item?.scope || '').trim() === 'kernel');
    const fallbackCode = String(fallback?.code || '').trim();
    return fallbackCode || undefined;
  };

  return {
    kernelCode,
    sqlCode,
    issues,
    fieldIssueSummary,
    getFieldFirstCode,
    getFieldKernelCode,
  };
}
