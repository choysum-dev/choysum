// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import { sql } from 'kysely';
import Job from '@/task/service/models/job';

type ModuleOriginType = 'local' | 'registry';

type RequestSyncParams = {
  originType?: ModuleOriginType;
  force?: boolean;
  ifStale?: boolean;
};

function normalizeOriginType(value?: string): ModuleOriginType {
  const raw = String(value || '')
    .trim()
    .toLowerCase();
  if (raw === 'registry') return 'registry';
  return 'local';
}

function ensureCurrentUserId(): string {
  const userId = String(getUserId() || '').trim();
  if (userId) return userId;
  const env = (import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv;
  const fallback = String((env as any)?.CHOYSUM_E2E_OPERATOR_USER_ID || (env as any)?.choysum_e2e_operator_user_id || '').trim();
  if (fallback) return fallback;
  throw new Error('current user is required');
}

function getModuleManagementBridge(): any {
  const root: any = (globalThis as any)?.$choysum;
  if (!root?.moduleManagement) {
    throw new Error('moduleManagement bridge is not injected');
  }
  return root.moduleManagement;
}

async function findRunningJobId(fullMethod: string): Promise<string> {
  const running = await Job.Search(
    {
      And: [
        ['TargetApp', '=', 'meta'],
        ['FullMethod', '=', fullMethod],
        ['Status', 'in', ['queued', 'dispatching'] as any],
      ],
    } as any,
    { limit: 1, orderBy: { field: 'CreatedAt', order: 'desc' } as any, fields: ['Id'] as any } as any
  );
  const jobId = String(running?.[0]?.Id || '').trim();
  return jobId;
}

@Model('IrModuleIndex', {
  tableName: 'meta_ir_module_index',
  autoMigrate: false,
})
export default class IrModuleIndex extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  ModuleName!: string;

  @Field({ type: 'varchar', column: { size: 32, notNull: true } })
  OriginType!: ModuleOriginType;

  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  OriginRef!: string;

  @Field({ type: 'boolean', column: { notNull: true } })
  Available!: boolean;

  @Field({ type: 'varchar', column: { size: 255 } })
  Version?: string;

  @Field({ type: 'jsonobject' })
  ManifestJson?: Record<string, unknown> | null;

  @Field({ type: 'varchar', column: { size: 512 } })
  LocalPath?: string;

  @Field({ type: 'datetime' })
  LastSyncAt!: Date;

  @Field({ type: 'datetime' })
  LastBatchSyncAt!: Date;

  @Field({ type: 'varchar', column: { size: 255 } })
  SyncRevision?: string;

  @Field({ type: 'text' })
  LastErrorMessage?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ selectFrom, col }) =>
        sql<string>`coalesce((${selectFrom('meta_ir_module as m')
          .select('m.status')
          .whereRef('m.name', '=', col('meta_ir_module_index', 'module_name'))
          .limit(1)}), 'uninstalled')`,
      size: 64,
    },
  })
  InstalledStatus?: string;

  @Field({
    type: 'varchar',
    select: {
      expr: ({ selectFrom, col }) =>
        selectFrom('meta_ir_module as m').select('m.version').whereRef('m.name', '=', col('meta_ir_module_index', 'module_name')).limit(1),
      size: 255,
    },
  })
  InstalledVersion?: string;

  static async Search<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: any[] | Record<string, any> = [],
    options?: any
  ): Promise<T[]> {
    const emptyArray = Array.isArray(condition) && condition.length === 0;
    const emptyObject = !Array.isArray(condition) && condition && typeof condition === 'object' && Object.keys(condition).length === 0;
    const normalized = emptyArray || emptyObject ? (['Available', '=', true] as any) : condition;
    return await (BaseModel as any).Search.call(this, normalized, options);
  }

  static async RequestSync(params: RequestSyncParams = {}): Promise<string> {
    const force = !!params.force;
    const ifStale = !!params.ifStale;
    if (!force && !ifStale) return '';

    const fullMethod = 'meta.IrModuleIndex/Sync';
    const runningJobId = await findRunningJobId(fullMethod);
    if (runningJobId) return runningJobId;

    const originType = normalizeOriginType(params.originType);
    if (ifStale && !force) {
      const repo = this.getRepository();
      let query = repo
        .selectQueryBuilder()
        .select((eb: any) => eb.fn.max('last_batch_sync_at').as('last_batch_sync_at'))
        .where('meta_ir_module_index.origin_type' as any, '=', originType as any);
      if (originType === 'local') {
        query = query.where('meta_ir_module_index.origin_ref' as any, '=', 'local');
      }
      const rows = await repo.execute(query);
      const row = rows?.[0] as any;
      const lastBatchSyncAt = row?.lastBatchSyncAt ?? row?.last_batch_sync_at ?? null;
      if (!lastBatchSyncAt) {
        // stale: proceed
      } else {
        return '';
      }
    }

    const userId = ensureCurrentUserId();
    const job = await Job.EnqueueJob('meta', fullMethod, { originType, force }, userId, userId, undefined, 0, 0);
    return String((job as any)?.Id || '').trim();
  }

  static async Sync(originType?: ModuleOriginType, force?: boolean): Promise<any> {
    const bridge = getModuleManagementBridge();
    const syncIndex = (bridge as any)?.syncIndex;
    if (typeof syncIndex !== 'function') {
      throw new Error('moduleManagement.syncIndex is not implemented');
    }
    return await syncIndex({ originType: normalizeOriginType(originType), force: !!force });
  }
}
