// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DefaultOperations } from './model_default';
import { OnchangeOperations } from './model_onchange';
import { OnchangeEngine } from '../../runtime/onchange/engine';
import { ComputeEngine } from '../../runtime/compute/engine';
import { ComputeCascadeEngine } from '../../runtime/compute/cascade';
import {
  defaultModelValues,
  recomputeModelMetadata,
  runModelOnchange,
  runModelOnchangePreviewEngine,
  triggerModelDownstream,
  triggerModelUpstream,
} from './model_runtime_service_facade';

test('model runtime service facade delegates default values without reshaping input', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const calls: any[] = [];
  const ctor = {} as any;
  const input = { Name: 'demo' };

  try {
    DefaultOperations.DefaultGet = (async (ModelCtor: any, value: any) => {
      calls.push({ ModelCtor, value });
      return { ...value, Code: 'C001' };
    }) as any;

    const result = await defaultModelValues(ctor, input as any);

    expect(result).toEqual({ Name: 'demo', Code: 'C001' });
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.value).toBe(input);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
  }
});

test('model runtime service facade delegates onchange without rewriting options', async () => {
  const originalOnchange = OnchangeOperations.Onchange;
  const calls: any[] = [];
  const ctor = {} as any;
  const draft = { Id: 'M1', Name: 'draft' };
  const changed = ['Name'] as any;
  const opts = { withCompute: false, maxIterations: 2, loopThreshold: 1 };

  try {
    OnchangeOperations.Onchange = (async (ModelCtor: any, nextDraft: any, nextChanged: any, nextOpts: any) => {
      calls.push({ ModelCtor, nextDraft, nextChanged, nextOpts });
      return { value: { Name: 'next' }, messages: [] };
    }) as any;

    const result = await runModelOnchange(ctor, draft, changed, opts);

    expect(result).toEqual({ value: { Name: 'next' }, messages: [] });
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.nextDraft).toBe(draft);
    expect(calls[0]?.nextChanged).toBe(changed);
    expect(calls[0]?.nextOpts).toBe(opts);
  } finally {
    OnchangeOperations.Onchange = originalOnchange;
  }
});

test('model runtime service facade delegates compute recompute with metadata unchanged', async () => {
  const originalRecompute = ComputeEngine.recompute;
  const calls: any[] = [];
  const meta = { computeGraph: { computeFields: new Set(['Name']) } } as any;
  const entity = { Id: 'M1', Name: 'draft' };
  const seed = new Set(['Name']);

  try {
    ComputeEngine.recompute = (async (nextMeta: any, nextEntity: any, nextSeed: any, nextMode: any) => {
      calls.push({ nextMeta, nextEntity, nextSeed, nextMode });
    }) as any;

    await recomputeModelMetadata(meta, entity, seed, 'persist');

    expect(calls.length).toBe(1);
    expect(calls[0]?.nextMeta).toBe(meta);
    expect(calls[0]?.nextEntity).toBe(entity);
    expect(calls[0]?.nextSeed).toBe(seed);
    expect(calls[0]?.nextMode).toBe('persist');
  } finally {
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model runtime service facade skips recompute when metadata has no compute graph', async () => {
  const originalRecompute = ComputeEngine.recompute;
  const calls: any[] = [];

  try {
    ComputeEngine.recompute = (async () => {
      calls.push('recompute');
    }) as any;

    await recomputeModelMetadata({} as any, { Id: 'M2' }, new Set(['Name']), 'persist');

    expect(calls).toEqual([]);
  } finally {
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model runtime service facade runs preview onchange engine with normalized defaults and preview recompute bridge', async () => {
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;
  const calls: any[] = [];
  const recomputeCalls: any[] = [];
  const meta = { computeGraph: { computeFields: new Set(['Name']) } } as any;
  const entity = { Id: 'M1', Name: 'draft' };
  const changedFields = ['Name'];
  const seed = new Set(['Name']);

  try {
    ComputeEngine.recompute = (async (nextMeta: any, nextEntity: any, nextSeed: any, nextMode: any) => {
      recomputeCalls.push({ nextMeta, nextEntity, nextSeed, nextMode });
    }) as any;
    OnchangeEngine.run = (async (nextMeta: any, nextEntity: any, nextChangedFields: any, nextOpts: any) => {
      calls.push({ nextMeta, nextEntity, nextChangedFields, nextOpts });
      await nextOpts.computePreview(nextEntity, seed);
      return { value: { Name: 'preview' } };
    }) as any;

    const result = await runModelOnchangePreviewEngine(meta, entity, changedFields, { withCompute: false, maxIterations: 2, loopThreshold: 1 });

    expect(result).toEqual({ value: { Name: 'preview' } });
    expect(calls.length).toBe(1);
    expect(calls[0]?.nextMeta).toBe(meta);
    expect(calls[0]?.nextEntity).toBe(entity);
    expect(calls[0]?.nextChangedFields).toBe(changedFields);
    expect(calls[0]?.nextOpts?.withCompute).toBe(false);
    expect(calls[0]?.nextOpts?.maxIterations).toBe(2);
    expect(calls[0]?.nextOpts?.loopThreshold).toBe(1);

    expect(recomputeCalls.length).toBe(1);
    expect(recomputeCalls[0]?.nextMeta).toBe(meta);
    expect(recomputeCalls[0]?.nextEntity).toBe(entity);
    expect(recomputeCalls[0]?.nextSeed).toBe(seed);
    expect(recomputeCalls[0]?.nextMode).toBe('preview');
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model runtime service facade delegates upstream and downstream cascade unchanged', async () => {
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const calls: any[] = [];
  const ctor = {} as any;
  const upstreamEvent = { childCtor: ctor, operation: 'update', changedFields: ['Name'] } as any;

  try {
    ComputeCascadeEngine.triggerUpstream = (async (event: any) => {
      calls.push({ kind: 'upstream', event });
    }) as any;
    ComputeCascadeEngine.triggerDownstream = (async (ModelCtor: any, changedFields: any, recordId: any) => {
      calls.push({ kind: 'downstream', ModelCtor, changedFields, recordId });
    }) as any;

    await triggerModelUpstream(upstreamEvent);
    await triggerModelDownstream(ctor, ['Name'], 'ROW-1');

    expect(calls.length).toBe(2);
    expect(calls[0]).toEqual({ kind: 'upstream', event: upstreamEvent });
    expect(calls[1]).toEqual({ kind: 'downstream', ModelCtor: ctor, changedFields: ['Name'], recordId: 'ROW-1' });
  } finally {
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});
