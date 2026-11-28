// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { estimateListHeaderMinWidth } from './listColumnSizeDefaults';
import {
  createListCanvasTextMeasure,
  formatListSampleCellText,
  getDefaultListTextMeasure,
  listTextWidthPx,
  readListSampleCellValue,
  resetDefaultListTextMeasureForTests,
  sampleSoftColumnMinWidths,
  type ListTextMeasureFn,
} from './listColumnTextMeasure';

const measure10: ListTextMeasureFn = (text) => String(text).length * 10;

test('estimateListHeaderMinWidth uses font measure when provided', () => {
  // 8 chars * 10 + 16 pad = 96 → snap 100; still capped by typeMin+80.
  expect(estimateListHeaderMinWidth('ABCDEFGH', 80, { measure: measure10 })).toBe(100);
  expect(estimateListHeaderMinWidth('En', 80, { measure: measure10 })).toBe(80);
  const long = estimateListHeaderMinWidth('非常长的状态标签名XYZ', 80, { measure: measure10 });
  expect(long).toBeGreaterThan(80);
  expect(long % 20).toBe(0);
  expect(long).toBeLessThanOrEqual(160);
});

test('listTextWidthPx falls back to char budget without measure', () => {
  expect(listTextWidthPx('ABCD', null)).toBe(48);
  expect(listTextWidthPx('', null)).toBe(0);
});

test('formatListSampleCellText reads scalars and M2O-like objects', () => {
  expect(formatListSampleCellText(null)).toBe('');
  expect(formatListSampleCellText('  hi  ')).toBe('hi');
  expect(formatListSampleCellText(12)).toBe('12');
  expect(formatListSampleCellText({ DisplayName: 'Acme' })).toBe('Acme');
  expect(formatListSampleCellText({ Name: 'Beta' })).toBe('Beta');
  expect(formatListSampleCellText({ other: 1 })).toBe('');
});

test('createListCanvasTextMeasure uses canvas measureText when available', () => {
  resetDefaultListTextMeasureForTests();
  const original = document.createElement.bind(document);
  document.createElement = ((tag: string, ...rest: unknown[]) => {
    if (String(tag).toLowerCase() === 'canvas') {
      return {
        getContext: () => ({
          font: '',
          measureText: (text: string) => ({ width: String(text).length * 7 }),
        }),
      } as unknown as HTMLCanvasElement;
    }
    return original(tag as any, ...(rest as []));
  }) as typeof document.createElement;
  try {
    const measure = createListCanvasTextMeasure('13px sans-serif');
    expect(measure).toBeTruthy();
    expect(measure!('ABCD')).toBe(28);
    expect(measure!('')).toBe(0);
    expect(getDefaultListTextMeasure()).toBeTruthy();
    expect(listTextWidthPx('XY', undefined)).toBe(14);
  } finally {
    document.createElement = original;
    resetDefaultListTextMeasureForTests();
  }
});

test('createListCanvasTextMeasure returns null when canvas context is missing', () => {
  const original = document.createElement.bind(document);
  document.createElement = ((tag: string, ...rest: unknown[]) => {
    if (String(tag).toLowerCase() === 'canvas') {
      return { getContext: () => null } as unknown as HTMLCanvasElement;
    }
    return original(tag as any, ...(rest as []));
  }) as typeof document.createElement;
  try {
    expect(createListCanvasTextMeasure()).toBeNull();
  } finally {
    document.createElement = original;
  }
});

test('createListCanvasTextMeasure returns null when canvas setup throws', () => {
  const original = document.createElement.bind(document);
  document.createElement = ((tag: string, ...rest: unknown[]) => {
    if (String(tag).toLowerCase() === 'canvas') {
      throw new Error('canvas unavailable');
    }
    return original(tag as any, ...(rest as []));
  }) as typeof document.createElement;
  try {
    expect(createListCanvasTextMeasure()).toBeNull();
  } finally {
    document.createElement = original;
  }
});

test('createListCanvasTextMeasure falls back when measureText width is non-positive', () => {
  const original = document.createElement.bind(document);
  document.createElement = ((tag: string, ...rest: unknown[]) => {
    if (String(tag).toLowerCase() === 'canvas') {
      return {
        getContext: () => ({
          font: '',
          measureText: () => ({ width: 0 }),
        }),
      } as unknown as HTMLCanvasElement;
    }
    return original(tag as any, ...(rest as []));
  }) as typeof document.createElement;
  try {
    const measure = createListCanvasTextMeasure();
    expect(measure!('ABCD')).toBe(48);
  } finally {
    document.createElement = original;
  }
});

test('readListSampleCellValue prefers literal keys then dotted paths', () => {
  expect(readListSampleCellValue({ 'Partner.Name': 'Flat' }, 'Partner.Name')).toBe('Flat');
  expect(
    readListSampleCellValue({ Partner: { Name: 'NestedAcmeCorpXXXXXXXX' } }, 'Partner.Name'),
  ).toBe('NestedAcmeCorpXXXXXXXX');
  expect(readListSampleCellValue({ Title: 'plain' }, 'Title')).toBe('plain');
});

test('sampleSoftColumnMinWidths walks nested dotted dataKeys', () => {
  const raised = sampleSoftColumnMinWidths(
    [
      {
        key: 'Partner.Name',
        dataKey: 'Partner.Name',
        title: 'Partner',
        fieldType: 'varchar',
        minWidth: 140,
        maxWidth: 350,
      },
    ],
    [{ Partner: { Name: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXX' } }],
    { measure: measure10 },
  );
  expect(raised.get('Partner.Name')).toBeGreaterThan(140);
  expect(raised.get('Partner.Name')).toBeLessThanOrEqual(350);
});

test('sampleSoftColumnMinWidths raises soft mins from first-screen cells', () => {
  const raised = sampleSoftColumnMinWidths(
    [
      {
        key: 'Title',
        dataKey: 'Title',
        title: '标题',
        fieldType: 'varchar',
        minWidth: 140,
        maxWidth: 350,
      },
      {
        key: 'Code',
        dataKey: 'Code',
        title: '编码',
        fieldType: 'varchar',
        minWidth: 100,
        maxWidth: 160,
      },
      {
        key: 'Qty',
        dataKey: 'Qty',
        title: '数量',
        fieldType: 'int',
        minWidth: 80,
      },
    ],
    [
      { Title: 'short', Code: 'A' },
      { Title: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXX', Code: 'SKU-LONG-CODE-999' },
    ],
    { measure: measure10 },
  );
  expect(raised.get('Title')).toBeGreaterThan(140);
  expect(raised.get('Title')).toBeLessThanOrEqual(350);
  expect(raised.get('Code')).toBeGreaterThan(100);
  expect(raised.get('Code')).toBeLessThanOrEqual(160);
  expect(raised.has('Qty')).toBe(false);
});

test('sampleSoftColumnMinWidths skips explicit width and author escape', () => {
  const raised = sampleSoftColumnMinWidths(
    [
      {
        key: 'Title',
        dataKey: 'Title',
        fieldType: 'varchar',
        minWidth: 140,
        maxWidth: 350,
        explicitWidth: 200,
      },
      {
        key: 'Name',
        dataKey: 'Name',
        fieldType: 'varchar',
        minWidth: 140,
        maxWidth: 350,
        skipSample: true,
      },
    ],
    [{ Title: 'XXXXXXXXXXXXXXXXXXXXXXXXXXXX', Name: 'YYYYYYYYYYYYYYYYYYYYYYYY' }],
    { measure: measure10 },
  );
  expect(raised.size).toBe(0);
});
