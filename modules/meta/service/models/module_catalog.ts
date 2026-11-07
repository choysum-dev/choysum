// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import { resolveChoysum } from '@/core/service/runtime/choysum_runtime';
import type JobModel from '@/task/service/models/job';
import { _t, _lt } from '../i18n';
import { publishModuleOpChangedTip } from '../tips';
import MetaModuleOperation from './module_operation';

const Job = createServiceByModel<typeof JobModel>('task.Job');

export type RequestSyncReq = {
  Force?: boolean;
  IfStale?: boolean;
};

export type RequestSyncResp = {
  Id: string;
};

const ACTIVE_SYNC_STATUSES = ['draft', 'queued', 'running'];
// RequestSync currently plans a single local-origin sync; use the local freshness window.
const LOCAL_STALE_MS = 60 * 1000;

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

  static async RequestSync(req: RequestSyncReq = {}): Promise<RequestSyncResp> {
    const force = req?.Force !== undefined ? !!req.Force : true;
    const ifStale = !!req?.IfStale;

    if (ifStale && !force) {
      const latest = await this.Search([], {
        limit: 1,
        orderBy: { field: 'LastSyncAt', order: 'desc' },
        fields: ['LastSyncAt'],
      });
      const lastSyncAt = latest?.[0]?.LastSyncAt ? new Date(latest[0].LastSyncAt as Date).getTime() : NaN;
      if (!isNaN(lastSyncAt) && Date.now() - lastSyncAt < LOCAL_STALE_MS) {
        return { Id: '' };
      }
    }

    const active = await MetaModuleOperation.Search(
      {
        And: [
          ['Action', '=', 'sync'],
          ['Status', 'in', ACTIVE_SYNC_STATUSES],
        ],
      },
      { limit: 1, orderBy: { field: 'CreatedAt', order: 'desc' }, fields: ['Id', 'Status'] }
    );
    const existingId = String(active?.[0]?.Id || '').trim();
    if (existingId) {
      return { Id: existingId };
    }

    const userId = BaseModel.ensureUserId();
    const bridge = getModuleManagementBridge();
    const operatorUserId = String(getUserId() || '').trim();
    const planResult = (await bridge.plan({
      action: 'sync',
      origin: 'local',
      operatorUserId,
    })) as { id?: string; ok?: boolean; errorMessage?: string };
    if (planResult && planResult.ok === false) {
      throw new Error(planResult.errorMessage || _t('sync plan failed', { scope: 'service/models/module_catalog' }));
    }
    const id = String(planResult?.id || '').trim();
    if (!id) {
      throw new Error(planResult?.errorMessage || _t('sync plan returned empty id', { scope: 'service/models/module_catalog' }));
    }

    const applyResult = (await bridge.apply({ id })) as { ok?: boolean; errorMessage?: string } | null | undefined;
    if (applyResult && applyResult.ok === false) {
      throw new Error(applyResult.errorMessage || _t('sync apply failed', { scope: 'service/models/module_catalog' }));
    }

    const job = await Job.EnqueueJob({
      TargetApp: 'meta',
      FullMethod: 'meta.MetaModuleOperation/Execute',
      Payload: { req: { Id: id } },
      MaxAttempts: 0,
      TimeoutMs: 0,
    });
    const jobId = String((job as { Id?: unknown })?.Id || '').trim();
    if (jobId) {
      await MetaModuleOperation.UpdateById(id, { JobId: jobId }, ['Id']);
      await publishModuleOpChangedTip({ operationId: id, jobId, userId, source: 'meta.MetaModuleCatalog.RequestSync' });
    }
    return { Id: id };
  }
}
