// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';

test('resolveValidationSummary prefers metadata kernelCode/sqlCode over issues fallback', () => {
  const error = new ChoysumError({
    domain: 'core.repository',
    code: 'validation_failed',
    message: 'validation failed',
  }).withMetadata({
    kernelCode: 'kernel_required_missing',
    sqlCode: 'sql_unique_violation',
    issues: JSON.stringify([
      { scope: 'kernel', code: 'kernel_decimal_invalid' },
      { scope: 'sql', code: 'sql_fk_violation' },
    ]),
  });

  const summary = resolveValidationSummary(error);
  expect(summary.kernelCode).toBe('kernel_required_missing');
  expect(summary.sqlCode).toBe('sql_unique_violation');
});

test('resolveValidationSummary supports fieldIssueSummary firstKernelCode and kernelCode fallback', () => {
  const summary = resolveValidationSummary({
    fieldIssueSummary: JSON.stringify({
      Code: {
        firstCode: 'platform_unknown_field',
        issueCount: '2',
        kernelCode: 'kernel_required_missing',
      },
    }),
  });

  expect(summary.getFieldFirstCode('Code')).toBe('platform_unknown_field');
  expect(summary.getFieldKernelCode('Code')).toBe('kernel_required_missing');
  expect(summary.fieldIssueSummary.Code?.firstKernelCode).toBe('kernel_required_missing');
});

test('resolveValidationSummary builds field summary from issues when metadata summary is missing', () => {
  const summary = resolveValidationSummary({
    issues: JSON.stringify([
      { scope: 'platform', field: 'Code', code: 'platform_unknown_field' },
      { scope: 'kernel', field: 'Code', code: 'kernel_required_missing' },
      { scope: 'kernel', code: 'kernel_decimal_invalid' },
      { scope: 'sql', code: 'sql_check_violation' },
    ]),
  });

  expect(summary.kernelCode).toBe('kernel_required_missing');
  expect(summary.sqlCode).toBe('sql_check_violation');
  expect(summary.getFieldFirstCode('Code')).toBe('platform_unknown_field');
  expect(summary.getFieldKernelCode('Code')).toBe('kernel_required_missing');
  expect(summary.fieldIssueSummary.Code?.issueCount).toBe(2);
});

test('resolveValidationSummary falls back to issues for kernel/sql classification when metadata codes are missing', () => {
  const error = new ChoysumError({
    domain: 'core.repository',
    code: 'validation_failed',
    message: 'validation failed',
  }).withMetadata({
    issues: JSON.stringify([
      { scope: 'platform', code: 'platform_unknown_field' },
      { scope: 'kernel', code: 'kernel_required_missing' },
      { scope: 'sql', code: 'sql_fk_violation' },
    ]),
  });

  const summary = resolveValidationSummary(error);
  expect(summary.kernelCode).toBe('kernel_required_missing');
  expect(summary.sqlCode).toBe('sql_fk_violation');
});

test('resolveValidationSummary accepts wrapper object metadata and normalizes invalid fieldIssueSummary payloads', () => {
  const summary = resolveValidationSummary({
    metadata: {
      issues: JSON.stringify({ not: 'array' }),
      fieldIssueSummary: JSON.stringify({
        Name: {
          firstCode: ' ',
          firstKernelCode: '',
          kernelCode: 'kernel_required_missing',
          issueCount: 'not-a-number',
        },
      }),
    },
  } as any);

  expect(summary.issues).toEqual([]);
  expect(summary.kernelCode).toBe(undefined);
  expect(summary.sqlCode).toBe(undefined);
  expect(summary.getFieldFirstCode('Name')).toBe(undefined);
  expect(summary.getFieldKernelCode('Name')).toBe('kernel_required_missing');
  expect(summary.fieldIssueSummary.Name?.firstKernelCode).toBe('kernel_required_missing');
  expect(summary.fieldIssueSummary.Name?.issueCount).toBe(0);
});

test('resolveValidationSummary tolerates malformed JSON and empty field names from issues', () => {
  const malformed = resolveValidationSummary({
    issues: '{bad-json',
    fieldIssueSummary: '{also-bad',
  });

  expect(malformed.issues).toEqual([]);
  expect(Object.keys(malformed.fieldIssueSummary)).toEqual([]);

  const fromIssues = resolveValidationSummary({
    issues: JSON.stringify([
      { scope: 'kernel', field: '  ', code: 'kernel_required_missing' },
      { scope: 'kernel', field: 'Name', code: 'kernel_decimal_invalid' },
      { scope: 'sql', field: 'Name', code: 'sql_check_violation' },
    ]),
  });

  expect(fromIssues.getFieldKernelCode('')).toBe(undefined);
  expect(fromIssues.getFieldFirstCode('Name')).toBe('kernel_decimal_invalid');
  expect(fromIssues.getFieldKernelCode('Name')).toBe('kernel_decimal_invalid');
});

test('resolveValidationSummary getFieldKernelCode falls back to issues when parsed summary lacks kernel markers', () => {
  const summary = resolveValidationSummary({
    fieldIssueSummary: JSON.stringify({
      Code: {
        firstCode: 'platform_unknown_field',
        issueCount: 1,
      },
    }),
    issues: JSON.stringify([{ scope: 'kernel', field: 'Code', code: 'kernel_required_missing' }]),
  });

  expect(summary.getFieldFirstCode('Code')).toBe('platform_unknown_field');
  expect(summary.getFieldKernelCode('Code')).toBe('kernel_required_missing');
  expect(summary.getFieldKernelCode('Unknown')).toBe(undefined);
});

test('resolveValidationSummary handles empty input and missing metadata payloads', () => {
  const empty = resolveValidationSummary();
  expect(empty.kernelCode).toBe(undefined);
  expect(empty.sqlCode).toBe(undefined);
  expect(empty.issues).toEqual([]);
  expect(Object.keys(empty.fieldIssueSummary)).toEqual([]);

  const withoutMetadata = resolveValidationSummary(
    new ChoysumError({
      domain: 'core.repository',
      code: 'validation_failed',
      message: 'failed-no-metadata',
    })
  );
  expect(withoutMetadata.kernelCode).toBe(undefined);
  expect(withoutMetadata.sqlCode).toBe(undefined);
  expect(withoutMetadata.issues).toEqual([]);
});

test('resolveValidationSummary backfills kernelCode from firstKernelCode and trims blank field lookups', () => {
  const summary = resolveValidationSummary({
    kernelCode: '   ',
    sqlCode: '',
    fieldIssueSummary: JSON.stringify({
      Name: {
        firstCode: null,
        firstKernelCode: 'kernel_from_first',
        issueCount: 1,
      },
      EmptyMeta: null,
    }),
    issues: JSON.stringify([
      { scope: 'sql', field: 'Name', code: 'sql_unique_violation' },
      { scope: 'kernel', field: 'Name', code: 'kernel_issue_fallback' },
    ]),
  } as any);

  expect(summary.kernelCode).toBe('kernel_issue_fallback');
  expect(summary.sqlCode).toBe('sql_unique_violation');
  expect(summary.getFieldFirstCode('Name')).toBe(undefined);
  expect(summary.getFieldKernelCode('Name')).toBe('kernel_from_first');
  expect(summary.fieldIssueSummary.Name?.kernelCode).toBe('kernel_from_first');
  expect(summary.getFieldFirstCode('   ')).toBe(undefined);
  expect(summary.getFieldKernelCode('   ')).toBe(undefined);
});
