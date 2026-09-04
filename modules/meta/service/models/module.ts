// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getCtxValue, getUserId } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import type JobModel from '@/task/service/models/job';
import { getBackendEnvText, isTruthyFlag } from '@/core/service/runtime/env/backend_env';
import { _t, _lt } from '../i18n';
import { publishModuleOpChangedTip } from '../tips';
import MetaApplication from './application';
import MetaComponent from './component';
import MetaModel from './model';
import MetaModuleDependency from './module_dependency';
import MetaUiResource from './ui_resource';
import ModuleManagementLog from './module_management_log';

const Job = createServiceByModel<typeof JobModel>('task.Job');

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

function pickErrString(err: any, primary: string, secondary: string): string {
  if (err == null) return '';
  const primaryValue = err[primary];
  if (primaryValue != null && String(primaryValue) !== '') return String(primaryValue);
  const secondaryValue = err[secondary];
  if (secondaryValue != null && String(secondaryValue) !== '') return String(secondaryValue);
  return '';
}

function classifyRetryability(err?: any): FailureKind {
  const domain = pickErrString(err, 'domain', 'errorDomain').toLowerCase();
  const code = pickErrString(err, 'code', 'errorCode').toUpperCase();
  if (domain === 'meta.lock' && code === 'LEASE_CONFLICT') return 'RETRYABLE';
  if (domain === 'module_management' && code === 'LOCK_LEASE_LOST') return 'RETRYABLE';
  return 'NON_RETRYABLE';
}

function resolveFailureSource(err: any, result: any): any {
  if (err != null) return err;
  if (pickErrString(result, 'domain', 'errorDomain')) return result;
  if (pickErrString(result, 'code', 'errorCode')) return result;
  return undefined;
}

