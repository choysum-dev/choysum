// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ExportMessageType, type ExportReport } from './pb/export_pb';

export function exportReportHasErrors(report: ExportReport | null | undefined): boolean {
  if (!report) {
    return true;
  }
  if ((report.stats?.error ?? 0) > 0) {
    return true;
  }
  return (report.messages ?? []).some(message => {
    const type = message.type_ ?? ExportMessageType.UNSPECIFIED;
    return type === ExportMessageType.ERROR || type === ExportMessageType.UNSPECIFIED;
  });
}

export function exportReportErrorText(report: ExportReport | null | undefined): string {
  const first = report?.messages?.find(message => {
    const type = message.type_ ?? ExportMessageType.UNSPECIFIED;
    return type === ExportMessageType.ERROR || (type === ExportMessageType.UNSPECIFIED && String(message.text ?? '').trim());
  });
  if (first?.text) {
    return first.text;
  }
  const statsError = report?.stats?.error;
  if (statsError != null && statsError > 0) {
    return `Export finished with ${statsError} error(s).`;
  }
  return 'Export failed.';
}

export function exportPreviewSummary(report: ExportReport | null | undefined): string {
  const stats = report?.stats;
  if (!stats) {
    return '';
  }
  const count = stats.error ?? 0;
  const errors = count > 0 ? `${count} errors` : exportReportHasErrors(report) ? 'errors' : '0 errors';
  return `Preview: ${stats.ok ?? 0} ok, ${errors}, ${stats.total ?? 0} total`;
}
