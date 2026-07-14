// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Search } from './search';

test('@Search registers handler metadata', () => {
  class SearchDecoratorModel extends BaseModel {
    Name?: string;

    @Search<SearchDecoratorModel>('Name')
    rewriteName() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(SearchDecoratorModel as any);
  const handler = meta.searchHandlers?.get('Name') as any;

  expect(handler).toEqual({
    field: 'Name',
    method: 'rewriteName',
  });
});

test('@Search validates parameterless method', () => {
  expect(() => {
    class SearchMethodArgsModel extends BaseModel {
      @Search<any>('Name')
      rewriteName(_v: unknown) {
        return undefined;
      }
    }
    return SearchMethodArgsModel;
  }).toThrow('method must be parameterless');
});
