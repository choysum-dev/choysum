// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DEFAULT_LOOP_THRESHOLD } from '../../runtime/onchange/constants';
import type { OnchangeResult } from '../../runtime/onchange/types';
import { executePreparedModelOnchangePreview } from './model_onchange_execute';
import type { ModelOnchangePreparation } from './model_onchange_prepare';

test('model onchange execute delegates prepared preview pipeline unchanged', async () => {
  const modelCtor = { name: 'DemoModel' } as any;
  const draft = { Id: 'ROW-1', Name: 'draft' };
  const previewResult: OnchangeResult<any> = { value: { Name: 'next' }, messages: [] };
  const finalTransport: OnchangeResult<any> = { value: { Name: 'transport' } };
  const calls: string[] = [];

  const prepared: ModelOnchangePreparation = {
    preview: {
      meta: { fullModelName: 'test.DemoModel' } as any,
      selParsed: { collectionRoots: new Set(), fieldSignals: new Map(), selectors: new Map(), normalizedSeeds: new Set(['Name']) } as any,
      changedFields: ['Name'],
      mergedDraft: { Id: 'ROW-1', Name: 'next' },
      previewProxy: { Id: 'ROW-1', Name: 'next' },
    },
    diagnostics: {
      missingCount: 2,
      usedCache: true,
      pathDepthMax: 3,
      cachedSignature: 'sig-1',
      execStats: { totalBatches: 1, totalRows: 2 } as any,
      readsRoot: new Set(['Name', 'Status']),
    },
  };

  const transport = await executePreparedModelOnchangePreview(
    {
      ModelCtor: modelCtor,
      draft,
      prepared,
      prefetchTimeMs: 12.5,
      opts: { withCompute: false, maxIterations: 2 },
    },
    {
      runPreviewEngine: (async (meta: any, previewProxy: any, changedFields: any, opts: any) => {
        calls.push('run');
        expect(meta).toBe(prepared.preview.meta);
        expect(previewProxy).toBe(prepared.preview.previewProxy);
        expect(changedFields).toBe(prepared.preview.changedFields);
        expect(opts).toEqual({ withCompute: false, maxIterations: 2 });
        return previewResult;
      }) as any,
      applyPreviewCascade: (async (params: any) => {
        calls.push('cascade');
        expect(params.meta).toBe(prepared.preview.meta);
        expect(params.previewProxy).toBe(prepared.preview.previewProxy);
        expect(params.changedFields).toBe(prepared.preview.changedFields);
        expect(params.selParsed).toBe(prepared.preview.selParsed);
        expect(params.opts).toEqual({ withCompute: false, maxIterations: 2 });
        expect(params.res).toBe(previewResult);
      }) as any,
      attachDiagnostics: ((params: any) => {
        calls.push('diagnostics');
        expect(params.res).toBe(previewResult);
        expect(params.missingCount).toBe(2);
        expect(params.prefetchTimeMs).toBe(12.5);
        expect(params.pathDepthMax).toBe(3);
        expect(params.readsRoot).toBe(prepared.diagnostics.readsRoot);
        expect(params.changedFields).toBe(prepared.preview.changedFields);
        expect(params.usedCache).toBe(true);
        expect(params.cachedSignature).toBe('sig-1');
        expect(params.execStats).toBe(prepared.diagnostics.execStats);
        expect(params.loopThreshold).toBe(DEFAULT_LOOP_THRESHOLD);
      }) as any,
      validatePreview: (async (params: any) => {
        calls.push('validate');
        expect(params.ModelCtor).toBe(modelCtor);
        expect(params.draft).toBe(draft);
        expect(params.meta).toBe(prepared.preview.meta);
        expect(params.previewProxy).toBe(prepared.preview.previewProxy);
        expect(params.mergedDraft).toBe(prepared.preview.mergedDraft);
        expect(params.changedFields).toBe(prepared.preview.changedFields);
        expect(params.res).toBe(previewResult);
      }) as any,
      finalizeTransport: ((res: OnchangeResult) => {
        calls.push('finalize');
        expect(res).toBe(previewResult);
        return finalTransport;
      }) as any,
    }
  );

  expect(transport).toBe(finalTransport);
  expect(calls).toEqual(['run', 'cascade', 'diagnostics', 'validate', 'finalize']);
});

