<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-preferences-dialog flex max-h-[90vh] w-full max-w-lg flex-col gap-0 overflow-hidden p-5"
      :show-close-button="false"
      :dismissible="!saving"
      data-testid="preferences-dialog"
    >
      <ChoyDialogHeader class="shrink-0 space-y-0 text-start">
        <ChoyDialogTitle>{{ _t('Preferences') }}</ChoyDialogTitle>
      </ChoyDialogHeader>

        <form v-if="!userId" class="mt-4 flex flex-col gap-4" @submit.prevent="onMissingUserSubmit">
          <ChoyDialogFooter>
            <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
            <ChoyButton type="submit">{{ _t('Save') }}</ChoyButton>
          </ChoyDialogFooter>
        </form>

        <ChoyFormView
          v-else
          class="mt-3 flex min-h-0 flex-1 flex-col overflow-hidden"
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
            <SelfProfileDraftSeeder
              :ready="!loading"
              :language-id="seedLanguageId"
              :timezone="seedTimezone"
              :theme="seedTheme"
              :density="seedDensity"
            />
            <ThemeDraftApplier
              :theme="formData?.Theme"
              :active="visible && !saving"
              @apply="onThemeDraftApply"
            />

            <!-- Compact identity chrome above tabs: fixed First | Last order. -->
            <div class="choy-preferences-dialog__profile flex shrink-0 items-start gap-3">
              <div
                v-if="isUserAvatarFieldVisible(collaborationCaps)"
                class="choy-preferences-dialog__avatar w-[72px] shrink-0"
              >
                <ChoyImageField
                  :store="userStore"
                  prop="Avatar"
                  :width="72"
                  :height="72"
                  fit="cover"
                  radius="xl"
                  :drag="false"
                  hide-label
                  hide-help
                  data-testid="preferences-avatar"
                />
              </div>
              <div class="min-w-0 flex-1 space-y-1">
                <div class="choy-preferences-dialog__names grid grid-cols-2 gap-2">
                  <ChoyVarcharField
                    :store="userStore"
                    prop="FirstName"
                    :label="_t('First name')"
                    hide-help
                    :show-word-limit="false"
                    orientation="vertical"
                    data-testid="preferences-first-name"
                  />
                  <ChoyVarcharField
                    :store="userStore"
                    prop="LastName"
                    :label="_t('Last name')"
                    hide-help
                    :show-word-limit="false"
                    orientation="vertical"
                    data-testid="preferences-last-name"
                  />
                </div>
                <div
                  v-if="profileIdentityMeta(formData)"
                  class="truncate text-sm text-foreground/60"
                  data-testid="preferences-identity-meta"
                >
                  {{ profileIdentityMeta(formData) }}
                </div>
              </div>
            </div>

            <ChoyTabs
              v-model="activeTab"
              equal-pane-height
              class="choy-preferences-dialog__tabs mt-2 min-h-0 flex-1 overflow-hidden"
              @mousedown.capture="armTabSwitch"
            >
              <ChoyTab :label="_t('General')" value="general" force-mount>
                <div class="choy-preferences-dialog__panel overflow-y-auto">
                  <div class="flex flex-col gap-3">
                    <div class="space-y-1.5">
                      <div class="text-sm font-medium text-foreground">{{ _t('Appearance') }}</div>
                      <div class="choy-preferences-dialog__setting">
                        <ChoySelectionField
                          :store="userStore"
                          prop="Theme"
                          :label="_t('Theme')"
                          :help="_t('UI color theme (light, dark, or follow the system).')"
                          :clearable="false"
                          :disabled="saving"
                          orientation="horizontal"
                          :select-props="{ 'data-testid': 'preferences-theme' }"
                        />
                      </div>
                      <div class="choy-preferences-dialog__setting">
                        <ChoySelectionField
                          :store="userStore"
                          prop="Density"
                          :label="_t('Density')"
                          :help="_t('Spacing density for lists and the app shell.')"
                          :clearable="false"
                          :disabled="saving"
                          orientation="horizontal"
                          :select-props="{ 'data-testid': 'preferences-density' }"
                        />
                      </div>
                    </div>

                    <div class="space-y-1.5">
                      <div class="text-sm font-medium text-foreground">{{ _t('Locale') }}</div>
                      <div class="choy-preferences-dialog__setting">
                        <ChoyManyToOneRefField
                          :store="userStore"
                          prop="LanguageId"
                          :label="_t('Language')"
                          :help="_t('Language for the UI and terminology.')"
                          :clearable="false"
                          orientation="horizontal"
                          data-testid="preferences-language"
                        />
                        <div
                          v-if="showLanguageSessionHint(formData)"
                          class="choy-preferences-dialog__hint text-xs text-foreground/60"
                        >
                          {{ _t('Using current session language') }}
                        </div>
                      </div>
                      <div class="choy-preferences-dialog__setting">
                        <ChoySelectionField
                          :store="userStore"
                          prop="Timezone"
                          :label="_t('Timezone')"
                          :help="_t('Display timezone; empty uses browser timezone on first login.')"
                          :clearable="false"
                          orientation="horizontal"
                          :placeholder="_t('Select timezone')"
                          :select-props="{ 'data-testid': 'preferences-timezone' }"
                        />
                        <div
                          v-if="showTimezoneBrowserHint(formData)"
                          class="choy-preferences-dialog__hint text-xs text-foreground/60"
                        >
                          {{ _t('Suggested from your browser') }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </ChoyTab>

              <ChoyTab :label="_t('Security')" value="security" force-mount>
                <div class="choy-preferences-dialog__panel overflow-y-auto">
                  <div class="choy-preferences-dialog__setting">
                    <ChoySlotField
                      :label="_t('Password')"
                      :help="_t('Update if compromised or rotating credentials.')"
                      prop="PasswordAction"
                      orientation="horizontal"
                    >
                      <ChoyButton
                        type="button"
                        variant="outline"
                        class="box-border h-control min-h-control shrink-0 px-3 py-0"
                        data-testid="preferences-open-change-password"
                        @click="onOpenChangePassword"
                      >
                        {{ _t('Change password') }}
                      </ChoyButton>
                    </ChoySlotField>
                  </div>
                </div>
              </ChoyTab>
            </ChoyTabs>

            <ChoyDialogFooter class="mt-2 shrink-0 border-t border-border pt-2">
              <ChoyButton type="button" variant="outline" @click="visible = false">{{ _t('Cancel') }}</ChoyButton>
              <ChoyButton type="submit" :disabled="saving || loading">{{ _t('Save') }}</ChoyButton>
            </ChoyDialogFooter>
          </template>
        </ChoyFormView>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  ChoyButton,
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogFooter,
  ChoyDialogHeader,
  ChoyDialogTitle,
  ChoyFormView,
  ChoyImageField,
  ChoyManyToOneRefField,
  ChoyMessage,
  ChoySelectionField,
  ChoySlotField,
  ChoyTab,
  ChoyTabs,
  ChoyVarcharField,
  openDialog,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useI18nStore, langToUiKey, afterLocaleChange, softLocaleRemount } from '@/web/web/stores/i18nStore';
