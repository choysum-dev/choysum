// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import type { AppSettingModelCtor } from '@/core/service';
import User from '@/auth/service/models/user';
import {
  PERSIST_BROWSER_TIMEZONE_KEY,
  persistBrowserTimezoneIfEmpty,
} from '@/auth/service/models/_user_lifecycle_auth';
import { withPermissionGraphBypass } from '@/auth/service/models/_user_authz_shared';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};

  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = {};

  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function setupAllowlistForAppSetting(): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, {
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.AppSetting:read',
      'auth.AppSetting:write',
      'auth.AppSetting:create',
      'auth.AppSetting:unlink',
      'AppSetting:read',
      'AppSetting:write',
      'AppSetting:create',
      'AppSetting:unlink',
    ],
  });
}

async function setPersistBrowserTimezoneFlag(value: string | null): Promise<void> {
  await withModelContext(
    {} as any,
    async () => {
      await withPermissionGraphBypass(async () => {
        await User.pool<AppSettingModelCtor>('AppSetting').Set(PERSIST_BROWSER_TIMEZONE_KEY, value);
      });
    },
    { merge: false }
  );
}

test('persist_browser_timezone: default/on still writes empty User.Timezone', async () => {
  resetRequestContext();
  setupAllowlistForAppSetting();
  await setPersistBrowserTimezoneFlag(null);

  const updates: Array<{ userId: string; timezone: string }> = [];
  const user = { Id: 'tz_on_1', Timezone: null as string | null };
  await withModelContext(
    {} as any,
    async () => {
      await persistBrowserTimezoneIfEmpty(user, {
        clientTimezone: 'Asia/Shanghai',
        updateTimezone: async (userId, timezone) => {
          updates.push({ userId, timezone });
        },
      });
    },
    { merge: false }
  );
  expect(updates).toEqual([{ userId: 'tz_on_1', timezone: 'Asia/Shanghai' }]);
  expect(user.Timezone).toBe('Asia/Shanghai');
});

test('persist_browser_timezone: Set(0) skips write; Set(1)/null reopen; key reusable', async () => {
  resetRequestContext();
  setupAllowlistForAppSetting();

  await setPersistBrowserTimezoneFlag('0');
  const closedUpdates: Array<{ userId: string; timezone: string }> = [];
  const closedUser = { Id: 'tz_off_1', Timezone: null as string | null };
  await withModelContext(
    {} as any,
    async () => {
      await persistBrowserTimezoneIfEmpty(closedUser, {
        clientTimezone: 'Europe/Berlin',
        updateTimezone: async (userId, timezone) => {
          closedUpdates.push({ userId, timezone });
        },
      });
    },
    { merge: false }
  );
  expect(closedUpdates).toEqual([]);
  expect(closedUser.Timezone).toBe(null);

  await setPersistBrowserTimezoneFlag('1');
  const openUpdates: Array<{ userId: string; timezone: string }> = [];
  const openUser = { Id: 'tz_on_2', Timezone: null as string | null };
  await withModelContext(
    {} as any,
    async () => {
      await persistBrowserTimezoneIfEmpty(openUser, {
        clientTimezone: 'UTC',
        updateTimezone: async (userId, timezone) => {
          openUpdates.push({ userId, timezone });
        },
      });
    },
    { merge: false }
  );
  expect(openUpdates).toEqual([{ userId: 'tz_on_2', timezone: 'UTC' }]);

  // Set(null) hard-deletes so the unique key can be written again.
  await setPersistBrowserTimezoneFlag(null);
  await setPersistBrowserTimezoneFlag('0');
  await setPersistBrowserTimezoneFlag('1');
  const reuseUpdates: Array<{ userId: string; timezone: string }> = [];
  const reuseUser = { Id: 'tz_reuse', Timezone: null as string | null };
  await withModelContext(
    {} as any,
    async () => {
      await persistBrowserTimezoneIfEmpty(reuseUser, {
        clientTimezone: 'America/New_York',
        updateTimezone: async (userId, timezone) => {
          reuseUpdates.push({ userId, timezone });
        },
      });
    },
    { merge: false }
  );
  expect(reuseUpdates).toEqual([{ userId: 'tz_reuse', timezone: 'America/New_York' }]);
});
