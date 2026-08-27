// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import DataTransferJob from './data_transfer_job';

type ImportBridge = {
  run?: (spec: Record<string, unknown> | string) => Promise<Record<string, any>>;
};

type ExportBridge = {
  run?: (spec: Record<string, unknown> | string) => Promise<Record<string, any>>;
};

function importBridge(): ImportBridge {
  const root: any = (globalThis as any)?.$choysum;
  return (root?.import ?? {}) as ImportBridge;
}

function exportBridge(): ExportBridge {
  const root: any = (globalThis as any)?.$choysum;
  return (root?.export ?? {}) as ExportBridge;
}

/** Task worker entry: replays SpecSnapshotJson via $choysum.import.run and writes report. */
export async function executeImport(dataTransferJobId: string): Promise<Record<string, any>> {
  const id = String(dataTransferJobId || '').trim();
  if (!id) {
    throw new Error('dataTransferJobId is required');
  }
  const row = await DataTransferJob.Browse(id, ['Id', 'SpecSnapshotJson', 'Direction'] as any);
  const direction = String((row as any)?.Direction || '').trim();
  if (direction && direction !== 'import') {
    throw new Error(`ExecuteImport requires Direction=import (got ${JSON.stringify(direction)})`);
  }
  const spec = (row as any)?.SpecSnapshotJson;
  if (!spec || typeof spec !== 'object') {
    throw new Error('data transfer job is missing spec snapshot');
  }
  const bridge = importBridge();
  if (typeof bridge.run !== 'function') {
    throw new Error('import bridge is not available');
  }
  const report = await bridge.run(spec);
  await DataTransferJob.FinalizeReport(id, report ?? {});
  return report ?? {};
}

/** Task worker entry: replays SpecSnapshotJson via $choysum.export.run and writes report. */
export async function executeExport(dataTransferJobId: string): Promise<Record<string, any>> {
  const id = String(dataTransferJobId || '').trim();
  if (!id) {
    throw new Error('dataTransferJobId is required');
  }
  const row = await DataTransferJob.Browse(id, ['Id', 'SpecSnapshotJson', 'Direction'] as any);
  const direction = String((row as any)?.Direction || '').trim();
  if (direction && direction !== 'export') {
    throw new Error(`ExecuteExport requires Direction=export (got ${JSON.stringify(direction)})`);
  }
  const spec = (row as any)?.SpecSnapshotJson;
  if (!spec || typeof spec !== 'object') {
    throw new Error('data transfer job is missing spec snapshot');
  }
  const bridge = exportBridge();
  if (typeof bridge.run !== 'function') {
    throw new Error('export bridge is not available');
  }
  const report = await bridge.run(spec);
  await DataTransferJob.FinalizeReport(id, report ?? {});
  return report ?? {};
}
