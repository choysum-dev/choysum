// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import DataTransferJob, {
  DATA_TRANSFER_JOB_EXECUTE_EXPORT_FULL_METHOD,
  DATA_TRANSFER_JOB_EXECUTE_IMPORT_FULL_METHOD,
} from '@/task/service/models/data_transfer_job';
import { getQueueStatus } from '@/task/service/models/data_transfer_job_queue';
import { executeExport, executeImport } from '@/task/service/models/data_transfer_job_worker';
import Job from '@/task/service/models/job';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum');
  }
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'task.Job:create',
      'task.Job:read',
      'task.Job:write',
      'task.Job:delete',
      'task.DataTransferJob:create',
      'task.DataTransferJob:read',
      'task.DataTransferJob:write',
      'task.DataTransferJob:delete',
      'DataTransferJob:create',
      'DataTransferJob:read',
      'DataTransferJob:write',
      'DataTransferJob:delete',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: 'admin' };
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  delete (jsCtx as any)[CTX_OVERRIDE_KEY];
  delete (jsCtx as any)[CTX_FROZEN_KEY];
}

function sampleSnapshot(sourceRef: string) {
  return {
    profile: 'record',
    caller: 'user',
    policy: 'atomic',
    model: 'base.Country',
    source: { format: 'csv', document_ref: sourceRef },
  };
}

function sampleExportSnapshot(model: string) {
  return {
    profile: 'record',
    caller: 'user',
    async: true,
    model,
    format: 'csv',
    mode: 'data',
  };
}

async function expectAsyncError(run: () => Promise<unknown>, pattern: RegExp): Promise<void> {
  let thrown: any;
  try {
    await run();
  } catch (err) {
    thrown = err;
  }
  expect(thrown).toBeTruthy();
  expect(String(thrown?.message || thrown)).toMatch(pattern);
}

test('DataTransferJob model has no State field metadata', () => {
  resetRequestContext();
  const fields = (DataTransferJob as any).$meta?.fields;
  expect(fields?.has?.('State') ?? fields?.State).toBeFalsy();
});

test('DataTransferJob Direction is required and defaults to import', async () => {
  resetRequestContext();

  const created = await DataTransferJob.Create({
    TargetModel: 'base.Country',
    SourceRef: 'doc-direction-default',
    SpecSnapshotJson: sampleSnapshot('doc-direction-default'),
  } as Partial<DataTransferJob>);
  const createdRow = await DataTransferJob.Browse(created.Id, ['Direction'] as any);
  expect((createdRow as any).Direction).toBe('import');

  const enqueued = await DataTransferJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-direction-required',
    specSnapshot: sampleSnapshot('doc-direction-required'),
  });
  const row = await DataTransferJob.Browse(enqueued.dataTransferJobId, ['Direction'] as any);
  expect((row as any).Direction).toBe('import');

  await expectAsyncError(
    () => DataTransferJob.ExecuteExport(enqueued.dataTransferJobId),
    /ExecuteExport requires Direction=export/
  );
});

test('DataTransferJob.EnqueueRecordImport creates linked task job', async () => {
  resetRequestContext();
  const result = await DataTransferJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-ref-1',
    companyId: 'cmp-1',
    policy: 'atomic',
    specSnapshot: sampleSnapshot('doc-ref-1'),
  });
  expect(result.dataTransferJobId).toBeTruthy();
  expect(result.taskJobId).toBeTruthy();

  const row = await DataTransferJob.Browse(result.dataTransferJobId, ['TaskJobId', 'TargetModel', 'Policy', 'Direction'] as any);
  expect((row as any).TaskJobId).toBe(result.taskJobId);
  expect((row as any).TargetModel).toBe('base.Country');
  expect((row as any).Policy).toBe('atomic');
  expect((row as any).Direction).toBe('import');

  const taskJob = await Job.GetJob(result.taskJobId, ['FullMethod', 'PayloadJson'] as any);
  expect((taskJob as any).FullMethod).toBe(DATA_TRANSFER_JOB_EXECUTE_IMPORT_FULL_METHOD);
  expect((taskJob as any).PayloadJson?.dataTransferJobId).toBe(result.dataTransferJobId);
});

