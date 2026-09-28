<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="o-preferences-dialog fixed inset-0 z-[100] flex items-center justify-center bg-black/50 p-4"
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

        <div v-if="currentUser" class="o-preferences-dialog__header mb-4 mt-3">
          <div class="o-preferences-dialog__identity">
            <div class="o-preferences-dialog__name font-semibold text-foreground">{{ displayName }}</div>
            <div class="o-preferences-dialog__email text-sm text-foreground/70">{{ currentUser.Email || '' }}</div>
          </div>
        </div>

        <form class="o-preferences-dialog__form flex flex-col gap-4" @submit.prevent="handleSave">
          <label class="flex flex-col gap-1 text-sm">
            <span class="font-medium">{{ _t('Language') }}</span>
            <select v-model="languageCode" class="rounded-md border border-border bg-background px-2 py-1.5 text-sm">
              <option v-for="opt in languageOptions" :key="opt.Code" :value="opt.Code">{{ opt.Name }}</option>
            </select>
            <div v-if="languageFromSession" class="o-preferences-dialog__hint text-xs text-foreground/60">
              {{ _t('Using current session language') }}
            </div>
          </label>

          <label class="flex flex-col gap-1 text-sm">
            <span class="font-medium">{{ _t('Timezone') }}</span>
            <select v-model="timezone" class="rounded-md border border-border bg-background px-2 py-1.5 text-sm">
              <option value="">{{ _t('Select timezone') }}</option>
              <option v-for="tz in timezoneOptions" :key="tz.value" :value="tz.value">{{ tz.label }}</option>
            </select>
            <div v-if="timezoneFromBrowser" class="o-preferences-dialog__hint text-xs text-foreground/60">
              {{ _t('Suggested from your browser') }}
            </div>
          </label>

          <div class="flex justify-end gap-2 pt-2">
            <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
            <ChoyButton type="submit" :disabled="saving">{{ _t('Update preferences') }}</ChoyButton>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useId, watch } from 'vue';
import { ChoyButton, ChoyMessage } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useI18nStore, langToUiKey, afterLocaleChange, softLocaleRemount } from '@/web/web/stores/i18nStore';
import { createStoreByModel } from '@/web/web/stores/registry';
import {
  detectBrowserTimezone,
  resolvePreferenceLanguage,
  resolvePreferenceTimezone,
} from './preferences_defaults';
import { trapDialogTabKey } from './dialog_focus_trap';
import { resolveLanguageCodeFromId } from './preferences_language';

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
  const root = dialogRef.value;
  if (root) trapDialogTabKey(event, root);
}

const currentUser = computed(() => authStore.currentUser as any);
const displayName = computed(() => {
  const u = currentUser.value;
  if (!u) return '';
  return String(u.DisplayName || u.Username || u.Email || '');
});

const languageCode = ref('');
const savedLanguageCode = ref('');
const timezone = ref('');
const languageFromSession = ref(false);
const timezoneFromBrowser = ref(false);
const languageOptions = ref<Array<{ Code: string; Name: string }>>([]);
const timezoneOptions = ref<Array<{ value: string; label: string }>>([]);
const saving = ref(false);

async function loadLanguageOptions() {
  try {
    const rows = await (languageStore as any).GetActiveLanguages();
    languageOptions.value = (rows || [])
      .map((r: any) => ({ Code: String(r.Code || ''), Name: String(r.Name || r.Code || '') }))
      .filter((r: { Code: string }) => !!r.Code);
  } catch {
    languageOptions.value = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
  }
}

async function loadTimezoneOptions() {
  try {
    const fields = await (userStore as any).FieldsGet?.(['Timezone']);
    const selection = fields?.Timezone?.selection || fields?.fields?.Timezone?.selection;
    if (Array.isArray(selection)) {
      timezoneOptions.value = selection
        .map((item: any) => ({
          value: String(item.value ?? item.Value ?? ''),
          label: String(item.label ?? item.Label ?? item.value ?? ''),
        }))
        .filter((item: { value: string }) => !!item.value);
      return;
    }
  } catch {
    // fall through
  }
  timezoneOptions.value = [];
}

