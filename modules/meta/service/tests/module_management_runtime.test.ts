// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getBackendEnv,
  getBackendEnvText,
  isTruthyFlag,
  ensureCurrentUserId,
  getModuleManagementBridge,
} from '@/meta/service/models/_module_management_runtime';

declare var globalThis: any;

function ensureRequestContext(): any {
  const root: any = globalThis.$choysum;
  if (!root) {
    throw new Error('missing global $choysum; module runtime tests must run under the QuickJS-first harness');
  }
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    companyMode: 'skip',
    recordRuleMode: 'skip',
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: 'admin' };
}

test('getBackendEnv returns a non-null object', () => {
  const env = getBackendEnv();
  expect(typeof env).toBe('object');
  expect(env).not.toBe(null);
});

test('getBackendEnvText returns empty string for nonexistent keys', () => {
  expect(getBackendEnvText('NONEXISTENT_KEY_12345')).toBe('');
});

test('isTruthyFlag recognizes truthy and falsy values', () => {
  expect(isTruthyFlag('1')).toBe(true);
  expect(isTruthyFlag('true')).toBe(true);
  expect(isTruthyFlag('TRUE')).toBe(true);
  expect(isTruthyFlag('yes')).toBe(true);
  expect(isTruthyFlag('YES')).toBe(true);
  expect(isTruthyFlag('on')).toBe(true);
  expect(isTruthyFlag('ON')).toBe(true);

  expect(isTruthyFlag('0')).toBe(false);
  expect(isTruthyFlag('false')).toBe(false);
  expect(isTruthyFlag('no')).toBe(false);
  expect(isTruthyFlag('off')).toBe(false);
  expect(isTruthyFlag('')).toBe(false);
  expect(isTruthyFlag('  random  ')).toBe(false);
});

test('ensureCurrentUserId returns user from request context', () => {
  resetRequestContext();
  expect(ensureCurrentUserId()).toBe('admin');
});

test('ensureCurrentUserId falls back to E2E env when no request user', () => {
  const jsCtx = ensureRequestContext();
  delete jsCtx.identity.userId;

  // In QuickJS, import.meta.env takes precedence over globalThis.__choysumBackendEnv.
  // The E2E fallback path reads from the backend env, which is import.meta.env
  // in the bundled runtime. When no E2E operator key is set in either source,
  // ensureCurrentUserId should throw.
  expect(() => ensureCurrentUserId()).toThrow('current user is required');
});

test('getModuleManagementBridge returns bridge when available', () => {
  const root: any = globalThis.$choysum;
  const saved = root?.moduleManagement;
  root.moduleManagement = { install: () => ({ ok: true }) };
  try {
    const bridge = getModuleManagementBridge();
    expect(typeof bridge.install).toBe('function');
  } finally {
    root.moduleManagement = saved;
  }
});

test('getModuleManagementBridge throws when bridge is missing', () => {
  const root: any = globalThis.$choysum;
  const saved = root?.moduleManagement;
  delete root.moduleManagement;
  try {
    expect(() => getModuleManagementBridge()).toThrow('moduleManagement bridge is not injected');
  } finally {
    root.moduleManagement = saved;
  }
});
