// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { downloadExportCsvBytes, suggestExportFileName } from './download_csv';

test('suggestExportFileName: uses model segment for file name', () => {
  expect(suggestExportFileName('partner.Partner')).toBe('Partner.csv');
  expect(suggestExportFileName('')).toBe('export.csv');
  expect(suggestExportFileName('single')).toBe('single.csv');
  expect(suggestExportFileName('partner.')).toBe('export.csv');
  expect(suggestExportFileName(undefined as unknown as string)).toBe('export.csv');
});

function installDownloadStubs(anchor: { click: () => void; remove: () => void; download?: string; href?: string; rel?: string }) {
  const createCalls: unknown[] = [];
  const revokeCalls: unknown[] = [];
  const appendCalls: unknown[] = [];

  const prevURL = (globalThis as any).URL;
  const prevDocument = (globalThis as any).document;
  const prevBlob = (globalThis as any).Blob;

  (globalThis as any).Blob = class Blob {
    constructor(_parts?: unknown[], _opts?: unknown) {}
  };
  (globalThis as any).URL = {
    createObjectURL(value: unknown) {
      createCalls.push(value);
      return 'blob:export';
    },
    revokeObjectURL(value: unknown) {
      revokeCalls.push(value);
    },
  };
  const body = {
    appendChild(node: unknown) {
      appendCalls.push(node);
      return node;
    },
  };
  (globalThis as any).document = {
    createElement() {
      return anchor;
    },
    body,
  };

  return {
    createCalls,
    revokeCalls,
    appendCalls,
    restore() {
      if (prevURL === undefined) Reflect.deleteProperty(globalThis, 'URL');
      else (globalThis as any).URL = prevURL;
      if (prevDocument === undefined) Reflect.deleteProperty(globalThis, 'document');
      else (globalThis as any).document = prevDocument;
      if (prevBlob === undefined) Reflect.deleteProperty(globalThis, 'Blob');
      else (globalThis as any).Blob = prevBlob;
    },
  };
}

test('downloadExportCsvBytes: creates a blob download link and revokes object url', () => {
  let clicked = 0;
  let removed = 0;
  const stubs = installDownloadStubs({
    click: () => {
      clicked += 1;
    },
    remove: () => {
      removed += 1;
    },
  });
  try {
    downloadExportCsvBytes(new Uint8Array([97, 98]), 'Partner.csv');
    expect(stubs.createCalls.length).toBe(1);
    expect(stubs.appendCalls.length).toBe(1);
    expect(clicked).toBe(1);
    expect(removed).toBe(1);
    expect(stubs.revokeCalls).toEqual(['blob:export']);
  } finally {
    stubs.restore();
  }
});

test('downloadExportCsvBytes: accepts ArrayBuffer input', () => {
  let clicked = 0;
  const stubs = installDownloadStubs({
    click: () => {
      clicked += 1;
    },
    remove: () => {},
    download: '',
  });
  try {
    downloadExportCsvBytes(new ArrayBuffer(2), '');
    expect(clicked).toBe(1);
  } finally {
    stubs.restore();
  }
});

test('downloadExportCsvBytes: defaults download file name when empty', () => {
  const anchor = { click: () => {}, remove: () => {}, download: '', href: '', rel: '' };
  const stubs = installDownloadStubs(anchor);
  try {
    downloadExportCsvBytes(new Uint8Array([1]), '');
    expect(anchor.download).toBe('export.csv');
    expect(anchor.rel).toBe('noopener');
  } finally {
    stubs.restore();
  }
});
