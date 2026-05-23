// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { ModelMetadata } from '../metadata';
import type { RuntimeModelCtor } from './types';
import { MetadataStorage } from '../metadata/storage';
import { buildComputeGraph } from '../../runtime/compute/graph';
import { EntityConverter } from '../utils/converter';
import type { ObjectRecord } from '../../../utils/types';

type ModelCtorMetadataCarrier = {
  metadata?: ModelMetadata;
};

type BaseModelRuntimeState = {
  entity: ObjectRecord;
  fields?: unknown;
};

export function getCachedModelMetadata<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>): ModelMetadata {
  const runtimeCtor = ModelCtor as unknown as ModelCtorMetadataCarrier;
  if (!runtimeCtor.metadata) {
    runtimeCtor.metadata = MetadataStorage.instance.getModelMetadata(ModelCtor);
    if (!runtimeCtor.metadata.computeGraph) {
      runtimeCtor.metadata.computeGraph = buildComputeGraph(runtimeCtor.metadata);
    }
  }
  return runtimeCtor.metadata;
}

export function markPlain<T>(val: T): T {
  if (val && typeof val === 'object') {
    try {
      Object.defineProperty(val as ObjectRecord, '__choysum_plain', {
        value: true,
        enumerable: false,
        configurable: false,
        writable: false,
      });
    } catch {}
  }
  return val as T;
}

export function markPlainShallow<T>(val: T): T {
  return markPlain(val) as T;
}

export function toTransportObject(instance: BaseModel): ObjectRecord {
  const ctor = instance.constructor as RuntimeModelCtor;
  const runtimeState = instance as unknown as BaseModelRuntimeState;
  const payload = EntityConverter.entityToPlainObject(ctor, runtimeState.entity, runtimeState.fields as never);
  return markPlainShallow(payload);
}
