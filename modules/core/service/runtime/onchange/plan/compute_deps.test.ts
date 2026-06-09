// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as computeDepsApi from './compute_deps';

test('plan compute deps module export surface stays limited to dependency extractors', () => {
  expect(Object.keys(computeDepsApi).sort()).toEqual(['extractComputeCollectionPathDeps', 'extractComputePathDeps']);
});

test('extractComputePathDeps returns empty map when compute graph is missing', () => {
  const meta = {} as any;
  const out = computeDepsApi.extractComputePathDeps(meta, new Set(['Name']));
  expect(out.size).toBe(0);
});

test('extractComputePathDeps collects path deps and deduplicates chains', () => {
  const parsedDeps = new Map<string, any[]>([
    [
      'DisplayName',
      [
        { kind: 'path', root: 'User', chain: ['Name'] },
        { kind: 'path', root: 'User', chain: ['Name'] },
        { kind: 'path', root: 'User', chain: ['Email'] },
        { kind: 'collection-path', root: 'Ignored', chain: ['X'] },
      ],
    ],
  ]);

  const meta = {
    computeGraph: {
      parsedDeps,
    },
  } as any;

  const out = computeDepsApi.extractComputePathDeps(meta, new Set(['DisplayName', 'MissingField']));
  expect(out.get('User')).toEqual([['Name'], ['Email']]);
  expect(out.has('Ignored')).toBe(false);
});

test('extractComputeCollectionPathDeps returns empty map when compute graph is missing', () => {
  const meta = {} as any;
  const out = computeDepsApi.extractComputeCollectionPathDeps(meta, new Set(['Total']));
  expect(out.size).toBe(0);
});

test('extractComputeCollectionPathDeps merges per field and deduplicates chains', () => {
  const computeCollectionPathDeps = new Map<string, any[]>([
    [
      'TotalAmount',
      [
        { collection: 'Lines', chain: ['Amount'] },
        { collection: 'Lines', chain: ['Amount'] },
        { collection: 'Lines', chain: ['Tax'] },
      ],
    ],
    ['TotalQty', [{ collection: 'Lines', chain: ['Qty'] }]],
  ]);

  const meta = {
    computeGraph: {
      computeCollectionPathDeps,
    },
  } as any;

  const out = computeDepsApi.extractComputeCollectionPathDeps(meta, new Set(['TotalAmount', 'TotalQty', 'MissingField']));
  expect(out.get('Lines')).toEqual([['Amount'], ['Tax'], ['Qty']]);
});

test('extractComputeCollectionPathDeps returns empty map when graph exists but collection deps map is absent', () => {
  const meta = {
    computeGraph: {
      parsedDeps: new Map(),
    },
  } as any;

  const out = computeDepsApi.extractComputeCollectionPathDeps(meta, new Set(['Total']));
  expect(out.size).toBe(0);
});

test('extractors dedupe empty chains consistently across fields', () => {
  const meta = {
    computeGraph: {
      parsedDeps: new Map<string, any[]>([
        [
          'ComputeA',
          [
            { kind: 'path', root: 'OwnerId', chain: [] },
            { kind: 'path', root: 'OwnerId', chain: [] },
          ],
        ],
      ]),
      computeCollectionPathDeps: new Map<string, any[]>([
        [
          'ComputeB',
          [
            { collection: 'Lines', chain: [] },
            { collection: 'Lines', chain: [] },
          ],
        ],
      ]),
    },
  } as any;

  const pathDeps = computeDepsApi.extractComputePathDeps(meta, new Set(['ComputeA']));
  const collectionDeps = computeDepsApi.extractComputeCollectionPathDeps(meta, new Set(['ComputeB']));

  expect(pathDeps.get('OwnerId')).toEqual([[]]);
  expect(collectionDeps.get('Lines')).toEqual([[]]);
});

test('extractors initialize and append chains for multiple roots across recompute fields', () => {
  const meta = {
    computeGraph: {
      parsedDeps: new Map<string, any[]>([
        ['ComputeName', [{ kind: 'path', root: 'OwnerId', chain: ['Name'] }]],
        ['ComputeCode', [{ kind: 'path', root: 'DepartmentId', chain: ['Code'] }]],
      ]),
      computeCollectionPathDeps: new Map<string, any[]>([
        ['ComputeLines', [{ collection: 'Lines', chain: ['Amount'] }]],
        ['ComputeTaxes', [{ collection: 'Taxes', chain: ['Value'] }]],
      ]),
    },
  } as any;

  const pathDeps = computeDepsApi.extractComputePathDeps(meta, new Set(['ComputeName', 'ComputeCode']));
  const collectionDeps = computeDepsApi.extractComputeCollectionPathDeps(meta, new Set(['ComputeLines', 'ComputeTaxes']));

  expect(pathDeps.get('OwnerId')).toEqual([['Name']]);
  expect(pathDeps.get('DepartmentId')).toEqual([['Code']]);
  expect(collectionDeps.get('Lines')).toEqual([['Amount']]);
  expect(collectionDeps.get('Taxes')).toEqual([['Value']]);
});

test('extractors merge same root deps across different recompute fields', () => {
  const meta = {
    computeGraph: {
      parsedDeps: new Map<string, any[]>([
        ['ComputeA', [{ kind: 'path', root: 'OwnerId', chain: ['Name'] }]],
        ['ComputeB', [{ kind: 'path', root: 'OwnerId', chain: ['Code'] }]],
      ]),
      computeCollectionPathDeps: new Map<string, any[]>([
        ['ComputeC', [{ collection: 'Lines', chain: ['Qty'] }]],
        ['ComputeD', [{ collection: 'Lines', chain: ['Price'] }]],
      ]),
    },
  } as any;

  const pathDeps = computeDepsApi.extractComputePathDeps(meta, new Set(['ComputeA', 'ComputeB']));
  const collectionDeps = computeDepsApi.extractComputeCollectionPathDeps(meta, new Set(['ComputeC', 'ComputeD']));

  expect(pathDeps.get('OwnerId')).toEqual([['Name'], ['Code']]);
  expect(collectionDeps.get('Lines')).toEqual([['Qty'], ['Price']]);
});
