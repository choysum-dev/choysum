// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { DeleteOperations } from './model_delete';
import { RepositoryFactory } from '../repository/repository_factory';
import { ComputeCascadeEngine } from '../../runtime/compute/cascade';

class DeleteOpsModel extends BaseModel {}

test('model delete snapshots upstream inverse fields and triggers upstream for each old row', async () => {
  const originalCollect = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalTrigger = ComputeCascadeEngine.triggerUpstream;

  const calls = {
    search: [] as any[],
    delete: [] as any[],
    upstream: [] as any[],
  };

  const repository = {
    async search(condition: any, options: any) {
      calls.search.push({ condition, options });
      return [
        { Id: 'r1', ParentId: 'p1' },
        { Id: 'r2', ParentId: 'p2' },
      ];
    },
    async delete(condition: any) {
      calls.delete.push(condition);
      return [{ Id: 'r1' }, { Id: 'r2' }];
    },
  };

  try {
    ComputeCascadeEngine.collectUpstreamInverseFields = (() => ['ParentId', 'ParentId']) as any;
    ComputeCascadeEngine.triggerUpstream = (async (event: any) => {
      calls.upstream.push(event);
    }) as any;

    RepositoryFactory.setRepository(DeleteOpsModel as any, repository as any);

    const count = await DeleteOperations.Delete(DeleteOpsModel as any, ['Status', '=', 'draft'] as any);

    expect(count).toBe(2);
    expect(calls.search.length).toBe(1);
    expect(calls.search[0]?.options?.fields).toEqual(['Id', 'ParentId']);
    expect(calls.delete).toEqual([['Status', '=', 'draft']]);
    expect(calls.upstream.length).toBe(2);
    expect(calls.upstream[0]?.operation).toBe('delete');
    expect(calls.upstream[0]?.childCtor).toBe(DeleteOpsModel);
  } finally {
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollect;
    ComputeCascadeEngine.triggerUpstream = originalTrigger;
  }
});

test('model delete swallows upstream failures with warning and keeps delete result', async () => {
  const originalCollect = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalTrigger = ComputeCascadeEngine.triggerUpstream;
  const originalWarn = console.warn;

  const warnCalls: any[] = [];
  const repository = {
    async search() {
      return [{ Id: 'r1' }];
    },
    async delete() {
      return [];
    },
  };

  try {
    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {
      throw new Error('upstream failed');
    }) as any;
    (console as any).warn = (...args: any[]) => warnCalls.push(args);

    RepositoryFactory.setRepository(DeleteOpsModel as any, repository as any);

    const count = await DeleteOperations.Delete(DeleteOpsModel as any, ['Id', '=', 'r1'] as any);
    expect(count).toBe(0);
    expect(warnCalls.length).toBe(1);
    expect(String(warnCalls[0]?.[0] || '')).toContain('upstream recompute failed');
  } finally {
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollect;
    ComputeCascadeEngine.triggerUpstream = originalTrigger;
    (console as any).warn = originalWarn;
  }
});

test('model delete supports onlyDeleted repository scope and oldRows fallback branch', async () => {
  const originalCollect = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalTrigger = ComputeCascadeEngine.triggerUpstream;

  const calls = {
    onlyDeleted: 0,
    search: 0,
    delete: 0,
    trigger: 0,
  };

  const scopedRepository = {
    async search() {
      calls.search += 1;
      return undefined;
    },
    async delete() {
      calls.delete += 1;
      return [{ Id: 'r1' }];
    },
  };

  const rootRepository = {
    onlyDeleted() {
      calls.onlyDeleted += 1;
      return scopedRepository;
    },
    withDeleted() {
      return scopedRepository;
    },
  };

  try {
    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {
      calls.trigger += 1;
    }) as any;

    RepositoryFactory.setRepository(DeleteOpsModel as any, rootRepository as any);

    const count = await DeleteOperations.Delete(
      DeleteOpsModel as any,
      ['Id', '=', 'r1'] as any,
      {
        onlyDeleted: true,
      } as any
    );

    expect(count).toBe(1);
    expect(calls.onlyDeleted).toBe(1);
    expect(calls.search).toBe(1);
    expect(calls.delete).toBe(1);
    expect(calls.trigger).toBe(0);
  } finally {
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollect;
    ComputeCascadeEngine.triggerUpstream = originalTrigger;
  }
});

test('model delete by id delegates with canonical id condition', async () => {
  const originalDelete = DeleteOperations.Delete;

  try {
    const calls: any[] = [];
    DeleteOperations.Delete = (async (_ctor: any, condition: any, options: any) => {
      calls.push({ condition, options });
      return 7;
    }) as any;

    const count = await DeleteOperations.DeleteById(DeleteOpsModel as any, 'row_99', {
      withDeleted: true,
    } as any);

    expect(count).toBe(7);
    expect(calls.length).toBe(1);
    expect(calls[0]?.condition).toEqual(['Id', '=', 'row_99']);
    expect(calls[0]?.options).toEqual({ withDeleted: true });
  } finally {
    DeleteOperations.Delete = originalDelete;
  }
});