import { createStoreByModel } from '@/web/web/stores/registry';
import {
  applyUserLanguagePreference,
  normalizeLanguageId,
} from '@/auth/web/stores/auth/language_preference';
import { useReactiveCollaborationCaps } from '@/web/web/platform/collaborationCaps';
import { isUserAvatarFieldVisible } from '@/auth/web/views/user_avatar_visibility';
import { resolveUserMenuMeta } from '@/auth/web/components/layout/user_menu_display_name';
import { SelfProfileDraftSeeder } from './SelfProfileDraftSeeder';
import { ThemeDraftApplier } from './ThemeDraftApplier';
import ChangePasswordDialog from '@/auth/web/dialogs/change-password/ChangePasswordDialog.vue';
import {
  needsSelfProfileLoad,
  resolveSelfProfileSeedLanguageId,
  resolveSelfProfileSeedTimezone,
} from '@/auth/web/stores/auth/self_profile_seed';
import { runSelfProfileSubmit } from '@/auth/web/stores/auth/self_profile_submit';
import {
  applyUserAppearance,
  resolveThemeMode,
  resolveUserDensity,
  resolveUserThemeMode,
  type ChoyThemeMode,
} from '@/auth/web/stores/auth/user_appearance';
import { readChoyThemePreference } from '@/web/web/composables/applyChoyThemePreference';

defineOptions({ name: 'PreferencesDialog' });

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [boolean]; closed: [] }>();

const { _t } = createTranslate('auth', { scope: 'web/dialogs/preferences/PreferencesDialog' });
const authStore = useAuthStore();
const i18nStore = useI18nStore();
const collaborationCaps = useReactiveCollaborationCaps();
const userStore = createStoreByModel('auth.User');
const languageStore = createStoreByModel('base.Language');

