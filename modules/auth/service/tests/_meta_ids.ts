// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withPermissionGraphBypass } from '@/core/service/api/authz_bypass';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';

const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');

/** Test helper: unique live meta_model id for (application, name). */
export async function metaModelId(appName: string, modelName: string): Promise<string> {
  return await withPermissionGraphBypass(async () => {
    const rows = await MetaModel.Search(
      { And: [['Application', '=', appName], ['Name', '=', modelName]] } as any,
      { fields: ['Id'], limit: 1 } as any
    );
    return String(rows?.[0]?.Id || '').trim();
  });
}

/** Test helper: unique live meta_application id by name. */
export async function metaApplicationId(appName: string): Promise<string> {
  return await withPermissionGraphBypass(async () => {
    const rows = await MetaApplication.Search(['Name', '=', appName] as any, {
      fields: ['Id'],
      limit: 1,
    } as any);
    return String(rows?.[0]?.Id || '').trim();
  });
}