test('DataTransferJob.EnqueueRecordExport creates linked export task job', async () => {
  resetRequestContext();
  const result = await DataTransferJob.EnqueueRecordExport({
    targetModel: 'base.Country',
    sourceRef: 'export:base.Country',
    companyId: 'cmp-1',
    specSnapshot: sampleExportSnapshot('base.Country'),
  });
  expect(result.dataTransferJobId).toBeTruthy();
  expect(result.taskJobId).toBeTruthy();

  const row = await DataTransferJob.Browse(result.dataTransferJobId, ['TaskJobId', 'TargetModel', 'Direction', 'SourceRef'] as any);
  expect((row as any).TaskJobId).toBe(result.taskJobId);
  expect((row as any).TargetModel).toBe('base.Country');
  expect((row as any).Direction).toBe('export');
  expect((row as any).SourceRef).toBe('export:base.Country');

  const taskJob = await Job.GetJob(result.taskJobId, ['FullMethod', 'PayloadJson'] as any);
  expect((taskJob as any).FullMethod).toBe(DATA_TRANSFER_JOB_EXECUTE_EXPORT_FULL_METHOD);
  expect((taskJob as any).PayloadJson?.dataTransferJobId).toBe(result.dataTransferJobId);
});

test('DataTransferJob.EnqueueRecordExport validation paths', async () => {
  resetRequestContext();
  const jsCtx = ensureRequestContext();
  const previousUserId = jsCtx.identity.userId;
  jsCtx.identity.userId = '';
  try {
    await expectAsyncError(
      () =>
        DataTransferJob.EnqueueRecordExport({
          targetModel: 'base.Country',
          sourceRef: 'export:base.Country',
          specSnapshot: sampleExportSnapshot('base.Country'),
        }),
      /authenticated user/
    );
  } finally {
    jsCtx.identity.userId = previousUserId;
  }

  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordExport({
        targetModel: '',
        sourceRef: 'export:base.Country',
        specSnapshot: sampleExportSnapshot('base.Country'),
      } as any),
    /targetModel and sourceRef/
  );
  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordExport({
        targetModel: 'base.Country',
        sourceRef: '',
        specSnapshot: sampleExportSnapshot('base.Country'),
      } as any),
    /targetModel and sourceRef/
  );
  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordExport({
        targetModel: 'base.Country',
        sourceRef: 'export:base.Country',
        specSnapshot: null as any,
      }),
    /specSnapshot/
  );
});

test('DataTransferJob.EnqueueRecordExport rejects non-record profiles', async () => {
  resetRequestContext();
  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordExport({
        targetModel: 'base.Country',
        sourceRef: 'export:base.Country',
        profile: 'terminology',
        specSnapshot: sampleExportSnapshot('base.Country'),
      }),
    /unsupported data transfer profile/
  );
  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordExport({
        targetModel: 'base.Country',
        sourceRef: 'export:base.Country',
        profile: 'initdata',
        specSnapshot: sampleExportSnapshot('base.Country'),
      }),
    /unsupported data transfer profile/
  );
});

test('DataTransferJob.EnqueueRecordExport rolls back row when EnqueueJob fails', async () => {
  resetRequestContext();
  const enqueueJob = (Job as any).EnqueueJob.bind(Job);
  const deleteById = (DataTransferJob as any).DeleteById.bind(DataTransferJob);
  let createdId = '';
  const create = (DataTransferJob as any).Create.bind(DataTransferJob);
  (Job as any).EnqueueJob = async () => {
    throw new Error('enqueue boom');
  };
  (DataTransferJob as any).Create = async (values: Partial<DataTransferJob>) => {
    const row = await create(values);
    createdId = row.Id;
    return row;
  };
  try {
    await expectAsyncError(
      () =>
        DataTransferJob.EnqueueRecordExport({
          targetModel: 'base.Country',
          sourceRef: 'export:base.Country',
          specSnapshot: sampleExportSnapshot('base.Country'),
        }),
      /enqueue boom/
    );
    await expectAsyncError(() => DataTransferJob.Browse(createdId, ['Id'] as any), /not found/i);
  } finally {
    (Job as any).EnqueueJob = enqueueJob;
    (DataTransferJob as any).Create = create;
    (DataTransferJob as any).DeleteById = deleteById;
  }
});

