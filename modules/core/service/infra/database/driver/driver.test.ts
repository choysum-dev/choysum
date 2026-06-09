// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumConnection } from './connection';
import { ChoysumDriver } from './driver';

test('driver acquireConnection returns ChoysumConnection instance', async () => {
  const driver = new ChoysumDriver();
  const connection = await driver.acquireConnection();
  expect(connection instanceof ChoysumConnection).toBe(true);
});

test('driver begin/commit/rollback delegate to connection methods', async () => {
  const calls = {
    begin: 0,
    commit: 0,
    rollback: 0,
  };

  const connection = {
    async beginTransaction() {
      calls.begin += 1;
    },
    async commitTransaction() {
      calls.commit += 1;
    },
    async rollbackTransaction() {
      calls.rollback += 1;
    },
  };

  const driver = new ChoysumDriver();
  await driver.beginTransaction(connection as any);
  await driver.commitTransaction(connection as any);
  await driver.rollbackTransaction(connection as any);

  expect(calls).toEqual({
    begin: 1,
    commit: 1,
    rollback: 1,
  });
});

test('driver lifecycle no-op methods resolve without throwing', async () => {
  const driver = new ChoysumDriver();
  await driver.init();
  await driver.releaseConnection({} as any);
  await driver.destroy();
  expect(true).toBe(true);
});
