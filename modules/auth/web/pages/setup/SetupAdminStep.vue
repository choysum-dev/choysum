<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Reserve owns inter-field rhythm; gap-0 avoids stacking group gap on error slots. -->
  <ChoyFieldGroup class="setup-fields setup-admin-step !gap-0">
    <div class="setup-username">
      <ChoyVarcharField
        :store="store"
        prop="Username"
        :label="_t('Username')"
        orientation="vertical"
        :placeholder="_t('Enter username')"
        autocomplete="username"
        name="username"
        id="setup-username"
        buffer-strategy="live"
        :nullable="false"
        :show-word-limit="false"
        show-inline-error
        inline-error-space="reserve"
        :rules="usernameRules"
      />
    </div>
    <div class="setup-password">
      <ChoyVarcharField
        :store="store"
        prop="Password"
        :label="_t('Password')"
        orientation="vertical"
        widget="password"
        :placeholder="_t('Enter password')"
        autocomplete="new-password"
        name="password"
        id="setup-password"
        buffer-strategy="live"
        :nullable="false"
        :show-word-limit="false"
        show-inline-error
        inline-error-space="reserve"
        :rules="passwordRules"
      />
    </div>
    <div class="setup-confirm">
      <ChoyVarcharField
        :store="store"
        prop="ConfirmPassword"
        :label="_t('Confirm Password')"
        orientation="vertical"
        widget="password"
        :placeholder="_t('Re-enter password')"
        autocomplete="new-password"
        name="confirmPassword"
        id="setup-confirm"
        buffer-strategy="live"
        :nullable="false"
        :show-word-limit="false"
        show-inline-error
        inline-error-space="reserve"
        :rules="confirmPasswordRules"
      />
    </div>
    <div class="setup-email">
      <ChoyVarcharField
        :store="store"
        prop="Email"
        :label="_t('Email')"
        orientation="vertical"
        :placeholder="_t('Optional')"
        autocomplete="email"
        name="email"
        id="setup-email"
        buffer-strategy="live"
        :show-word-limit="false"
        show-inline-error
        inline-error-space="reserve"
        :rules="emailRules"
      />
    </div>
  </ChoyFieldGroup>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChoyFieldGroup, ChoyVarcharField, type LocalFormStore } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import {
  setupConfirmPasswordRules,
  setupEmailRules,
  setupPasswordRules,
  setupUsernameRules,
  setupRuleTranslate,
  withSetupServerError,
} from '../setup_form';

const { _t } = createTranslate('auth', { scope: 'web/pages/setup/SetupAdminStep' });

/** Administrator account fields of the setup wizard. */
const props = defineProps<{
  store: LocalFormStore;
  formData: Record<string, unknown>;
  /** Server field errors keyed by local field name. */
  serverErrors: Record<string, string>;
}>();

const usernameRules = computed(() => withSetupServerError(setupUsernameRules(setupRuleTranslate), props.serverErrors.Username));
const passwordRules = computed(() => withSetupServerError(setupPasswordRules(setupRuleTranslate), props.serverErrors.Password));
const confirmPasswordRules = computed(() => {
  const password = props.formData.Password;
  return setupConfirmPasswordRules(setupRuleTranslate, () => password);
});
const emailRules = computed(() => withSetupServerError(setupEmailRules(setupRuleTranslate), props.serverErrors.Email));
</script>
