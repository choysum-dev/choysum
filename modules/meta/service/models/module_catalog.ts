// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { createServiceByModel } from '@/core/service/rpc';
import { resolveChoysum } from '@/core/service/runtime/choysum_runtime';
import type JobModel from '@/task/service/models/job';
import { _t, _lt } from '../i18n';
import { publishModuleOpChangedTip } from '../tips';
import MetaModuleOperation from './module_operation';

const Job = createServiceByModel<typeof JobModel>('task.Job');

export type RequestSyncReq = {
  /** Reserved for callers that always intend a refresh; defaults to true. */
  Force?: boolean;
};

export type RequestSyncResp = {
  Id: string;
};

const ACTIVE_SYNC_STATUSES = ['draft', 'queued', 'running'];
// Drop abandoned non-terminal sync rows so RequestSync cannot be blocked forever.
const ACTIVE_SYNC_STALE_MS = 10 * 60 * 1000;

function getModuleManagementBridge() {
  const root = resolveChoysum();
  if (!root?.moduleManagement) {
    throw new Error('moduleManagement bridge is not injected');
  }
  return root.moduleManagement;
}

@Model('MetaModuleCatalog', {
  tableName: 'meta_module_catalog',
  autoMigrate: false,
})
export default class MetaModuleCatalog extends BaseModel {
  @Field({ type: 'varchar', size: 255, unique: true, notNull: true, string: _lt('Module Name', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  ModuleName!: string;

  @Field({ type: 'text', string: _lt('Summary', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Summary?: string;

  @Field({ type: 'text', string: _lt('Description', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Description?: string;

  @Field({ type: 'varchar', size: 255, index: true, string: _lt('Category', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Category?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Application', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Application?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Author', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Author?: string;

  @Field({ type: 'text', string: _lt('License', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  License?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Local Version', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  LocalVersion?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Registry Version', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  RegistryVersion?: string;

  @Field({ type: 'boolean', index: true, string: _lt('Installed', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Installed?: boolean;

  @Field({ type: 'varchar', size: 255, string: _lt('Installed Version', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  InstalledVersion?: string;

  @Field({ type: 'boolean', index: true, string: _lt('Update Available', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  UpdateAvailable?: boolean;

  @Field({ type: 'boolean', string: _lt('Registry Newer Than Local', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  RegistryNewerThanLocal?: boolean;

  @Field({ type: 'boolean', index: true, string: _lt('Available', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  Available?: boolean;

  @Field({ type: 'text', string: _lt('Unavailable Reason', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  UnavailableReason?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Local Path', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  LocalPath?: string;

  @Field({ type: 'jsonobject', string: _lt('Depends', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  DependsJson?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'jsonobject', string: _lt('Dependents', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  DependentsJson?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'boolean', string: _lt('Has Demo', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  HasDemo?: boolean;

  @Field({ type: 'boolean', string: _lt('Demo Loaded', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  DemoLoaded?: boolean;

  @Field({ type: 'datetime', string: _lt('Last Sync At', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  LastSyncAt?: Date;

  @Field({ type: 'varchar', size: 20, index: true, string: _lt('Active Operation', { scope: 'meta.model.MetaModuleCatalog.fields' }) })
  ActiveOperationId?: string;

  /**
   * Enqueue a catalog sync (local + registry snapshots, then materialize).
   * Reuses a non-stale in-flight sync operation when one exists.
   */
  static async RequestSync(_req: RequestSyncReq = {}): Promise<RequestSyncResp> {
    const active = await MetaModuleOperation.Search(
      {
        And: [
          ['Action', '=', 'sync'],
          ['Status', 'in', ACTIVE_SYNC_STATUSES],
        ],
      },
      { limit: 1, orderBy: { field: 'CreatedAt', order: 'desc' }, fields: ['Id', 'Status', 'CreatedAt'] }
    );
    const activeRow = active?.[0];
    const existingId = String(activeRow?.Id || '').trim();
    if (existingId) {
      const activeCreatedAt = activeRow?.CreatedAt ? new Date(activeRow.CreatedAt as Date).getTime() : NaN;
      // Missing/invalid CreatedAt cannot age out otherwise, so treat it as stale.
      const activeIsStale = isNaN(activeCreatedAt) || Date.now() - activeCreatedAt > ACTIVE_SYNC_STALE_MS;
      if (!activeIsStale) {
        return { Id: existingId };
      }
      await MetaModuleOperation.UpdateById(
        existingId,
        {
          Status: 'failed',
          ErrorMessage: _t('sync operation timed out', { scope: 'service/models/module_catalog' }),
        },
        ['Id'],
      );
    }

    const userId = BaseModel.ensureUserId();
    const bridge = getModuleManagementBridge();
    // Plan ignores origin for sync; Execute runs SyncSourceSnapshotsThenMaterialize("all").
    const planResult = (await bridge.plan({
      action: 'sync',
      operatorUserId: userId,
    })) as { id?: string; ok?: boolean; errorMessage?: string };
    if (planResult && planResult.ok === false) {
      throw new Error(planResult.errorMessage || _t('sync plan failed', { scope: 'service/models/module_catalog' }));
    }
    const id = String(planResult?.id || '').trim();
    if (!id) {
      throw new Error(planResult?.errorMessage || _t('sync plan returned empty id', { scope: 'service/models/module_catalog' }));
    }

    let applyResult: { ok?: boolean; errorMessage?: string } | null | undefined;
    try {
      applyResult = (await bridge.apply({ id })) as { ok?: boolean; errorMessage?: string } | null | undefined;
    } catch (applyErr) {
      await MetaModuleOperation.UpdateById(
        id,
        {
          Status: 'failed',
          ErrorMessage: String((applyErr as Error)?.message || applyErr),
        },
        ['Id'],
      );
      throw applyErr;
    }
    if (applyResult && applyResult.ok === false) {
      const message = applyResult.errorMessage || _t('sync apply failed', { scope: 'service/models/module_catalog' });
      await MetaModuleOperation.UpdateById(id, { Status: 'failed', ErrorMessage: message }, ['Id']);
      throw new Error(message);
    }

    let job: unknown;
    try {
      job = await Job.EnqueueJob({
        TargetApp: 'meta',
        FullMethod: 'meta.MetaModuleOperation/Execute',
        Payload: { req: { Id: id } },
        MaxAttempts: 0,
        TimeoutMs: 0,
      });
    } catch (enqueueErr) {
      // Avoid leaving an applied sync op stuck in ACTIVE_SYNC_STATUSES with no worker.
      await MetaModuleOperation.UpdateById(
        id,
        {
          Status: 'failed',
          ErrorMessage: String((enqueueErr as Error)?.message || enqueueErr),
        },
        ['Id'],
      );
      throw enqueueErr;
    }
    const jobId = String((job as { Id?: unknown })?.Id || '').trim();
    if (!jobId) {
      const message = _t('enqueue returned empty job id', { scope: 'service/models/module_catalog' });
      await MetaModuleOperation.UpdateById(id, { Status: 'failed', ErrorMessage: message }, ['Id']);
      throw new Error(message);
    }
    await MetaModuleOperation.UpdateById(id, { JobId: jobId }, ['Id']);
    await publishModuleOpChangedTip({ operationId: id, jobId, userId, source: 'meta.MetaModuleCatalog.RequestSync' });
    return { Id: id };
  }
}
