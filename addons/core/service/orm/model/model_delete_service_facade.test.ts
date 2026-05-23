// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DeleteOperations } from './model_delete';
import { deleteModelById, deleteModels } from './model_delete_service_facade';

test('model delete service facade delegates delete condition unchanged', async () => {
  const originalDelete = DeleteOperations.Delete;
  const calls: any[] = [];
  const ctor = {} as any;
  const condition = ['Status', '=', 'draft'] as any;
  const options = { withDeleted: true } as any;

  try {
    DeleteOperations.Delete = (async (ModelCtor: any, nextCondition: any, nextOptions: any) => {
      calls.push({ ModelCtor, nextCondition, nextOptions });
      return 3;
    }) as any;

    const result = await deleteModels(ctor, condition, options);

    expect(result).toBe(3);
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.nextCondition).toBe(condition);
    expect(calls[0]?.nextOptions).toBe(options);
  } finally {
    DeleteOperations.Delete = originalDelete;
  }
});

test('model delete service facade delegates delete-by-id unchanged', async () => {
  const originalDeleteById = DeleteOperations.DeleteById;
  const calls: any[] = [];
  const ctor = {} as any;
  const options = { onlyDeleted: true } as any;

  try {
    DeleteOperations.DeleteById = (async (ModelCtor: any, id: string, nextOptions: any) => {
      calls.push({ ModelCtor, id, nextOptions });
      return 1;
    }) as any;

    const result = await deleteModelById(ctor, 'M1', options);

    expect(result).toBe(1);
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(ctor);
    expect(calls[0]?.id).toBe('M1');
    expect(calls[0]?.nextOptions).toBe(options);
  } finally {
    DeleteOperations.DeleteById = originalDeleteById;
  }
});
