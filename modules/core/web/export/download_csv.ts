// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Triggers a browser download for export CSV bytes returned by ExportHub.
 */
export function downloadExportCsvBytes(bytes: Uint8Array | ArrayBuffer, fileName: string): void {
  const part: BlobPart = bytes instanceof Uint8Array ? (bytes as BlobPart) : bytes;
  const blob = new Blob([part], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName || 'export.csv';
  anchor.rel = 'noopener';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

/**
 * Suggest a CSV file name from the target model (e.g. partner.Partner → Partner.csv).
 */
export function suggestExportFileName(model: string): string {
  const segment = String(model ?? '')
    .split('.')
    .pop()
    ?.trim();
  return segment ? `${segment}.csv` : 'export.csv';
}
