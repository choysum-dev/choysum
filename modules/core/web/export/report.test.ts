// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { ExportMessageType } from './pb/export_pb';
import { exportReportErrorText, exportReportHasErrors, exportPreviewSummary } from './report';

describe('exportReportHasErrors', () => {
  it('treats missing report as error', () => {
    expect(exportReportHasErrors(null)).toBe(true);
    expect(exportReportHasErrors(undefined)).toBe(true);
  });

  it('detects stats and message errors', () => {
    expect(exportReportHasErrors({ stats: { error: 1 } })).toBe(true);
    expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.ERROR, text: 'bad row' }] })).toBe(true);
    expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.UNSPECIFIED, text: '' }] })).toBe(true);
    expect(exportReportHasErrors({ stats: { ok: 1 }, messages: [{ type_: ExportMessageType.WARNING, text: 'ok' }] })).toBe(false);
  });
});

describe('exportReportErrorText', () => {
  it('returns first error message text', () => {
    expect(exportReportErrorText({ messages: [{ type_: ExportMessageType.ERROR, text: 'duplicate code' }] })).toBe('duplicate code');
  });

  it('falls back to stats error count', () => {
    expect(exportReportErrorText({ stats: { error: 2 }, messages: [{ text: '' }] })).toBe('Export finished with 2 error(s).');
  });

  it('returns generic failure text', () => {
    expect(exportReportErrorText({ stats: { ok: 0 } })).toBe('Export failed.');
  });

  it('uses unspecified message text when present', () => {
    expect(
      exportReportErrorText({
        messages: [{ type_: ExportMessageType.UNSPECIFIED, text: '  row failed  ' }],
      }),
    ).toBe('  row failed  ');
  });
});

describe('exportPreviewSummary', () => {
  it('uses numeric error counts when present', () => {
    expect(exportPreviewSummary({ stats: { ok: 1, error: 2, total: 3 } })).toBe('Preview: 1 ok, 2 errors, 3 total');
  });

  it('reflects message-only errors without numeric count', () => {
    expect(
      exportPreviewSummary({
        stats: { ok: 0, error: 0, total: 1 },
        messages: [{ type_: ExportMessageType.ERROR, text: 'bad row' }],
      }),
    ).toBe('Preview: 0 ok, errors, 1 total');
  });

  it('returns empty summary without stats', () => {
    expect(exportPreviewSummary({ messages: [] })).toBe('');
  });
});

describe('exportReportHasErrors skip messages', () => {
  it('ignores skip messages', () => {
    expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.SKIP, text: 'skipped' }] })).toBe(false);
  });
});
