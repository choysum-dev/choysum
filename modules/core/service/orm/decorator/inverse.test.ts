// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Inverse } from './inverse';

test('@Inverse registers handler metadata', () => {
  class InverseDecoratorModel extends BaseModel {
    Name?: string;

    @Inverse<InverseDecoratorModel>('Name')
    writeName() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(InverseDecoratorModel as any);
  const handler = meta.inverseHandlers?.get('Name') as any;

  expect(handler).toEqual({
    field: 'Name',
    method: 'writeName',
  });
});

test('@Inverse validates parameterless method', () => {
  expect(() => {
    class InverseMethodArgsModel extends BaseModel {
      @Inverse<any>('Name')
      writeName(_v: unknown) {
        return undefined;
      }
    }
    return InverseMethodArgsModel;
  }).toThrow('method must be parameterless');
});
