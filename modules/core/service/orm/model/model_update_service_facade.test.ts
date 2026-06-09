// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { UpdateOperations } from './model_update';
import { updateModelById, updateModels } from './model_update_service_facade';

test('model update service facade delegates update payload unchanged', async () => {
  const originalUpdate = UpdateOperations.Update;
  const calls: any[] = [];
  const ctor = {} as any;
  const condition = ['Status', '=', 'draft'] as any;
  const values = { Name: 'next' };
  const returnFields = ['Id', 'Name'] as any;
  const options = { withDeleted: true } as any;

  try {
    UpdateOperations.Update = (async (ModelCtor: any, nextCondition: any, nextValues: any, nextReturnFields: any, nextOptions: any) => {
      calls.push({ ModelCtor, nextCondition, nextValues, nextReturnFields, nextOptions });
      return [{ Id: 'M1', Name: 'next' }];
    }) as any;

    const result = await updateModels(ctor, condition, values as any, returnFields, options);

    expect(result).toEqual([{ Id: 'M1', Name: 'next' }]);
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.nextCondition).toBe(condition);
    expect(calls[0]?.nextValues).toBe(values);
    expect(calls[0]?.nextReturnFields).toBe(returnFields);
    expect(calls[0]?.nextOptions).toBe(options);
  } finally {
    UpdateOperations.Update = originalUpdate;
  }
});

test('model update service facade delegates update-by-id payload unchanged', async () => {
  const originalUpdateById = UpdateOperations.UpdateById;
  const calls: any[] = [];
  const ctor = {} as any;
  const values = { Name: 'other' };
  const returnFields = ['Name'] as any;
  const options = { onlyDeleted: true } as any;

  try {
    UpdateOperations.UpdateById = (async (ModelCtor: any, id: string, nextValues: any, nextReturnFields: any, nextOptions: any) => {
      calls.push({ ModelCtor, id, nextValues, nextReturnFields, nextOptions });
      return { Id: 'M2', Name: 'other' };
    }) as any;

    const result = await updateModelById(ctor, 'M2', values as any, returnFields, options);

    expect(result).toEqual({ Id: 'M2', Name: 'other' });
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.id).toBe('M2');
    expect(calls[0]?.nextValues).toBe(values);
    expect(calls[0]?.nextReturnFields).toBe(returnFields);
    expect(calls[0]?.nextOptions).toBe(options);
  } finally {
    UpdateOperations.UpdateById = originalUpdateById;
  }
});