test('DataTransferJob.EnqueueRecordExport throws when rollback DeleteById fails', async () => {
  resetRequestContext();
  const enqueueJob = (Job as any).EnqueueJob.bind(Job);
  const deleteById = (DataTransferJob as any).DeleteById.bind(DataTransferJob);
  (Job as any).EnqueueJob = async () => {
    throw new Error('enqueue boom');
  };
  (DataTransferJob as any).DeleteById = async () => {
    throw new Error('delete boom');
  };
  try {
    await expectAsyncError(
      () =>
        DataTransferJob.EnqueueRecordExport({
          targetModel: 'base.Country',
          sourceRef: 'export:base.Country',
          specSnapshot: sampleExportSnapshot('base.Country'),
        }),
      /enqueue boom/
    );
  } finally {
    (Job as any).EnqueueJob = enqueueJob;
    (DataTransferJob as any).DeleteById = deleteById;
  }
});

test('DataTransferJob.EnqueueRecordExport omits blank companyId', async () => {
  resetRequestContext();
  const result = await DataTransferJob.EnqueueRecordExport({
    targetModel: 'base.Country',
    sourceRef: 'export:base.Country',
    companyId: '   ',
    specSnapshot: sampleExportSnapshot('base.Country'),
  });
  const row = await DataTransferJob.Browse(result.dataTransferJobId, ['CompanyId'] as any);
  expect((row as any).CompanyId == null).toBe(true);
});

test('DataTransferJob.EnqueueRecordExport keeps row when UpdateById fails after enqueue', async () => {
  resetRequestContext();
  const updateById = (DataTransferJob as any).UpdateById.bind(DataTransferJob);
  let createdId = '';
  let taskJobId = '';
  const create = (DataTransferJob as any).Create.bind(DataTransferJob);
  const enqueueJob = (Job as any).EnqueueJob.bind(Job);
  (DataTransferJob as any).Create = async (values: Partial<DataTransferJob>) => {
    const row = await create(values);
    createdId = row.Id;
    return row;
  };
  (Job as any).EnqueueJob = async (...args: any[]) => {
    const job = await enqueueJob(...args);
    taskJobId = job.Id;
    return job;
  };
  (DataTransferJob as any).UpdateById = async () => {
    throw new Error('update boom');
  };
  try {
    await expectAsyncError(
      () =>
        DataTransferJob.EnqueueRecordExport({
          targetModel: 'base.Country',
          sourceRef: 'export:base.Country',
          specSnapshot: sampleExportSnapshot('base.Country'),
        }),
      /update boom/
    );
    const row = await DataTransferJob.Browse(createdId, ['TaskJobId'] as any);
    expect(row).toBeTruthy();
    expect((row as any).TaskJobId).toBeFalsy();
    const taskJob = await Job.GetJob(taskJobId, ['FullMethod'] as any);
    expect((taskJob as any).FullMethod).toBe(DATA_TRANSFER_JOB_EXECUTE_EXPORT_FULL_METHOD);
  } finally {
    (DataTransferJob as any).UpdateById = updateById;
    (DataTransferJob as any).Create = create;
    (Job as any).EnqueueJob = enqueueJob;
  }
});

test('DataTransferJob.EnqueueRecordImport validation paths', async () => {
  resetRequestContext();
  const jsCtx = ensureRequestContext();
  const previousUserId = jsCtx.identity.userId;
  jsCtx.identity.userId = '';
  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        specSnapshot: sampleSnapshot('doc'),
      }),
    /authenticated user/
  );
  jsCtx.identity.userId = previousUserId;

  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordImport({
        targetModel: '',
        sourceRef: 'doc',
        specSnapshot: sampleSnapshot('doc'),
      } as any),
    /targetModel and sourceRef/
  );

  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: '',
        specSnapshot: sampleSnapshot('doc'),
      } as any),
    /targetModel and sourceRef/
  );

  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        specSnapshot: null as any,
      }),
    /specSnapshot/
  );

  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        profile: 'nope',
        specSnapshot: sampleSnapshot('doc'),
      }),
    /unsupported data transfer profile/
  );

  await expectAsyncError(
    () =>
      DataTransferJob.EnqueueRecordImport({
        targetModel: 'base.Country',
        sourceRef: 'doc',
        policy: 'nope',
        specSnapshot: sampleSnapshot('doc'),
      }),
    /unsupported data transfer policy/
  );

  const defaults = await DataTransferJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-defaults',
    profile: '  ',
    policy: '',
    specSnapshot: sampleSnapshot('doc-defaults'),
  });
  const row = await DataTransferJob.Browse(defaults.dataTransferJobId, ['Profile', 'Policy'] as any);
  expect((row as any).Profile).toBe('record');
  expect((row as any).Policy).toBe('atomic');
});

