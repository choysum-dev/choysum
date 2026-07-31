// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { validateImageFieldFile } from './imageFieldLimits';

function makeFile(size: number, type = 'image/png'): File {
  const buffer = new Uint8Array(size);
  return new File([buffer], 'photo.png', { type });
}

describe('imageFieldLimits (PR-P2-F3)', () => {
  const limits = {
    maxUploadBytes: 1024,
    maxWidth: 100,
    maxHeight: 80,
  };

  it('rejects oversized file', async () => {
    const result = await validateImageFieldFile(makeFile(2048), limits, async () => ({ width: 50, height: 50 }));
    expect(result).toEqual({ ok: false, reason: 'fileTooLarge', detail: '1 KB' });
  });

  it('rejects oversized width', async () => {
    const result = await validateImageFieldFile(makeFile(512), limits, async () => ({ width: 200, height: 50 }));
    expect(result).toEqual({ ok: false, reason: 'widthTooLarge', detail: '100' });
  });

  it('rejects oversized height', async () => {
    const result = await validateImageFieldFile(makeFile(512), limits, async () => ({ width: 50, height: 120 }));
    expect(result).toEqual({ ok: false, reason: 'heightTooLarge', detail: '80' });
  });

  it('accepts file within limits', async () => {
    const readDimensions = vi.fn(async () => ({ width: 80, height: 60 }));
    const result = await validateImageFieldFile(makeFile(512), limits, readDimensions);
    expect(result).toEqual({ ok: true });
    expect(readDimensions).toHaveBeenCalled();
  });
});
