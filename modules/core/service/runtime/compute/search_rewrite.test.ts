// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model, Search, SqlCompute } from '@/core/service';
import { MetadataStorage } from '../../orm/metadata/storage';
import { rewriteSearchCondition } from './search_rewrite';

@Model('test.SearchRewriteTarget')
class SearchRewriteTarget extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.SearchRewriteModel')
class SearchRewriteModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  override DisplayName!: string;

  @SqlCompute<SearchRewriteModel>('DisplayName')
  sqlDisplayName() {
    const sql = this.$sql as any;
    return sql.str.concat(sql.field(SearchRewriteModel, 'Name'), '#');
  }

  @Search<SearchRewriteModel>('DisplayName')
  searchDisplayName() {
    const search = this.$search as any;
    return search.cmp('Name', search.op(), `s:${search.value()}`);
  }
}

test('search rewrite keeps physical field condition unchanged when no handler is declared', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SearchRewriteTarget as any);
  const rewritten = rewriteSearchCondition(meta, 'Name', '=', 'A', 'postgres', 'query');
  expect(rewritten).toBeUndefined();
});

test('search rewrite throws SEARCH_HANDLER_REQUIRED for virtual sql field without @Search', () => {
  class MissingSearchModel extends BaseModel {}

  const meta = {
    type: MissingSearchModel,
    modelName: 'MissingSearchModel',
    fields: new Map([
      [
        'DisplayName',
        {
          type: 'varchar',
          column: { name: 'DisplayName' },
        },
      ],
    ]),
    sqlComputeHandlers: new Map([
      [
        'DisplayName',
        {
          field: 'DisplayName',
          method: 'sqlDisplayName',
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    rewriteSearchCondition(meta, 'DisplayName', '=', 'A', 'postgres', 'query');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('SEARCH_HANDLER_REQUIRED')).toBe(true);
  expect(message.includes('MissingSearchModel.DisplayName')).toBe(true);
});

test('search rewrite executes @Search instance handler through bridge context', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SearchRewriteModel as any);
  const rewritten = rewriteSearchCondition(meta, 'DisplayName', 'ilike', 'ABC', 'postgres', 'query');

  expect(rewritten).toEqual({
    kind: 'domain',
    domain: ['Name', 'ilike', 's:ABC'],
  });
});