test('DataTransferJob field defaults apply on minimal Create', async () => {
  resetRequestContext();
  const row = await DataTransferJob.Create({
    TargetModel: 'base.Country',
    SourceRef: 'doc-field-defaults',
    SpecSnapshotJson: sampleSnapshot('doc-field-defaults'),
  } as Partial<DataTransferJob>);
  const loaded = await DataTransferJob.Browse(row.Id, ['Profile', 'Policy', 'DryRun', 'Direction'] as any);
  expect((loaded as any).Profile).toBe('record');
  expect((loaded as any).Policy).toBe('atomic');
  expect((loaded as any).DryRun).toBe(false);
  expect((loaded as any).Direction).toBe('import');
});

test('getQueueStatus joins DataTransferJob with Job.Status', async () => {
  resetRequestContext();
  const enqueued = await DataTransferJob.EnqueueRecordImport({
    targetModel: 'base.Country',
    sourceRef: 'doc-ref-2',
    specSnapshot: sampleSnapshot('doc-ref-2'),
  });
  const status = await getQueueStatus(enqueued.dataTransferJobId);
  expect(status.queueStatus).toBe('queued');
  expect(status.taskJobId).toBe(enqueued.taskJobId);
});

test('getQueueStatus error paths', async () => {
  resetRequestContext();
  await expectAsyncError(() => getQueueStatus(''), /dataTransferJobId is required/);
  await expectAsyncError(() => getQueueStatus('missing-import-job'), /not found/i);

  const browse = (DataTransferJob as any).Browse.bind(DataTransferJob);
  (DataTransferJob as any).Browse = async () => null;
  try {
    await expectAsyncError(() => getQueueStatus('ghost-import-job'), /data transfer job ghost-import-job not found/);
  } finally {
    (DataTransferJob as any).Browse = browse;
  }

  const row = await DataTransferJob.Create({
    Profile: 'record',
    Policy: 'atomic',
    DryRun: false,
    TargetModel: 'base.Country',
    SourceRef: 'doc-unlinked',
    SpecSnapshotJson: sampleSnapshot('doc-unlinked'),
    Direction: 'import',
    ProgressDone: 0,
    ProgressTotal: 0,
  } as Partial<DataTransferJob>);
  await expectAsyncError(() => getQueueStatus(row.Id), /missing task job link/);
});

test('getQueueStatus joins empty progress and report fallbacks', async () => {
  resetRequestContext();
  const browse = (DataTransferJob as any).Browse.bind(DataTransferJob);
  const getJob = (Job as any).GetJob.bind(Job);
  (DataTransferJob as any).Browse = async () =>
    ({
      Id: 'import-fallback',
      TaskJobId: 'task-fallback',
      ProgressDone: null,
      ProgressTotal: undefined,
      ReportJson: null,
      ReportRef: '',
    }) as any;
  (Job as any).GetJob = async () => ({ Id: 'task-fallback', Status: '' }) as any;
  try {
    const status = await getQueueStatus('import-fallback');
    expect(status.queueStatus).toBe('');
    expect(status.progressDone).toBe(0);
    expect(status.progressTotal).toBe(0);
    expect(status.reportJson).toBeUndefined();
    expect(status.reportRef).toBeUndefined();
  } finally {
    (DataTransferJob as any).Browse = browse;
    (Job as any).GetJob = getJob;
  }
});

test('executeImport writes report via import bridge', async () => {
  resetRequestContext();
  const root: any = (globalThis as any).$choysum;
  const previousImport = root.import;
  try {
    root.import = {
      run: async () => ({
        profile: 'record',
        policy: 'atomic',
        stats: { total: 2, ok: 2, error: 0, skip: 0 },
        messages: [],
        artifact_ref: 'art-1',
      }),
    };

    const enqueued = await DataTransferJob.EnqueueRecordImport({
      targetModel: 'base.Country',
      sourceRef: 'doc-ref-3',
      specSnapshot: sampleSnapshot('doc-ref-3'),
    });

    const report = await DataTransferJob.ExecuteImport(enqueued.dataTransferJobId);
    expect(report?.stats?.ok).toBe(2);

    const row = await DataTransferJob.Browse(enqueued.dataTransferJobId, [
      'ReportJson',
      'ReportRef',
      'ProgressDone',
      'ProgressTotal',
    ] as any);
    expect((row as any).ReportJson?.stats?.ok).toBe(2);
    expect((row as any).ReportRef).toBe('art-1');
    expect((row as any).ProgressDone).toBe(2);
    expect((row as any).ProgressTotal).toBe(2);

    const status = await getQueueStatus(enqueued.dataTransferJobId);
    expect(status.reportRef).toBe('art-1');
    expect(status.reportJson?.stats?.ok).toBe(2);
  } finally {
    if (previousImport === undefined) {
      delete root.import;
    } else {
      root.import = previousImport;
    }
  }
});

