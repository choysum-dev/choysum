// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import productLogoUrl from '../assets/logo.svg';

function asRecord(raw: unknown): Record<string, unknown> | null {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
  return raw as Record<string, unknown>;
}

function normalizeOptionalString(raw: unknown): string | undefined {
  if (typeof raw !== 'string') return undefined;
  const trimmed = raw.trim();
  return trimmed || undefined;
}

function resolveDescriptor(raw: Record<string, unknown>): Record<string, unknown> | null {
  return asRecord(raw.descriptor) || asRecord(raw.Descriptor);
}

/**
 * Resolve a display URL for `base.Company.Logo`.
 * Prefers attachment preview/download URLs from Search/Browse enrichment;
 * falls back to the product mark (`logo.svg`) when empty.
 */
export function resolveCompanyLogoUrl(logo: unknown): string {
  if (logo == null || logo === '') return productLogoUrl;

  if (typeof logo === 'string') {
    const trimmed = logo.trim();
    if (!trimmed) return productLogoUrl;
    // Plain URL or data/blob from a prior local preview.
    if (/^(https?:|data:|blob:|\/)/i.test(trimmed)) return trimmed;
    return productLogoUrl;
  }

  const record = asRecord(logo);
  if (!record) return productLogoUrl;

  const descriptor = resolveDescriptor(record);
  const candidates = [
    record.previewUrl,
    record.url,
    record.thumbnailUrl,
    record.downloadUrl,
    descriptor?.previewUrl,
    descriptor?.downloadUrl,
    descriptor?.url,
  ];
  for (const candidate of candidates) {
    const url = normalizeOptionalString(candidate);
    if (url && /^(https?:|data:|blob:|\/)/i.test(url)) return url;
  }
  return productLogoUrl;
}

export { productLogoUrl };
