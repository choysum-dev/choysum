// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model, Search, SqlCompute } from '@/core/service';
import { MetadataStorage } from '../../orm/metadata/storage';
import { rewriteSearchCondition } from './search_rewrite';

@Model('test.SearchRewriteTarget')
class SearchRewriteTarget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.SearchRewriteModel')
class SearchRewriteModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'varchar', size: 64 })
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

test('search rewrite keeps sql-compute field condition unchanged when @Search is not declared', () => {
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

  const rewritten = rewriteSearchCondition(meta, 'DisplayName', '=', 'A', 'postgres', 'query');
  expect(rewritten).toBeUndefined();
});

test('search rewrite executes @Search instance handler through bridge context', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SearchRewriteModel as any);
  const rewritten = rewriteSearchCondition(meta, 'DisplayName', 'ilike', 'ABC', 'postgres', 'query');

  expect(rewritten).toEqual({
    kind: 'domain',
    domain: ['Name', 'ilike', 's:ABC'],
  });
});

test('search rewrite auto-resolves virtual related field to related.path when @Search is missing', () => {
  class RelatedVirtualModel extends BaseModel {}

  const meta = {
    type: RelatedVirtualModel,
    modelName: 'RelatedVirtualModel',
    fields: new Map([
      [
        'PartnerName',
        {
          type: 'varchar',
          related: {
            path: 'PartnerId.Name',
            store: false,
          },
        },
      ],
    ]),
  } as any;

  const rewritten = rewriteSearchCondition(meta, 'PartnerName', 'ilike', 'ALICE', 'postgres', 'query');

  expect(rewritten).toEqual({
    kind: 'domain',
    domain: ['PartnerId.Name', 'ilike', 'ALICE'],
  });
});