test('executeExport writes report via export bridge', async () => {
  resetRequestContext();
  const root: any = (globalThis as any).$choysum;
  const previousExport = root.export;
  try {
    root.export = {
      run: async () => ({
        profile: 'record',
        stats: { total: 2, ok: 2, error: 0, skip: 0 },
        messages: [],
        artifact_ref: 'export-art-1',
      }),
    };

    const enqueued = await DataTransferJob.EnqueueRecordExport({
      targetModel: 'base.Country',
      sourceRef: 'export:base.Country',
      specSnapshot: sampleExportSnapshot('base.Country'),
    });

    const report = await DataTransferJob.ExecuteExport(enqueued.dataTransferJobId);
    expect(report?.stats?.ok).toBe(2);

    const row = await DataTransferJob.Browse(enqueued.dataTransferJobId, [
      'ReportJson',
      'ReportRef',
      'ProgressDone',
      'ProgressTotal',
      'Direction',
    ] as any);
    expect((row as any).Direction).toBe('export');
    expect((row as any).ReportJson?.stats?.ok).toBe(2);
    expect((row as any).ReportRef).toBe('export-art-1');
    expect((row as any).ProgressDone).toBe(2);
    expect((row as any).ProgressTotal).toBe(2);
  } finally {
    if (previousExport === undefined) {
      delete root.export;
    } else {
      root.export = previousExport;
    }
  }
});

