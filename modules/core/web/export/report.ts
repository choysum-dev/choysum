// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ExportMessageType, type ExportMessage, type ExportReport, type ExportStats } from './pb/export_pb';

/** Wire-shaped report fragments accepted by UI helpers (partial protobuf messages). */
export type ExportReportLike = {
  stats?: Partial<ExportStats> | null;
  messages?: Array<Partial<ExportMessage> | null> | null;
} | null | undefined;

export function exportReportHasErrors(report: ExportReportLike | ExportReport): boolean {
  if (!report) {
    return true;
  }
  if ((report.stats?.error ?? 0) > 0) {
    return true;
  }
  return (report.messages ?? []).some(message => {
    if (!message) return false;
    const type = message.type_ ?? ExportMessageType.UNSPECIFIED;
    return type === ExportMessageType.ERROR || (type === ExportMessageType.UNSPECIFIED && Boolean(String(message.text ?? '').trim()));
  });
}

export function exportReportErrorText(report: ExportReportLike | ExportReport): string {
  const first = report?.messages?.find(message => {
    if (!message) return false;
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

export function exportPreviewSummary(report: ExportReportLike | ExportReport): string {
  const stats = report?.stats;
  if (!stats) {
    return '';
  }
  const count = stats.error ?? 0;
  const errors =
    count === 1 ? '1 error' : count > 1 ? `${count} errors` : exportReportHasErrors(report) ? 'errors' : '0 errors';
  return `Preview: ${stats.ok ?? 0} ok, ${errors}, ${stats.total ?? 0} total`;
}
