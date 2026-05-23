// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DatabaseConnection, ConnectionProvider, Driver } from 'kysely';

export class ChoysumConnectionProvider implements ConnectionProvider {
  readonly #driver: Driver;

  constructor(driver: Driver) {
    this.#driver = driver;
  }

  async provideConnection<T>(consumer: (connection: DatabaseConnection) => Promise<T>): Promise<T> {
    const connection = await this.#driver.acquireConnection();

    try {
      return await consumer(connection);
    } finally {
      await this.#driver.releaseConnection(connection);
    }
  }
}
