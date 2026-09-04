// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model, SqlCompute } from '@/core/service';
import { getModelRepository } from '@/core/service/orm/model';
import { createServiceByModel } from '@/core/service/rpc';
import { sql } from 'kysely';
import type JobModel from '@/task/service/models/job';
import { getBackendEnvText, isTruthyFlag } from '@/core/service/runtime/env/backend_env';
import { normalizeFields, normalizeLimit, normalizeOffset } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import MetaModule from './module';
import {
  aggregateRows,
  applySoftDeleteOptions,
  buildModuleNamesCondition,
  buildSortPushdownPlan,
  canReuseRunningSync,
  compareBySpecs,
  extractGroupedModuleNames,
  assertOriginType,
  assertSearchCondition,
  DEFAULT_MODULE_INDEX_SEARCH,
  parseSortSpecs,
  projectFields,
  toPlainRecord,
  type ModuleIndexRecord,
  type ModuleOriginType,
  type ModuleSyncOriginType,
  type RequestSyncParams,
} from './_module_index_query';

const Job = createServiceByModel<typeof JobModel>('task.Job');

async function findRunningJobId(fullMethod: string, requestedOrigin: ModuleSyncOriginType): Promise<string> {
  const running = await Job.Search(
    {
      And: [
        ['TargetApp', '=', 'meta'],
        ['FullMethod', '=', fullMethod],
        ['Status', 'in', ['queued', 'dispatching'] as any],
      ],
    } as any,
    { limit: 20, orderBy: { field: 'CreatedAt', order: 'desc' } as any, fields: ['Id', 'PayloadJson'] as any } as any
  );
  for (const row of running || []) {
    const jobId = String((row as any)?.Id || '').trim();
    if (!jobId) continue;

    let payload = (row as any)?.PayloadJson;
    if (typeof payload === 'string') {
      try {
        payload = JSON.parse(payload);
      } catch {
        payload = undefined;
      }
    }
    const originValue = (payload as any)?.originType;
    if (!String(originValue || '').trim()) continue;

    let runningOrigin: ModuleSyncOriginType;
    try {
      runningOrigin = assertOriginType(String(originValue));
    } catch {
      continue;
    }
    if (canReuseRunningSync(requestedOrigin, runningOrigin)) {
      return jobId;
    }
  }
  return '';
}