async function syncLanguageFromUser() {
  const code = await resolveLanguageCodeFromId(currentUser.value?.LanguageId, (id, fields) =>
    (languageStore as any).Browse(id, fields)
  );
  savedLanguageCode.value = code;
  const resolved = resolvePreferenceLanguage(code || null, i18nStore.terminologyLang);
  languageCode.value = resolved.code;
  languageFromSession.value = resolved.fromSession;
}

function applyTimezoneFromUserOrBrowser() {
  const allowed = timezoneOptions.value.map(opt => opt.value);
  const resolved = resolvePreferenceTimezone(currentUser.value?.Timezone, detectBrowserTimezone(), allowed);
  timezone.value = resolved.timezone ?? '';
  timezoneFromBrowser.value = resolved.fromBrowser;
  if (resolved.fromBrowser && resolved.timezone && !allowed.includes(resolved.timezone)) {
    timezoneOptions.value = [{ value: resolved.timezone, label: resolved.timezone }, ...timezoneOptions.value];
  }
}

async function openAndLoad() {
  if (authStore.isAuthenticated) {
    try {
      await authStore.loadUser(true);
    } catch {
      // Fall back to whatever is already in auth state / identity.
    }
  }
  await syncLanguageFromUser();
  await Promise.all([loadLanguageOptions(), loadTimezoneOptions()]);
  applyTimezoneFromUserOrBrowser();
}

watch(
  () => props.modelValue,
  async open => {
    if (open) {
      lastFocused = document.activeElement as HTMLElement | null;
      await openAndLoad();
      await nextTick();
      dialogRef.value?.focus();
      return;
    }
    lastFocused?.focus?.();
    lastFocused = null;
  }
);

watch(languageCode, code => {
  if (!languageFromSession.value) return;
  const saved = savedLanguageCode.value;
  if (code !== saved && code !== String(i18nStore.terminologyLang || '').trim()) {
    languageFromSession.value = false;
  }
});

watch(timezone, value => {
  if (!timezoneFromBrowser.value) return;
  const browserTz = detectBrowserTimezone();
  if (value !== browserTz) {
    timezoneFromBrowser.value = false;
  }
});

onMounted(async () => {
  if (!props.modelValue) return;
  lastFocused = document.activeElement as HTMLElement | null;
  await openAndLoad();
  await nextTick();
  dialogRef.value?.focus();
});

function resolveUserId(): string {
  return String(currentUser.value?.Id || authStore.identity?.userId || '').trim();
}

async function handleSave() {
  let userId = resolveUserId();
  if (!userId) {
    ChoyMessage.error(_t('Cannot update preferences: missing user id'));
    return;
  }
  saving.value = true;
  try {
    if (!currentUser.value?.Id) {
      await authStore.loadUser(true);
      userId = resolveUserId();
      if (!userId) {
        throw new Error(_t('Cannot update preferences: missing user id'));
      }
    }
    const nextLang = String(languageCode.value || '').trim();
    const nextTz = timezone.value ? String(timezone.value).trim() : null;
    let languageId: string | null = null;
    if (nextLang) {
      const rows = (await (languageStore as any).Search(
        { And: [['Code', '=', nextLang], ['IsActive', '=', true]] } as any,
        { fields: ['Id'], limit: 1 } as any
      )) as Array<{ Id?: string }>;
      languageId = String(rows?.[0]?.Id || '').trim() || null;
      if (!languageId) {
        throw new Error(_t('Invalid or inactive language'));
      }
    }
    await userStore.UpdateById(
      userId,
      { LanguageId: languageId, Timezone: nextTz } as any,
      ['Id', 'LanguageId', 'Timezone'] as any
    );
    if (authStore.currentUser) {
      (authStore.currentUser as any).LanguageId = languageId;
      (authStore.currentUser as any).Timezone = nextTz;
    }
    savedLanguageCode.value = nextLang;
    if (nextLang) {
      await i18nStore.setUiKey(langToUiKey(nextLang));
    }
    i18nStore.setDisplayOverrides((authStore.currentUser as any)?.Preferences?.display ?? null);
    await authStore.refreshToken(true);
    await afterLocaleChange({ remount: softLocaleRemount });
    ChoyMessage.success(_t('Preferences updated'));
    visible.value = false;
  } catch (err: any) {
    ChoyMessage.error(String(err?.message || err || _t('Failed to update preferences')));
  } finally {
    saving.value = false;
  }
}
</script>
