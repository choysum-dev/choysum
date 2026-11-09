// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyMessage } from '@/web';
import { moduleCatalogProgressNotifyHooks } from './moduleCatalogProgressSessionHooks';

const msgs = {
  jobStillRunning: 'still-running',
  serviceRestarting: 'restarting',
  failedToGetStatus: 'no-status',
};

test('moduleCatalogProgressNotifyHooks: timeout and network toasts', () => {
  const warnings: string[] = [];
  const orig = ChoyMessage.warning;
  ChoyMessage.warning = (m: string) => {
    warnings.push(m);
  };
  const hooks = moduleCatalogProgressNotifyHooks(msgs);
  hooks.onTimeout();
  hooks.onTransientNetworkError();
  expect(warnings).toEqual([msgs.jobStillRunning, msgs.serviceRestarting]);
  ChoyMessage.warning = orig;
});

test('moduleCatalogProgressNotifyHooks: hard errors use generic or status fallback', () => {
  const errors: string[] = [];
  const orig = ChoyMessage.error;
  ChoyMessage.error = (m: string) => {
    errors.push(m);
  };
  const hooks = moduleCatalogProgressNotifyHooks(msgs);
  hooks.onHardError('');
  hooks.onHardError('Failed to get status');
  hooks.onHardError('disk full');
  expect(errors).toEqual([msgs.failedToGetStatus, msgs.failedToGetStatus, 'disk full']);
  ChoyMessage.error = orig;
});
