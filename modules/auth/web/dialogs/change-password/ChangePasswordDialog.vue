<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-change-password-dialog max-w-md gap-4 p-5"
      :show-close-button="false"
      :dismissible="!saving"
      data-testid="change-password-dialog"
    >
      <ChoyDialogHeader class="space-y-1 text-start">
        <ChoyDialogTitle>{{ _t('Change password') }}</ChoyDialogTitle>
        <ChoyDialogDescription>
          {{ _t('Update if compromised or rotating credentials.') }}
        </ChoyDialogDescription>
      </ChoyDialogHeader>

      <!-- Local FormView (not auth.User): same command-form pattern as Login/Register. -->
      <ChoyFormView
        :store="formStore"
        view-mode="create"
        embedded
        :show-header="false"
        :show-actions="false"
        :show-messages="false"
        :resolve-record-id-from-route="false"
        :initial-values="emptyValues"
        :submit-handler="onSubmit"
      >
        <template #default="{ formData, loading }">
          <div class="flex flex-col gap-2.5">
            <ChoyVarcharField
              :store="formStore"
              prop="CurrentPassword"
              :label="_t('Current password')"
              orientation="vertical"
              widget="password"
              autocomplete="current-password"
              id="change-password-current"
              name="current-password"
              buffer-strategy="live"
              :nullable="false"
              :show-word-limit="false"
              show-inline-error
              :rules="currentPasswordRules"
              data-testid="change-password-current"
            />
            <ChoyVarcharField
              :store="formStore"
              prop="NewPassword"
              :label="_t('New password')"
              orientation="vertical"
              widget="password"
              autocomplete="new-password"
              id="change-password-new"
              name="new-password"
              buffer-strategy="live"
              :nullable="false"
              :show-word-limit="false"
              show-inline-error
              :rules="newPasswordRules(() => formData.CurrentPassword)"
              data-testid="change-password-new"
            />
            <ChoyVarcharField
              :store="formStore"
              prop="ConfirmPassword"
              :label="_t('Confirm new password')"
              orientation="vertical"
              widget="password"
              autocomplete="new-password"
              id="change-password-confirm"
              name="confirm-password"
              buffer-strategy="live"
              :nullable="false"
              :show-word-limit="false"
              show-inline-error
              :rules="confirmPasswordRules(() => formData.NewPassword)"
              data-testid="change-password-confirm"
            />
          </div>

          <ChoyDialogFooter class="mt-4">
            <ChoyButton type="button" variant="outline" :disabled="saving || loading" @click="close">{{
              _t('Cancel')
            }}</ChoyButton>
            <ChoyButton
              type="submit"
              :disabled="saving || loading"
              data-testid="change-password-submit"
            >
              {{ _t('Change password') }}
            </ChoyButton>
          </ChoyDialogFooter>
        </template>
      </ChoyFormView>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import {
  ChoyButton,
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogDescription,
  ChoyDialogFooter,
  ChoyDialogHeader,
  ChoyDialogTitle,
  ChoyFormView,
  ChoyMessage,
  ChoyVarcharField,
  createLocalFormStore,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { hashPasswordClient } from '@/auth/web/stores/auth/utils';
import { createStoreByModel } from '@/web/web/stores/registry';
import {
  changePasswordConfirmRules,
  changePasswordCurrentRules,
  changePasswordNewRules,
} from './change_password';

defineOptions({ name: 'ChangePasswordDialog' });

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [boolean] }>();

const { _t } = createTranslate('auth', {
  scope: 'web/dialogs/change-password/ChangePasswordDialog',
});
const authStore = useAuthStore();
const userStore = createStoreByModel('auth.User');
const saving = ref(false);

const emptyValues = {
  CurrentPassword: '',
  NewPassword: '',
  ConfirmPassword: '',
};

const formStore = createLocalFormStore({
  storeId: 'auth.change-password',
  fields: [
    { name: 'CurrentPassword', label: 'Current password', type: 'varchar' },
    { name: 'NewPassword', label: 'New password', type: 'varchar' },
    { name: 'ConfirmPassword', label: 'Confirm new password', type: 'varchar' },
  ],
  initialValues: emptyValues,
});

const currentPasswordRules = computed(() => changePasswordCurrentRules(_t));
const newPasswordRules = (getCurrent: () => unknown) =>
  changePasswordNewRules(_t, getCurrent);
const confirmPasswordRules = (getNew: () => unknown) =>
  changePasswordConfirmRules(_t, getNew);

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

function close(): void {
  visible.value = false;
}

function resetForm(): void {
  formStore.setField('CurrentPassword', '');
  formStore.setField('NewPassword', '');
  formStore.setField('ConfirmPassword', '');
}

watch(
  () => props.modelValue,
  open => {
    if (!open) return;
    resetForm();
  },
);

async function onSubmit(ctx: { formData: Record<string, unknown> }) {
  // Field rules + FormView validate run first; this path is RPC-only.
  const formData = ctx.formData || {};
  const currentPlain = String(formData.CurrentPassword ?? '');
  const newPlain = String(formData.NewPassword ?? '');
  const username = String((authStore.currentUser as any)?.Username || '').trim();
  if (!username) {
    ChoyMessage.error(_t('Cannot change password: missing username'));
    return { handled: true, skipSuccessMessage: true };
  }
  saving.value = true;
  try {
    const currentPassword = await hashPasswordClient(currentPlain, username);
    const newPassword = await hashPasswordClient(newPlain, username);
    await (userStore as any).ChangePassword({
      CurrentPassword: currentPassword,
      NewPassword: newPassword,
    });
    try {
      await authStore.refreshToken(true);
    } catch {
      // Password already changed; keep session if refresh fails.
    }
    ChoyMessage.success(_t('Password changed'));
    close();
  } catch (err) {
    const message = err instanceof Error ? String(err.message || '').trim() : String(err ?? '').trim();
    ChoyMessage.error(message || _t('Failed to change password'));
  } finally {
    saving.value = false;
  }
  return { handled: true, skipSuccessMessage: true };
}
</script>
