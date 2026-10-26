// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Map Revoked group labels for chart axes / legend (Kanban uses the same wording).
 */
export function formatTokenRevokedChartLabel(
  label: string,
  labels: { revoked: string; notRevoked: string },
): string {
  const raw = String(label || '').trim();
  if (/^Revoked\s*=\s*true$/i.test(raw) || raw === 'true') return labels.revoked;
  if (/^Revoked\s*=\s*false$/i.test(raw) || raw === 'false') return labels.notRevoked;
  return raw;
}
