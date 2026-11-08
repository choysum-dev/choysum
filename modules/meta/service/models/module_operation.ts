// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { createServiceByModel } from '@/core/service/rpc';
import { resolveChoysum } from '@/core/service/runtime/choysum_runtime';
import type JobModel from '@/task/service/models/job';
import { _t, _lt } from '../i18n';
import { publishModuleOpChangedTip } from '../tips';

const Job = createServiceByModel<typeof JobModel>('task.Job');

export type ModuleOpAction = 'install' | 'upgrade' | 'uninstall' | 'sync' | 'load-demo';
export type ModuleOpOrigin = 'local' | 'registry';

export type PlanIssue = {
  Code: string;
  Level: string;
  Message?: string;
};

export type ModuleOrderEntry = {
  ModuleName: string;
  Reason?: string;
  CurrentVersion?: string;
  TargetVersion?: string;
  Origin?: string;
};

export type PlanModuleOperationReq = {
  ModuleName: string;
  Action: 'install' | 'upgrade' | 'uninstall';
  Origin?: ModuleOpOrigin;
};

export type PlanModuleOperationResp = {
  Id: string;
  Action: string;
  ModuleName: string;
  ModuleOrder: ModuleOrderEntry[];
  AffectedApps: string[];
  Risks: PlanIssue[];
  Blockers: PlanIssue[];
  BaseRevision?: string;
};

export type ApplyModuleOperationReq = {
  Id: string;
};

export type ApplyModuleOperationResp = {
  Id: string;
};

export type ExecuteModuleOperationReq = {
  Id: string;
  WithDemo?: boolean;
};

export type ExecuteModuleOperationResp = {
  Id: string;
  Action: string;
  ModuleName: string;
  Status: string;
  Phase: string;
  ErrorDomain?: string;
  ErrorCode?: string;
  ErrorMessage?: string;
  ReloadTriggered: boolean;
  ReloadFailed: boolean;
};

export type PlanDemoModuleOperationReq = Record<string, never>;

/** Empty envelope for interactive sync load-demo (Plan → Apply → Run via bridge). */
export type RequestLoadDemoReq = Record<string, never>;

export type RequestLoadDemoResp = {
  Ok: boolean;
  Id?: string;
  ErrorDomain?: string;
  ErrorCode?: string;
  ErrorMessage?: string;
};

type BridgePlanResult = {
  id?: string;
  ok?: boolean;
  action?: string;
  moduleName?: string;
  moduleOrder?: Array<{
    moduleName?: string;
    reason?: string;
    currentVersion?: string;
    targetVersion?: string;
    origin?: string;
  }>;
  affectedApps?: string[];
  risks?: Array<{ code?: string; level?: string; message?: string }>;
  blockers?: Array<{ code?: string; level?: string; message?: string }>;
  baseRevision?: string;
  errorDomain?: string;
  errorCode?: string;
  errorMessage?: string;
};

type BridgeOpResult = {
  ok?: boolean;
  id?: string;
  errorDomain?: string;
  errorCode?: string;
  errorMessage?: string;
};

const TERMINAL_OP_STATUSES = new Set(['succeeded', 'failed', 'cancelled']);

function mapPlanResult(result: BridgePlanResult | null | undefined, fallbackAction: string, fallbackModuleName = ''): PlanModuleOperationResp {
  if (result && result.ok === false) {
    throw new Error(result.errorMessage || _t('plan failed', { scope: 'service/models/module_operation' }));
  }
  const id = String(result?.id || '').trim();
  if (!id) {
    throw new Error(result?.errorMessage || _t('plan returned empty id', { scope: 'service/models/module_operation' }));
  }
  return {
    Id: id,
    Action: String(result?.action || fallbackAction),
    ModuleName: String(result?.moduleName || fallbackModuleName),
    ModuleOrder: mapModuleOrder(result?.moduleOrder),
    AffectedApps: (result?.affectedApps || []).map((name) => String(name || '').trim()).filter(Boolean),
    Risks: mapIssues(result?.risks),
    Blockers: mapIssues(result?.blockers),
    BaseRevision: result?.baseRevision != null ? String(result.baseRevision) : undefined,
  };
}

function mapIssues(list: BridgePlanResult['risks']): PlanIssue[] {
  return (list || []).map((issue) => ({
    Code: String(issue?.code || '').trim(),
    Level: String(issue?.level || '').trim(),
    Message: issue?.message != null ? String(issue.message) : undefined,
  }));
}

