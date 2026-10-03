<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="choy-preferences-dialog fixed inset-0 z-[100] flex items-center justify-center bg-black/50 p-4"
      role="presentation"
      @click.self="visible = false"
    >
      <div
        ref="dialogRef"
        tabindex="-1"
        class="w-full max-w-lg rounded-lg border border-border bg-background p-6 shadow-lg outline-none"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        @click.stop
        @keydown="onDialogKeydown"
      >
        <h2 :id="titleId" class="text-lg font-semibold">{{ _t('Edit Profile') }}</h2>

        <div v-if="currentUser" class="choy-preferences-dialog__header mb-4 mt-3">
          <div class="choy-preferences-dialog__identity">
            <div class="choy-preferences-dialog__name font-semibold text-foreground">{{ displayName }}</div>
            <div class="choy-preferences-dialog__email text-sm text-foreground/70">{{ currentUser.Email || '' }}</div>
          </div>
        </div>

        <form v-if="!userId" class="choy-preferences-dialog__form flex flex-col gap-4" @submit.prevent="onMissingUserSubmit">
          <div class="flex justify-end gap-2 pt-2">
            <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
            <ChoyButton type="submit">{{ _t('Update preferences') }}</ChoyButton>
          </div>
        </form>

        <ChoyFormView
          v-else
          :store="userStore"
          :record-id="userId"
          view-mode="edit"
          embedded
          :show-header="false"
          :show-actions="false"
          :show-messages="false"
          :resolve-record-id-from-route="false"
          :submit-handler="onPreferencesSubmit"
        >
          <template #default="{ formData, loading }">
            <PreferenceDraftSeeder
              :ready="!loading"
              :language-id="seedLanguageId"
              :timezone="seedTimezone"
            />
            <ChoyFieldGroup class="choy-preferences-dialog__form gap-4">
              <div class="choy-preferences-dialog__language">
                <ChoyManyToOneRefField
                  :store="userStore"
                  prop="LanguageId"
                  data-testid="preferences-language"
                  show-inline-error
                />
                <div v-if="showLanguageSessionHint(formData)" class="choy-preferences-dialog__hint text-xs text-foreground/60">
                  {{ _t('Using current session language') }}
                </div>
              </div>
              <div class="choy-preferences-dialog__timezone">
                <ChoySelectionField
                  :store="userStore"
                  prop="Timezone"
                  :placeholder="_t('Select timezone')"
                  :select-props="{ 'data-testid': 'preferences-timezone' }"
                  show-inline-error
                />
                <div v-if="showTimezoneBrowserHint(formData)" class="choy-preferences-dialog__hint text-xs text-foreground/60">
                  {{ _t('Suggested from your browser') }}
                </div>
              </div>
              <div class="flex justify-end gap-2 pt-2">
                <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
                <ChoyButton type="submit" :disabled="saving || loading">{{ _t('Update preferences') }}</ChoyButton>
              </div>
            </ChoyFieldGroup>
          </template>
        </ChoyFormView>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useId, watch } from 'vue';
import {
  ChoyButton,
  ChoyFieldGroup,
  ChoyFormView,
  ChoyManyToOneRefField,
  ChoyMessage,
  ChoySelectionField,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useI18nStore, langToUiKey, afterLocaleChange, softLocaleRemount } from '@/web/web/stores/i18nStore';
import { createStoreByModel } from '@/web/web/stores/registry';
import { applyUserLanguagePreference } from '@/auth/web/stores/auth/language_preference';
import {
  detectBrowserTimezone,
  resolvePreferenceLanguage,
  resolvePreferenceTimezone,
} from './preferences_defaults';
import { restoreDialogFocus } from './dialog_focus_restore';
import { trapDialogTabKey } from './dialog_focus_trap';
import { PreferenceDraftSeeder } from './PreferenceDraftSeeder';
import { runPreferencesSubmit } from './preferences_submit';

defineOptions({ name: 'PreferencesDialog' });

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [boolean]; closed: [] }>();

const { _t } = createTranslate('auth', { scope: 'web/components/preferences/PreferencesDialog' });
const authStore = useAuthStore();
const i18nStore = useI18nStore();
const userStore = createStoreByModel('auth.User');
const languageStore = createStoreByModel('base.Language');
const titleId = useId();
const dialogRef = ref<HTMLElement | null>(null);
let lastFocused: HTMLElement | null = null;

const visible = computed({
  get: () => props.modelValue,
  set: v => {
    emit('update:modelValue', v);
    if (!v) emit('closed');
  },
});

/** Escape dismisses; Tab stays inside the dialog until it closes. */
function onDialogKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    visible.value = false;
    return;
  }
  const root = dialogRef.value as HTMLElement | null;
  if (root) trapDialogTabKey(event, root);
}

const currentUser = computed(() => authStore.currentUser as any);
const displayName = computed(() => {
  const u = currentUser.value;
  if (!u) return '';
  return String(u.DisplayName || u.Username || u.Email || '');
});

const userId = computed(() => String(currentUser.value?.Id || authStore.identity?.userId || '').trim());
const seedLanguageId = ref('');
const seedTimezone = ref('');
const languageFromSession = ref(false);
const timezoneFromBrowser = ref(false);
const saving = ref(false);

