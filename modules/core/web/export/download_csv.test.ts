// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from 'vitest';
import { downloadExportCsvBytes, suggestExportFileName } from './download_csv';

describe('suggestExportFileName', () => {
  it('uses model segment for file name', () => {
    expect(suggestExportFileName('partner.Partner')).toBe('Partner.csv');
    expect(suggestExportFileName('')).toBe('export.csv');
    expect(suggestExportFileName('single')).toBe('single.csv');
  });
});

describe('downloadExportCsvBytes', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('creates a blob download link and revokes object url', () => {
    const click = vi.fn();
    const remove = vi.fn();
    const appendChild = vi.fn();
    const createObjectURL = vi.fn(() => 'blob:export');
    const revokeObjectURL = vi.fn();
    vi.spyOn(URL, 'createObjectURL').mockImplementation(createObjectURL);
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(revokeObjectURL);
    vi.spyOn(document, 'createElement').mockReturnValue({
      click,
      remove,
    } as unknown as HTMLAnchorElement);
    vi.spyOn(document.body, 'appendChild').mockImplementation(appendChild);

    downloadExportCsvBytes(new Uint8Array([97, 98]), 'Partner.csv');

    expect(createObjectURL).toHaveBeenCalled();
    expect(appendChild).toHaveBeenCalled();
    expect(click).toHaveBeenCalled();
    expect(remove).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:export');
  });

  it('accepts ArrayBuffer input', () => {
    const click = vi.fn();
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:export');
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
    vi.spyOn(document, 'createElement').mockReturnValue({ click, remove: vi.fn(), download: '' } as unknown as HTMLAnchorElement);
    vi.spyOn(document.body, 'appendChild').mockImplementation(() => undefined as unknown as Node);

    downloadExportCsvBytes(new ArrayBuffer(2), '');
    expect(click).toHaveBeenCalled();
  });

  it('defaults download file name when empty', () => {
    const anchor = { click: vi.fn(), remove: vi.fn(), download: '', href: '', rel: '' };
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:export');
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
    vi.spyOn(document, 'createElement').mockReturnValue(anchor as unknown as HTMLAnchorElement);
    vi.spyOn(document.body, 'appendChild').mockImplementation(() => undefined as unknown as Node);
    downloadExportCsvBytes(new Uint8Array([1]), '');
    expect(anchor.download).toBe('export.csv');
  });
});
