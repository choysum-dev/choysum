// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * HC6: omitting createServiceByModel / dial / pool type args must default to
 * `never` (unusable), not fall back to ModelConstructor.
 */

import type { ModelService } from '../../rpc/types';
import { createServiceByModel } from '../rpc/service_factory';
import { dial, pool } from '../orm/model/model_pool';

type Expect<T extends true> = T;

if (false) {
  const omittedDial = dial('hc6.OmitCtor');
  const omittedPool = pool('hc6', 'OmitCtor');
  const omittedFactory = createServiceByModel('hc6.OmitCtor');

  type _DialOmitsCtor = Expect<typeof omittedDial extends ModelService<never> ? true : false>;
  type _PoolOmitsCtor = Expect<[typeof omittedPool] extends [never] ? true : false>;
  type _FactoryOmitsCtor = Expect<typeof omittedFactory extends ModelService<never> ? true : false>;

  const _keep: [_DialOmitsCtor, _PoolOmitsCtor, _FactoryOmitsCtor] | undefined = undefined;
  void _keep;
}
