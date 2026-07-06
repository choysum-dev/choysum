// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getCtxValue, getUserId } from '@/core/service/api/context';
import { getOperatorUserId, getModuleManagement } from '@/core/service/utils/bridge';
import Job from '@/task/service/models/job';
import IrApplication from './ir_application';
import IrComponent from './ir_component';
import IrModel from './ir_model';
import IrModuleDependency from './ir_module_dependency';
import IrUiResource from './ir_ui_resource';
import ModuleManagementLog from './module_management_log';

type ModuleAction = 'install' | 'uninstall' | 'upgrade';
type FailureKind = 'RETRYABLE' | 'NON_RETRYABLE' | 'NONE';

type PlanOperationReq = {
  action: ModuleAction;
  moduleName: string;
  withDemo?: boolean;
  baseRevision?: string;
};

type PlanOperationResp = {
  baseRevision: string;
  affectedModules: Array<{ moduleName: string; reason?: string; currentVersion?: string; targetVersion?: string }>;
  risks: Array<{ code: string; level: string; message?: string; params?: Record<string, any> }>;
  blockers: Array<{ code: string; level: string; message?: string; params?: Record<string, any> }>;
};

type OpStatusResp = {
  status: string;
  summary?: any;
  resultStatus?: 'SUCCEEDED' | 'FAILED';
  failureKind?: FailureKind;
  createdAt?: Date;
  startedAt?: Date;
  finishedAt?: Date;
  reload_web?: boolean;
  reload_triggered?: boolean;
  reload_failed?: boolean;
  moduleName?: string;
  action?: ModuleAction;
  operatorUserId?: string;
  attempt?: number;
  maxAttempts?: number;
  nextRetryAt?: Date;
  retryAfterMs?: number;
  errorDomain?: string;
  errorCode?: string;
  errorMessage?: string;
};

type ModuleOpBridgeResult = {
  ok: boolean;
  errorDomain?: string;
  errorCode?: string;
  errorMessage?: string;
};

function ensureModuleName(name?: string): string {
  const trimmed = String(name || '').trim();
  if (!trimmed) throw new Error('moduleName cannot be empty');
  return trimmed;
}


function normalizeFailureKind(status: string, err?: any): FailureKind {
  if (status === 'cancelled') return 'NON_RETRYABLE';
  if (status !== 'failed') return 'NONE';
  const domain = String(err?.domain || err?.errorDomain || '').toLowerCase();
  const code = String(err?.code || err?.errorCode || '').toUpperCase();
  if (domain === 'meta.lock' && code === 'LEASE_CONFLICT') return 'RETRYABLE';
  if (domain === 'module_management' && code === 'LOCK_LEASE_LOST') return 'RETRYABLE';
  return 'NON_RETRYABLE';
}

async function loadExecutionTimes(jobId: string): Promise<{ startedAt?: Date; finishedAt?: Date }> {
  if (!jobId) return {};
  const root: any = (globalThis as any)?.$choysum;
  if (!root?.db?.query) return {};
  const raw = await root.db.query(
    'SELECT started_at, finished_at FROM task_job_execution WHERE job_id = ? ORDER BY created_at DESC LIMIT 1',
    JSON.stringify([jobId])
  );
  const rows = JSON.parse(raw || '[]');
  const row = rows?.[0] || {};
  return {
    startedAt: row.started_at ? new Date(row.started_at) : undefined,
    finishedAt: row.finished_at ? new Date(row.finished_at) : undefined,
  };
}

async function findModuleLogByJobId(jobId: string): Promise<any> {
  if (!jobId) return undefined;
  const existing = await ModuleManagementLog.Search(['JobId', '=', jobId] as any, { limit: 1 } as any);
  return existing?.[0];
}

async function upsertModuleLog(values: Partial<ModuleManagementLog>): Promise<void> {
  if (!values.JobId) return;
  const existing = await findModuleLogByJobId(values.JobId);
  if (existing?.Id) {
    await (ModuleManagementLog as any).UpdateById(existing.Id, values as any);
    return;
  }
  try {
    await ModuleManagementLog.Create(values as any);
  } catch (err) {
    const record = await findModuleLogByJobId(values.JobId);
    if (record?.Id) {
      await (ModuleManagementLog as any).UpdateById(record.Id, values as any);
      return;
    }
    throw err;
  }
}