/** Last Theme value applied to the shell from this dialog (avoids re-apply loops). */
let lastAppliedTheme = '';
/** Target shell theme on dismiss: opening capture, or the last successfully saved mode. */
let openedTheme = '';
let openGeneration = 0;

const visible = computed({
  get: () => props.modelValue,
  set: v => {
    emit('update:modelValue', v);
    if (!v) emit('closed');
  },
});

let suppressFormSubmitUntil = 0;
function armTabSwitch(event: MouseEvent): void {
  const el = event.target as HTMLElement | null;
  if (!el?.closest?.('[data-slot="tabs-trigger"]')) return;
  suppressFormSubmitUntil = Date.now() + 300;
}

function onOpenChangePassword(): void {
  // replace: dialog service unmounts preferences (silent draft discard).
  openDialog({ component: ChangePasswordDialog, mode: 'replace' });
}

const currentUser = computed(() => authStore.currentUser as any);
const userId = computed(() => String(currentUser.value?.Id || authStore.identity?.userId || '').trim());
const seedLanguageId = ref('');
const seedTimezone = ref('');
const seedTheme = ref('');
const seedDensity = ref('');
const languageFromSession = ref(false);
const timezoneFromBrowser = ref(false);
const saving = ref(false);
const activeTab = ref('general');

/** Readonly identity under name fields: username · email (whichever is set). */
function profileIdentityMeta(formData: Record<string, unknown> | null | undefined): string {
  const user = currentUser.value || {};
  return resolveUserMenuMeta({
    email: formData?.Email ?? user.Email,
    username: formData?.Username ?? user.Username,
  });
}

function fieldText(value: unknown): string {
  return value == null ? '' : String(value).trim();
}

function showLanguageSessionHint(formData: Record<string, unknown>): boolean {
  const seed = normalizeLanguageId(seedLanguageId.value);
  if (!languageFromSession.value || !seed) return false;
  return normalizeLanguageId(formData?.LanguageId) === seed;
}

function showTimezoneBrowserHint(formData: Record<string, unknown>): boolean {
  const seed = fieldText(seedTimezone.value);
  if (!timezoneFromBrowser.value || !seed) return false;
  return fieldText(formData?.Timezone) === seed;
}

/** Live-apply Theme from the User form draft to the shell (still persisted on submit). */
function onThemeDraftApply(mode: ChoyThemeMode): void {
  if (mode === lastAppliedTheme) return;
  lastAppliedTheme = mode;
  const density = resolveUserDensity(currentUser.value, readChoyThemePreference().density);
  applyUserAppearance(mode, density);
}

function restoreOpeningThemeIfNeeded(): void {
  // Always sync shell to openedTheme (open capture or last successful save).
  const mode = resolveThemeMode(openedTheme);
  if (mode === lastAppliedTheme) return;
  lastAppliedTheme = mode;
  const density = resolveUserDensity(currentUser.value, readChoyThemePreference().density);
  applyUserAppearance(mode, density);
}

async function openAndLoad() {
  const gen = ++openGeneration;
  activeTab.value = 'general';
  // Capture shell theme before any await so early dismiss restores correctly.
  const local = readChoyThemePreference();
  openedTheme = resolveThemeMode(local.theme);
  lastAppliedTheme = openedTheme;

  if (authStore.isAuthenticated && needsSelfProfileLoad(currentUser.value)) {
    try {
      await authStore.loadUser(false);
    } catch {
      // Fall back to whatever is already in auth state / identity.
    }
  }
  // Dismissed (or a newer open) while loading: do not overwrite restore state.
  if (gen !== openGeneration || !props.modelValue) return;

  seedTheme.value = resolveUserThemeMode(currentUser.value, local.theme);
  seedDensity.value = resolveUserDensity(currentUser.value, local.density);

  const tz = resolveSelfProfileSeedTimezone(currentUser.value?.Timezone);
  seedTimezone.value = tz.seedTimezone;
  timezoneFromBrowser.value = tz.timezoneFromBrowser;

  const lang = await resolveSelfProfileSeedLanguageId({
    languageId: currentUser.value?.LanguageId,
    sessionTerminologyLang: i18nStore.terminologyLang,
    searchLanguageByCode: async code => {
      const rows = (await (languageStore as any).Search(
        { And: [['Code', '=', code], ['IsActive', '=', true]] } as any,
        { fields: ['Id'], limit: 1 } as any,
      )) as Array<{ Id?: string }>;
      return String(rows?.[0]?.Id || '').trim();
    },
  });
  if (gen !== openGeneration || !props.modelValue) return;
  seedLanguageId.value = lang.seedLanguageId;
  languageFromSession.value = lang.languageFromSession;
}

