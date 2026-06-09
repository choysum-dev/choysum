// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CreateOperations } from './model_create';
import { createManyModels, createModel } from './model_create_service_facade';

test('model create service facade delegates create payload unchanged', async () => {
  const originalCreate = CreateOperations.Create;
  const calls: any[] = [];
  const ctor = {} as any;
  const value = { Name: 'demo' };
  const returnFields = ['Id', 'Name'] as any;

  try {
    CreateOperations.Create = (async (ModelCtor: any, nextValue: any, nextReturnFields: any) => {
      calls.push({ ModelCtor, nextValue, nextReturnFields });
      return { Id: 'M1' };
    }) as any;

    const result = await createModel(ctor, value as any, returnFields);

    expect(result).toEqual({ Id: 'M1' });
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.nextValue).toBe(value);
    expect(calls[0]?.nextReturnFields).toBe(returnFields);
  } finally {
    CreateOperations.Create = originalCreate;
  }
});

test('model create service facade delegates create-many payload unchanged', async () => {
  const originalCreateMany = CreateOperations.CreateMany;
  const calls: any[] = [];
  const ctor = {} as any;
  const values = [{ Name: 'A' }, { Name: 'B' }];
  const returnFields = ['Id'] as any;

  try {
    CreateOperations.CreateMany = (async (ModelCtor: any, nextValues: any, nextReturnFields: any) => {
      calls.push({ ModelCtor, nextValues, nextReturnFields });
      return [{ Id: 'A' }, { Id: 'B' }];
    }) as any;

    const result = await createManyModels(ctor, values as any, returnFields);

    expect(result).toEqual([{ Id: 'A' }, { Id: 'B' }]);
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.nextValues).toBe(values);
    expect(calls[0]?.nextReturnFields).toBe(returnFields);
  } finally {
    CreateOperations.CreateMany = originalCreateMany;
  }
});
