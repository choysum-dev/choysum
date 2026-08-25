// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import ImportJob from './import_job';

type ImportBridge = {
  run?: (spec: Record<string, unknown> | string) => Promise<Record<string, any>>;
};

function importBridge(): ImportBridge {
  const root: any = (globalThis as any)?.$choysum;
  return (root?.import ?? {}) as ImportBridge;
}

/** Task worker entry: replays SpecSnapshotJson via $choysum.import.run and writes report. */
export async function executeImportJob(importJobId: string): Promise<Record<string, any>> {
  const id = String(importJobId || '').trim();
  if (!id) {
    throw new Error('importJobId is required');
  }
  const row = await ImportJob.Browse(id, ['Id', 'SpecSnapshotJson'] as any);
  const spec = (row as any)?.SpecSnapshotJson;
  if (!spec || typeof spec !== 'object') {
    throw new Error('import job is missing spec snapshot');
  }
  const bridge = importBridge();
  if (typeof bridge.run !== 'function') {
    throw new Error('import bridge is not available');
  }
  const report = await bridge.run(spec);
  await ImportJob.FinalizeReport(id, report ?? {});
  return report ?? {};
}
