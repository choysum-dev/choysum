<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//div[@class='o-header__actions-primary']" position="inside">
    <el-divider direction="vertical" class="o-header__action-divider" />
    <el-button v-if="!isAuthenticated" text class="o-header__action-btn o-header__action-item" aria-label="Log in" @click="handleLogin"> Log In </el-button>
    <el-button v-if="isAuthenticated" text class="o-header__action-btn o-header__action-item" aria-label="Notifications" @click="handleNotificationClick">
      <el-icon :size="20"><Bell /></el-icon>
    </el-button>
    <OSwitchCompany v-if="isAuthenticated" />
    <el-dropdown v-if="isAuthenticated" trigger="click" class="o-header__action-item" placement="bottom-end">
      <el-button text class="o-header__action-btn" aria-label="User menu">
        <el-icon :size="20"><User /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item>
            <span @click="handleProfileClick">Profile</span>
          </el-dropdown-item>
          <el-dropdown-item>
            <span @click="handleSettingsClick">Settings</span>
          </el-dropdown-item>
          <el-dropdown-item divided>
            <span @click="handleLogout">Log Out</span>
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </Xpath>
</template>

<script lang="ts" _name="OHeader">
import { defineComponent, computed } from 'vue';
import { Xpath } from '@/core/web';
import OHeader from '@/web/web/components/layout/OHeader.vue';
import { Bell, User } from '@element-plus/icons-vue';
import { ElDivider, ElButton, ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/auth/web/stores/auth';
import OSwitchCompany from './OSwitchCompany.vue';

/**
 * Extend the shared header with auth-specific actions and menus.
 */
export default defineComponent({
  extends: OHeader,
  components: {
    Xpath,
    Bell,
    User,
    ElDivider,
    ElButton,
    ElDropdown,
    ElDropdownMenu,
    ElDropdownItem,
    OSwitchCompany,
  },
  setup(props, ctx) {
    const baseSetup = OHeader?.setup?.(props, ctx) || {};
    const router = useRouter();
    const authStore = useAuthStore();
    const isAuthenticated = computed(() => authStore.isAuthenticated);

    /**
     * Navigate to the login page.
     */
    function handleLogin() {
      router.push({ name: 'login' });
    }

    /**
     * Ask the parent layout to open notifications.
     */
    function handleNotificationClick() {
      ctx.emit('show-notifications');
    }

    /**
     * Ask the parent layout to open the profile panel.
     */
    function handleProfileClick() {
      ctx.emit('show-profile');
    }

    /**
     * Ask the parent layout to open settings.
     */
    function handleSettingsClick() {
      ctx.emit('show-settings');
    }

    /**
     * Navigate to the logout page.
     */
    function handleLogout() {
      router.push({ name: 'logout' });
    }

    return {
      ...baseSetup,
      isAuthenticated,
      handleLogin,
      handleNotificationClick,
      handleProfileClick,
      handleSettingsClick,
      handleLogout,
    };
  },
});
</script>

<style lang="scss" scoped>
.o-header__action-btn {
  position: relative;
  height: 36px;
  width: 36px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  border-radius: var(--el-border-radius-base);
  cursor: pointer;

  &:hover {
    color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
  }
}

.o-header__action-divider {
  margin: 0 8px;
  height: 24px;
  align-self: center;
}
</style>
