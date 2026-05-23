// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import type { OnchangeHandlerMeta } from '../metadata/model';
import type BaseModel from '../model/model';
import type { OnchangeTrigger } from '../metadata/field';
import { ONCHANGE_DEFAULT_PRIORITY } from '../../runtime/onchange/constants';
import { asObjectRecord } from '@/core/utils/object';

interface OnchangeOptions {
  priority?: number;
  reads?: string[]; // Additional read-only dependencies, such as scalar fields or paths like 'PartnerId.Name'.
}

/**
 * Usage:
 *  @Onchange<SaleOrder>('CustomerId','OrderLines')
 *  _onchange_customer_or_lines() { ... }
 */
export function Onchange<T extends BaseModel = BaseModel>(...args: (OnchangeTrigger<T> | OnchangeOptions)[]): MethodDecorator {
  return (target, key) => {
    // Split the option bag: treat the last object argument as OnchangeOptions.
    let opt: OnchangeOptions = {};
    const lastArg = args.length ? args[args.length - 1] : undefined;
    const lastRecord = asObjectRecord(lastArg);
    if (lastRecord && !Array.isArray(lastArg)) {
      opt = {
        priority: typeof lastRecord.priority === 'number' ? lastRecord.priority : undefined,
        reads: Array.isArray(lastRecord.reads) ? lastRecord.reads.filter((x): x is string => typeof x === 'string') : undefined,
      };
      args = args.slice(0, -1);
    }

    // Normalize triggers from string and string-array arguments.
    const triggers: string[] = [];
    for (const a of args) {
      if (!a) continue;
      if (Array.isArray(a)) {
        for (const x of a) {
          if (typeof x === 'string' && x) triggers.push(x);
        }
      } else if (typeof a === 'string') {
        triggers.push(a);
      }
    }
    const uniqueTriggers = [...new Set(triggers)];

    // Normalize reads.
    let reads: string[] | undefined;
    if (Array.isArray(opt.reads)) {
      const tmp: string[] = [];
      for (const r of opt.reads) {
        if (typeof r === 'string' && r.trim()) {
          tmp.push(r.trim());
        }
      }
      if (tmp.length) reads = [...new Set(tmp)];
    }

    const ctor = target.constructor as unknown as typeof BaseModel;
    const meta = MetadataStorage.instance.getModelMetadata(ctor);
    const list: OnchangeHandlerMeta[] = [...(meta.onchangeHandlers || [])];

    list.push({
      method: key as string,
      triggers: uniqueTriggers,
      priority: typeof opt.priority === 'number' ? opt.priority : ONCHANGE_DEFAULT_PRIORITY,
      reads,
    });

    MetadataStorage.instance.setModelMetadata(ctor, { ...meta, onchangeHandlers: list });
  };
}
