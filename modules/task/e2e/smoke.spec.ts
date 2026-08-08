// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { createClient, type Interceptor } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import { ValueSchema, ListValueSchema, StructSchema, NullValue, type Value } from '@bufbuild/protobuf/wkt';
import { loginAsE2EAdmin } from '../../auth/e2e/utils/login.ts';

/**
 * Runtime metadata injected by the task e2e harness.
 */
type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
};

/**
 * Minimal task protobuf module surface needed by the e2e smoke test.
 */
type TaskPbModule = {
  Job: any;
  Schedule: any;
  JobGetJobReqSchema: any;
  ScheduleCreateScheduleReqSchema: any;
  ScheduleTriggerScheduleReqSchema: any;
  ScheduleDeleteScheduleReqSchema: any;
};

let taskPbModulePromise: Promise<TaskPbModule> | null = null;

/**
 * Reads the task e2e runtime metadata from the harness JSON file.
 */
function readRuntimeInfo(): RuntimeInfo {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) {
    throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  }
  const raw = fs.readFileSync(runtimePath, 'utf-8');
  return JSON.parse(raw) as RuntimeInfo;
}

/**
 * Loads the generated task protobuf module staged under specsDir/.generated.
 */
async function loadTaskPbModule(): Promise<TaskPbModule> {
  const runtime = readRuntimeInfo();
  const staged = path.join(runtime.specsDir, '.generated', 'task_pb.ts');
  if (!fs.existsSync(staged)) {
    throw new Error(`Cannot find staged task_pb.ts at ${staged} (e2e runner should link it)`);
  }
  const mod = await import('./.generated/task_pb.ts');
  return mod as TaskPbModule;
}

/**
 * Reuses the generated task protobuf module across test steps.
 */
async function getTaskPbModule(): Promise<TaskPbModule> {
  if (!taskPbModulePromise) {
    taskPbModulePromise = loadTaskPbModule();
  }
  return await taskPbModulePromise;
}

/**
 * Reads the persisted auth state from browser storage.
 */
async function readAuthState(page: any): Promise<{ accessToken: string; identity: any }> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return { accessToken: '', identity: null };
    try {
      const data = JSON.parse(raw);
      return {
        accessToken: String(data?.tokens?.accessToken || ''),
        identity: data?.identity ?? null,
      };
    } catch {
      return { accessToken: '', identity: null };
    }
  });
}

/**
 * Decodes the JSON payload from a JWT access token.
 */
function decodeJwtPayload(token: string): any {
  const parts = String(token || '').split('.');
  if (parts.length < 2) return null;
  const b64url = parts[1];
  const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const pad = b64.length % 4 === 0 ? '' : '='.repeat(4 - (b64.length % 4));
  const json = Buffer.from(b64 + pad, 'base64').toString('utf-8');
  try {
    return JSON.parse(json);
  } catch {
    return null;
  }
}

/**
 * Resolves the current user id from auth identity or JWT claims.
 */
function getUserId(identity: any, accessToken: string): string {
  if (identity && identity.userId) return String(identity.userId);
  const payload = decodeJwtPayload(accessToken);
  return String(payload?.sub || payload?.user_id || payload?.userId || '');
}

/**
 * Converts plain JavaScript values into protobuf Value messages.
 */
function toValue(val: any): Value {
  if (val === null || val === undefined) {
    return create(ValueSchema, {
      kind: { case: 'nullValue', value: NullValue.NULL_VALUE },
    });
  }

  if (typeof val === 'string') {
    return create(ValueSchema, { kind: { case: 'stringValue', value: val } });
  }

  if (typeof val === 'number') {
    return create(ValueSchema, { kind: { case: 'numberValue', value: val } });
  }

  if (typeof val === 'boolean') {
    return create(ValueSchema, { kind: { case: 'boolValue', value: val } });
  }

  if (Array.isArray(val)) {
    const values = val.map(item => toValue(item));
    return create(ValueSchema, {
      kind: {
        case: 'listValue',
        value: create(ListValueSchema, { values }),
      },
    });
  }

  if (typeof val === 'object') {
    const fields: Record<string, any> = {};
    for (const [k, v] of Object.entries(val)) {
      fields[k] = toValue(v);
    }
    return create(ValueSchema, {
      kind: {
        case: 'structValue',
        value: create(StructSchema, { fields }),
      },
    });
  }

  return create(ValueSchema, {
    kind: { case: 'nullValue', value: NullValue.NULL_VALUE },
  });
}