@Model('IrModule', {
  tableName: 'meta_ir_module',
  autoMigrate: false,
})
export default class IrModule extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, unique: true, notNull: true } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 1024 } })
  ShortDesc?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Version?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Tarball?: string;

  @Field({ type: 'text' })
  Summary?: string;

  @Field({ type: 'text' })
  Description?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ApplicationStr?: string;

  @Field({ type: 'jsonobject' })
  EntryPoints?: Record<string, unknown> | null;

  @Field({ type: 'varchar', column: { size: 512 } })
  WebEntryPoint?: string;

  @Field({ type: 'varchar', column: { size: 512 } })
  ServiceEntryPoint?: string;

  @Field({ type: 'jsonobject' })
  DependsStr?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'jsonobject' })
  ExternalDependencies?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'varchar', column: { size: 255 } })
  Author?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  License?: string;

  @Field({ type: 'text' })
  Homepage?: string;

  @Field({ type: 'text' })
  Repository?: string;

  @Field({ type: 'varchar', column: { size: 512 } })
  Path?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Status?: string;

  @Field({ type: 'varchar', column: { size: 255, index: true } })
  Category?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrApplication } })
  ApplicationId?: IrApplication;

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrModel, inverseField: 'ModuleId' } })
  Models?: IrModel[];

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrComponent, inverseField: 'ModuleId' } })
  Components?: IrComponent[];

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrUiResource, inverseField: 'ModuleId' } })
  UiResources?: IrUiResource[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => IrModuleDependency,
      targetModel: () => IrModule,
      joinField: 'ModuleId',
      inverseJoinField: 'DependModuleId',
    },
  })
  Dependencies?: IrModule[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => IrModuleDependency,
      targetModel: () => IrModule,
      joinField: 'DependModuleId',
      inverseJoinField: 'ModuleId',
    },
  })
  Dependents?: IrModule[];

  static async PlanOperation(req: PlanOperationReq): Promise<PlanOperationResp> {
    const action = (req?.action || 'install') as ModuleAction;
    const moduleName = ensureModuleName(req?.moduleName);
    const risks: PlanOperationResp['risks'] = [];
    const blockers: PlanOperationResp['blockers'] = [];
    const affectedModules: PlanOperationResp['affectedModules'] = [];

    const latest = await this.Search([] as any, {
      limit: 1,
      orderBy: { field: 'UpdatedAt', order: 'desc' } as any,
      fields: ['UpdatedAt', 'Version', 'Status', 'Name'] as any,
    });
    const baseRevision = latest?.[0]?.UpdatedAt ? String(new Date(latest[0].UpdatedAt).getTime()) : '0';

    if (req?.baseRevision && req.baseRevision !== baseRevision) {
      risks.push({
        code: 'PLAN_REVISION_MISMATCH',
        level: 'WARN',
        message: 'plan was regenerated from the latest module state',
        params: { baseRevision },
      });
    }

    const existing = await this.Search(['Name', '=', moduleName] as any, { limit: 1, fields: ['Version', 'Status'] as any });
    const current = existing?.[0];

    if ((action === 'uninstall' || action === 'upgrade') && !current) {
      blockers.push({ code: 'MODULE_NOT_FOUND', level: 'BLOCKER', message: 'module not found', params: { moduleName } });
    }

    affectedModules.push({
      moduleName,
      reason: action,
      currentVersion: current?.Version,
    });

    return { baseRevision, affectedModules, risks, blockers };
  }

  static async RequestInstall(moduleName: string, withDemo?: boolean): Promise<string> {
    const name = ensureModuleName(moduleName);
    const userId = getOperatorUserId();
    const env = (import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {};
    const forceLockConflict = Boolean((env as any).CHOYSUM_E2E_FORCE_LOCK_CONFLICT || (env as any).choysum_e2e_force_lock_conflict);
    const job = await Job.EnqueueJob(
      'meta',
      'meta.IrModule/ExecuteInstall',
      { moduleName: name, withDemo: !!withDemo, operatorUserId: userId },
      userId,
      userId,
      undefined,
      0,
      0
    );
    if (forceLockConflict && (job as any)?.Id) {
      const retryAfterMs = 2500;
      await (Job as any).UpdateById((job as any).Id, {
        Status: 'failed',
        RunAfter: new Date(Date.now() + retryAfterMs),
        LastErrorJson: {
          domain: 'meta.lock',
          code: 'LEASE_CONFLICT',
          message: 'lease conflict',
          details: { retry_after_ms: retryAfterMs },
        },
      });
    }
    return String((job as any)?.Id || '').trim();
  }

  static async RequestUninstall(moduleName: string): Promise<string> {
    const name = ensureModuleName(moduleName);
    const userId = getOperatorUserId();
    const env = (import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {};
    const forceLockConflict = Boolean((env as any).CHOYSUM_E2E_FORCE_LOCK_CONFLICT || (env as any).choysum_e2e_force_lock_conflict);
    const job = await Job.EnqueueJob('meta', 'meta.IrModule/ExecuteUninstall', { moduleName: name, operatorUserId: userId }, userId, userId, undefined, 0, 0);
    if (forceLockConflict && (job as any)?.Id) {
      const retryAfterMs = 2500;
      await (Job as any).UpdateById((job as any).Id, {
        Status: 'failed',
        RunAfter: new Date(Date.now() + retryAfterMs),
        LastErrorJson: {
          domain: 'meta.lock',
          code: 'LEASE_CONFLICT',
          message: 'lease conflict',
          details: { retry_after_ms: retryAfterMs },
        },
      });
    }
    return String((job as any)?.Id || '').trim();
  }

  static async RequestUpgrade(moduleName: string): Promise<string> {
    const name = ensureModuleName(moduleName);
    const userId = getOperatorUserId();
    const env = (import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {};
    const forceLockConflict = Boolean((env as any).CHOYSUM_E2E_FORCE_LOCK_CONFLICT || (env as any).choysum_e2e_force_lock_conflict);
    const job = await Job.EnqueueJob('meta', 'meta.IrModule/ExecuteUpgrade', { moduleName: name, operatorUserId: userId }, userId, userId, undefined, 0, 0);
    if (forceLockConflict && (job as any)?.Id) {
      const retryAfterMs = 2500;
      await (Job as any).UpdateById((job as any).Id, {
        Status: 'failed',
        RunAfter: new Date(Date.now() + retryAfterMs),
        LastErrorJson: {
          domain: 'meta.lock',
          code: 'LEASE_CONFLICT',
          message: 'lease conflict',
          details: { retry_after_ms: retryAfterMs },
        },
      });
    }
    return String((job as any)?.Id || '').trim();
  }

  static async GetOpStatus(jobId: string): Promise<OpStatusResp> {
    const id = String(jobId || '').trim();
    if (!id) throw new Error('jobId cannot be empty');

    const job = await Job.GetJob(id, [
      'Id',
      'Status',
      'ResultJson',
      'LastErrorJson',
      'CreatedAt',
      'FinishedAt',
      'RunAfter',
      'Attempt',
      'MaxAttempts',
      'PayloadJson',
      'FullMethod',
    ] as any);
    const exec = await loadExecutionTimes(id);
    const result = (job as any)?.ResultJson && typeof (job as any).ResultJson === 'object' ? (job as any).ResultJson : {};
    const err = (job as any)?.LastErrorJson && typeof (job as any).LastErrorJson === 'object' ? (job as any).LastErrorJson : undefined;
    const payload = (job as any)?.PayloadJson && typeof (job as any).PayloadJson === 'object' ? (job as any).PayloadJson : {};

    const status = String((job as any)?.Status || '').toLowerCase();
    const method = String((job as any)?.FullMethod || '');
    let action: ModuleAction | undefined = result?.action || payload?.action;
    if (!action) {
      if (method.endsWith('ExecuteInstall')) action = 'install';
      else if (method.endsWith('ExecuteUninstall')) action = 'uninstall';
      else if (method.endsWith('ExecuteUpgrade')) action = 'upgrade';
    }

    const retryAfterMs = err?.details?.retry_after_ms ? Number(err.details.retry_after_ms) : undefined;
    const nextRetryAt = retryAfterMs && (job as any)?.RunAfter ? new Date((job as any).RunAfter) : undefined;

    const resultStatus = result?.resultStatus || (status === 'failed' || status === 'cancelled' ? 'FAILED' : undefined);
    let failureKind = normalizeFailureKind(status, err);
    if (resultStatus === 'FAILED' && failureKind === 'NONE') {
      failureKind = normalizeFailureKind('failed', err);
    }
    const summary = result?.summary || (resultStatus === 'FAILED' ? { code: 'MODULE_OPERATION_FAILED', message: err?.message } : undefined);

    return {
      status,
      summary,
      resultStatus,
      failureKind,
      createdAt: (job as any)?.CreatedAt,
      startedAt: exec?.startedAt,
      finishedAt: (job as any)?.FinishedAt || exec?.finishedAt,
      reload_web: result?.reload_web,
      reload_triggered: result?.reload_triggered,
      reload_failed: result?.reload_failed,
      moduleName: result?.moduleName || payload?.moduleName,
      action,
      operatorUserId: result?.operatorUserId || payload?.operatorUserId,
      attempt: (job as any)?.Attempt,
      maxAttempts: (job as any)?.MaxAttempts,
      nextRetryAt,
      retryAfterMs,
      errorDomain: err?.domain || result?.errorDomain,
      errorCode: err?.code || result?.errorCode,
      errorMessage: err?.message || result?.errorMessage,
    } as OpStatusResp;
  }

  static async ExecuteInstall(moduleName: string, withDemo?: boolean, operatorUserId?: string): Promise<any> {
    return await this.executeModuleOp('install', moduleName, { withDemo, operatorUserId });
  }

  static async ExecuteUninstall(moduleName: string, operatorUserId?: string): Promise<any> {
    return await this.executeModuleOp('uninstall', moduleName, { operatorUserId });
  }

  static async ExecuteUpgrade(moduleName: string, operatorUserId?: string): Promise<any> {
    return await this.executeModuleOp('upgrade', moduleName, { operatorUserId });
  }

  private static async executeModuleOp(action: ModuleAction, moduleName: string, opts: { withDemo?: boolean; operatorUserId?: string }): Promise<any> {
    const name = ensureModuleName(moduleName);
    const operatorUserId = String(opts?.operatorUserId || getUserId() || '').trim();
    const jobId = String(getCtxValue('jobId') || '').trim();
    const bridge = getModuleManagement();

    const env = (import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {};
    const forceResultStatus = String((env as any).CHOYSUM_E2E_FORCE_RESULT_STATUS || (env as any).choysum_e2e_force_result_status || '').toUpperCase();
    const forceReloadFailed = Boolean((env as any).CHOYSUM_E2E_FORCE_RELOAD_FAILED || (env as any).choysum_e2e_force_reload_failed);

    let bridgeResult: ModuleOpBridgeResult;
    if (forceResultStatus === 'FAILED') {
      bridgeResult = { ok: false, errorDomain: 'MODULE_MANAGEMENT', errorCode: 'OP_FAILED', errorMessage: 'E2E forced failure' };
    } else {
      bridgeResult = (await bridge[action]({
        moduleName: name,
        withDemo: !!opts?.withDemo,
        operatorUserId,
        jobId,
        action,
      })) as ModuleOpBridgeResult;
    }

    let resultStatus: 'SUCCEEDED' | 'FAILED' = bridgeResult.ok ? 'SUCCEEDED' : 'FAILED';
    let summary: any;
    let errorDomain = bridgeResult.errorDomain;
    let errorCode = bridgeResult.errorCode;
    let errorMessage = bridgeResult.errorMessage;

    if (bridgeResult.ok) {
      if (action === 'install') summary = { code: 'MODULE_INSTALLED', params: { moduleName: name } };
      else if (action === 'uninstall') summary = { code: 'MODULE_UNINSTALLED', params: { moduleName: name } };
      else summary = { code: 'MODULE_UPGRADED', params: { moduleName: name } };
    } else {
      summary = {
        code: 'MODULE_OPERATION_FAILED',
        params: { moduleName: name, action },
        message: errorMessage,
      };
      if (!errorDomain) errorDomain = 'MODULE_MANAGEMENT';
      if (!errorCode) errorCode = 'OP_FAILED';
    }

    let reload_triggered = false;
    let reload_failed = false;
    let reload_web = false;
    const skipReload = Boolean((env as any)?.CHOYSUM_E2E_SKIP_RELOAD || (env as any)?.choysum_e2e_skip_reload);
    if (bridgeResult.ok && !skipReload) {
      try {
        const reloadResult = await bridge.reload();
        reload_triggered = !!reloadResult?.triggered;
        reload_failed = !!reloadResult?.failed;
        reload_web = reload_triggered && !reload_failed;
      } catch (err) {
        reload_triggered = true;
        reload_failed = true;
        reload_web = false;
      }
    }
    if (bridgeResult.ok && forceReloadFailed) {
      reload_triggered = true;
      reload_failed = true;
      reload_web = false;
    }

    let job: any;
    if (jobId) {
      try {
        job = await Job.GetJob(jobId, ['Id', 'CreatedAt', 'FinishedAt', 'Attempt', 'MaxAttempts'] as any);
      } catch {
        job = undefined;
      }
    }

    await upsertModuleLog({
      JobId: jobId,
      ModuleName: name,
      Action: action,
      OperatorUserId: operatorUserId,
      ResultStatus: resultStatus,
      SummaryJson: summary,
      ErrorDomain: errorDomain,
      ErrorCode: errorCode,
      LastErrorJson: bridgeResult.ok ? undefined : { message: errorMessage, domain: errorDomain, code: errorCode },
      JobCreatedAt: job?.CreatedAt,
      JobFinishedAt: job?.FinishedAt,
      Attempt: job?.Attempt,
      MaxAttempts: job?.MaxAttempts,
    });

    return {
      resultStatus,
      summary,
      errorDomain,
      errorCode,
      errorMessage,
      reload_triggered,
      reload_failed,
      reload_web,
      moduleName: name,
      action,
      operatorUserId,
    };
  }
}
