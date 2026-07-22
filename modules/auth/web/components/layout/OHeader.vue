<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//div[@class='o-header__actions-primary']" position="inside">
    <el-divider direction="vertical" class="o-header__action-divider" />
    <el-button v-if="!isAuthenticated" text class="o-header__action-btn o-header__action-item" :aria-label="_t('Log in')" @click="handleLogin">
      {{ _t('Log In') }}
    </el-button>
    <el-button v-if="isAuthenticated" text class="o-header__action-btn o-header__action-item" :aria-label="_t('Notifications')" @click="handleNotificationClick">
      <el-icon :size="20"><Bell /></el-icon>
    </el-button>
    <OSwitchCompany v-if="isAuthenticated" />
    <el-dropdown v-if="isAuthenticated" trigger="click" class="o-header__action-item" placement="bottom-end">
      <el-button text class="o-header__action-btn" :aria-label="_t('User menu')">
        <el-icon :size="20"><User /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item @click="openPreferences">{{ _t('Profile') }}</el-dropdown-item>
          <el-dropdown-item @click="openPreferences">{{ _t('Settings') }}</el-dropdown-item>
          <el-dropdown-item divided @click="handleLogout">{{ _t('Log Out') }}</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <OPreferencesDialog v-if="isAuthenticated" v-model="preferencesVisible" />
  </Xpath>
</template>

<script lang="ts" _name="OHeader">
import { defineComponent, computed, ref } from 'vue';
import { Xpath } from '@/core/web';
import OHeader from '@/web/web/components/layout/OHeader.vue';
import { Bell, User } from '@element-plus/icons-vue';
import { ElDivider, ElButton, ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/auth/web/stores/auth';
import OSwitchCompany from './OSwitchCompany.vue';
import OPreferencesDialog from '../preferences/OPreferencesDialog.vue';
import { createTranslate } from '@/web/web/i18n';

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
    OPreferencesDialog,
  },
  setup(props, ctx) {
    const baseSetup = OHeader?.setup?.(props, ctx) || {};
    const { _t } = createTranslate('auth', { scope: 'web/components/layout/OHeader' });
    const router = useRouter();
    const authStore = useAuthStore();
    const isAuthenticated = computed(() => authStore.isAuthenticated);
    const preferencesVisible = ref(false);

    function handleLogin() {
      router.push({ name: 'login' });
    }

    function handleNotificationClick() {
      ctx.emit('show-notifications');
    }

    /** Profile and Settings open the same Preferences dialog (L13 / L18). */
    function openPreferences() {
      preferencesVisible.value = true;
    }

    function handleLogout() {
      router.push({ name: 'logout' });
    }

    return {
      ...baseSetup,
      _t,
      isAuthenticated,
      preferencesVisible,
      handleLogin,
      handleNotificationClick,
      openPreferences,
      handleLogout,
    };
  },
});
</script>

<style lang="scss" scoped>
.o-header {
  &__action-divider {
    height: 20px;
    margin: 0 4px;
  }
}
</style>