/**
 * Converts protobuf Value messages back into plain JavaScript values.
 */
function fromValue(v?: Value): any {
  const toJs = (x: any): any => {
    if (!x || typeof x !== 'object' || !x.kind) return x;
    switch (x.kind.case) {
      case 'nullValue':
        return null;
      case 'stringValue':
      case 'numberValue':
      case 'boolValue':
        return x.kind.value;
      case 'listValue': {
        const arr = x.kind.value?.values ?? [];
        return arr.map((it: any) => toJs(it));
      }
      case 'structValue': {
        const fields = x.kind.value?.fields ?? {};
        const obj: Record<string, any> = {};
        for (const [k, vv] of Object.entries(fields)) obj[k] = toJs(vv);
        return obj;
      }
      default:
        return null;
    }
  };
  return toJs(v);
}

/**
 * Builds an auth interceptor for gRPC-web clients.
 */
function makeAuthInterceptor(accessToken: string): Interceptor {
  return next => async req => {
    if (accessToken) {
      req.header.set('Authorization', `Bearer ${accessToken}`);
    }
    return await next(req);
  };
}

/**
 * Creates task schedule and job gRPC-web clients.
 */
function makeTaskClients(baseURL: string, accessToken: string, services: { Schedule: any; Job: any }): { scheduleClient: any; jobClient: any } {
  const transport = createGrpcWebTransport({
    baseUrl: baseURL,
    interceptors: [makeAuthInterceptor(accessToken)],
  });
  return {
    scheduleClient: createClient(services.Schedule as any, transport) as any,
    jobClient: createClient(services.Job as any, transport) as any,
  };
}

test('task: create schedule and trigger job via gRPC-web', async ({ page }) => {
  test.setTimeout(120_000);

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;
  const taskPb = await getTaskPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken, identity } = await readAuthState(page);
  expect(accessToken).not.toBe('');

  const userId = getUserId(identity, accessToken);
  expect(userId).not.toBe('');

  const clients = makeTaskClients(baseURL, accessToken, {
    Schedule: taskPb.Schedule,
    Job: taskPb.Job,
  });
  const scheduleClient: any = clients.scheduleClient;
  const jobClient: any = clients.jobClient;

  const name = `e2e-schedule-${Date.now()}`;
  const createResp: any = await (scheduleClient as any).createSchedule(
    create(taskPb.ScheduleCreateScheduleReqSchema, {
      name,
      targetApp: 'auth',
      fullMethod: 'auth.User/Login',
      payloadTemplate: toValue({ email: 'e2e@choysum.test' }),
      schedulerUserId: userId,
      triggeredByUserId: userId,
      cronExpr: '* * * * *',
      timezone: 'UTC',
    })
  );

  const schedule = fromValue(createResp.result);
  const scheduleId = String(schedule?.Id || schedule?.id || '');
  expect(scheduleId).not.toBe('');

  const triggerResp: any = await (scheduleClient as any).triggerSchedule(
    create(taskPb.ScheduleTriggerScheduleReqSchema, {
      scheduleId,
      payloadOverride: toValue({ email: 'e2e-trigger@choysum.test' }),
      schedulerUserIdOverride: userId,
      triggeredByUserId: userId,
    })
  );

  const triggerResult = fromValue(triggerResp.result);
  const jobId = String(triggerResult?.jobId || triggerResult?.JobId || triggerResult?.jobID || '');
  expect(jobId).not.toBe('');

  const jobResp: any = await (jobClient as any).getJob(
    create(taskPb.JobGetJobReqSchema, {
      jobId,
      fields: toValue(['Id', 'TargetApp', 'FullMethod', 'PayloadJson']),
    })
  );

  const job = fromValue(jobResp.result);
  expect(job?.TargetApp || job?.targetApp).toBe('auth');
  expect(job?.FullMethod || job?.fullMethod).toBe('auth.User/Login');

  await (scheduleClient as any).deleteSchedule(
    create(taskPb.ScheduleDeleteScheduleReqSchema, {
      scheduleId,
    })
  );
});
