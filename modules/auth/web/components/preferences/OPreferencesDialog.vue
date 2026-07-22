<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dialog
    v-model="visible"
    :title="_t('Edit Profile')"
    width="520px"
    destroy-on-close
    class="o-preferences-dialog"
    @closed="emit('closed')"
  >
    <div v-if="currentUser" class="o-preferences-dialog__header">
      <div class="o-preferences-dialog__identity">
        <div class="o-preferences-dialog__name">{{ displayName }}</div>
        <div class="o-preferences-dialog__email">{{ currentUser.Email || '' }}</div>
      </div>
    </div>

    <el-form label-position="top" class="o-preferences-dialog__form" @submit.prevent>
      <el-form-item :label="_t('Language')">
        <el-select v-model="languageCode" filterable style="width: 100%">
          <el-option v-for="opt in languageOptions" :key="opt.Code" :label="opt.Name" :value="opt.Code" />
        </el-select>
      </el-form-item>
      <el-form-item :label="_t('Timezone')">
        <el-select v-model="timezone" filterable clearable style="width: 100%" :placeholder="_t('Select timezone')">
          <el-option v-for="tz in timezoneOptions" :key="tz.value" :label="tz.label" :value="tz.value" />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">{{ _t('Cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="handleSave">{{ _t('Update preferences') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { ElButton, ElDialog, ElForm, ElFormItem, ElMessage, ElOption, ElSelect } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useI18nStore, langToUiKey, afterLocaleChange, softLocaleRemount } from '@/web/web/stores/i18nStore';
import { createStoreByModel } from '@/web/web/stores/registry';

defineOptions({ name: 'OPreferencesDialog' });

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [boolean]; closed: [] }>();

const { _t } = createTranslate('auth', { scope: 'web/components/preferences/OPreferencesDialog' });
const authStore = useAuthStore();
const i18nStore = useI18nStore();
const userStore = createStoreByModel('auth.User');
const languageStore = createStoreByModel('base.Language');

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

const currentUser = computed(() => authStore.currentUser as any);
const displayName = computed(() => {
  const u = currentUser.value;
  if (!u) return '';
  return String(u.DisplayName || u.Username || u.Email || '');
});

const languageCode = ref('');
const timezone = ref<string | null>(null);
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
      timezoneOptions.value = selection.map((item: any) => ({
        value: String(item.value ?? item.Value ?? ''),
        label: String(item.label ?? item.Label ?? item.value ?? ''),
      })).filter((item: { value: string }) => !!item.value);
      return;
    }
  } catch {
    // fall through
  }
  timezoneOptions.value = [];
}

function syncFromUser() {
  const u = currentUser.value;
  languageCode.value = String(u?.Language || i18nStore.terminologyLang || 'en_US');
  timezone.value = u?.Timezone ? String(u.Timezone) : null;
}

watch(
  () => props.modelValue,
  async open => {
    if (!open) return;
    syncFromUser();
    await Promise.all([loadLanguageOptions(), loadTimezoneOptions()]);
  }
);

onMounted(() => {
  if (props.modelValue) {
    syncFromUser();
    void loadLanguageOptions();
    void loadTimezoneOptions();
  }
});

async function handleSave() {
  const userId = currentUser.value?.Id;
  if (!userId) {
    return;
  }
  saving.value = true;
  try {
    const nextLang = String(languageCode.value || '').trim();
    const nextTz = timezone.value ? String(timezone.value).trim() : null;
    await userStore.UpdateById(
      userId,
      { Language: nextLang || null, Timezone: nextTz } as any,
      ['Id', 'Language', 'Timezone'] as any
    );
    if (authStore.currentUser) {
      (authStore.currentUser as any).Language = nextLang || null;
      (authStore.currentUser as any).Timezone = nextTz;
    }
    if (nextLang) {
      await i18nStore.setUiKey(langToUiKey(nextLang));
    }
    i18nStore.setDisplayOverrides((authStore.currentUser as any)?.Preferences?.display ?? null);
    await authStore.refreshToken(true);
    await afterLocaleChange({ remount: softLocaleRemount });
    ElMessage.success(_t('Preferences updated'));
    visible.value = false;
  } catch (err: any) {
    ElMessage.error(String(err?.message || err || _t('Failed to update preferences')));
  } finally {
    saving.value = false;
  }
}
</script>

<style scoped>
.o-preferences-dialog__header {
  margin-bottom: 16px;
}
.o-preferences-dialog__name {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.o-preferences-dialog__email {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>