function languageRefId(value: unknown): string {
  if (value == null || Array.isArray(value)) return '';
  if (typeof value !== 'object') return String(value).trim();
  if (!('Id' in value)) return '';
  return String((value as { Id?: unknown }).Id ?? '').trim();
}

function timezoneText(value: unknown): string {
  return value == null ? '' : String(value).trim();
}

function showLanguageSessionHint(formData: Record<string, unknown>): boolean {
  if (!languageFromSession.value) return false;
  return languageRefId(formData?.LanguageId) === languageRefId(seedLanguageId.value);
}

function showTimezoneBrowserHint(formData: Record<string, unknown>): boolean {
  if (!timezoneFromBrowser.value) return false;
  return timezoneText(formData?.Timezone) === seedTimezone.value;
}

async function resolveSeedLanguageId(): Promise<void> {
  const savedId = languageRefId(currentUser.value?.LanguageId);
  if (savedId) {
    seedLanguageId.value = savedId;
    languageFromSession.value = false;
    return;
  }
  const session = resolvePreferenceLanguage(null, i18nStore.terminologyLang);
  languageFromSession.value = session.fromSession;
  const code = String(session.code || '').trim();
  if (!code) {
    seedLanguageId.value = '';
    return;
  }
  try {
    const rows = (await (languageStore as any).Search(
      { And: [['Code', '=', code], ['IsActive', '=', true]] } as any,
      { fields: ['Id'], limit: 1 } as any,
    )) as Array<{ Id?: string }>;
    seedLanguageId.value = String(rows?.[0]?.Id || '').trim();
  } catch {
    seedLanguageId.value = '';
  }
}

async function resolveSeedTimezone(): Promise<void> {
  let allowed: string[] = [];
  try {
    const fields = await (userStore as any).FieldsGet?.(['Timezone']);
    const selection = fields?.Timezone?.selection || fields?.fields?.Timezone?.selection;
    if (Array.isArray(selection)) {
      allowed = selection
        .map((item: any) => String(item.value ?? item.Value ?? ''))
        .filter((value: string) => !!value);
    }
  } catch {
    allowed = [];
  }
  const resolved = resolvePreferenceTimezone(currentUser.value?.Timezone, detectBrowserTimezone(), allowed);
  seedTimezone.value = resolved.timezone ?? '';
  timezoneFromBrowser.value = resolved.fromBrowser;
}

async function openAndLoad() {
  if (authStore.isAuthenticated) {
    try {
      await authStore.loadUser(true);
    } catch {
      // Fall back to whatever is already in auth state / identity.
    }
  }
  await Promise.all([resolveSeedLanguageId(), resolveSeedTimezone()]);
}

watch(
  () => props.modelValue,
  async open => {
    if (open) {
      lastFocused = document.activeElement as HTMLElement | null;
      // Focus the dialog before network work so keyboard users are not stuck
      // on the trigger behind the backdrop while preferences load.
      await nextTick();
      dialogRef.value?.focus();
      await openAndLoad();
      return;
    }
    restoreDialogFocus(lastFocused);
    lastFocused = null;
  },
);

onMounted(async () => {
  if (!props.modelValue) return;
  lastFocused = document.activeElement as HTMLElement | null;
  await nextTick();
  dialogRef.value?.focus();
  await openAndLoad();
});

function onMissingUserSubmit() {
  ChoyMessage.error(_t('Cannot update preferences: missing user id'));
}

async function onPreferencesSubmit(ctx: {
  formData: Record<string, unknown>;
  defaultSubmit: () => Promise<Record<string, unknown> | null>;
}) {
  saving.value = true;
  try {
    await runPreferencesSubmit({
      userId: userId.value,
      hasCurrentUserId: Boolean(currentUser.value?.Id),
      loadUser: () => authStore.loadUser(true),
      missingUserMessage: _t('Cannot update preferences: missing user id'),
      defaultSubmit: async () => (await ctx.defaultSubmit()) as Record<string, unknown> | null,
      formData: (ctx.formData || {}) as Record<string, unknown>,
      patchCurrentUser: (languageId, timezone) => {
        if (!authStore.currentUser) return;
        (authStore.currentUser as any).LanguageId = languageId;
        (authStore.currentUser as any).Timezone = timezone;
      },
      applyLanguage: languageId =>
        applyUserLanguagePreference({
          languageId,
          displayOverrides: (authStore.currentUser as any)?.Preferences?.display,
          browseLanguage: (id, fields) => (languageStore as any).Browse(id, fields),
          setUiKey: k => i18nStore.setUiKey(k),
          setDisplayOverrides: o => i18nStore.setDisplayOverrides(o as never),
          langToUiKey,
        }),
      refreshToken: () => authStore.refreshToken(true),
      afterLocaleChange: () => afterLocaleChange({ remount: softLocaleRemount }),
      onSuccess: () => {
        ChoyMessage.success(_t('Preferences updated'));
        visible.value = false;
      },
      onError: message => {
        ChoyMessage.error(message);
      },
      failedMessage: _t('Failed to update preferences'),
    });
  } finally {
    saving.value = false;
  }
  return { handled: true, skipSuccessMessage: true };
};
</script>
