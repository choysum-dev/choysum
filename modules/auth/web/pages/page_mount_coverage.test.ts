// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import Login from './Login.vue';
import Role from './Role.vue';
import RoleList from './RoleList.vue';
import RoleFieldRule from './RoleFieldRule.vue';
import RoleFieldRuleList from './RoleFieldRuleList.vue';
import RoleMethodAccess from './RoleMethodAccess.vue';
import RoleMethodAccessList from './RoleMethodAccessList.vue';
import RoleRecordRule from './RoleRecordRule.vue';
import RoleRecordRuleList from './RoleRecordRuleList.vue';
import RoleUiResource from './RoleUiResource.vue';
import RoleUiResourceList from './RoleUiResourceList.vue';
import Session from './Session.vue';
import SessionList from './SessionList.vue';
import Token from './Token.vue';
import TokenChart from './TokenChart.vue';
import TokenKanban from './TokenKanban.vue';
import TokenList from './TokenList.vue';
import User from './User.vue';
import UserList from './UserList.vue';

test('auth page mount: mounts real Login.vue under choysumMount', async () => {
  const wrapper = mount(Login as any, {
    global: buildPageMountGlobal({ route: { path: '/login', query: {} } }),
  });
  await flushPromises();
  // Assert the real login affordances; generic stubs alone must not satisfy this test.
  expect(wrapper.find('.login-username').exists()).toBe(true);
  expect(wrapper.find('.login-password').exists()).toBe(true);
  wrapper.unmount();
});

/** ChoyPage hosts that call createModelStore in script setup (patch coverage). */
const modelStorePages: Array<[string, any, string]> = [
  ['Role', Role, '/auth/roles/1'],
  ['RoleList', RoleList, '/auth/roles'],
  ['RoleFieldRule', RoleFieldRule, '/auth/role-field-rules/1'],
  ['RoleFieldRuleList', RoleFieldRuleList, '/auth/role-field-rules'],
  ['RoleMethodAccess', RoleMethodAccess, '/auth/role-method-accesses/1'],
  ['RoleMethodAccessList', RoleMethodAccessList, '/auth/role-method-accesses'],
  ['RoleRecordRule', RoleRecordRule, '/auth/role-record-rules/1'],
  ['RoleRecordRuleList', RoleRecordRuleList, '/auth/role-record-rules'],
  ['RoleUiResource', RoleUiResource, '/auth/role-ui-resources/1'],
  ['RoleUiResourceList', RoleUiResourceList, '/auth/role-ui-resources'],
  ['Session', Session, '/auth/sessions/1'],
  ['SessionList', SessionList, '/auth/sessions'],
  ['Token', Token, '/auth/tokens/1'],
  ['TokenChart', TokenChart, '/auth/tokens/chart'],
  ['TokenKanban', TokenKanban, '/auth/tokens/kanban'],
  ['TokenList', TokenList, '/auth/tokens'],
  ['User', User, '/auth/users/1'],
  ['UserList', UserList, '/auth/users'],
];

test('auth page mount: every createModelStore ChoyPage host mounts under choysumMount', async () => {
  const failures: string[] = [];
  for (const [name, Comp, path] of modelStorePages) {
    let wrapper: ReturnType<typeof mount> | null = null;
    try {
      wrapper = mount(Comp as any, {
        global: buildPageMountGlobal({ route: { path, fullPath: path } }),
      });
      await flushPromises();
      const ok =
        wrapper.find('[data-anchor="choy.page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-child-view"]').exists();
      if (!ok) failures.push(name);
    } catch (error) {
      failures.push(`${name}: ${(error as Error).message || String(error)}`);
    } finally {
      wrapper?.unmount();
    }
  }
  if (failures.length) throw new Error(`auth page mount failed: ${failures.join(', ')}`);
});
