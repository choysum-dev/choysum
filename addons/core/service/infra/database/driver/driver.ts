// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DatabaseConnection, Driver } from 'kysely';
import { ChoysumConnection } from './connection';

export class ChoysumDriver implements Driver {
  async init(): Promise<void> {}

  /*        Async methods            */
  async acquireConnection(): Promise<DatabaseConnection> {
    return new ChoysumConnection();
  }

  async beginTransaction(conn: ChoysumConnection) {
    await conn.beginTransaction();
  }

  async commitTransaction(conn: ChoysumConnection) {
    await conn.commitTransaction();
  }

  async rollbackTransaction(conn: ChoysumConnection) {
    await conn.rollbackTransaction();
  }

  async releaseConnection(_connection: DatabaseConnection): Promise<void> {}

  async destroy(): Promise<void> {}
}
