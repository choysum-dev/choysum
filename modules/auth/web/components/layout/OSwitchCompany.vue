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
            :teleported="false"
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
            :teleported="false"
            class="o-switch-company__select"
            :placeholder="_t('Select available companies')"
            data-testid="company-enabled-select"
            @change="onEnabledChange"
            @remove-tag="onRemoveEnabledTag"
          >
            <el-option v-for="c in companies" :key="c.Id" :label="c.DisplayName || c.Id" :value="c.Id" />
          </el-select>
        </el-form-item>

        <div v-if="applyDisabledReason" class="o-switch-company__hint" data-testid="company-switch-hint">
          {{ applyDisabledReason }}
        </div>

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
import { computed, nextTick, ref, watch } from 'vue';
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
/** Guards async open-sync so a late refresh cannot reset an in-progress selection. */
let panelOpenGeneration = 0;

const meta = computed(() => ((authStore.identity as any)?.metadata ?? {}) as any);
const currentActiveCompanyId = computed(() => String(meta.value?.activeCompanyId ?? '').trim());
const currentEnabledCompanyIds = computed(() =>
  Array.isArray(meta.value?.enabledCompanyIds) ? meta.value.enabledCompanyIds.map((x: any) => String(x ?? '').trim()).filter(Boolean) : ([] as string[])
);
/** Latest allowlist from User.CompanyId/CompanyIds (may be newer than JWT metadata). */
const liveAllowedCompanyIds = ref<string[]>([]);
const allowedCompanyIds = computed(() => {
  const xs = Array.isArray(meta.value?.allowedCompanyIds) ? meta.value.allowedCompanyIds : [];
  const ids = xs.map((x: any) => String(x ?? '').trim()).filter(Boolean);
  // Keep the current active/enabled companies and any freshly loaded user allowlist visible.
  const merged = new Set<string>(
    [...ids, ...liveAllowedCompanyIds.value, ...currentEnabledCompanyIds.value, currentActiveCompanyId.value].filter(Boolean)
  );
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
    // Panel-open drafts are user-owned (seeded in the visible watcher). A token
    // refresh while the popover is open must not clobber an in-progress selection,
    // or isDirty/canApply flip back to "No changes to apply".
    if (visible.value) return;
    draftActiveCompanyId.value = active;
    draftEnabledCompanyIds.value = uniq(enabled);
    ensureActiveInEnabled();
  },
  { immediate: true }
);

/**
 * Ensure active ∈ enabled when syncing scope drafts (server rule).
 */
function ensureActiveInEnabled(): void {
  const active = draftActiveCompanyId.value;
  if (!active) return;
  if (!draftEnabledCompanyIds.value.includes(active)) {
    draftEnabledCompanyIds.value = uniq([active, ...draftEnabledCompanyIds.value]);
  }
}

/**
 * Keep the current company in available companies after select changes (incl. backspace).
 */
function onEnabledChange(): void {
  ensureActiveInEnabled();
}

/**
 * Current company cannot leave the available set (server: active ∈ enabled).
 */
function onRemoveEnabledTag(id: unknown): void {
  if (String(id ?? '') !== draftActiveCompanyId.value) return;
  void nextTick(() => ensureActiveInEnabled());
}

watch(
  draftActiveCompanyId,
  () => {
    ensureActiveInEnabled();
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
    ensureActiveInEnabled();
  },
  { flush: 'sync' }
);

const companies = ref<CompanyRow[]>([]);
const fetchedSig = ref('');
/** True after a company-label fetch finishes (success or fail-soft). */
const labelsReady = ref(false);

/**
 * Seed select options from allowed ids immediately so el-select does not drop draft values
 * while DisplayName rows are still loading.
 */
function seedCompanyOptions(ids: string[]): void {
  const prev = new Map(companies.value.map(c => [c.Id, c] as const));
  companies.value = ids.map(id => prev.get(id) ?? { Id: id, DisplayName: '' });
}

/**
 * Load company labels for the currently allowed company set.
 */
