// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure image limit helpers. Mount upload/display wiring lives in OImageField.test.ts.
 */

import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from '@/core/service/orm/upload_limits';
import {
  formatImageByteLimit,
  imageFieldLimitErrorMessage,
  readImageNaturalDimensions,
  reportImageFieldValidation,
  resolveImageFieldLimits,
  resolveImageFieldLimitsFromSources,
  validateImageFieldFile,
} from './imageFieldLimits';

function makeFile(size: number, type = 'image/png'): File {
  return new File([new Uint8Array(size)], 'photo.png', { type });
}

describe('imageFieldLimits', () => {
  test('resolveImageFieldLimits caps bytes and ignores invalid values', () => {
    expect(resolveImageFieldLimits(null)).toEqual({});
    expect(resolveImageFieldLimits({ maxUploadBytes: -1, maxWidth: 0 })).toEqual({});
    expect(resolveImageFieldLimits({ maxUploadBytes: 100, maxWidth: 40, maxHeight: 30 })).toEqual({
      maxUploadBytes: 100,
      maxWidth: 40,
      maxHeight: 30,
    });
    expect(resolveImageFieldLimits({ maxUploadBytes: DEFAULT_GLOBAL_MAX_UPLOAD_BYTES + 10 }).maxUploadBytes).toBe(
      DEFAULT_GLOBAL_MAX_UPLOAD_BYTES
    );
  });

  test('formatImageByteLimit covers B/KB/MB', () => {
    expect(formatImageByteLimit(512)).toBe('512 B');
    expect(formatImageByteLimit(2048)).toBe('2 KB');
    expect(formatImageByteLimit(2 * 1024 * 1024)).toBe('2 MB');
  });

  test('validateImageFieldFile rejects size / width / height and accepts valid files', async () => {
    expect(await validateImageFieldFile(makeFile(200), { maxUploadBytes: 100 })).toMatchObject({
      ok: false,
      reason: 'fileTooLarge',
    });
    expect(
      await validateImageFieldFile(makeFile(20), { maxWidth: 40 }, async () => ({ width: 50, height: 10 }))
    ).toMatchObject({ ok: false, reason: 'widthTooLarge' });
    expect(
      await validateImageFieldFile(makeFile(20), { maxHeight: 30 }, async () => ({ width: 10, height: 40 }))
    ).toMatchObject({ ok: false, reason: 'heightTooLarge' });
    expect(
      await validateImageFieldFile(makeFile(20), { maxUploadBytes: 100, maxWidth: 40, maxHeight: 30 }, async () => ({
        width: 20,
        height: 20,
      }))
    ).toEqual({ ok: true });
  });

  test('skips dimension checks when no dimension limits or probe missing', async () => {
    expect(await validateImageFieldFile(makeFile(8), { maxUploadBytes: 100 })).toEqual({ ok: true });
    expect(await validateImageFieldFile(makeFile(8), { maxWidth: 40 }, async () => undefined)).toEqual({
      ok: true,
    });
  });

  test('resolveImageFieldLimitsFromSources prefers store meta then binding meta', () => {
    expect(
      resolveImageFieldLimitsFromSources({
        bindingProp: 'Photo',
        bindingStore: {
          getFieldMeta: (name: string) => (name === 'Photo' ? { maxUploadBytes: 50, maxWidth: 10 } : undefined),
        },
        bindingMeta: { maxUploadBytes: 999, maxHeight: 99 },
      })
    ).toEqual({ maxUploadBytes: 50, maxWidth: 10 });

    expect(
      resolveImageFieldLimitsFromSources({
        propsProp: 'Photo',
        bindingMeta: { maxUploadBytes: 80 },
      })
    ).toEqual({ maxUploadBytes: 80 });
  });

  test('imageFieldLimitErrorMessage covers all reasons', () => {
    expect(imageFieldLimitErrorMessage({ ok: false, reason: 'fileTooLarge', detail: '1 KB' })).toMatch(/1 KB/);
    expect(imageFieldLimitErrorMessage({ ok: false, reason: 'widthTooLarge', detail: '40' })).toMatch(/40/);
    expect(imageFieldLimitErrorMessage({ ok: false, reason: 'heightTooLarge', detail: '30' })).toMatch(/30/);
  });

  test('reportImageFieldValidation invokes onError on failure and returns true on success', async () => {
    const errors: string[] = [];
    expect(await reportImageFieldValidation(makeFile(200), { maxUploadBytes: 100 }, msg => errors.push(msg))).toBe(
      false
    );
    expect(errors.length).toBe(1);

    expect(await reportImageFieldValidation(makeFile(20), { maxUploadBytes: 100 }, msg => errors.push(msg))).toBe(
      true
    );
    expect(errors.length).toBe(1);
  });

  test('readImageNaturalDimensions uses createImageBitmap when available', async () => {
    const original = (globalThis as any).createImageBitmap;
    const closeCalls: number[] = [];
    (globalThis as any).createImageBitmap = async () => ({
      width: 11,
      height: 22,
      close: () => {
        closeCalls.push(1);
      },
    });
    try {
      expect(await readImageNaturalDimensions(makeFile(8))).toEqual({ width: 11, height: 22 });
      expect(closeCalls.length).toBe(1);

      (globalThis as any).createImageBitmap = async () => {
        throw new Error('bad image');
      };
      expect(await readImageNaturalDimensions(makeFile(8))).toBeUndefined();
    } finally {
      if (original) (globalThis as any).createImageBitmap = original;
      else delete (globalThis as any).createImageBitmap;
    }
  });
});
