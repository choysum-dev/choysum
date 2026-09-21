// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {  BaseModel, Field, Model, SqlCompute, type ModelCtor, type RowOf } from '@/core/service';
import { getModelRepository } from '@/core/service/orm/model';
import type { QueryCondition, SearchOptions, SoftDeleteOptions } from '@/core/service/api/query';
import type { FieldSelection, RowOrProjected } from '@/core/service/api/selection';
import { createServiceByModel } from '@/core/service/rpc';
import { sql } from 'kysely';
import type JobModel from '@/task/service/models/job';
import { getBackendEnvText, isTruthyFlag } from '@/core/service/runtime/env/backend_env';
import { resolveChoysum } from '@/core/service/runtime/choysum_runtime';
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
  originTypeOrAll,
  assertSearchCondition,
  DEFAULT_MODULE_INDEX_SEARCH,
  parseSortSpecs,
  projectFields,
  toPlainRecord,
  type ModuleIndexRecord,
  type ModuleOriginType,
  type ModuleSyncOriginType,
  type RequestSyncReq,
} from './_module_index_query';

export type SyncModuleIndexReq = {
  OriginType?: ModuleSyncOriginType;
  Force?: boolean;
};

export type SyncModuleIndexResp = {
  Ok: boolean;
};

const Job = createServiceByModel<typeof JobModel>('task.Job');

async function findRunningJobId(fullMethod: string, requestedOrigin: ModuleSyncOriginType): Promise<string> {
  const running = await Job.Search(
    {
      And: [
        ['TargetApp', '=', 'meta'],
        ['FullMethod', '=', fullMethod],
        ['Status', 'in', ['queued', 'dispatching']],
      ],
    },
    { limit: 20, orderBy: { field: 'CreatedAt', order: 'desc' }, fields: ['Id', 'PayloadJson'] }
  );
  for (const row of running || []) {
    const jobId = String(row?.Id || '').trim();
    if (!jobId) continue;

    let payload: unknown = row?.PayloadJson;
    if (typeof payload === 'string') {
      try {
        payload = JSON.parse(payload);
      } catch {
        payload = undefined;
      }
    }
    const originValue =
      payload && typeof payload === 'object'
        ? (payload as { req?: { OriginType?: unknown } }).req?.OriginType
        : undefined;
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
  OriginType!: string;

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
  LastSyncAt?: Date | null;

  @Field({ type: 'datetime', string: _lt('Batch Synced At', { scope: 'meta.model.MetaModuleIndex.fields' }) })
  LastBatchSyncAt?: Date | null;

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

  static override async Search<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    condition: QueryCondition<RowOf<C>> | [] = DEFAULT_MODULE_INDEX_SEARCH as QueryCondition<RowOf<C>>,
    options?: Omit<SearchOptions<RowOf<C>>, 'fields'> & { fields?: F }
  ): Promise<Array<RowOrProjected<RowOf<C>, F>>> {
    const normalized = assertSearchCondition(condition);
    const rawOptions = { ...(options || {}) } as Record<string, unknown>;
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
    const repository = getModelRepository(this);
    const groupedRows = await repository.readGroup({
      ...readGroupOptions,
      condition: normalized,
    } as Parameters<typeof repository.readGroup>[0]);
    const groupedModuleNames = extractGroupedModuleNames(groupedRows);
    if (groupedModuleNames.length === 0) {
      return [];
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

    const search = BaseModel.Search as (
      condition: QueryCondition<BaseModel> | [],
      options?: SearchOptions<BaseModel>
    ) => Promise<BaseModel[]>;
    const detailRows = (await search.call(
      this,
      buildModuleNamesCondition(normalized, groupedModuleNames) as QueryCondition<BaseModel>,
      detailOptions as SearchOptions<BaseModel>
    )) as RowOf<C>[];

    const installedByName = new Map<string, { status?: string; version?: string }>();
    if (groupedModuleNames.length > 0) {
      const installedRows = await MetaModule.Search(['Name', 'in', groupedModuleNames], {
        fields: ['Name', 'Status', 'Version'],
        limit: groupedModuleNames.length * 2,
      });
      for (const module of installedRows || []) {
        const moduleName = String(module?.Name || '').trim();
        if (!moduleName) continue;
        installedByName.set(moduleName, {
          status: String(module?.Status || '').trim() || undefined,
          version: String(module?.Version || '').trim() || undefined,
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
    const hydrateFields = requestedFields.length > 0 ? (requestedFields as FieldSelection<RowOf<C>>) : undefined;
    return projected.map(row => this.hydrate(row as Record<string, unknown>, hydrateFields)) as Array<RowOrProjected<RowOf<C>, F>>;
  }

  static async Count<C extends ModelCtor>(
    this: C,
    condition: QueryCondition<RowOf<C>> | [] = DEFAULT_MODULE_INDEX_SEARCH as QueryCondition<RowOf<C>>,
    options?: SoftDeleteOptions
  ): Promise<number> {
    const normalized = assertSearchCondition(condition);
    const readGroupCountOptions: Record<string, unknown> = {
      groupby: 'ModuleName',
      condition: normalized,
    };
    applySoftDeleteOptions(readGroupCountOptions, { ...(options || {}) });
    const repository = getModelRepository(this);
    return repository.readGroupCount(readGroupCountOptions as unknown as Parameters<typeof repository.readGroupCount>[0]);
  }

  static async RequestSync(req: RequestSyncReq = {}): Promise<string> {
    const originType = originTypeOrAll(req.OriginType);
    const force = !!req.Force;
    const ifStale = !!req.IfStale;
    if (!force && !ifStale) return '';

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
      const repo = getModelRepository(this);
      const isOriginStale = async (target: ModuleOriginType): Promise<boolean> => {
        // Qualified column names are not in the Kysely table schema; cast for the raw index query.
        const originTypeCol = 'meta_module_index.origin_type' as never;
        const originRefCol = 'meta_module_index.origin_ref' as never;
        let query = repo
          .selectQueryBuilder()
          .select(eb => eb.fn.max('last_batch_sync_at' as never).as('last_batch_sync_at'))
          .where(originTypeCol, '=', target as never);
        if (target === 'local') {
          query = query.where(originRefCol, '=', 'local' as never);
        }
        const rows = await repo.execute(query);
        const row = (rows?.[0] ?? {}) as { lastBatchSyncAt?: unknown; last_batch_sync_at?: unknown };
        const lastBatchSyncAt = row.lastBatchSyncAt ?? row.last_batch_sync_at ?? null;
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

    BaseModel.ensureUserId();
    const job = await Job.EnqueueJob({
      TargetApp: 'meta',
      FullMethod: fullMethod,
      Payload: { req: { OriginType: originType, Force: force } },
      MaxAttempts: 0,
      TimeoutMs: 0,
    });
    return String((job as { Id?: unknown })?.Id || '').trim();
  }

  private static getModuleManagementBridge() {
    const root = resolveChoysum();
    if (!root?.moduleManagement) {
      throw new Error('moduleManagement bridge is not injected');
    }
    return root.moduleManagement;
  }

  static async Sync(req: SyncModuleIndexReq = {}): Promise<SyncModuleIndexResp> {
    const bridge = this.getModuleManagementBridge();
    const syncIndex = bridge.syncIndex as (params: {
      originType?: ModuleSyncOriginType;
      force?: boolean;
    }) => Promise<unknown>;
    if (typeof syncIndex !== 'function') {
      throw new Error('moduleManagement.syncIndex is not implemented');
    }
    await syncIndex({ originType: originTypeOrAll(req.OriginType), force: !!req.Force });
    return { Ok: true };
  }
}
