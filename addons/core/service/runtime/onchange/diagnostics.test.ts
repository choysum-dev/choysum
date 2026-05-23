// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DiagnosticsBuilder, attachDiagnostics, buildDiagnostics, endTimer, startTimer } from './diagnostics';

test('onchange diagnostics timer helpers expose stable no-op timer behavior', () => {
  const t = startTimer();
  t.start();

  expect(t.elapsed()).toBe(0);
  expect(endTimer(t)).toBe(0);
});

test('onchange diagnostics builder and attachDiagnostics keep payload shape', () => {
  const diag = new DiagnosticsBuilder()
    .setMissingCount(1)
    .setPrefetchTime(2)
    .setPathDepthMax(3)
    .setComputeRecomputed(['Total'])
    .setReadsRoots(['PartnerId'])
    .setChangedSeeds(['Name'])
    .setIterations(4)
    .setLoopThreshold(5)
    .enablePlanCache(true)
    .pushMessage({ level: 'info', message: 'ok' })
    .build();

  const result: any = {};
  attachDiagnostics(result, diag);

  expect(result.diagnostics?.missingCount).toBe(1);
  expect(Array.isArray(result.messages)).toBe(true);

  const merged = buildDiagnostics({
    readsRoots: ['PartnerId'],
  } as any);
  expect(merged.readsRoots).toEqual(['PartnerId']);
  expect(merged.pathDepthMax).toBe(1);
});
