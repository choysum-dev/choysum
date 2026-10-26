// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  DEFAULT_IMAGE_AVATAR_SIZE_PX,
  formatAttachmentSize,
  hasAttachment,
  hasPendingLocalFile,
  isAcceptAllowed,
  resolveBindingId,
  resolveBinaryMediaIconKind,
  resolveDownloadUrl,
  resolveFileName,
  resolveImageAvatarBoxStyle,
  resolveImageAvatarContainBoxStyle,
  resolveImageAvatarRadiusClass,
  resolveObjectId,
  resolvePreviewUrl,
  toAttachmentCompactMetaText,
  toAttachmentDisplayText,
  toAttachmentMetaText,
  toAttachmentTooltipText,
} from './attachmentFieldHelpers';

describe('attachmentFieldHelpers', () => {
  test('resolves binding/object/fileName and download/preview URL chains', () => {
    expect(
      resolveBindingId({
        attachmentBindingId: '  bind-1  ',
        kind: 'set',
      })
    ).toBe('bind-1');
    expect(resolveObjectId({ attachmentObjectId: '  obj-1  ' })).toBe('obj-1');
    expect(resolveObjectId({ objectId: '  alias  ' })).toBe('alias');
    expect(resolveFileName('  report.pdf  ')).toBe('report.pdf');
    expect(resolveFileName({ fileName: '  a.bin  ' })).toBe('a.bin');

    expect(resolveDownloadUrl({ downloadUrl: '  /dl  ', kind: 'set' })).toBe('/dl');
    expect(resolveDownloadUrl({ url: '  /via-url  ', kind: 'set' })).toBe('/via-url');
    expect(resolveDownloadUrl({ previewUrl: '  /via-preview  ', kind: 'set' })).toBe('/via-preview');
    expect(resolveDownloadUrl({ thumbnailUrl: '  /via-thumb  ', kind: 'set' })).toBe('/via-thumb');
    expect(resolveDownloadUrl({ descriptor: { downloadUrl: '  /desc-dl  ' }, kind: 'set' })).toBe('/desc-dl');
    expect(resolveDownloadUrl({ descriptor: { previewUrl: '  /desc-pv  ' }, kind: 'set' })).toBe('/desc-pv');

    expect(resolvePreviewUrl({ previewUrl: '  /pv  ', kind: 'set' })).toBe('/pv');
    expect(resolvePreviewUrl({ downloadUrl: '  /dl-only  ', kind: 'set' })).toBe('/dl-only');
  });

  test('hasAttachment respects clear/noop and string/object presence', () => {
    expect(hasAttachment(null)).toBe(false);
    expect(hasAttachment('  ')).toBe(false);
    expect(hasAttachment('name.bin')).toBe(true);
    expect(hasAttachment({ kind: 'clear' })).toBe(false);
    expect(hasAttachment({ kind: 'noop' })).toBe(false);
    expect(hasAttachment({ kind: 'set', fileName: 'a.bin' })).toBe(true);
    expect(hasAttachment({ kind: 'set', previewUrl: '/p.png' })).toBe(true);
    expect(hasAttachment(1)).toBe(true);
  });

  test('formatAttachmentSize and meta text', () => {
    expect(formatAttachmentSize(undefined)).toBeUndefined();
    expect(formatAttachmentSize(100)).toBe('100 B');
    expect(formatAttachmentSize(5 * 1024 * 1024)).toMatch(/MB/);
    expect(formatAttachmentSize(2 * 1024 * 1024 * 1024)).toMatch(/GB/);
    expect(
      toAttachmentMetaText({
        mimeType: 'application/pdf',
        sizeBytes: 1536,
        kind: 'set',
      })
    ).toMatch(/application\/pdf/);
    expect(toAttachmentMetaText({ kind: 'set' })).toBeUndefined();
    expect(
      toAttachmentCompactMetaText({
        mimeType: 'application/pdf',
        sizeBytes: 1536,
        kind: 'set',
      })
    ).toMatch(/KB/);
    expect(
      toAttachmentCompactMetaText({
        mimeType: 'application/pdf',
        sizeBytes: 1536,
        kind: 'set',
      })
    ).not.toMatch(/application\/pdf/);
    expect(
      toAttachmentTooltipText({
        mimeType: 'application/pdf',
        sizeBytes: 1536,
        kind: 'set',
      })
    ).toMatch(/application\/pdf/);
  });

  test('isAcceptAllowed enforces tokens', () => {
    const pdf = new File([new Uint8Array([1])], 'ok.pdf', { type: 'application/pdf' });
    const txt = new File([new Uint8Array([1])], 'no.txt', { type: 'text/plain' });
    expect(isAcceptAllowed(pdf, '')).toBe(true);
    expect(isAcceptAllowed(pdf, '.pdf,application/pdf')).toBe(true);
    expect(isAcceptAllowed(txt, '.pdf,application/pdf')).toBe(false);
    expect(isAcceptAllowed(new File([new Uint8Array([1])], 'a.PNG', { type: '' }), '*.png')).toBe(true);
    expect(isAcceptAllowed(new File([new Uint8Array([1])], 'a.bin', { type: 'image/png' }), 'image/*')).toBe(true);
  });

  test('toAttachmentDisplayText uses empty-object label', () => {
    expect(toAttachmentDisplayText(null, '[binary]')).toBe('');
    expect(toAttachmentDisplayText({ fileName: 'n.bin' }, '[binary]')).toBe('n.bin');
    expect(toAttachmentDisplayText({ kind: 'set' }, '[binary]')).toBe('[binary]');
    expect(toAttachmentDisplayText(7, '[image]')).toBe('7');
  });

  test('resolveBinaryMediaIconKind and hasPendingLocalFile', () => {
    expect(resolveBinaryMediaIconKind({ mimeType: 'image/png', kind: 'set' })).toBe('image');
    expect(resolveBinaryMediaIconKind({ fileName: 'a.PDF', kind: 'set' })).toBe('pdf');
    expect(resolveBinaryMediaIconKind({ fileName: 'sheet.xlsx', kind: 'set' })).toBe('spreadsheet');
    expect(resolveBinaryMediaIconKind({ mimeType: 'text/csv', kind: 'set' })).toBe('spreadsheet');
    expect(resolveBinaryMediaIconKind({ fileName: 'pack.zip', kind: 'set' })).toBe('archive');
    expect(resolveBinaryMediaIconKind({ mimeType: 'video/mp4', kind: 'set' })).toBe('video');
    expect(resolveBinaryMediaIconKind({ mimeType: 'audio/mpeg', kind: 'set' })).toBe('audio');
    expect(resolveBinaryMediaIconKind({ fileName: 'app.ts', kind: 'set' })).toBe('code');
    expect(resolveBinaryMediaIconKind({ fileName: 'a.bin', kind: 'set' })).toBe('file');
    expect(hasPendingLocalFile({ kind: 'set', fileName: 'x' })).toBe(false);
    expect(hasPendingLocalFile({ kind: 'set', file: new File([], 'x.bin') })).toBe(true);
  });

  test('resolveImageAvatarBoxStyle defaults and coerces lengths', () => {
    const fallback = `${DEFAULT_IMAGE_AVATAR_SIZE_PX}px`;
    expect(resolveImageAvatarBoxStyle()).toEqual({ width: fallback, height: fallback });
    expect(resolveImageAvatarBoxStyle(160)).toEqual({ width: '160px', height: '160px' });
    expect(resolveImageAvatarBoxStyle(undefined, 90)).toEqual({ width: '90px', height: '90px' });
    expect(resolveImageAvatarBoxStyle(120, 80)).toEqual({ width: '120px', height: '80px' });
    expect(resolveImageAvatarBoxStyle('10rem', '8rem')).toEqual({ width: '10rem', height: '8rem' });
    expect(resolveImageAvatarBoxStyle('96')).toEqual({ width: '96px', height: '96px' });
  });

  test('resolveImageAvatarContainBoxStyle preserves natural ratio within max box', () => {
    expect(resolveImageAvatarContainBoxStyle({})).toEqual({
      width: `${DEFAULT_IMAGE_AVATAR_SIZE_PX}px`,
      height: `${DEFAULT_IMAGE_AVATAR_SIZE_PX}px`,
    });
    expect(
      resolveImageAvatarContainBoxStyle({
        width: 112,
        height: 112,
        naturalWidth: 1600,
        naturalHeight: 900,
      })
    ).toEqual({ width: '112px', height: '63px' });
    expect(resolveImageAvatarRadiusClass('xl')).toBe('rounded-xl');
    expect(resolveImageAvatarRadiusClass('md')).toBe('rounded-md');
    expect(resolveImageAvatarRadiusClass(undefined)).toBe('rounded-md');
  });
});