function resolveOpFailureKind(status: string, resultStatus: string | undefined, err?: any): FailureKind {
  if (status === 'cancelled') return 'NON_RETRYABLE';
  if (status === 'failed' || resultStatus === 'FAILED') return classifyRetryability(err);
  return 'NONE';
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

@Model('MetaModule', {
  tableName: 'meta_module',
  autoMigrate: false,
})
export default class MetaModule extends BaseModel {
  @Field({ type: 'varchar', size: 255, unique: true, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaModule.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 1024, string: _lt('Short Description', { scope: 'meta.model.MetaModule.fields' }) })
  ShortDesc?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Version', { scope: 'meta.model.MetaModule.fields' }) })
  Version?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Tarball', { scope: 'meta.model.MetaModule.fields' }) })
  Tarball?: string;

  @Field({ type: 'text', string: _lt('Summary', { scope: 'meta.model.MetaModule.fields' }) })
  Summary?: string;

  @Field({ type: 'text', string: _lt('Description', { scope: 'meta.model.MetaModule.fields' }) })
  Description?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Application', { scope: 'meta.model.MetaModule.fields' }) })
  ApplicationStr?: string;

  @Field({ type: 'jsonobject', string: _lt('Entry Points', { scope: 'meta.model.MetaModule.fields' }) })
  EntryPoints?: Record<string, unknown> | null;

  @Field({ type: 'varchar', size: 512, string: _lt('Web Entry Point', { scope: 'meta.model.MetaModule.fields' }) })
  WebEntryPoint?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Service Entry Point', { scope: 'meta.model.MetaModule.fields' }) })
  ServiceEntryPoint?: string;

  @Field({ type: 'jsonobject', string: _lt('Dependencies', { scope: 'meta.model.MetaModule.fields' }) })
  DependsStr?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'jsonobject', string: _lt('External Dependencies', { scope: 'meta.model.MetaModule.fields' }) })
  ExternalDependencies?: Record<string, unknown> | unknown[] | null;

  @Field({ type: 'varchar', size: 255, string: _lt('Author', { scope: 'meta.model.MetaModule.fields' }) })
  Author?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('License', { scope: 'meta.model.MetaModule.fields' }) })
  License?: string;

  @Field({ type: 'text', string: _lt('Homepage', { scope: 'meta.model.MetaModule.fields' }) })
  Homepage?: string;

  @Field({ type: 'text', string: _lt('Repository', { scope: 'meta.model.MetaModule.fields' }) })
  Repository?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Path', { scope: 'meta.model.MetaModule.fields' }) })
  Path?: string;

  @Field({ type: 'varchar', size: 64, string: _lt('Status', { scope: 'meta.model.MetaModule.fields' }) })
  Status?: string;

  @Field({ type: 'varchar', size: 255, index: true, string: _lt('Category', { scope: 'meta.model.MetaModule.fields' }) })
  Category?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaApplication }, string: _lt('Application', { scope: 'meta.model.MetaModule.fields' }) })
  ApplicationId?: MetaApplication;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaModel, inverseField: 'ModuleId' },
    string: _lt('Models', { scope: 'meta.model.MetaModule.fields' }),
  })
  Models?: MetaModel[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaComponent, inverseField: 'ModuleId' },
    string: _lt('Components', { scope: 'meta.model.MetaModule.fields' }),
  })
  Components?: MetaComponent[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => MetaUiResource, inverseField: 'ModuleId' },
    string: _lt('UI Resources', { scope: 'meta.model.MetaModule.fields' }),
  })
  UiResources?: MetaUiResource[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => MetaModuleDependency,
      targetModel: () => MetaModule,
      joinField: 'ModuleId',
      inverseJoinField: 'DependModuleId',
    },
    string: _lt('Dependencies', { scope: 'meta.model.MetaModule.fields' }),
  })
  Dependencies?: MetaModule[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => MetaModuleDependency,
      targetModel: () => MetaModule,
      joinField: 'DependModuleId',
      inverseJoinField: 'ModuleId',
    },
    string: _lt('Dependents', { scope: 'meta.model.MetaModule.fields' }),
  })
  Dependents?: MetaModule[];

  static async PlanOperation(req: PlanOperationReq): Promise<PlanOperationResp> {
    const action = (req?.action || 'install') as ModuleAction;
    const moduleName = this.ensureModuleName(req?.moduleName);
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
        message: _t('plan was regenerated from the latest module state', { scope: 'service/models/module' }),
        params: { baseRevision },
      });
    }

    const existing = await this.Search(['Name', '=', moduleName] as any, { limit: 1, fields: ['Version', 'Status'] as any });
    const current = existing?.[0];

    if ((action === 'uninstall' || action === 'upgrade') && !current) {
      blockers.push({
        code: 'MODULE_NOT_FOUND',
        level: 'BLOCKER',
        message: _t('module not found', { scope: 'service/models/module' }),
        params: { moduleName },
      });
    }

    affectedModules.push({
      moduleName,
      reason: action,
      currentVersion: current?.Version,
    });

    return { baseRevision, affectedModules, risks, blockers };
  }

  static async RequestInstall(moduleName: string, withDemo?: boolean): Promise<string> {
    return this.enqueueModuleOp('install', moduleName, { withDemo: !!withDemo });
  }

  static async RequestUninstall(moduleName: string): Promise<string> {
    return this.enqueueModuleOp('uninstall', moduleName);
  }

  static async RequestUpgrade(moduleName: string): Promise<string> {
    return this.enqueueModuleOp('upgrade', moduleName);
  }

  private static ensureModuleName(name?: string): string {
    const trimmed = String(name || '').trim();
    if (!trimmed) throw new Error(_t('moduleName cannot be empty', { scope: 'service/models/module' }));
    return trimmed;
  }

  private static async enqueueModuleOp(action: ModuleAction, moduleName: string, extraPayload?: Record<string, unknown>): Promise<string> {
    const name = this.ensureModuleName(moduleName);
    const userId = BaseModel.ensureUserId();
    const method = `meta.MetaModule/Execute${action.charAt(0).toUpperCase() + action.slice(1)}`;
    const forceLockConflict = isTruthyFlag(getBackendEnvText('CHOYSUM_E2E_FORCE_LOCK_CONFLICT', 'choysum_e2e_force_lock_conflict'));

    const payload: Record<string, unknown> = { moduleName: name, operatorUserId: userId, ...(extraPayload || {}) };
    const job = await Job.EnqueueJob('meta', method, payload, userId, userId, undefined, 0, 0);
    const jobId = String((job as any)?.Id || '').trim();

    if (forceLockConflict && jobId) {
      const retryAfterMs = 2500;
      await (Job as any).UpdateById(jobId, {
        Status: 'failed',
        RunAfter: new Date(Date.now() + retryAfterMs),
        LastErrorJson: {
          domain: 'meta.lock',
          code: 'LEASE_CONFLICT',
          message: _t('lease conflict', { scope: 'service/models/module' }),
          details: { retry_after_ms: retryAfterMs },
        },
      });
      await publishModuleOpChangedTip({ jobId, userId, source: 'meta.MetaModule.leaseConflict' });
    } else if (jobId) {
      await publishModuleOpChangedTip({ jobId, userId });
    }

    return jobId;
  }

  static async GetOpStatus(jobId: string): Promise<OpStatusResp> {
    const id = String(jobId || '').trim();
    if (!id) throw new Error(_t('jobId cannot be empty', { scope: 'service/models/module' }));

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
    const rawResult = (job as any)?.ResultJson && typeof (job as any).ResultJson === 'object' ? (job as any).ResultJson : {};
    const result = rawResult?.result && typeof rawResult.result === 'object' ? rawResult.result : rawResult;
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
    const failureKind = resolveOpFailureKind(status, resultStatus, resolveFailureSource(err, result));
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

  private static getModuleManagementBridge(): any {
    const root: any = (globalThis as any)?.$choysum;
    if (!root?.moduleManagement) {
      throw new Error('moduleManagement bridge is not injected');
    }
    return root.moduleManagement;
  }

  private static async executeModuleOp(action: ModuleAction, moduleName: string, opts: { withDemo?: boolean; operatorUserId?: string }): Promise<any> {
    const name = this.ensureModuleName(moduleName);
    const operatorUserId = String(opts?.operatorUserId || getUserId() || '').trim();
    const jobId = String(getCtxValue('jobId') || '').trim();
    const bridge = this.getModuleManagementBridge();

    const forceResultStatus = getBackendEnvText('CHOYSUM_E2E_FORCE_RESULT_STATUS', 'choysum_e2e_force_result_status').toUpperCase();
    const forceReloadFailed = isTruthyFlag(getBackendEnvText('CHOYSUM_E2E_FORCE_RELOAD_FAILED', 'choysum_e2e_force_reload_failed'));

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

    // E2E-only seam: reload-failed scenario should exercise
    // "operation succeeded but reload failed" deterministically,
    // even if the underlying module operation has incidental failures.
    if (forceReloadFailed) {
      resultStatus = 'SUCCEEDED';
      errorDomain = undefined;
      errorCode = undefined;
      errorMessage = undefined;
      if (action === 'install') summary = { code: 'MODULE_INSTALLED', params: { moduleName: name } };
      else if (action === 'uninstall') summary = { code: 'MODULE_UNINSTALLED', params: { moduleName: name } };
      else summary = { code: 'MODULE_UPGRADED', params: { moduleName: name } };
    }

    let reload_triggered = false;
    let reload_failed = false;
    let reload_web = false;
    const skipReload = isTruthyFlag(getBackendEnvText('CHOYSUM_E2E_SKIP_RELOAD', 'choysum_e2e_skip_reload'));
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
    if (forceReloadFailed) {
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

    const operationOK = resultStatus === 'SUCCEEDED';

    await upsertModuleLog({
      JobId: jobId,
      ModuleName: name,
      Action: action,
      OperatorUserId: operatorUserId,
      ResultStatus: resultStatus,
      SummaryJson: summary,
      ErrorDomain: errorDomain,
      ErrorCode: errorCode,
      LastErrorJson: operationOK ? undefined : { message: errorMessage, domain: errorDomain, code: errorCode },
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
