<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-popover
    v-model:visible="visible"
    placement="bottom-end"
    trigger="click"
    :teleported="true"
    popper-class="o-switch-company__popover"
    :popper-style="{ width: '320px' }"
    @before-enter="ensureCompanies"
  >
    <template #reference>
      <el-button text class="o-switch-company__trigger o-header__action-item" :aria-label="_t('Switch company')" data-testid="company-switch-trigger" @click.stop>
        {{ currentCompanyLabel }}
      </el-button>
    </template>

    <div class="o-switch-company__panel" data-testid="company-switch-panel" @click.stop>
      <el-form label-position="top" class="o-switch-company__form">
        <el-form-item :label="_t('Current Company')">
          <el-select
            v-model="draftActiveCompanyId"
            filterable
            class="o-switch-company__select"
            :placeholder="_t('Select company')"
            data-testid="company-active-select"
          >
            <el-option v-for="c in companies" :key="c.Id" :label="c.DisplayName || c.Id" :value="c.Id" />
          </el-select>
        </el-form-item>

        <el-form-item :label="_t('Available Companies')">
          <el-select
            v-model="draftEnabledCompanyIds"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            class="o-switch-company__select"
            :placeholder="_t('Select available companies')"
            data-testid="company-enabled-select"
          >
            <el-option v-for="c in companies" :key="c.Id" :label="c.DisplayName || c.Id" :value="c.Id" />
          </el-select>
        </el-form-item>

        <div class="o-switch-company__actions">
          <el-button type="primary" size="small" :disabled="!canApply" data-testid="company-switch-apply" @click.stop="apply">
            {{ _t('Apply') }}
          </el-button>
        </div>
      </el-form>
    </div>
  </el-popover>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { ElPopover, ElButton, ElForm, ElFormItem, ElSelect, ElOption } from 'element-plus';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createStoreByModel } from '@/web/web/stores/registry';
import type Company from '@/base/service/models/company';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('auth', { scope: 'web/components/layout/OSwitchCompany' });

type CompanyRow = { Id: string; DisplayName?: string };

const authStore = useAuthStore();
let globalCompanyStore: any | null = null;

/**
 * Reuse a shared company store for company-switch queries.
 */
function getGlobalCompanyStore(): any {
  if (globalCompanyStore) return globalCompanyStore;
  globalCompanyStore = createStoreByModel<typeof Company>('base.Company');
  return globalCompanyStore;
}

const visible = ref(false);

const meta = computed(() => ((authStore.identity as any)?.metadata ?? {}) as any);
const currentActiveCompanyId = computed(() => String(meta.value?.activeCompanyId ?? '').trim());
const currentEnabledCompanyIds = computed(() =>
  Array.isArray(meta.value?.enabledCompanyIds) ? meta.value.enabledCompanyIds.map((x: any) => String(x ?? '').trim()).filter(Boolean) : ([] as string[])
);
const allowedCompanyIds = computed(() => {
  const xs = Array.isArray(meta.value?.allowedCompanyIds) ? meta.value.allowedCompanyIds : [];
  const ids = xs.map((x: any) => String(x ?? '').trim()).filter(Boolean);
  // Keep the current active and enabled companies visible even if metadata lags.
  const merged = new Set<string>([...ids, ...currentEnabledCompanyIds.value, currentActiveCompanyId.value].filter(Boolean));
  return Array.from(merged);
});

const draftActiveCompanyId = ref('');
const draftEnabledCompanyIds = ref<string[]>([]);

/**
 * Normalize a company id list into unique non-empty values.
 */
function uniq(xs: string[]): string[] {
  return Array.from(new Set(xs.map(x => String(x ?? '').trim()).filter(Boolean)));
}

/**
 * Compare two company id lists as sets.
 */
function setEq(a: string[], b: string[]): boolean {
  const sa = new Set(uniq(a));
  const sb = new Set(uniq(b));
  if (sa.size !== sb.size) return false;
  for (const x of sa) if (!sb.has(x)) return false;
  return true;
}

watch(
  [currentActiveCompanyId, currentEnabledCompanyIds],
  ([active, enabled]) => {
    draftActiveCompanyId.value = active;
    draftEnabledCompanyIds.value = uniq(enabled);
    if (draftActiveCompanyId.value && !draftEnabledCompanyIds.value.includes(draftActiveCompanyId.value)) {
      draftEnabledCompanyIds.value = uniq([draftActiveCompanyId.value, ...draftEnabledCompanyIds.value]);
    }
  },
  { immediate: true }
);

