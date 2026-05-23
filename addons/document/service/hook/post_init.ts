// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostInit } from '@/core/service/api/model';
import { createServiceByModel } from '@/core/service/rpc';
import type Schedule from '@/task/service/models/schedule';

const ScheduleService = createServiceByModel<typeof Schedule>('task.Schedule');

type ScheduleRecord = {
  Id?: string;
  Active?: boolean;
  CronExpr?: string;
  Timezone?: string;
  TargetApp?: string;
  FullMethod?: string;
  PayloadTemplateJson?: Record<string, any> | null;
};

const scheduleName = 'document.attachment.gc';
const targetApp = 'document';
const fullMethod = 'document.AttachmentContent/RunGarbageCollection';
const cronExpr = '*/5 * * * *';
const timezone = 'UTC';
const payloadTemplate: Record<string, any> = {};

function normalizePayload(value: unknown): Record<string, any> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, any>;
}

function payloadEquals(left: unknown, right: unknown): boolean {
  return JSON.stringify(normalizePayload(left)) === JSON.stringify(normalizePayload(right));
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

export class DocumentAttachmentHooks {
  @HookPostInit()
  static async ensureAttachmentGcSchedule(): Promise<void> {
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
