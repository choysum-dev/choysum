// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumConnectionProvider } from './connection-provider';

test('connection provider acquires connection, runs consumer and always releases', async () => {
  const calls = {
    acquire: 0,
    release: 0,
  };

  const connection = { tag: 'conn_1' };
  const driver = {
    async acquireConnection() {
      calls.acquire += 1;
      return connection;
    },
    async releaseConnection(conn: unknown) {
      if (conn === connection) {
        calls.release += 1;
      }
    },
  };

  const provider = new ChoysumConnectionProvider(driver as any);
  const result = await provider.provideConnection(async conn => {
    expect(conn).toBe(connection as any);
    return 'ok';
  });

  expect(result).toBe('ok');
  expect(calls).toEqual({ acquire: 1, release: 1 });
});

test('connection provider releases connection when consumer throws', async () => {
  const calls = {
    acquire: 0,
    release: 0,
  };

  const connection = { tag: 'conn_2' };
  const driver = {
    async acquireConnection() {
      calls.acquire += 1;
      return connection;
    },
    async releaseConnection(conn: unknown) {
      if (conn === connection) {
        calls.release += 1;
      }
    },
  };

  const provider = new ChoysumConnectionProvider(driver as any);
  let message = '';
  try {
    await provider.provideConnection(async () => {
      throw new Error('consumer boom');
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toContain('consumer boom');
  expect(calls).toEqual({ acquire: 1, release: 1 });
});
