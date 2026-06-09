// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DeduplicateJoinsPlugin, KyselyPlugin, PluginTransformResultArgs, QueryResult } from 'kysely';
import type { ObjectRecord } from '../../../../../utils/types';

export class ChoysumDeduplicateJoinsPlugin extends DeduplicateJoinsPlugin implements KyselyPlugin {
  async transformResult(args: PluginTransformResultArgs): Promise<QueryResult<ObjectRecord>> {
    const res = args.result;
    const rows = Array.isArray(res.rows) ? res.rows.length : 0;
    void rows;
    return res;
  }
}
