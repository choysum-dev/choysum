// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { hydrateAccessUiResourceIds } from '@/auth/service/models/_role_ui_projection';
import RoleUiResource from '@/auth/service/models/role_ui_resource';

test('hydrateAccessUiResourceIds maps allow MetaUiResourceId grants into AccessUiResourceIds', async () => {
  const orig = (RoleUiResource as any).Search;
  try {
    (RoleUiResource as any).Search = async () => [
      {
        RoleId: 'role_1',
        Mode: 'allow',
        MetaApplicationId: null,
        MetaUiResourceId: 'res_allow',
      },
      {
        RoleId: 'role_1',
        Mode: 'deny',
        MetaApplicationId: null,
        MetaUiResourceId: 'res_deny',
      },
      {
        RoleId: 'role_1',
        Mode: 'allow',
        MetaApplicationId: 'app_1',
        MetaUiResourceId: null,
      },
      {
        RoleId: 'role_2',
        Mode: 'allow',
        MetaApplicationId: null,
        MetaUiResourceId: 'res_other',
      },
    ];

    const rows = [{ Id: 'role_1' }, { Id: 'role_2' }, { Id: 'role_3' }];
    await hydrateAccessUiResourceIds(rows);

    expect(rows[0].AccessUiResourceIds).toEqual(['res_allow']);
    expect(rows[1].AccessUiResourceIds).toEqual(['res_other']);
    expect(rows[2].AccessUiResourceIds).toEqual([]);
  } finally {
    (RoleUiResource as any).Search = orig;
  }
});
