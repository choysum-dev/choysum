// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ExportMessageType } from './pb/export_pb';
import { exportReportErrorText, exportReportHasErrors, exportPreviewSummary } from './report';

test('exportReportHasErrors: treats missing report as error', () => {
  expect(exportReportHasErrors(null)).toBe(true);
  expect(exportReportHasErrors(undefined)).toBe(true);
});

test('exportReportHasErrors: detects stats and message errors', () => {
  expect(exportReportHasErrors({ stats: { error: 1 } })).toBe(true);
  expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.ERROR, text: 'bad row' }] })).toBe(true);
  expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.UNSPECIFIED, text: '' }] })).toBe(true);
  expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.UNSPECIFIED, text: 'message only' }] })).toBe(true);
  expect(exportReportHasErrors({ stats: { ok: 1 }, messages: [{ type_: ExportMessageType.WARNING, text: 'ok' }] })).toBe(false);
  expect(exportReportHasErrors({ stats: { ok: 1 }, messages: [{ text: 'ok' }] })).toBe(true);
});

test('exportReportErrorText: returns first error message text', () => {
  expect(exportReportErrorText({ messages: [{ type_: ExportMessageType.ERROR, text: 'duplicate code' }] })).toBe('duplicate code');
});

test('exportReportErrorText: falls back to stats error count', () => {
  expect(exportReportErrorText({ stats: { error: 2 }, messages: [{ text: '' }] })).toBe('Export finished with 2 error(s).');
  expect(exportReportErrorText({ stats: { error: 5 } })).toBe('Export finished with 5 error(s).');
  expect(exportReportErrorText({ stats: {}, messages: [{ type_: ExportMessageType.ERROR, text: '' }] })).toBe('Export failed.');
  expect(
    exportReportErrorText({ stats: { error: 1 }, messages: [{ type_: ExportMessageType.WARNING, text: 'warn' }] }),
  ).toBe('Export finished with 1 error(s).');
});

test('exportReportErrorText: returns generic failure text', () => {
  expect(exportReportErrorText({ stats: { ok: 0 } })).toBe('Export failed.');
  expect(exportReportErrorText(null)).toBe('Export failed.');
});

test('exportReportErrorText: skips blank unspecified messages when locating error text', () => {
  expect(
    exportReportErrorText({
      stats: { error: 1 },
      messages: [{ type_: ExportMessageType.UNSPECIFIED, text: '   ' }],
    }),
  ).toBe('Export finished with 1 error(s).');
});

test('exportReportErrorText: uses unspecified message text when present', () => {
  expect(
    exportReportErrorText({
      messages: [{ type_: ExportMessageType.UNSPECIFIED, text: '  row failed  ' }],
    }),
  ).toBe('  row failed  ');
});

test('exportPreviewSummary: uses numeric error counts when present', () => {
  expect(exportPreviewSummary({ stats: { ok: 1, error: 1, total: 2 } })).toBe('Preview: 1 ok, 1 error, 2 total');
  expect(exportPreviewSummary({ stats: { ok: 1, error: 2, total: 3 } })).toBe('Preview: 1 ok, 2 errors, 3 total');
});

test('exportPreviewSummary: reflects message-only errors without numeric count', () => {
  expect(
    exportPreviewSummary({
      stats: { ok: 0, error: 0, total: 1 },
      messages: [{ type_: ExportMessageType.ERROR, text: 'bad row' }],
    }),
  ).toBe('Preview: 0 ok, errors, 1 total');
});

test('exportPreviewSummary: returns empty summary without stats', () => {
  expect(exportPreviewSummary({ messages: [] })).toBe('');
});

test('exportPreviewSummary: shows a clean zero-error preview summary', () => {
  expect(exportPreviewSummary({ stats: { ok: 2, error: 0, total: 2 } })).toBe('Preview: 2 ok, 0 errors, 2 total');
  expect(exportPreviewSummary({ stats: {} })).toBe('Preview: 0 ok, 0 errors, 0 total');
});

test('exportReportHasErrors skip messages: ignores skip messages', () => {
  expect(exportReportHasErrors({ messages: [{ type_: ExportMessageType.SKIP, text: 'skipped' }] })).toBe(false);
});
