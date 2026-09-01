// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveMethodAccessMeta } from '@/auth/service/models/user/_method_access';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';
import type MetaServiceModel from '@/meta/service/models/service';

const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');

test('resolveMethodAccessMeta returns undefined when MetaModel has no id', async () => {
  const orig = (MetaModel as any).Search;
  try {
    (MetaModel as any).Search = async () => [];
    expect(await resolveMethodAccessMeta('auth', 'User', 'Browse')).toBeUndefined();

    (MetaModel as any).Search = async () => [{}];
    expect(await resolveMethodAccessMeta('auth', 'User', 'Browse')).toBeUndefined();

    (MetaModel as any).Search = async () => null;
    expect(await resolveMethodAccessMeta('auth', 'User', 'Browse')).toBeUndefined();
  } finally {
    (MetaModel as any).Search = orig;
  }
});

test('resolveMethodAccessMeta skips application scope when MetaApplication id empty', async () => {
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;
  const origService = (MetaService as any).Search;
  try {
    (MetaModel as any).Search = async () => [{ Id: 'm1' }];
    (MetaService as any).Search = async () => [{ Id: 's1', Name: 'Browse' }];
    (MetaApplication as any).Search = async () => [];

    const meta = await resolveMethodAccessMeta('auth', 'User', 'Browse');
    expect(meta).toMatchObject({ modelId: 'm1', irServiceId: 's1', irApplicationId: '' });
    expect(JSON.stringify(meta!.scopeOr).includes('"MetaApplicationId","=","')).toBe(false);

    (MetaApplication as any).Search = async () => [{ Id: '' }];
    const metaEmpty = await resolveMethodAccessMeta('demo', 'Widget', 'Browse');
    expect(metaEmpty?.irApplicationId).toBe('');
  } finally {
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
    (MetaService as any).Search = origService;
  }
});

test('resolveMethodAccessMeta inserts application scope when MetaApplication id present', async () => {
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;
  const origService = (MetaService as any).Search;
  try {
    (MetaModel as any).Search = async () => [{ Id: 'm1' }];
    (MetaService as any).Search = async () => [{ Id: 's1', Name: 'Browse' }];
    (MetaApplication as any).Search = async () => [{ Id: 'a1' }];

    const meta = await resolveMethodAccessMeta('auth', 'User', 'Browse');
    expect(meta?.irApplicationId).toBe('a1');
    expect(meta!.scopeOr.some((c: any) => Array.isArray(c.And) && c.And.some((t: any) => t[0] === 'MetaApplicationId' && t[2] === 'a1'))).toBe(
      true
    );
  } finally {
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
    (MetaService as any).Search = origService;
  }
});
