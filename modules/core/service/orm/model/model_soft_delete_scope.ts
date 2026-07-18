// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import { getModelRepository } from './model_internal_facade';
import type { Repository } from '../repository';
import type { RuntimeModelCtor } from './types';
import { _t } from '@/core/service/i18n_binder';

type SoftDeleteOptionLike =
  | {
      withDeleted?: boolean;
      onlyDeleted?: boolean;
    }
  | undefined;

function resolveErrorDomain(ModelCtor: unknown): string {
  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor as Parameters<typeof MetadataStorage.instance.getModelMetadata>[0]);
  return typeof meta?.application === 'string' && meta.application.trim() ? meta.application.trim() : 'core';
}

function assertSoftDeleteOptionsValid(ModelCtor: unknown, options?: SoftDeleteOptionLike): void {
  if (!options) return;
  if (options.withDeleted && options.onlyDeleted) {
    throw new ChoysumError({
      domain: resolveErrorDomain(ModelCtor),
      code: 'InvalidArgument',
      message: _t('withDeleted and onlyDeleted cannot both be true', { scope: 'service/orm/model/model_soft_delete_scope' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
}

export function resolveRepositoryWithSoftDeleteOptions(ModelCtor: unknown, options?: SoftDeleteOptionLike): Repository {
  const repository = getModelRepository(ModelCtor as RuntimeModelCtor);
  assertSoftDeleteOptionsValid(ModelCtor, options);
  if (options?.onlyDeleted) return repository.onlyDeleted();
  if (options?.withDeleted) return repository.withDeleted();
  return repository;
}
