// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import BaseModel from './model';
import {
  isWritableStoredField,
  nameCreateModels,
  resolveNameCreateField,
} from './model_namecreate';
import { getModelRuntimeMetadata } from './model_runtime_service_facade';

@Model('NameCreateWidget', { application: 'demo' })
class NameCreateWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;
}

@Model('NameCreateNoNameWidget', { application: 'demo' })
class NameCreateNoNameWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Title!: string;
}

@Model('NameCreateOverrideWidget', { application: 'demo' })
class NameCreateOverrideWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;

  static override async NameCreate<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    name: string,
    values?: any,
    options?: any
  ): Promise<T> {
    const kw = String(name ?? '').trim();
    return (await (this as any).Create({ ...(values || {}), Name: kw, Code: `C-${kw}` }, options?.returnFields)) as T;
  }
}

test('isWritableStoredField: Name yes, DisplayName no, missing no', () => {
  const meta = getModelRuntimeMetadata(NameCreateWidget as any);
  expect(isWritableStoredField(meta, 'Name')).toBe(true);
  expect(isWritableStoredField(meta, 'Code')).toBe(true);
  expect(isWritableStoredField(meta, 'DisplayName')).toBe(false);
  expect(isWritableStoredField(meta, 'Missing')).toBe(false);
  expect(isWritableStoredField(meta, '')).toBe(false);
});

test('resolveNameCreateField defaults to Name and honors nameField', () => {
  const meta = getModelRuntimeMetadata(NameCreateWidget as any);
  expect(resolveNameCreateField(meta)).toBe('Name');
  expect(resolveNameCreateField(meta, 'Code')).toBe('Code');
  expect(() => resolveNameCreateField(meta, 'DisplayName')).toThrow(/nameField/);
  expect(() => resolveNameCreateField(meta, 'Nope')).toThrow(/nameField/);
});

test('resolveNameCreateField throws when model has no writable Name', () => {
  const meta = getModelRuntimeMetadata(NameCreateNoNameWidget as any);
  expect(() => resolveNameCreateField(meta)).toThrow(/no writable Name/);
  expect(resolveNameCreateField(meta, 'Title')).toBe('Title');
});

test('NameCreate rejects empty name', async () => {
  let emptyErr: unknown;
  try {
    await nameCreateModels(NameCreateWidget as any, '   ');
  } catch (e) {
    emptyErr = e;
  }
  expect(String(emptyErr)).toMatch(/name is empty/);

  let blankErr: unknown;
  try {
    await nameCreateModels(NameCreateWidget as any, '');
  } catch (e) {
    blankErr = e;
  }
  expect(String(blankErr)).toMatch(/name is empty/);
});

test('NameCreate writes Name and lets name override values', async () => {
  const calls: unknown[] = [];
  const original = NameCreateWidget.Create;
  NameCreateWidget.Create = (async (value: unknown, returnFields: unknown) => {
    calls.push({ value, returnFields });
    return { Id: '1', ...(value as object) };
  }) as any;
  try {
    const row = await NameCreateWidget.NameCreate('  alice  ', { Name: 'ignored', Code: 'X' } as any, {
      returnFields: ['Id', 'Name'] as any,
    });
    expect(row).toEqual({ Id: '1', Name: 'alice', Code: 'X' });
    expect(calls).toEqual([
      {
        value: { Name: 'alice', Code: 'X' },
        returnFields: ['Id', 'Name'],
      },
    ]);
  } finally {
    NameCreateWidget.Create = original;
  }
});

test('NameCreate with explicit nameField writes that field', async () => {
  const original = NameCreateWidget.Create;
  const seen: unknown[] = [];
  NameCreateWidget.Create = (async (value: any) => {
    seen.push(value);
    return { Id: '2', ...value };
  }) as any;
  try {
    await NameCreateWidget.NameCreate('bob', {}, { nameField: 'Code' });
    expect(seen).toEqual([{ Code: 'bob' }]);
  } finally {
    NameCreateWidget.Create = original;
  }
});

test('static override NameCreate replaces default payload', async () => {
  const original = NameCreateOverrideWidget.Create;
  const seen: unknown[] = [];
  NameCreateOverrideWidget.Create = (async (value: any) => {
    seen.push(value);
    return { Id: 'o1', ...value };
  }) as any;
  try {
    const row = await NameCreateOverrideWidget.NameCreate('Acme');
    expect(row).toEqual({ Id: 'o1', Name: 'Acme', Code: 'C-Acme' });
    expect(seen).toEqual([{ Name: 'Acme', Code: 'C-Acme' }]);
  } finally {
    NameCreateOverrideWidget.Create = original;
  }
});

test('NameCreateNoNameWidget without nameField throws before Create', async () => {
  const original = NameCreateNoNameWidget.Create;
  let called = false;
  NameCreateNoNameWidget.Create = (async (value: any) => {
    called = true;
    return { Id: 't1', ...value };
  }) as any;
  try {
    let err: unknown;
    try {
      await NameCreateNoNameWidget.NameCreate('x');
    } catch (e) {
      err = e;
    }
    expect(String(err)).toMatch(/no writable Name/);
    expect(called).toBe(false);
    const row = await NameCreateNoNameWidget.NameCreate('x', {}, { nameField: 'Title' });
    expect(row).toEqual({ Id: 't1', Title: 'x' });
    expect(called).toBe(true);
  } finally {
    NameCreateNoNameWidget.Create = original;
  }
});
