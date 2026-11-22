// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { Code, ConnectError } from '@connectrpc/connect';
import {
  resolveViewSurfaceErrorCopy,
  resolveViewSurfaceStaleBannerCopy,
} from './resolveViewSurfaceErrorCopy';
import { useConnectivityStore } from '../stores/connectivityStore';

function connectError(code: Code, message: string): ConnectError {
  return Object.assign(Object.create(ConnectError.prototype), {
    code,
    message,
    rawMessage: message,
    findDetails: () => [],
  }) as ConnectError;
}

beforeEach(() => {
  setActivePinia(createPinia());
  useConnectivityStore().resetForTests();
});

test('resolveViewSurfaceErrorCopy: unreachable session uses Action-specific detail', () => {
  useConnectivityStore().markUnreachable();
  const copy = resolveViewSurfaceErrorCopy(new Error('anything'));
  expect(copy.title).toBe('Connection lost');
  expect(copy.detail).toContain('Could not load this data');
  expect(copy.detail).not.toContain('Cannot reach the server');
});

test('resolveViewSurfaceErrorCopy: transport outage uses Action-specific detail', () => {
  const copy = resolveViewSurfaceErrorCopy(connectError(Code.Unavailable, 'unavailable'));
  expect(copy.title).toBe('Connection lost');
  expect(copy.detail).toContain('Could not load this data');
  expect(copy.detail).not.toContain('Cannot reach the server');
});

test('resolveViewSurfaceStaleBannerCopy: short refresh-failed line', () => {
  expect(resolveViewSurfaceStaleBannerCopy()).toBe('Could not refresh.');
});

test('resolveViewSurfaceErrorCopy: business error uses fallback title and message', () => {
  const copy = resolveViewSurfaceErrorCopy(connectError(Code.NotFound, 'missing row'), {
    fallbackTitle: 'Failed to load list',
  });
  expect(copy.title).toBe('Failed to load list');
  expect(copy.detail).toBe('missing row');
});

test('resolveViewSurfaceErrorCopy: generic Error without fallback title', () => {
  const copy = resolveViewSurfaceErrorCopy(new Error('boom'));
  expect(copy.title).toBe('Failed to load');
  expect(copy.detail).toBe('boom');
});
