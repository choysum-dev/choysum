// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookReady } from '@/core/service/api/model';
import { getUserId } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import type Schedule from '@/task/service/models/schedule';
import { condition } from '@/core/service/api/query';

const ScheduleService = createServiceByModel<typeof Schedule>('task.Schedule');

/** Worker routing key for attachment GC (not a public interactive RPC). */
export const DOCUMENT_ATTACHMENT_GC_FULL_METHOD = 'document.AttachmentContent/RunGarbageCollection';

const scheduleName = 'document.attachment.gc';
const targetApp = 'document';
const fullMethod = DOCUMENT_ATTACHMENT_GC_FULL_METHOD;
const cronExpr = '*/5 * * * *';
const timezone = 'UTC';
const payloadTemplate: Record<string, unknown> = {};

function normalizePayload(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}

function payloadEquals(left: unknown, right: unknown): boolean {
  return JSON.stringify(normalizePayload(left)) === JSON.stringify(normalizePayload(right));
}

function needsUpdate(existing: Partial<Schedule>, actorUserId: string): boolean {
  if (existing.Active !== true) return true;
  if ((existing.CronExpr || '').trim() !== cronExpr) return true;
  if ((existing.Timezone || '').trim() !== timezone) return true;
  if ((existing.TargetApp || '').trim() !== targetApp) return true;
  if ((existing.FullMethod || '').trim() !== fullMethod) return true;
  if (!payloadEquals(existing.PayloadTemplateJson, payloadTemplate)) return true;
  if ((existing.SchedulerUserId || '').trim() !== actorUserId) return true;
  if ((existing.TriggeredByUserId || '').trim() !== actorUserId) return true;
  return false;
}

async function listScheduleByName(name: string): Promise<Array<Partial<Schedule>>> {
  const items = await ScheduleService.Search(condition<Schedule>({ And: [['Name', '=', name]] }), { limit: 1 });
  return Array.isArray(items) ? items : [];
}

async function createSchedule(actorUserId: string): Promise<void> {
  await ScheduleService.Create({
    Active: true,
    Name: scheduleName,
    TargetApp: targetApp,
    FullMethod: fullMethod,
    PayloadTemplateJson: payloadTemplate,
    SchedulerUserId: actorUserId,
    TriggeredByUserId: actorUserId,
    CronExpr: cronExpr,
    Timezone: timezone,
    TimeoutMs: 0,
  });
}

async function updateSchedule(scheduleId: string, actorUserId: string): Promise<void> {
  await ScheduleService.UpdateById(scheduleId, {
    Active: true,
    CronExpr: cronExpr,
    Timezone: timezone,
    TargetApp: targetApp,
    FullMethod: fullMethod,
    PayloadTemplateJson: payloadTemplate,
    SchedulerUserId: actorUserId,
    TriggeredByUserId: actorUserId,
  });
}

/**
 * Ensures the attachment GC cron via task.Schedule API after application services
 * are registered. See `@/core/service/orm/decorator/LIFECYCLE_HOOKS.md`.
 */
export class DocumentAttachmentHooks {
  @HookReady()
  static async ensureAttachmentGcSchedule(): Promise<void> {
    const actorUserId = String(getUserId() || '').trim();
    if (!actorUserId) {
      throw new Error('SchedulerUserId requires an authenticated principal (bootstrap admin User.Id)');
    }
    const items = await listScheduleByName(scheduleName);
    const existing = items[0];
    if (!existing?.Id) {
      await createSchedule(actorUserId);
      return;
    }
    if (needsUpdate(existing, actorUserId)) {
      await updateSchedule(existing.Id, actorUserId);
    }
  }
}
