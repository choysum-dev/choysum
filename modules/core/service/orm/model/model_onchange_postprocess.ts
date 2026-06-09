// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ValidationIssue } from '../metadata/constraint';
import type { OnchangeEngineResult, OnchangeMessage, OnchangeResult, PrefetchExecStats } from '../../runtime/onchange/types';
import { DiagnosticsBuilder, attachDiagnostics } from '../../runtime/onchange/diagnostics';
import type { ObjectRecord } from '../../../utils/types';

export function attachModelOnchangeDiagnostics(params: {
  res: OnchangeEngineResult;
  missingCount: number;
  prefetchTimeMs: number;
  pathDepthMax: number;
  readsRoot: Set<string>;
  changedFields: string[];
  usedCache: boolean;
  cachedSignature: string;
  execStats?: PrefetchExecStats;
  loopThreshold?: number;
}): void {
  const { res, missingCount, prefetchTimeMs, pathDepthMax, readsRoot, changedFields, usedCache, cachedSignature, execStats, loopThreshold } = params;

  const diagBuilder = new DiagnosticsBuilder()
    .setMissingCount(missingCount)
    .setPrefetchTime(prefetchTimeMs)
    .setPathDepthMax(pathDepthMax)
    .setComputeRecomputed(res.computeRecomputed)
    .setReadsRoots(readsRoot)
    .setChangedSeeds(changedFields)
    .setIterations(res.iterations)
    .setLoopThreshold(loopThreshold)
    .enablePlanCache(usedCache)
    .pushMessage({ code: 'plan.signature', signature: cachedSignature });

  if (execStats) {
    diagBuilder.pushMessage({
      code: 'prefetch.stats',
      totalBatches: execStats.totalBatches ?? 0,
      totalRows: execStats.totalRows ?? 0,
    });
  }

  attachDiagnostics(res, diagBuilder.build());
}

export function appendModelOnchangeValidationIssues(res: OnchangeResult, issues: ValidationIssue[]): void {
  if (!issues.length) return;

  const messages: OnchangeMessage[] = issues.map(issue => ({
    level: issue.severity === 'warning' ? 'warn' : 'error',
    message: issue.message,
    field: issue.field as unknown as OnchangeMessage['field'],
    blocking: issue.severity !== 'warning',
    title: issue.method,
  }));

  res.messages = [...(res.messages || []), ...messages];
}

export function finalizeModelOnchangeTransport(res: OnchangeResult): OnchangeResult {
  const hasErrorFinal = Array.isArray(res?.messages) && res.messages.some(m => m?.level === 'error');
  if (hasErrorFinal) {
    res.value = {};
    try {
      if (res && res.value && '__collectionPatch' in (res.value as ObjectRecord)) {
        delete (res.value as ObjectRecord).__collectionPatch;
      }
    } catch {
      // ignore
    }
  }

  const transport: OnchangeResult = {};
  if (res.value && Object.keys(res.value).length > 0) {
    transport.value = res.value;
  }
  if (res.messages && res.messages.length > 0) {
    transport.messages = res.messages;
  }
  if (res.condition && res.condition.length > 0) {
    transport.condition = res.condition;
  }
  if (res.selection && res.selection.length > 0) {
    transport.selection = res.selection;
  }
  return transport;
}
