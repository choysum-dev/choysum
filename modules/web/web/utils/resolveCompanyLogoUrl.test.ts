// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { productLogoUrl, resolveCompanyLogoUrl } from './resolveCompanyLogoUrl';

test('resolveCompanyLogoUrl: empty falls back to product logo.svg', () => {
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
  expect(resolveCompanyLogoUrl('data:image/png;base64,abc')).toBe('data:image/png;base64,abc');
  expect(resolveCompanyLogoUrl('blob:https://example.com/1')).toBe('blob:https://example.com/1');
});

test('resolveCompanyLogoUrl: rejects opaque strings and empty attachment envelopes', () => {
  expect(resolveCompanyLogoUrl('relative/path.png')).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl('javascript:alert(1)')).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl('//evil.example/x.png')).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl([])).toBe(productLogoUrl);
  expect(resolveCompanyLogoUrl({ kind: 'attachment' })).toBe(productLogoUrl);
  expect(
    resolveCompanyLogoUrl({
      previewUrl: 'javascript:alert(1)',
    }),
  ).toBe(productLogoUrl);
  expect(
    resolveCompanyLogoUrl({
      previewUrl: '//evil.example/x.png',
    }),
  ).toBe(productLogoUrl);
});

test('resolveCompanyLogoUrl: reads Descriptor.downloadUrl and thumbnailUrl', () => {
  expect(
    resolveCompanyLogoUrl({
      Descriptor: { downloadUrl: 'https://cdn.example/from-descriptor.png' },
    }),
  ).toBe('https://cdn.example/from-descriptor.png');
  expect(
    resolveCompanyLogoUrl({
      thumbnailUrl: 'https://cdn.example/thumb.png',
    }),
  ).toBe('https://cdn.example/thumb.png');
});
