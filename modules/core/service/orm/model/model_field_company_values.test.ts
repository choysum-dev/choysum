// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
import { UpdateOperations } from './model_update';
import {
  getModelFieldCompanyValues,
  updateModelFieldCompanyValues,
} from './model_field_company_values';

@Model('FieldCompanyValuesWidget', { application: 'demo' })
class FieldCompanyValuesWidget extends BaseModel {
  @Field({ type: 'number', companyDependent: true } as any)
  Cost!: number;

  @Field({ type: 'varchar', size: 40 })
  Code!: string;
}

test('getModelFieldCompanyValues returns map and optional company filter', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Cost: { C1: 12.5, C2: 11 } }];
      },
    })) as any;

    const all = await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost');
    expect(all).toEqual({ C1: 12.5, C2: 11 });

    const filtered = await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', [
      'C2',
      'missing',
      '',
      '  ',
    ]);
    expect(filtered).toEqual({ C2: 11 });
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('field company values helpers reject non-companyDependent fields', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Cost: { C1: 1 }, Code: 'x' }];
      },
    })) as any;

    let codeGetErr: unknown;
    try {
      await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Code');
    } catch (err) {
      codeGetErr = err;
    }
    expect(String((codeGetErr as Error)?.message || codeGetErr)).toMatch(/not a company-dependent field/);

    let codeUpdateErr: unknown;
    try {
      await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Code', { C1: 9 });
    } catch (err) {
      codeUpdateErr = err;
    }
    expect(String((codeUpdateErr as Error)?.message || codeUpdateErr)).toMatch(/not a company-dependent field/);
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('updateModelFieldCompanyValues writes patched map and rejects empty ids', async () => {
  const originalRepo = RepositoryFactory.getRepository;
  const originalUpdate = UpdateOperations.Update;
  let written: { condition: any; values: any } | undefined;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [
          {
            Id: 'w1',
            UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
            Cost: { C1: 1, C2: 2 },
          },
        ];
      },
    })) as any;
    UpdateOperations.Update = (async (_ctor: any, condition: any, values: any) => {
      written = { condition, values };
      return [{ Id: 'w1' }];
    }) as any;

    const ok = await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', {
      C1: 9,
      C2: false,
    });
    expect(ok).toBe(true);
    expect(written?.condition).toEqual({
      And: [
        ['Id', '=', 'w1'],
        ['UpdatedAt', '=', new Date('2024-01-01T00:00:00.000Z')],
      ],
    });
    expect(written?.values?.Cost).toEqual({ C1: 9 });

    UpdateOperations.Update = (async () => []) as any;
    let staleErr: unknown;
    try {
      await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', { C1: 99 });
    } catch (err) {
      staleErr = err;
    }
    expect(String((staleErr as Error)?.message || staleErr)).toMatch(/has been modified/);

    let emptyIdErr: unknown;
    try {
      await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, '', 'Cost');
    } catch (err) {
      emptyIdErr = err;
    }
    expect(String((emptyIdErr as Error)?.message || emptyIdErr)).toMatch(/non-empty id/);

    let emptyFieldErr: unknown;
    try {
      await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', '  ');
    } catch (err) {
      emptyFieldErr = err;
    }
    expect(String((emptyFieldErr as Error)?.message || emptyFieldErr)).toMatch(/non-empty fieldName/);
  } finally {
    RepositoryFactory.getRepository = originalRepo;
    UpdateOperations.Update = originalUpdate;
  }
});

test('getModelFieldCompanyValues parses JSON strings, null, and garbage', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Cost: '{"C1":1.5,"C2":2}' }];
      },
    })) as any;
    expect(await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', [])).toEqual({
      C1: 1.5,
      C2: 2,
    });

    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Cost: null }];
      },
    })) as any;
    expect(await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost')).toEqual({});

    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Cost: 'not-json' }];
      },
    })) as any;
    expect(await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost')).toEqual({});
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('updateModelFieldCompanyValues accepts UpdatedAt string/number and rejects empty update id', async () => {
  const originalRepo = RepositoryFactory.getRepository;
  const originalUpdate = UpdateOperations.Update;
  let written: { condition: any; values: any } | undefined;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: '2024-06-01T00:00:00.000Z', Cost: { C1: 1 } }];
      },
    })) as any;
    UpdateOperations.Update = (async (_ctor: any, condition: any, values: any) => {
      written = { condition, values };
      return [{ Id: 'w1' }];
    }) as any;

    expect(await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', { C1: 3 })).toBe(true);
    expect(written?.condition?.And?.[1]?.[2]).toEqual(new Date('2024-06-01T00:00:00.000Z'));

    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: Date.parse('2024-07-01T00:00:00.000Z'), Cost: null }];
      },
    })) as any;
    expect(await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', { C2: 4 })).toBe(true);
    expect(written?.values?.Cost).toEqual({ C2: 4 });
    expect(written?.condition?.And?.[1]?.[2]).toEqual(new Date('2024-07-01T00:00:00.000Z'));

    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: 'not-a-date', Cost: { C1: 1 } }];
      },
    })) as any;
    written = undefined;
    expect(await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', { C1: 5 })).toBe(true);
    expect(written?.condition).toEqual(['Id', '=', 'w1']);

    let emptyUpdateIdErr: unknown;
    try {
      await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, '  ', 'Cost', { C1: 1 });
    } catch (err) {
      emptyUpdateIdErr = err;
    }
    expect(String((emptyUpdateIdErr as Error)?.message || emptyUpdateIdErr)).toMatch(/non-empty id/);

    const got = await (FieldCompanyValuesWidget as any).GetFieldCompanyValues('w1', 'Cost', ['C1']);
    expect(got).toEqual({ C1: 1 });
    expect(await (FieldCompanyValuesWidget as any).UpdateFieldCompanyValues('w1', 'Cost', { C1: 6 })).toBe(true);
  } finally {
    RepositoryFactory.getRepository = originalRepo;
    UpdateOperations.Update = originalUpdate;
  }
});

test('get/update company values cover undefined fieldName/id coercions', async () => {
  let err: unknown;
  try {
    await getModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', undefined as any);
  } catch (e) {
    err = e;
  }
  expect(String((err as Error)?.message || err)).toMatch(/non-empty fieldName/);

  err = undefined;
  try {
    await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, undefined as any, 'Cost', { C1: 1 });
  } catch (e) {
    err = e;
  }
  expect(String((err as Error)?.message || err)).toMatch(/non-empty id/);
});

test('updateModelFieldCompanyValues ignores Invalid Date UpdatedAt', async () => {
  const originalRepo = RepositoryFactory.getRepository;
  const originalUpdate = UpdateOperations.Update;
  let written: { condition: any } | undefined;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: new Date('invalid'), Cost: { C1: 1 } }];
      },
    })) as any;
    UpdateOperations.Update = (async (_c: any, condition: any) => {
      written = { condition };
      return [{ Id: 'w1' }];
    }) as any;
    expect(await updateModelFieldCompanyValues(FieldCompanyValuesWidget as any, 'w1', 'Cost', { C1: 2 })).toBe(true);
    expect(written?.condition).toEqual(['Id', '=', 'w1']);
  } finally {
    RepositoryFactory.getRepository = originalRepo;
    UpdateOperations.Update = originalUpdate;
  }
});
