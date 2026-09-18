// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveChoysum } from '@/core/service/runtime/choysum_runtime';
import DataTransferJob from './data_transfer_job';

type ImportBridge = {
  run?: (spec: Record<string, unknown> | string) => Promise<Record<string, any>>;
};

type ExportBridge = {
  run?: (spec: Record<string, unknown> | string) => Promise<Record<string, any>>;
};

type ChoysumWithExport = NonNullable<ReturnType<typeof resolveChoysum>> & {
  export?: ExportBridge;
};

function importBridge(): ImportBridge {
  return resolveChoysum()?.import ?? {};
}

function exportBridge(): ExportBridge {
  return (resolveChoysum() as ChoysumWithExport | undefined)?.export ?? {};
}

function asSpecSnapshot(value: unknown): Record<string, unknown> | string | undefined {
  if (typeof value === 'string') return value;
  if (value && typeof value === 'object') return value as Record<string, unknown>;
  return undefined;
}

/** Task worker entry: replays SpecSnapshotJson via $choysum.import.run and writes report. */
export async function executeImport(dataTransferJobId: string): Promise<Record<string, any>> {
  const id = String(dataTransferJobId || '').trim();
  if (!id) {
    throw new Error('dataTransferJobId is required');
  }
  const row = await DataTransferJob.Browse(id, ['Id', 'SpecSnapshotJson', 'Direction']);
  const direction = String(row?.Direction || '').trim();
  if (direction && direction !== 'import') {
    throw new Error(`ExecuteImport requires Direction=import (got ${JSON.stringify(direction)})`);
  }
  const spec = asSpecSnapshot(row?.SpecSnapshotJson);
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
  const row = await DataTransferJob.Browse(id, ['Id', 'SpecSnapshotJson', 'Direction']);
  const direction = String(row?.Direction || '').trim();
  if (direction && direction !== 'export') {
    throw new Error(`ExecuteExport requires Direction=export (got ${JSON.stringify(direction)})`);
  }
  const spec = asSpecSnapshot(row?.SpecSnapshotJson);
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
