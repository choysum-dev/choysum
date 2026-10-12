// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { productLogoUrl, resolveCompanyLogoUrl } from './resolveCompanyLogoUrl';

test('resolveCompanyLogoUrl: empty falls back to product logo-32.png', () => {
  expect(resolveCompanyLogoUrl(undefined)).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl(null)).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl('')).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl('   ')).toBe(productLogoUrl);
  expect(Boolean(productLogoUrl)).toBe(true);
});

test('resolveCompanyLogoUrl: prefers previewUrl from attachment envelope', () => {
  expect(
    resolveCompanyLogoUrl({
      kind: 'attachment',
      previewUrl: '/_document/bindings/b1/content',
    }),
  ).toBe('/_document/bindings/b1/content');
});

test('resolveCompanyLogoUrl: reads nested descriptor previewUrl', () => {
  expect(
    resolveCompanyLogoUrl({
      descriptor: { previewUrl: 'https://cdn.example/logo.png' },
    }),
  ).toBe('https://cdn.example/logo.png');
});

test('resolveCompanyLogoUrl: accepts absolute and root-relative strings', () => {
  expect(resolveCompanyLogoUrl('/static/x.png')).toBe('/static/x.png');
  expect(resolveCompanyLogoUrl('https://example.com/a.png')).toBe('https://example.com/a.png');
});