watch(
  () => props.modelValue,
  async open => {
    if (open) {
      await openAndLoad();
      return;
    }
    restoreOpeningThemeIfNeeded();
  },
);

onMounted(async () => {
  if (!props.modelValue) return;
  await openAndLoad();
});

const missingUserMessage = () => _t('Cannot update preferences: missing user id');

function onMissingUserSubmit() {
  ChoyMessage.error(missingUserMessage());
}

async function onPreferencesSubmit(ctx: {
  formData: Record<string, unknown>;
  defaultSubmit: () => Promise<Record<string, unknown> | null>;
}) {
  if (Date.now() < suppressFormSubmitUntil) {
    return { handled: true, skipSuccessMessage: true };
  }
  saving.value = true;
  try {
    await runSelfProfileSubmit({
      userId: userId.value,
      hasCurrentUserId: Boolean(currentUser.value?.Id),
      loadUser: () => authStore.loadUser(true),
      missingUserMessage: missingUserMessage(),
      defaultSubmit: async () => (await ctx.defaultSubmit()) as Record<string, unknown> | null,
      formData: (ctx.formData || {}) as Record<string, unknown>,
      patchCurrentUser: (languageId, timezone, theme, density) => {
        const user = authStore.currentUser as Record<string, unknown> | null | undefined;
        if (!user) return;
        user.LanguageId = languageId;
        user.Timezone = timezone;
        user.Theme = theme;
        user.Density = density;
        const firstName = ctx.formData?.FirstName;
        const lastName = ctx.formData?.LastName;
        const avatar = ctx.formData?.Avatar;
        if (firstName != null) user.FirstName = firstName;
        if (lastName != null) user.LastName = lastName;
        if (avatar != null) user.Avatar = avatar;
      },
      applyLanguage: languageId =>
        applyUserLanguagePreference({
          languageId,
          browseLanguage: (id, fields) => (languageStore as any).Browse(id, fields),
          setUiKey: k => i18nStore.setUiKey(k),
          langToUiKey,
          getCurrentUiKey: () => i18nStore.localeCode,
        }),
      applyAppearance: (theme, density) => {
        const applied = applyUserAppearance(theme, density);
        lastAppliedTheme = applied.theme;
        // Remember the saved mode so dismiss after a later draft tweak restores it.
        openedTheme = applied.theme;
      },
      refreshToken: () => authStore.refreshToken(true),
      afterLocaleChange: () => afterLocaleChange({ remount: softLocaleRemount }),
      onSuccess: () => {
        openedTheme = lastAppliedTheme;
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
}
</script>

<style scoped>
/* Circular 72px avatar chrome; labels/help are hidden via field props. */
.choy-preferences-dialog__avatar :deep(.choy-field-base),
.choy-preferences-dialog__avatar :deep([data-slot='form-item']) {
  gap: 0;
}
.choy-preferences-dialog__avatar :deep(.choy-image-field),
.choy-preferences-dialog__avatar :deep(.choy-image-avatar),
.choy-preferences-dialog__avatar :deep(.choy-image-avatar__media) {
  width: 72px;
  height: 72px;
  border-radius: 9999px;
}

/* Compact name labels above inputs. */
.choy-preferences-dialog__names :deep(.choy-field-base) {
  gap: 0.25rem;
}
.choy-preferences-dialog__names :deep(.choy-field-base__label-row) {
  min-height: 0;
}

/*
 * Beat ChoyTabs line rule (.choy-tabs[data-variant=line] … padding-top: 1rem)
 * which outranks a single-class :deep selector.
 */
.choy-preferences-dialog__tabs.choy-tabs[data-variant='line'] :deep([data-slot='tabs-content']) {
  padding-top: 0.5rem;
}

/* Settings rows: Field chrome owns label|control; keep label column compact. */
.choy-preferences-dialog__setting :deep([data-anchor='choy.field-base'][data-orientation='horizontal']) {
  gap: 0.75rem;
}
.choy-preferences-dialog__setting :deep(.choy-field-base__label-row) {
  width: 4.5rem;
  flex-shrink: 0;
  padding-top: 0.5rem;
  font-size: 0.875rem;
  line-height: 1.25rem;
}
.choy-preferences-dialog__setting :deep(.choy-field-base__content) {
  min-width: 0;
}
.choy-preferences-dialog__hint {
  margin-top: 0.25rem;
  margin-inline-start: calc(4.5rem + 0.75rem);
}
</style>
