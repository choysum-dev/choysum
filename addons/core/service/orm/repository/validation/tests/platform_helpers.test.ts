// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { recordRepositoryPlatformCreateWhitelistAudit, resolveRepositoryPlatformCreateWriteWhitelist, resolveRepositoryPlatformRejectUnknownFields } from '..';

test('validation platform helper resolves rejectUnknownFields from validation and root context', () => {
  expect(resolveRepositoryPlatformRejectUnknownFields(undefined)).toBe(false);
  expect(resolveRepositoryPlatformRejectUnknownFields({ platformRejectUnknownFields: true })).toBe(true);
  expect(resolveRepositoryPlatformRejectUnknownFields({ validation: { platformRejectUnknownFields: false }, platformRejectUnknownFields: true })).toBe(false);
});

test('validation platform helper records create whitelist audit in request context bucket', () => {
  const ctx: any = { requestId: 'req_1' };
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: 'demo.Model', modelName: '', name: '' } as any, ctx, 'create', [' Name ', '', 'Name', 'Code']);

  expect(ctx.__validationAudit?.version).toBe(1);
  const entries = ctx.__validationAudit?.platformCreateWhitelistHits || [];
  expect(entries.length).toBe(1);
  expect(entries[0].source).toBe('request_context');
  expect(entries[0].model).toBe('demo.Model');
  expect(entries[0].fields).toEqual(['Code', 'Name']);
  expect(entries[0].requestId).toBe('req_1');
});

test('validation platform helper uses global fallback bucket when context is not extensible', () => {
  const key = '__choysumValidationAudit';
  const root = globalThis as any;
  const previous = root[key];

  try {
    delete root[key];
    const frozen = Object.preventExtensions({ RequestId: 'req_2' }) as any;
    recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: '', modelName: '', name: '' } as any, frozen, 'create', ['A']);

    const entries = root[key]?.platformCreateWhitelistHits || [];
    expect(entries.length).toBe(1);
    expect(entries[0].source).toBe('global_fallback');
    expect(entries[0].model).toBe('unknown');
  } finally {
    if (previous === undefined) delete root[key];
    else root[key] = previous;
  }
});

test('validation platform helper skips audit on non-create mode or empty fields', () => {
  const ctx: any = {};
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: 'demo.Model' } as any, ctx, 'update', ['A']);
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: 'demo.Model' } as any, ctx, 'create', []);
  expect(ctx.__validationAudit).toBeUndefined();
});

test('validation platform helper resolves create whitelist by precedence and normalizes list', () => {
  const model = { fullModelName: 'demo.Model', modelName: '', name: '' } as any;

  expect(
    resolveRepositoryPlatformCreateWriteWhitelist(model, {
      validation: { platformCreateWriteWhitelistByModel: { 'demo.Model': [' Name ', '', 'Name', 'Code'] } },
      platformCreateWriteWhitelistByModel: { 'demo.Model': ['RootOnly'] },
      validation2: {},
    })
  ).toEqual(['Name', 'Code']);

  expect(
    resolveRepositoryPlatformCreateWriteWhitelist(model, {
      platformCreateWriteWhitelistByModel: { 'demo.Model': [' RootA ', 'RootA', 'RootB'] },
    })
  ).toEqual(['RootA', 'RootB']);

  expect(
    resolveRepositoryPlatformCreateWriteWhitelist(model, {
      validation: { platformCreateWriteWhitelist: [' V1 ', '', 'V1', 'V2'] },
      platformCreateWriteWhitelist: ['R1'],
    })
  ).toEqual(['V1', 'V2']);

  expect(resolveRepositoryPlatformCreateWriteWhitelist(model, { platformCreateWriteWhitelist: [' R1 ', '', 'R1', 'R2'] })).toEqual(['R1', 'R2']);
  expect(resolveRepositoryPlatformCreateWriteWhitelist(model, {})).toEqual([]);
});

test('validation platform helper resolves requestId fallback and model name fallback chain', () => {
  const ctx: any = { RequestId: 'req_2' };
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: '', modelName: 'ModelOnly', name: '' } as any, ctx, 'create', ['A']);
  const entries = ctx.__validationAudit?.platformCreateWhitelistHits || [];
  expect(entries.length).toBe(1);
  expect(entries[0].requestId).toBe('req_2');
  expect(entries[0].model).toBe('ModelOnly');

  const ctx2: any = {};
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: '', modelName: '', name: 'NameOnly' } as any, ctx2, 'create', ['B']);
  const entries2 = ctx2.__validationAudit?.platformCreateWhitelistHits || [];
  expect(entries2.length).toBe(1);
  expect(entries2[0].requestId).toBeUndefined();
  expect(entries2[0].model).toBe('NameOnly');
});

test('validation platform helper handles undefined context and empty model name branch', () => {
  expect(resolveRepositoryPlatformCreateWriteWhitelist({ fullModelName: '', modelName: '', name: '' } as any, undefined)).toEqual([]);
});

test('validation platform helper handles falsy primitive context and normalizes invalid request bucket', () => {
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: '', modelName: '', name: 'NameFallback' } as any, 0 as any, 'create', ['A']);

  const ctx: any = { __validationAudit: 1 };
  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: '', modelName: '', name: 'NameFallback' } as any, ctx, 'create', ['A']);

  const entries = ctx.__validationAudit?.platformCreateWhitelistHits || [];
  expect(entries.length).toBe(1);
  expect(entries[0].source).toBe('request_context');
  expect(entries[0].model).toBe('NameFallback');
});

test('validation platform helper reuses existing object bucket without overwriting version', () => {
  const ctx: any = {
    __validationAudit: {
      version: 7,
      platformCreateWhitelistHits: [],
    },
  };

  recordRepositoryPlatformCreateWhitelistAudit({ fullModelName: 'demo.Model', modelName: '', name: '' } as any, ctx, 'create', ['A']);

  expect(ctx.__validationAudit.version).toBe(7);
  expect(ctx.__validationAudit.platformCreateWhitelistHits.length).toBe(1);
  expect(ctx.__validationAudit.platformCreateWhitelistHits[0].source).toBe('request_context');
});

test('validation platform helper reads by-model whitelist even when model key is empty string', () => {
  const list = resolveRepositoryPlatformCreateWriteWhitelist({ fullModelName: '', modelName: '', name: '' } as any, {
    validation: {
      platformCreateWriteWhitelistByModel: {
        '': [' X ', 'Y', 'X'],
      },
    },
  });
  expect(list).toEqual(['X', 'Y']);
});