@Model('MetaModuleIndex', {
  tableName: 'meta_module_index',
  autoMigrate: false,
})
export default class MetaModuleIndex extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Module Name', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  ModuleName!: string;

  @Field({ type: 'varchar', size: 32, notNull: true, string: _lt('Origin Type', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  OriginType!: ModuleOriginType;

  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Origin Ref', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  OriginRef!: string;

  @Field({ type: 'boolean', notNull: true, string: _lt('Available', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  Available!: boolean;

  @Field({ type: 'varchar', size: 255, string: _lt('Version', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  Version?: string;

  @Field({ type: 'jsonobject', string: _lt('Manifest JSON', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  ManifestJson?: Record<string, unknown> | null;

  @Field({ type: 'varchar', size: 512, string: _lt('Local Path', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  LocalPath?: string;

  @Field({ type: 'datetime', string: _lt('Last Synced At', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  LastSyncAt!: Date;

  @Field({ type: 'datetime', string: _lt('Batch Synced At', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  LastBatchSyncAt!: Date;

  @Field({ type: 'varchar', size: 255, string: _lt('Sync Revision', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  SyncRevision?: string;

  @Field({ type: 'text', string: _lt('Error Message', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  LastErrorMessage?: string;

  @Field({
    type: 'varchar',
    size: 64,
    string: _lt('Origin Types', { scope: 'meta.model.MetaModuleIndex.fields' }),
  })
  OriginTypes?: string;

  @SqlCompute<MetaModuleIndex>('OriginTypes')
  sqlOriginTypes() {
    return this.$sql.field('OriginType');
  }

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Local Version', { scope: 'meta.model.MetaModuleIndex.fields' }),
  })
  LocalVersion?: string;

  @SqlCompute<MetaModuleIndex>('LocalVersion')
  sqlLocalVersion() {
    return this.$sql.field('Version');
  }

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Registry Version', { scope: 'meta.model.MetaModuleIndex.fields' }),
  })
  RegistryVersion?: string;

  @SqlCompute<MetaModuleIndex>('RegistryVersion')
  sqlRegistryVersion() {
    return this.$sql.field('Version');
  }

  @Field({
    type: 'varchar',
    size: 64,
    string: _lt('Install Status', { scope: 'meta.model.MetaModuleIndex.fields' }),
  })
  InstalledStatus?: string;

  @SqlCompute<MetaModuleIndex>('InstalledStatus')
  sqlInstalledStatus() {
    const moduleStatus = this.$sql
      .selectFrom('meta_module as m')
      .select('m.status')
      .whereRef('m.name', '=', this.$sql.col('meta_module_index', 'module_name'))
      .limit(1);
    return sql<string>`coalesce((${moduleStatus}), 'uninstalled')`;
  }

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Installed Version', { scope: 'meta.model.MetaModuleIndex.fields' }),
  })
  InstalledVersion?: string;

  @SqlCompute<MetaModuleIndex>('InstalledVersion')
  sqlInstalledVersion() {
    return this.$sql
      .selectFrom('meta_module as m')
      .select('m.version')
      .whereRef('m.name', '=', this.$sql.col('meta_module_index', 'module_name'))
      .limit(1);
  }

  static async Search<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: any[] | Record<string, any> = DEFAULT_MODULE_INDEX_SEARCH,
    options?: any
  ): Promise<T[]> {
    const normalized = assertSearchCondition(condition);
    const rawOptions = { ...(options || {}) };
    const requestedFields = normalizeFields(rawOptions.fields);
    const sortSpecs = parseSortSpecs(rawOptions.orderBy);
    const offset = normalizeOffset(rawOptions.offset);
    const limit = normalizeLimit(rawOptions.limit);
    const sortPlan = buildSortPushdownPlan(sortSpecs);

    const readGroupOptions: Record<string, unknown> = {
      groupby: 'ModuleName',
      fields: sortPlan.aggregateFields.map(item => `${item.field}:${item.agg}`),
    };
    if (sortPlan.supported) {
      readGroupOptions.offset = offset;
      if (limit != null) {
        readGroupOptions.limit = limit;
      }
      readGroupOptions.orderBy = sortPlan.orderBy;
    }
    applySoftDeleteOptions(readGroupOptions, rawOptions);
    const repository = getModelRepository(this as any);
    const groupedRows = await repository.readGroup({
      ...readGroupOptions,
      condition: normalized,
    } as any);
    const groupedModuleNames = extractGroupedModuleNames(groupedRows as any[]);
    if (groupedModuleNames.length === 0) {
      return [] as unknown as T[];
    }

    const detailFields = [
      'Id',
      'ModuleName',
      'OriginType',
      'OriginRef',
      'Available',
      'Version',
      'ManifestJson',
      'LocalPath',
      'LastSyncAt',
      'LastBatchSyncAt',
      'SyncRevision',
      'LastErrorMessage',
    ];
    const detailOptions: Record<string, unknown> = {
      fields: detailFields,
      limit: groupedModuleNames.length * 2,
      orderBy: [{ field: 'ModuleName', order: 'asc' }],
    };
    applySoftDeleteOptions(detailOptions, rawOptions);

    const detailRows = (await (BaseModel as any).Search.call(this, buildModuleNamesCondition(normalized, groupedModuleNames), detailOptions)) as any[];

    const installedByName = new Map<string, { status?: string; version?: string }>();
    if (groupedModuleNames.length > 0) {
      const installedRows = await MetaModule.Search(
        ['Name', 'in', groupedModuleNames] as any,
        {
          fields: ['Name', 'Status', 'Version'] as any,
          limit: groupedModuleNames.length * 2,
        } as any
      );
      for (const module of installedRows || []) {
        const moduleName = String((module as any)?.Name || '').trim();
        if (!moduleName) continue;
        installedByName.set(moduleName, {
          status: String((module as any)?.Status || '').trim() || undefined,
          version: String((module as any)?.Version || '').trim() || undefined,
        });
      }
    }

    const mergedByModule = new Map<string, ModuleIndexRecord>();
    for (const rawRow of aggregateRows((detailRows || []).map(toPlainRecord))) {
      const row = { ...rawRow };
      const moduleName = String(row?.ModuleName || '').trim();
      if (!moduleName) continue;

      const installed = installedByName.get(moduleName);
      row.InstalledStatus = installed?.status || row.InstalledStatus || 'uninstalled';
      row.InstalledVersion = installed?.version || row.InstalledVersion;

      mergedByModule.set(moduleName, row);
    }

    const ordered: ModuleIndexRecord[] = [];
    for (const moduleName of groupedModuleNames) {
      const hit = mergedByModule.get(moduleName);
      if (hit) ordered.push(hit);
    }

    let finalRows = ordered;
    if (!sortPlan.supported) {
      ordered.sort((a, b) => compareBySpecs(a, b, sortSpecs));
      const start = offset;
      const end = limit == null ? undefined : start + limit;
      finalRows = ordered.slice(start, end);
    }

    const projected = projectFields(finalRows, requestedFields);
    const hydrateFields = requestedFields.length > 0 ? (requestedFields as any) : undefined;
    return projected.map(row => this.hydrate(row as any, hydrateFields)) as unknown as T[];
  }

  static async Count<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: any[] | Record<string, any> = DEFAULT_MODULE_INDEX_SEARCH,
    options?: any
  ): Promise<number> {
    const normalized = assertSearchCondition(condition);
    const readGroupCountOptions: Record<string, unknown> = {
      groupby: 'ModuleName',
      condition: normalized,
    };
    applySoftDeleteOptions(readGroupCountOptions, { ...(options || {}) });
    const repository = getModelRepository(this as any);
    return repository.readGroupCount(readGroupCountOptions as any);
  }

  static async RequestSync(params: RequestSyncParams = {}): Promise<string> {
    const force = !!params.force;
    const ifStale = !!params.ifStale;
    if (!force && !ifStale) return '';
    let originType: ModuleSyncOriginType = assertOriginType('all');
    if (params.originType != null) {
      originType = assertOriginType(params.originType);
    }

    if (ifStale && !force && isTruthyFlag(getBackendEnvText('CHOYSUM_E2E_SKIP_INDEX_STALE_SYNC', 'choysum_e2e_skip_index_stale_sync'))) {
      return '';
    }

    const fullMethod = 'meta.MetaModuleIndex/Sync';

    // Reuse in-flight jobs for non-force requests to reduce contention.
    if (!force) {
      const runningJobId = await findRunningJobId(fullMethod, originType);
      if (runningJobId) return runningJobId;
    }
    if (ifStale && !force) {
      const repo = getModelRepository(this as any);
      const isOriginStale = async (target: ModuleOriginType): Promise<boolean> => {
        let query = repo
          .selectQueryBuilder()
          .select((eb: any) => eb.fn.max('last_batch_sync_at').as('last_batch_sync_at'))
          .where('meta_module_index.origin_type' as any, '=', target as any);
        if (target === 'local') {
          query = query.where('meta_module_index.origin_ref' as any, '=', 'local');
        }
        const rows = await repo.execute(query);
        const row = rows?.[0] as any;
        const lastBatchSyncAt = row?.lastBatchSyncAt ?? row?.last_batch_sync_at ?? null;
        if (!lastBatchSyncAt) {
          return true;
        }

        const lastTime = new Date(lastBatchSyncAt as string).getTime();
        if (isNaN(lastTime)) {
          return true;
        }

        const ttlMs = target === 'registry' ? 10 * 60 * 1000 : 1 * 60 * 1000;
        return Date.now() - lastTime >= ttlMs;
      };

      if (originType === 'all') {
        const [registryStale, localStale] = await Promise.all([isOriginStale('registry'), isOriginStale('local')]);
        if (!registryStale && !localStale) {
          return ''; // both origins are still fresh
        }
      } else {
        const stale = await isOriginStale(originType);
        if (!stale) {
          return ''; // within staleness window, skip
        }
      }
    }

    const userId = BaseModel.ensureUserId();
    const job = await Job.EnqueueJob('meta', fullMethod, { originType, force }, userId, userId, undefined, 0, 0);
    return String((job as any)?.Id || '').trim();
  }

  private static getModuleManagementBridge(): any {
    const root: any = (globalThis as any)?.$choysum;
    if (!root?.moduleManagement) {
      throw new Error('moduleManagement bridge is not injected');
    }
    return root.moduleManagement;
  }

  static async Sync(originType?: ModuleSyncOriginType, force?: boolean): Promise<any> {
    const bridge = this.getModuleManagementBridge();
    const syncIndex = (bridge as any)?.syncIndex;
    if (typeof syncIndex !== 'function') {
      throw new Error('moduleManagement.syncIndex is not implemented');
    }
    let normalizedOriginType: ModuleSyncOriginType = assertOriginType('all');
    if (originType != null) {
      normalizedOriginType = assertOriginType(originType);
    }
    return await syncIndex({ originType: normalizedOriginType, force: !!force });
  }
}
