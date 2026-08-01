// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostInit } from '@/core/service/api/model';
import { createServiceByModel } from '@/core/service/rpc';
import type Schedule from '@/task/service/models/schedule';

const ScheduleService = createServiceByModel<typeof Schedule>('task.Schedule');

type ScheduleRecord = {
  Id?: string;
  Name?: string;
  Active?: boolean;
  CronExpr?: string;
  Timezone?: string;
  TargetApp?: string;
  FullMethod?: string;
  PayloadTemplateJson?: Record<string, any> | null;
};

const scheduleName = 'meta.module_index.daily_sync';
const targetApp = 'meta';
const fullMethod = 'meta.MetaModuleIndex/Sync';
const cronExpr = '0 0 * * *';
const timezone = 'UTC';
const payloadTemplate = { originType: 'local', force: true };

function normalizePayload(value: unknown): Record<string, any> {
  if (!value || typeof value !== 'object') return {};
  return value as Record<string, any>;
}

function payloadEquals(left: unknown, right: unknown): boolean {
  const a = normalizePayload(left);
  const b = normalizePayload(right);
  return JSON.stringify(a) === JSON.stringify(b);
}

function needsUpdate(existing: ScheduleRecord): boolean {
  if (existing.Active !== true) return true;
  if ((existing.CronExpr || '').trim() !== cronExpr) return true;
  if ((existing.Timezone || '').trim() !== timezone) return true;
  if ((existing.TargetApp || '').trim() !== targetApp) return true;
  if ((existing.FullMethod || '').trim() !== fullMethod) return true;
  if (!payloadEquals(existing.PayloadTemplateJson, payloadTemplate)) return true;
  return false;
}

async function listScheduleByName(name: string): Promise<ScheduleRecord[]> {
  const items = await ScheduleService.ListSchedules({ And: [['Name', '=', name]] } as any, { limit: 1 } as any);
  return Array.isArray(items) ? (items as ScheduleRecord[]) : [];
}

async function createSchedule(): Promise<void> {
  await ScheduleService.CreateSchedule(scheduleName, targetApp, fullMethod, payloadTemplate, 'admin', 'admin', cronExpr, timezone, 0);
}

async function updateSchedule(scheduleId: string): Promise<void> {
  await ScheduleService.UpdateSchedule(scheduleId, {
    Active: true,
    CronExpr: cronExpr,
    Timezone: timezone,
    TargetApp: targetApp,
    FullMethod: fullMethod,
    PayloadTemplateJson: payloadTemplate,
  } as any);
}

/**
 * Sample host kept static + no-`this` for lifecycle contract.
 * Daily Cron replaced by on-demand Lazy Sync from ModuleKanbanView;
 * decorator stays disabled on purpose. See `@/core/service/orm/decorator/LIFECYCLE_HOOKS.md`.
 */
export class MetaModuleIndexHooks {
  // @HookPostInit()
  static async ensureModuleIndexDailySync(): Promise<void> {
    const items = await listScheduleByName(scheduleName);
    const existing = items[0];
    if (!existing?.Id) {
      await createSchedule();
      return;
    }
    if (needsUpdate(existing)) {
      await updateSchedule(existing.Id);
    }
  }
}
