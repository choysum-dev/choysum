<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//*[@data-anchor='choy.shell.header-actions']" position="inside">
    <ChoyNotificationBell v-if="collaborationCaps.message" />
  </Xpath>
  <Xpath expr="//*[@data-anchor='choy.shell.sidebar-header']" position="inside">
    <SwitchCompany variant="sidebar" />
  </Xpath>
  <Xpath expr="//*[@data-anchor='choy.shell.sidebar-footer']" position="inside">
    <!-- Identity + session connectivity — not a SidebarMenu ul/li (footer is not a nav list). -->
    <AuthSidebarUserMenu />
  </Xpath>
</template>

<script lang="ts" _name="ChoyAppShell">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import { ChoyNotificationBell } from '@/web';
import ChoyAppShell from '@/web/web/components/layout/ChoyAppShell.vue';
import { useReactiveCollaborationCaps } from '@/web/web/platform/collaborationCaps';
import { reuseParentSetupState } from './reuse_parent_setup_state';
import AuthSidebarUserMenu from './AuthSidebarUserMenu.vue';
import SwitchCompany from './SwitchCompany.vue';

/**
 * Extends the app shell so session chrome merges into shell anchors at build time:
 * notifications → header-actions; company → sidebar-header; user → sidebar-footer.
 */
export default defineComponent({
  name: 'ChoyAppShell',
  extends: ChoyAppShell,
  components: {
    Xpath,
    ChoyNotificationBell,
    SwitchCompany,
    AuthSidebarUserMenu,
  },
  setup(props, ctx) {
    const baseSetupFn = (ChoyAppShell as any)?.setup;
    if (typeof baseSetupFn !== 'function') {
      throw new Error('auth ChoyAppShell: base web ChoyAppShell exposes no setup() to merge');
    }
    const baseSetup = reuseParentSetupState(baseSetupFn(props, ctx));
    const collaborationCaps = useReactiveCollaborationCaps();
    return {
      ...baseSetup,
      collaborationCaps,
    };
  },
});
</script>