test('model onchange execute swallows diagnostics errors and still finalizes transport', async () => {
  const previewResult: OnchangeResult<any> = { value: { Name: 'next' } };
  const finalTransport: OnchangeResult<any> = { value: { Name: 'done' } };
  const calls: string[] = [];

  const prepared: ModelOnchangePreparation = {
    preview: {
      meta: {} as any,
      selParsed: { collectionRoots: new Set(), fieldSignals: new Map(), selectors: new Map(), normalizedSeeds: new Set(['Name']) } as any,
      changedFields: ['Name'],
      mergedDraft: { Name: 'next' },
      previewProxy: { Name: 'next' },
    },
    diagnostics: {
      missingCount: 0,
      usedCache: false,
      pathDepthMax: 1,
      cachedSignature: 'sig-2',
      readsRoot: new Set(['Name']),
    },
  };

  const transport = await executePreparedModelOnchangePreview(
    {
      ModelCtor: {} as any,
      draft: { Name: 'draft' },
      prepared,
      prefetchTimeMs: 1,
    },
    {
      runPreviewEngine: (async () => {
        calls.push('run');
        return previewResult;
      }) as any,
      applyPreviewCascade: (async () => {
        calls.push('cascade');
      }) as any,
      attachDiagnostics: (() => {
        calls.push('diagnostics');
        throw new Error('boom');
      }) as any,
      validatePreview: (async () => {
        calls.push('validate');
      }) as any,
      finalizeTransport: (() => {
        calls.push('finalize');
        return finalTransport;
      }) as any,
    }
  );

  expect(transport).toBe(finalTransport);
  expect(calls).toEqual(['run', 'cascade', 'diagnostics', 'validate', 'finalize']);
});

test('model onchange execute produces consistent transport shape regardless of handler signature', async () => {
  const previewResult: OnchangeResult<any> = {
    value: { Code: 'CHANGED' },
    messages: [{ level: 'warn', message: 'test-warn' }],
    condition: [{ field: 'Code', condition: ['Code', '=', 'CHANGED'] }],
    selection: [{ field: 'Code', selection: ['A', 'CHANGED'] }],
  };
  const finalTransport: OnchangeResult<any> = { ...previewResult };

  const prepared: ModelOnchangePreparation = {
    preview: {
      meta: { fullModelName: 'test.SigModel' } as any,
      selParsed: { collectionRoots: new Set(), fieldSignals: new Map(), selectors: new Map(), normalizedSeeds: new Set(['Name']) } as any,
      changedFields: ['Name'],
      mergedDraft: { Name: 'A' },
      previewProxy: { Name: 'A' },
    },
    diagnostics: {
      missingCount: 0,
      usedCache: false,
      pathDepthMax: 1,
      cachedSignature: 'sig',
      readsRoot: new Set(['Name']),
    },
  };

  const callsA: string[] = [];
  const callsB: string[] = [];

  const deps = {
    runPreviewEngine: (async () => { callsA.push('run'); return previewResult; }) as any,
    applyPreviewCascade: (async () => { callsA.push('cascade'); }) as any,
    attachDiagnostics: (() => { callsA.push('diagnostics'); }) as any,
    validatePreview: (async () => { callsA.push('validate'); }) as any,
    finalizeTransport: (() => { callsA.push('finalize'); return finalTransport; }) as any,
  };

  const t1 = await executePreparedModelOnchangePreview(
    { ModelCtor: {} as any, draft: {}, prepared, prefetchTimeMs: 0 },
    deps
  );

  const depsB = {
    runPreviewEngine: (async () => { callsB.push('run'); return previewResult; }) as any,
    applyPreviewCascade: (async () => { callsB.push('cascade'); }) as any,
    attachDiagnostics: (() => { callsB.push('diagnostics'); }) as any,
    validatePreview: (async () => { callsB.push('validate'); }) as any,
    finalizeTransport: (() => { callsB.push('finalize'); return finalTransport; }) as any,
  };

  const t2 = await executePreparedModelOnchangePreview(
    { ModelCtor: {} as any, draft: {}, prepared, prefetchTimeMs: 0 },
    depsB
  );

  expect(t1).toEqual(t2);
  expect(callsA).toEqual(['run', 'cascade', 'diagnostics', 'validate', 'finalize']);
  expect(callsB).toEqual(callsA);
});
