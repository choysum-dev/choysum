// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { EntityConverter } from '../utils/converter';
import { RepositoryFactory } from '../repository/repository_factory';
import { toEntity, toPlainObject, withModelSavepoint } from './model_edge_facade';

test('model edge facade delegates withSavepoint directly to repository', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const calls: Array<{ fn: () => Promise<string>; name: string | undefined }> = [];
  const repo = {
    async withSavepoint(fn: () => Promise<string>, name?: string) {
      calls.push({ fn, name });
      return await fn();
    },
  };

  try {
    RepositoryFactory.getRepository = (() => repo) as any;

    const value = await withModelSavepoint({} as any, async () => 'ok', 'sp-1');

    expect(value).toBe('ok');
    expect(calls.length).toBe(1);
    expect(calls[0]?.name).toBe('sp-1');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model edge facade delegates plain serialization and keeps toEntity as a shallow entity copy', () => {
  const original = EntityConverter.modelToPlainObject;
  const calls: any[] = [];
  const instance = {
    entity: {
      Id: 'E1',
      Name: 'demo',
      nested: { keep: true },
    },
    fields: ['Name'],
  } as any;

  try {
    EntityConverter.modelToPlainObject = ((input: any, fields: any) => {
      calls.push({ input, fields });
      return { Id: 'P1', fields };
    }) as any;

    const plain = toPlainObject(instance as any);
    const entity = toEntity(instance as any);

    expect(plain).toEqual({ Id: 'P1', fields: ['Name'] });
    expect(calls.length).toBe(1);
    expect(calls[0]?.input).toBe(instance as any);
    expect(calls[0]?.fields).toEqual(['Name']);

    expect(entity).toEqual({ Id: 'E1', Name: 'demo', nested: { keep: true } });
    expect(entity === instance.entity).toBe(false);

    entity.Name = 'changed';
    expect(instance.entity.Name).toBe('demo');
  } finally {
    EntityConverter.modelToPlainObject = original;
  }
});
