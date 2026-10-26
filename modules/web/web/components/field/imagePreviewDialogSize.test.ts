// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveImagePreviewDialogSize } from './imagePreviewDialogSize';

describe('resolveImagePreviewDialogSize', () => {
  test('preserves landscape aspect ratio within viewport caps', () => {
    const size = resolveImagePreviewDialogSize(5120, 2880, 1280, 800);
    const ratio = size.width / size.height;
    const expected = 5120 / 2880;
    expect(Math.abs(ratio - expected)).toBeLessThan(0.02);
    expect(size.width <= Math.floor(1280 * 0.92)).toBe(true);
    expect(size.height <= Math.floor(800 * 0.85)).toBe(true);
  });

  test('does not upscale small images', () => {
    expect(resolveImagePreviewDialogSize(320, 180, 1280, 800)).toEqual({ width: 320, height: 180 });
  });

  test('rejects non-positive natural sizes', () => {
    expect(resolveImagePreviewDialogSize(0, 100, 1280, 800)).toEqual({ width: 0, height: 0 });
  });
});
