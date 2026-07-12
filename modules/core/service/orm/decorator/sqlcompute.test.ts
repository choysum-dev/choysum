// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { SqlCompute } from './sqlcompute';

test('@SqlCompute registers handler metadata', () => {
  class SqlComputeDecoratorModel extends BaseModel {
    Name?: string;

    @SqlCompute<SqlComputeDecoratorModel>('Name', {
      deps: ['Name', 'Name'],
    })
    sqlName() {
      return 'expr' as any;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(SqlComputeDecoratorModel as any);
  const handler = meta.sqlComputeHandlers?.get('Name') as any;

  expect(handler).toEqual({
    field: 'Name',
    method: 'sqlName',
    deps: ['Name'],
  });
});

test('@SqlCompute validates parameterless method', () => {
  expect(() => {
    class SqlComputeMethodArgsModel extends BaseModel {
      @SqlCompute<any>('Name')
      sqlName(_v: unknown) {
        return 'expr' as any;
      }
    }
    return SqlComputeMethodArgsModel;
  }).toThrow('method must be parameterless');
});
