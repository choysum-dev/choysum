// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { parseChangedSelectors } from './selectors';

test('parseChangedSelectors keeps non-selector paths and ignores empty entries', () => {
  const result = parseChangedSelectors(['Name', '', 'Partner.Email']);

  expect(Array.from(result.normalizedSeeds).sort()).toEqual(['Name', 'Partner.Email']);
  expect(Array.from(result.collectionRoots)).toEqual([]);
  expect(Array.from(result.fieldSignals.keys())).toEqual([]);
  expect(Array.from(result.selectors.keys())).toEqual([]);
});

test('parseChangedSelectors collects collection roots, field signals and selector unions', () => {
  const result = parseChangedSelectors([
    'Lines[2].Qty',
    'Lines[5].Qty',
    'Lines(id=L-1).Qty',
    'Lines(id=L-1).Batches[0].Price',
    'Lines(id=L-1).Batches(id=B-2)',
  ]);

  expect(result.normalizedSeeds.has('Lines')).toBe(true);
  expect(result.normalizedSeeds.has('Lines.Qty')).toBe(true);
  expect(result.normalizedSeeds.has('Lines.Batches')).toBe(true);
  expect(result.normalizedSeeds.has('Lines.Batches.Price')).toBe(true);

  expect(Array.from(result.collectionRoots).sort()).toEqual(['Lines', 'Lines.Batches']);

  const lineSignals = Array.from(result.fieldSignals.get('Lines') || []).sort();
  const batchSignals = Array.from(result.fieldSignals.get('Lines.Batches') || []).sort();
  expect(lineSignals).toEqual(['Qty']);
  expect(batchSignals).toEqual(['Price']);

  const lineSelector = result.selectors.get('Lines');
  const batchesSelector = result.selectors.get('Lines.Batches');

  expect(lineSelector?.kind).toBe('all');
  expect(batchesSelector?.kind).toBe('all');
});

test('parseChangedSelectors merges same-kind selectors without escalating to all', () => {
  const result = parseChangedSelectors(['Lines[1].Qty', 'Lines[3].Qty', 'Lines(id=A).Product(id=P1).Name', 'Lines(id=B).Product(id=P2).Code']);

  const lineSelector = result.selectors.get('Lines');
  const productSelector = result.selectors.get('Lines.Product');

  expect(lineSelector?.kind).toBe('all');
  expect(productSelector?.kind).toBe('id');
  expect(Array.from((productSelector as any).ids).sort()).toEqual(['P1', 'P2']);

  const productSignals = Array.from(result.fieldSignals.get('Lines.Product') || []).sort();
  expect(productSignals).toEqual(['Code', 'Name']);
});

test('parseChangedSelectors preserves malformed selector-like segment as normalized seed', () => {
  const result = parseChangedSelectors(['Lines[abc].Qty']);

  expect(Array.from(result.normalizedSeeds)).toEqual(['Lines[abc].Qty']);
  expect(Array.from(result.collectionRoots)).toEqual([]);
  expect(Array.from(result.selectors.keys())).toEqual([]);
});

test('parseChangedSelectors keeps pos selector union when root only uses positional selectors', () => {
  const result = parseChangedSelectors(['Lines[1].Qty', 'Lines[3].Code']);

  const lineSelector = result.selectors.get('Lines') as any;
  expect(lineSelector?.kind).toBe('pos');
  expect(Array.from(lineSelector?.positions || []).sort()).toEqual([1, 3]);

  const lineSignals = Array.from(result.fieldSignals.get('Lines') || []).sort();
  expect(lineSignals).toEqual(['Code', 'Qty']);
});

test('parseChangedSelectors tolerates nullish changed input and keeps empty result maps', () => {
  const result = parseChangedSelectors(undefined as any);

  expect(Array.from(result.normalizedSeeds)).toEqual([]);
  expect(Array.from(result.collectionRoots)).toEqual([]);
  expect(Array.from(result.fieldSignals.keys())).toEqual([]);
  expect(Array.from(result.selectors.keys())).toEqual([]);
});