async function ensureCompanies(): Promise<void> {
  const ids = allowedCompanyIds.value;
  seedCompanyOptions(ids);
  const sig = ids.slice().sort().join(',');
  if (!sig) {
    labelsReady.value = false;
    return;
  }
  if (sig === fetchedSig.value) return;
  fetchedSig.value = sig;
  labelsReady.value = false;

  try {
    const companyStore = getGlobalCompanyStore();
    const rows = (await companyStore.Search(['Id', 'in', ids] as any, { fields: ['Id', 'DisplayName'], limit: 1000 } as any)) as any[];
    // A newer ensureCompanies call owns fetchedSig; drop this stale response.
    if (fetchedSig.value !== sig) return;

    const out: CompanyRow[] = (rows || [])
      .map(r => ({ Id: String((r as any)?.Id ?? '').trim(), DisplayName: String((r as any)?.DisplayName ?? '').trim() }))
      .filter(r => !!r.Id);

    // Preserve the current allowlist ordering; use latest ids in case the set grew mid-flight.
    const map = new Map(out.map(r => [r.Id, r] as const));
    companies.value = allowedCompanyIds.value.map(id => map.get(id) ?? { Id: id, DisplayName: '' });
    labelsReady.value = true;
  } catch {
    if (fetchedSig.value !== sig) return;
    // Allow a later popover open (or metadata change) to retry the fetch.
    fetchedSig.value = '';
    seedCompanyOptions(allowedCompanyIds.value);
    labelsReady.value = true;
  }
}

// Prefetch labels for the header trigger; do not wait until the popover opens.
watch(
  allowedCompanyIds,
  ids => {
    seedCompanyOptions(ids);
    void ensureCompanies();
  },
  { immediate: true }
);

/**
 * Load the current user's CompanyId/CompanyIds so the switcher does not stay stuck
 * on stale JWT allowedCompanyIds after the user record was edited.
 */
async function syncAllowedCompaniesFromUser(): Promise<void> {
  const userId = String((authStore.identity as any)?.userId || '').trim();
  if (!userId) return;
  try {
    const userStore = createStoreByModel('auth.User');
    const user = (await userStore.Browse(userId, ['Id', 'CompanyId', 'CompanyIds'] as any)) as any;
    liveAllowedCompanyIds.value = uniq([
      String(user?.CompanyId ?? '').trim(),
      ...(Array.isArray(user?.CompanyIds) ? user.CompanyIds.map((x: any) => String(x ?? '').trim()) : []),
    ]);
  } catch {
    // Fail soft: keep metadata-derived allowlist.
  }
}

/**
 * Re-sync drafts each time the panel opens, then refresh the allowlist in the background.
 * Do not reset drafts after the async refresh — that races with user selection and keeps Apply disabled.
 */
watch(visible, async isOpen => {
  if (!isOpen) return;
  const openGen = ++panelOpenGeneration;
  draftActiveCompanyId.value = currentActiveCompanyId.value;
  draftEnabledCompanyIds.value = uniq(currentEnabledCompanyIds.value);
  ensureActiveInEnabled();
  void ensureCompanies();

  try {
    await authStore.refreshToken(true);
  } catch {
    // Fail soft: keep the existing token metadata when refresh is unavailable.
  }
  if (openGen !== panelOpenGeneration || !visible.value) return;

  await syncAllowedCompaniesFromUser();
  if (openGen !== panelOpenGeneration || !visible.value) return;

  // Expand labels for any newly discovered allowed companies without clobbering drafts.
  fetchedSig.value = '';
  await ensureCompanies();
});

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
  const name = companyNameById.value.get(id);
  if (name) return name;
  // Wait until the label fetch settles before falling back to the raw id.
  // Do not use fetchedSig here: it is set before Search completes and would flash the id.
  return labelsReady.value ? id : _t('Company');
});

/** Normalize enabled scope the same way drafts do (active is always included). */
const effectiveCurrentEnabledCompanyIds = computed(() => {
  const active = currentActiveCompanyId.value;
  const enabled = uniq(currentEnabledCompanyIds.value);
  if (active && !enabled.includes(active)) return uniq([active, ...enabled]);
  return enabled;
});

const isDirty = computed(() => {
  if (draftActiveCompanyId.value !== currentActiveCompanyId.value) return true;
  if (!setEq(draftEnabledCompanyIds.value, effectiveCurrentEnabledCompanyIds.value)) return true;
  return false;
});

const canApply = computed(() => {
  const active = draftActiveCompanyId.value;
  if (!active) return false;
  if (!draftEnabledCompanyIds.value.includes(active)) return false;
  if (!isDirty.value) return false;
  return true;
});

const applyDisabledReason = computed(() => {
  if (canApply.value) return '';
  const active = draftActiveCompanyId.value;
  if (!active) return _t('Select a current company');
  if (!draftEnabledCompanyIds.value.includes(active)) {
    return _t('Available companies must include the current company');
  }
  if (companies.value.length < 2) {
    return _t('Only one company is available; nothing to apply');
  }
  return _t('No changes to apply');
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

.o-switch-company__hint {
  margin: 0 0 8px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
}

.o-switch-company__select {
  width: 100%;
}
</style>

<style lang="scss">
/* Teleported popover: keep non-teleported select dropdowns visible and interactive. */
.o-switch-company__popover {
  overflow: visible;
}
</style>