watch(
  draftActiveCompanyId,
  active => {
    if (!active) return;
    if (!draftEnabledCompanyIds.value.includes(active)) {
      draftEnabledCompanyIds.value = uniq([active, ...draftEnabledCompanyIds.value]);
    }
  },
  { flush: 'sync' }
);

watch(
  allowedCompanyIds,
  allowed => {
    if (!allowed.length) return;
    draftEnabledCompanyIds.value = uniq(draftEnabledCompanyIds.value.filter(id => allowed.includes(id)));
    if (draftActiveCompanyId.value && !allowed.includes(draftActiveCompanyId.value)) {
      draftActiveCompanyId.value = allowed[0];
    }
    if (draftActiveCompanyId.value && !draftEnabledCompanyIds.value.includes(draftActiveCompanyId.value)) {
      draftEnabledCompanyIds.value = uniq([draftActiveCompanyId.value, ...draftEnabledCompanyIds.value]);
    }
  },
  { flush: 'sync' }
);

const companies = ref<CompanyRow[]>([]);
const fetchedSig = ref('');

/**
 * Load company labels for the currently allowed company set.
 */
async function ensureCompanies(): Promise<void> {
  const ids = allowedCompanyIds.value;
  const sig = ids.slice().sort().join(',');
  if (!sig || sig === fetchedSig.value) return;
  fetchedSig.value = sig;

  try {
    const companyStore = getGlobalCompanyStore();
    const rows = (await companyStore.Search(['Id', 'in', ids] as any, { fields: ['Id', 'DisplayName'], limit: 1000 } as any)) as any[];
    const out: CompanyRow[] = (rows || [])
      .map(r => ({ Id: String((r as any)?.Id ?? '').trim(), DisplayName: String((r as any)?.DisplayName ?? '').trim() }))
      .filter(r => !!r.Id);

    // Preserve the permission-derived company ordering in the selector.
    const map = new Map(out.map(r => [r.Id, r] as const));
    companies.value = ids.map(id => map.get(id) ?? { Id: id, DisplayName: '' });
  } catch {
    // Allow a later popover open (or metadata change) to retry the fetch.
    fetchedSig.value = '';
    companies.value = ids.map(id => ({ Id: id, DisplayName: '' }));
  }
}

// Prefetch labels for the header trigger; do not wait until the popover opens.
watch(
  allowedCompanyIds,
  () => {
    void ensureCompanies();
  },
  { immediate: true }
);

const companyNameById = computed(() => {
  const m = new Map<string, string>();
  for (const c of companies.value) {
    const name = String(c.DisplayName || '').trim();
    if (name) m.set(String(c.Id), name);
  }
  return m;
});

const currentCompanyLabel = computed(() => {
  const id = currentActiveCompanyId.value;
  if (!id) return _t('Company');
  // Prefer DisplayName; avoid flashing the raw company id while labels are still loading.
  return companyNameById.value.get(id) ?? (companies.value.length ? id : _t('Company'));
});

const isDirty = computed(() => {
  if (draftActiveCompanyId.value !== currentActiveCompanyId.value) return true;
  if (!setEq(draftEnabledCompanyIds.value, currentEnabledCompanyIds.value)) return true;
  return false;
});

const canApply = computed(() => {
  const active = draftActiveCompanyId.value;
  if (!active) return false;
  if (!draftEnabledCompanyIds.value.includes(active)) return false;
  if (!isDirty.value) return false;
  return true;
});

/**
 * Persist the selected company scope back to the auth store.
 */
async function apply(): Promise<void> {
  if (!canApply.value) return;
  await authStore.switchCompanyScope(draftActiveCompanyId.value, uniq(draftEnabledCompanyIds.value));
  visible.value = false;
}
</script>

<style lang="scss" scoped>
.o-switch-company__trigger {
  height: 36px;
  padding: 0 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--el-border-radius-base);
  color: var(--el-text-color-regular);

  &:hover {
    color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
  }
}

.o-switch-company__panel {
  padding: 8px;
}

.o-switch-company__actions {
  display: flex;
  justify-content: flex-end;
}

.o-switch-company__select {
  width: 100%;
}
</style>