function mapModuleOrder(list: BridgePlanResult['moduleOrder']): ModuleOrderEntry[] {
  return (list || []).map((entry) => ({
    ModuleName: String(entry?.moduleName || '').trim(),
    Reason: entry?.reason != null ? String(entry.reason) : undefined,
    CurrentVersion: entry?.currentVersion != null ? String(entry.currentVersion) : undefined,
    TargetVersion: entry?.targetVersion != null ? String(entry.targetVersion) : undefined,
    Origin: entry?.origin != null ? String(entry.origin) : undefined,
  }));
}

function getModuleManagementBridge() {
  const root = resolveChoysum();
  if (!root?.moduleManagement) {
    throw new Error('moduleManagement bridge is not injected');
  }
  return root.moduleManagement;
}

@Model('MetaModuleOperation', {
  tableName: 'meta_module_operation',
  autoMigrate: false,
})
export default class MetaModuleOperation extends BaseModel {
  @Field({ type: 'varchar', size: 32, index: true, notNull: true, string: _lt('Action', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  Action!: string;

  @Field({ type: 'varchar', size: 255, index: true, string: _lt('Module Name', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  ModuleName?: string;

  @Field({ type: 'jsonobject', string: _lt('Plan', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  PlanJson?: Record<string, unknown> | null;

  @Field({ type: 'varchar', size: 32, index: true, notNull: true, string: _lt('Status', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  Status!: string;

  @Field({ type: 'varchar', size: 32, index: true, string: _lt('Phase', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  Phase?: string;

  @Field({ type: 'varchar', size: 128, string: _lt('Error Domain', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  ErrorDomain?: string;

  @Field({ type: 'varchar', size: 128, string: _lt('Error Code', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  ErrorCode?: string;

  @Field({ type: 'text', string: _lt('Error Message', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  ErrorMessage?: string;

  @Field({ type: 'boolean', string: _lt('Reload Triggered', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  ReloadTriggered?: boolean;

  @Field({ type: 'boolean', string: _lt('Reload Failed', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  ReloadFailed?: boolean;

  @Field({ type: 'varchar', size: 64, index: true, string: _lt('Operator', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  OperatorUserId?: string;

  @Field({ type: 'varchar', size: 64, unique: true, string: _lt('Job', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  JobId?: string;

  @Field({ type: 'datetime', index: true, string: _lt('Started At', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  StartedAt?: Date;

  @Field({ type: 'datetime', index: true, string: _lt('Finished At', { scope: 'meta.model.MetaModuleOperation.fields' }) })
  FinishedAt?: Date;

  static async Plan(req: PlanModuleOperationReq): Promise<PlanModuleOperationResp> {
    const operatorUserId = BaseModel.ensureUserId();
    const action = String(req?.Action || '').trim().toLowerCase();
    if (action !== 'install' && action !== 'upgrade' && action !== 'uninstall') {
      throw new Error(_t('action must be install, upgrade, or uninstall', { scope: 'service/models/module_operation' }));
    }
    const moduleName = String(req?.ModuleName || '').trim();
    if (!moduleName) {
      throw new Error(_t('moduleName cannot be empty', { scope: 'service/models/module_operation' }));
    }
    const origin = String(req?.Origin || '').trim().toLowerCase();
    const bridge = getModuleManagementBridge();
    const result = (await bridge.plan({
      moduleName,
      action,
      origin: origin === 'registry' ? 'registry' : 'local',
      operatorUserId,
    })) as BridgePlanResult;

    return mapPlanResult(result, action, moduleName);
  }

  static async PlanDemo(_req: PlanDemoModuleOperationReq = {}): Promise<PlanModuleOperationResp> {
    void _req;
    const operatorUserId = BaseModel.ensureUserId();
    const bridge = getModuleManagementBridge();
    const result = (await bridge.plan({
      action: 'load-demo',
      operatorUserId,
    })) as BridgePlanResult;

    return mapPlanResult(result, 'load-demo');
  }

  /**
   * Load pending demo data for installed modules (HasDemo && !DemoLoaded).
   * Runs Plan → Apply → Run synchronously via the moduleManagement bridge;
   * soft-succeeds when nothing is pending.
   */
  static async RequestLoadDemo(_req: RequestLoadDemoReq = {}): Promise<RequestLoadDemoResp> {
    void _req;
    const operatorUserId = BaseModel.ensureUserId();
    const bridge = getModuleManagementBridge();
    const result = (await bridge.loadDemo({ operatorUserId })) as BridgeOpResult;
    if (result && result.ok === false) {
      return {
        Ok: false,
        Id: result.id != null ? String(result.id) : undefined,
        ErrorDomain: result.errorDomain != null ? String(result.errorDomain) : undefined,
        ErrorCode: result.errorCode != null ? String(result.errorCode) : undefined,
        ErrorMessage:
          result.errorMessage != null
            ? String(result.errorMessage)
            : _t('load demo failed', { scope: 'service/models/module_operation' }),
      };
    }
    return {
      Ok: true,
      Id: result?.id != null ? String(result.id) : undefined,
    };
  }

  static async Apply(req: ApplyModuleOperationReq): Promise<ApplyModuleOperationResp> {
    const id = String(req?.Id || '').trim();
    if (!id) {
      throw new Error(_t('operation id cannot be empty', { scope: 'service/models/module_operation' }));
    }
    const bridge = getModuleManagementBridge();
    let applyResult: BridgeOpResult | null | undefined;
    try {
      applyResult = (await bridge.apply({ id })) as BridgeOpResult | null | undefined;
    } catch (applyErr) {
      // Plan left a draft; mark failed so callers are not stuck on a non-runnable op.
      await this.UpdateById(
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
      const message = applyResult.errorMessage || _t('apply failed', { scope: 'service/models/module_operation' });
      await this.UpdateById(id, { Status: 'failed', ErrorMessage: message }, ['Id']);
      throw new Error(message);
    }

    const userId = BaseModel.ensureUserId();
    let job: unknown;
    try {
      // Apply never carries WithDemo; Execute defaults WithDemo to false.
      job = await Job.EnqueueJob({
        TargetApp: 'meta',
        FullMethod: 'meta.MetaModuleOperation/Execute',
        Payload: { req: { Id: id, WithDemo: false } },
        MaxAttempts: 0,
        TimeoutMs: 0,
      });
    } catch (enqueueErr) {
      // bridge.apply already advanced the row; mark failed so it does not hang active forever.
      await this.UpdateById(
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
      const message = _t('enqueue returned empty job id', { scope: 'service/models/module_operation' });
      await this.UpdateById(id, { Status: 'failed', ErrorMessage: message }, ['Id']);
      throw new Error(message);
    }
    await this.UpdateById(id, { JobId: jobId }, ['Id']);
    await publishModuleOpChangedTip({ operationId: id, jobId, userId });
    return { Id: id };
  }

  /** Task-worker FullMethod target for queued module operations (not interactive client API). */
  static async Execute(req: ExecuteModuleOperationReq): Promise<ExecuteModuleOperationResp> {
    const id = String(req?.Id || '').trim();
    if (!id) {
      throw new Error(_t('operation id cannot be empty', { scope: 'service/models/module_operation' }));
    }
    const withDemo = !!req?.WithDemo;
    const bridge = getModuleManagementBridge();
    let runErr: unknown;
    try {
      const runResult = (await bridge.run({ id, withDemo })) as BridgeOpResult | null | undefined;
      if (runResult && runResult.ok === false) {
        // Prefer persisted terminal row below; if status stays non-terminal, rethrow.
        runErr = new Error(
          runResult.errorMessage || _t('run failed', { scope: 'service/models/module_operation' }),
        );
      }
    } catch (err) {
      // ops.Run usually persists failure; if the row stays non-terminal, rethrow so the job fails.
      runErr = err;
    }

    const rows = await this.Search(['Id', '=', id], {
      limit: 1,
      fields: [
        'Id',
        'Action',
        'ModuleName',
        'Status',
        'Phase',
        'ErrorDomain',
        'ErrorCode',
        'ErrorMessage',
        'ReloadTriggered',
        'ReloadFailed',
      ],
    });
    const row = rows?.[0];
    if (!row) {
      if (runErr) throw runErr;
      throw new Error(_t('operation not found', { scope: 'service/models/module_operation' }));
    }
    const status = String(row.Status || '').toLowerCase();
    if (runErr && !TERMINAL_OP_STATUSES.has(status)) {
      throw runErr;
    }
    return {
      Id: String(row.Id || id),
      Action: String(row.Action || ''),
      ModuleName: String(row.ModuleName || ''),
      Status: String(row.Status || ''),
      Phase: String(row.Phase || ''),
      ErrorDomain: row.ErrorDomain != null ? String(row.ErrorDomain) : undefined,
      ErrorCode: row.ErrorCode != null ? String(row.ErrorCode) : undefined,
      ErrorMessage: row.ErrorMessage != null ? String(row.ErrorMessage) : undefined,
      ReloadTriggered: !!row.ReloadTriggered,
      ReloadFailed: !!row.ReloadFailed,
    };
  }
}
