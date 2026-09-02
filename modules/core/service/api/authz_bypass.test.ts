// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withPermissionGraphBypass } from './authz_bypass';

function setRequest(req: any): () => void {
  const previous = (globalThis as any).$choysum;
  const current = previous || {};
  (globalThis as any).$choysum = { ...current, request: req };
  return () => {
    if (previous === undefined) {
      delete (globalThis as any).$choysum;
      return;
    }
    (globalThis as any).$choysum = previous;
  };
}

test('authz_bypass: withPermissionGraphBypass runs without request context', async () => {
  const previous = (globalThis as any).$choysum;
  delete (globalThis as any).$choysum;
  try {
    const got = await withPermissionGraphBypass(async () => 'ok');
    if (got !== 'ok') {
      throw new Error(`expected ok, got ${String(got)}`);
    }
  } finally {
    if (previous === undefined) {
      delete (globalThis as any).$choysum;
    } else {
      (globalThis as any).$choysum = previous;
    }
  }
});

test('authz_bypass: withPermissionGraphBypass toggles companyMode and bypass depths', async () => {
  const req: any = { companyMode: 'enforce', __choysumServiceState: {} };
  const restore = setRequest({ context: { req } });
  try {
    await withPermissionGraphBypass(async () => {
      if (req.companyMode !== 'skip') {
        throw new Error(`companyMode = ${String(req.companyMode)}, want skip`);
      }
      if (req.__choysumServiceState.recordRuleBypassDepth !== 1) {
        throw new Error(`recordRuleBypassDepth = ${String(req.__choysumServiceState.recordRuleBypassDepth)}`);
      }
      if (req.__choysumServiceState.fieldRuleBypassDepth !== 1) {
        throw new Error(`fieldRuleBypassDepth = ${String(req.__choysumServiceState.fieldRuleBypassDepth)}`);
      }
      return 'done';
    });
    if (req.companyMode !== 'enforce') {
      throw new Error(`restored companyMode = ${String(req.companyMode)}, want enforce`);
    }
    if ((req.__choysumServiceState.recordRuleBypassDepth ?? 0) !== 0) {
      throw new Error('recordRuleBypassDepth not restored');
    }
    if ((req.__choysumServiceState.fieldRuleBypassDepth ?? 0) !== 0) {
      throw new Error('fieldRuleBypassDepth not restored');
    }
  } finally {
    restore();
  }
});

test('authz_bypass: withPermissionGraphBypass noops when req is not an object record', async () => {
  const restore = setRequest({ context: { req: true } });
  try {
    const got = await withPermissionGraphBypass(async () => 'plain');
    if (got !== 'plain') {
      throw new Error(`expected plain, got ${String(got)}`);
    }
  } finally {
    restore();
  }
});

test('authz_bypass: withPermissionGraphBypass removes companyMode when absent before call', async () => {
  const req: any = { __choysumServiceState: {} };
  const restore = setRequest({ context: { req } });
  try {
    await withPermissionGraphBypass(async () => undefined);
    if ('companyMode' in req) {
      throw new Error('companyMode should be removed when it was absent initially');
    }
  } finally {
    restore();
  }
});

test('authz_bypass: withPermissionGraphBypass returns fn result', async () => {
  const req: any = { __choysumServiceState: {} };
  const restore = setRequest({ context: { req } });
  try {
    const got = await withPermissionGraphBypass(async () => 'payload');
    if (got !== 'payload') {
      throw new Error(`expected payload, got ${String(got)}`);
    }
  } finally {
    restore();
  }
});

test('authz_bypass: withPermissionGraphBypass noops when service state cannot init', async () => {
  const restore = setRequest({ context: { req: null } });
  try {
    const got = await withPermissionGraphBypass(async () => 'null-req');
    if (got !== 'null-req') {
      throw new Error(`expected null-req, got ${String(got)}`);
    }
  } finally {
    restore();
  }
});

test('authz_bypass: withPermissionGraphBypass noops when req is numeric carrier', async () => {
  const restore = setRequest({ context: { req: 1 } });
  try {
    const got = await withPermissionGraphBypass(async () => 'num');
    if (got !== 'num') {
      throw new Error(`expected num, got ${String(got)}`);
    }
  } finally {
    restore();
  }
});
