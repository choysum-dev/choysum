// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '@/core/service/error';
import BaseModel from './model';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import { resolveRepositoryWithSoftDeleteOptions } from './model_soft_delete_scope';

class SoftDeleteScopeCoreModel extends BaseModel {}
class SoftDeleteScopeAuthModel extends BaseModel {}

test('resolveRepositoryWithSoftDeleteOptions routes to base/withDeleted/onlyDeleted repository', () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  const calls: string[] = [];
  const baseRepository = {
    tag: 'base',
    withDeleted() {
      calls.push('withDeleted');
      return { tag: 'withDeleted' };
    },
    onlyDeleted() {
      calls.push('onlyDeleted');
      return { tag: 'onlyDeleted' };
    },
  };

  try {
    RepositoryFactory.getRepository = (() => baseRepository as any) as any;

    const base = resolveRepositoryWithSoftDeleteOptions(SoftDeleteScopeCoreModel as any);
    const withDeleted = resolveRepositoryWithSoftDeleteOptions(SoftDeleteScopeCoreModel as any, { withDeleted: true });
    const onlyDeleted = resolveRepositoryWithSoftDeleteOptions(SoftDeleteScopeCoreModel as any, { onlyDeleted: true });

    expect(base).toBe(baseRepository as any);
    expect(withDeleted).toEqual({ tag: 'withDeleted' });
    expect(onlyDeleted).toEqual({ tag: 'onlyDeleted' });
    expect(calls).toEqual(['withDeleted', 'onlyDeleted']);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('resolveRepositoryWithSoftDeleteOptions throws InvalidArgument ChoysumError with resolved domain', () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    MetadataStorage.instance.setModelMetadata(SoftDeleteScopeAuthModel as any, { application: 'auth' } as any);
    RepositoryFactory.getRepository = (() => ({ withDeleted: () => ({}), onlyDeleted: () => ({}) }) as any) as any;

    let error: unknown;
    try {
      resolveRepositoryWithSoftDeleteOptions(SoftDeleteScopeAuthModel as any, {
        withDeleted: true,
        onlyDeleted: true,
      });
    } catch (e) {
      error = e;
    }

    expect(error instanceof ChoysumError).toBe(true);
    expect((error as ChoysumError).domain).toBe('auth');
    expect((error as ChoysumError).grpcCode).toBe(GrpcCode.InvalidArgument);
    expect((error as ChoysumError).message).toContain('withDeleted and onlyDeleted cannot both be true');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('resolveRepositoryWithSoftDeleteOptions falls back to core domain when application is empty', () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    MetadataStorage.instance.setModelMetadata(SoftDeleteScopeCoreModel as any, { application: '   ' } as any);
    RepositoryFactory.getRepository = (() => ({ withDeleted: () => ({}), onlyDeleted: () => ({}) }) as any) as any;

    let domain = '';
    try {
      resolveRepositoryWithSoftDeleteOptions(SoftDeleteScopeCoreModel as any, {
        withDeleted: true,
        onlyDeleted: true,
      });
    } catch (e) {
      domain = String((e as ChoysumError).domain || '');
    }

    expect(domain).toBe('core');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});