test('executeImport and FinalizeReport error paths', async () => {
  resetRequestContext();
  const root: any = (globalThis as any).$choysum;
  const previousImport = root.import;
  try {
    await expectAsyncError(() => executeImport(''), /dataTransferJobId is required/);
    await expectAsyncError(() => executeImport('   '), /dataTransferJobId is required/);
    await expectAsyncError(() => executeImport(null as any), /dataTransferJobId is required/);
    await expectAsyncError(() => executeExport(''), /dataTransferJobId is required/);
    await expectAsyncError(() => executeExport('   '), /dataTransferJobId is required/);
    await expectAsyncError(() => executeExport(undefined as any), /dataTransferJobId is required/);
    await expectAsyncError(() => DataTransferJob.FinalizeReport('', {}), /dataTransferJobId is required/);

    const exportDirection = await DataTransferJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-export-direction',
      SpecSnapshotJson: sampleSnapshot('doc-export-direction'),
      Direction: 'export',
    } as Partial<DataTransferJob>);
    await expectAsyncError(
      () => executeImport(exportDirection.Id),
      /ExecuteImport requires Direction=import/
    );
    delete root.export;
    await expectAsyncError(
      () => executeExport(exportDirection.Id),
      /export bridge is not available/
    );

    const importDirection = await DataTransferJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-import-direction',
      SpecSnapshotJson: sampleSnapshot('doc-import-direction'),
      Direction: 'import',
    } as Partial<DataTransferJob>);
    await expectAsyncError(
      () => executeExport(importDirection.Id),
      /ExecuteExport requires Direction=export/
    );

    const missingDirection = await DataTransferJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-missing-direction',
      SpecSnapshotJson: sampleSnapshot('doc-missing-direction'),
      Direction: 'import',
    } as Partial<DataTransferJob>);
    const browse = (DataTransferJob as any).Browse.bind(DataTransferJob);
    (DataTransferJob as any).Browse = async (_id: string, fields?: any) => {
      const row = await browse(_id, fields);
      if (!row) return row;
      return { ...(row as any), Direction: '' };
    };
    root.import = {
      run: async () => ({ stats: { total: 0, ok: 0, error: 0, skip: 0 } }),
    };
    try {
      const report = await executeImport(missingDirection.Id);
      expect(report?.stats?.total).toBe(0);
    } finally {
      (DataTransferJob as any).Browse = browse;
    }

    const missingExportDirection = await DataTransferJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-missing-export-direction',
      SpecSnapshotJson: sampleExportSnapshot('base.Country'),
      Direction: 'export',
    } as Partial<DataTransferJob>);
    (DataTransferJob as any).Browse = async (_id: string, fields?: any) => {
      const row = await browse(_id, fields);
      if (!row) return row;
      return { ...(row as any), Direction: '' };
    };
    root.export = {
      run: async () => ({ stats: { total: 0, ok: 0, error: 0, skip: 0 } }),
    };
    try {
      const exportReport = await executeExport(missingExportDirection.Id);
      expect(exportReport?.stats?.total).toBe(0);
    } finally {
      (DataTransferJob as any).Browse = browse;
    }

    const withoutSnapshot = await DataTransferJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-no-snapshot',
      SpecSnapshotJson: { ok: true },
      Direction: 'import',
    } as Partial<DataTransferJob>);
    await (DataTransferJob as any).UpdateById(withoutSnapshot.Id, { SpecSnapshotJson: 0 } as any);
    await expectAsyncError(() => executeImport(withoutSnapshot.Id), /missing spec snapshot/);

    const withoutExportSnapshot = await DataTransferJob.Create({
      Profile: 'record',
      Policy: 'atomic',
      DryRun: false,
      TargetModel: 'base.Country',
      SourceRef: 'doc-no-export-snapshot',
      SpecSnapshotJson: sampleExportSnapshot('base.Country'),
      Direction: 'export',
    } as Partial<DataTransferJob>);
    await (DataTransferJob as any).UpdateById(withoutExportSnapshot.Id, { SpecSnapshotJson: 0 } as any);
    await expectAsyncError(() => executeExport(withoutExportSnapshot.Id), /missing spec snapshot/);

    delete root.import;
    const enqueued = await DataTransferJob.EnqueueRecordImport({
      targetModel: 'base.Country',
      sourceRef: 'doc-no-bridge',
      specSnapshot: sampleSnapshot('doc-no-bridge'),
    });
    await expectAsyncError(() => executeImport(enqueued.dataTransferJobId), /import bridge is not available/);

    root.import = {
      run: async () => null,
    };
    const nullReportJob = await DataTransferJob.EnqueueRecordImport({
      targetModel: 'base.Country',
      sourceRef: 'doc-null-report',
      specSnapshot: sampleSnapshot('doc-null-report'),
    });
    const empty = await executeImport(nullReportJob.dataTransferJobId);
    expect(empty).toEqual({});

    await DataTransferJob.FinalizeReport(nullReportJob.dataTransferJobId, {
      Stats: { Total: 3 },
      artifactRef: 'art-alt',
    });
    const finalized = await DataTransferJob.Browse(nullReportJob.dataTransferJobId, ['ProgressTotal', 'ReportRef'] as any);
    expect((finalized as any).ProgressTotal).toBe(3);
    expect((finalized as any).ReportRef).toBe('art-alt');

    await DataTransferJob.FinalizeReport(nullReportJob.dataTransferJobId, null as any);
    const cleared = await DataTransferJob.Browse(nullReportJob.dataTransferJobId, ['ReportJson', 'ProgressTotal'] as any);
    expect((cleared as any).ReportJson).toEqual({});
    expect((cleared as any).ProgressTotal).toBe(0);

    delete root.export;
    const exportJob = await DataTransferJob.EnqueueRecordExport({
      targetModel: 'base.Country',
      sourceRef: 'export:base.Country',
      specSnapshot: sampleExportSnapshot('base.Country'),
    });
    await expectAsyncError(() => executeExport(exportJob.dataTransferJobId), /export bridge is not available/);

    root.export = {
      run: async () => null,
    };
    const nullExportJob = await DataTransferJob.EnqueueRecordExport({
      targetModel: 'base.Country',
      sourceRef: 'export:base.Country',
      specSnapshot: sampleExportSnapshot('base.Country'),
    });
    const emptyExport = await executeExport(nullExportJob.dataTransferJobId);
    expect(emptyExport).toEqual({});

    root.export = {
      run: async () => undefined,
    };
    const undefinedExportJob = await DataTransferJob.EnqueueRecordExport({
      targetModel: 'base.Country',
      sourceRef: 'export:base.Country',
      specSnapshot: sampleExportSnapshot('base.Country'),
    });
    const undefinedExport = await executeExport(undefinedExportJob.dataTransferJobId);
    expect(undefinedExport).toEqual({});
  } finally {
    if (previousImport === undefined) {
      delete root.import;
    } else {
      root.import = previousImport;
    }
    if (root.export) {
      delete root.export;
    }
  }
});
