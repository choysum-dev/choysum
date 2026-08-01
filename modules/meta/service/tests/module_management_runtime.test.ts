// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getBackendEnv, getBackendEnvText, isTruthyFlag } from '@/core/service/runtime/env/backend_env';
import MetaModule from '@/meta/service/models/module';
import MetaModuleIndex from '@/meta/service/models/module_index';

declare var globalThis: any;

test('getBackendEnv returns a non-null object', () => {
  expect(getBackendEnv()).toBeTruthy();
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

test('MetaModule.getModuleManagementBridge returns bridge when available', () => {
  const root: any = globalThis.$choysum;
  const saved = root?.moduleManagement;
  root.moduleManagement = { install: () => ({ ok: true }) };
  try {
    const bridge = (MetaModule as any).getModuleManagementBridge();
    expect(typeof bridge.install).toBe('function');
  } finally {
    root.moduleManagement = saved;
  }
});

test('MetaModule.getModuleManagementBridge throws when bridge is missing', () => {
  const root: any = globalThis.$choysum;
  const saved = root?.moduleManagement;
  delete root.moduleManagement;
  try {
    expect(() => (MetaModule as any).getModuleManagementBridge()).toThrow('moduleManagement bridge is not injected');
  } finally {
    root.moduleManagement = saved;
  }
});

test('MetaModuleIndex.getModuleManagementBridge returns bridge when available', () => {
  const root: any = globalThis.$choysum;
  const saved = root?.moduleManagement;
  root.moduleManagement = { syncIndex: () => ({ ok: true }) };
  try {
    const bridge = (MetaModuleIndex as any).getModuleManagementBridge();
    expect(typeof bridge.syncIndex).toBe('function');
  } finally {
    root.moduleManagement = saved;
  }
});

test('MetaModuleIndex.getModuleManagementBridge throws when bridge is missing', () => {
  const root: any = globalThis.$choysum;
  const saved = root?.moduleManagement;
  delete root.moduleManagement;
  try {
    expect(() => (MetaModuleIndex as any).getModuleManagementBridge()).toThrow('moduleManagement bridge is not injected');
  } finally {
    root.moduleManagement = saved;
  }
});
