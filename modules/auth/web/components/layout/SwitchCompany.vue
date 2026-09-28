<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div ref="rootRef" class="relative">
    <ChoyButton
      variant="ghost"
      size="sm"
      class="o-switch-company__trigger o-header__action-item max-w-[12rem] truncate"
      :aria-expanded="visible"
      :aria-label="_t('Switch company')"
      data-testid="company-switch-trigger"
      @click.stop="togglePanel"
    >
      {{ currentCompanyLabel }}
    </ChoyButton>

    <div
      v-if="visible"
      class="absolute end-0 top-full z-50 mt-1 w-80 rounded-md border border-border bg-background p-3 shadow-lg"
      data-testid="company-switch-panel"
      @click.stop
    >
      <div class="o-switch-company__panel flex flex-col gap-3">
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium">{{ _t('Current Company') }}</span>
          <select
            v-model="draftActiveCompanyId"
            class="o-switch-company__select rounded-md border border-border bg-background px-2 py-1.5 text-sm"
            data-testid="company-active-select"
          >
            <option v-for="c in companies" :key="c.Id" :value="c.Id">{{ c.DisplayName || c.Id }}</option>
          </select>
        </label>

        <div class="flex flex-col gap-1 text-sm">
          <span class="font-medium">{{ _t('Available Companies') }}</span>
          <div
            class="o-switch-company__select flex min-h-[5.5rem] flex-col gap-1 rounded-md border border-border bg-background px-2 py-1.5 text-sm"
            data-testid="company-enabled-select"
          >
            <label v-for="c in companies" :key="'enabled-' + c.Id" class="flex items-center gap-2">
              <input
                v-model="draftEnabledCompanyIds"
                type="checkbox"
                class="size-4 rounded border-border"
                :value="c.Id"
                @change="onEnabledChange"
              />
              <span>{{ c.DisplayName || c.Id }}</span>
            </label>
          </div>
        </div>

        <div v-if="applyDisabledReason" class="o-switch-company__hint text-xs text-foreground/60" data-testid="company-switch-hint">
          {{ applyDisabledReason }}
        </div>

        <div class="o-switch-company__actions flex justify-end">
          <ChoyButton size="sm" :disabled="!canApply" data-testid="company-switch-apply" @click.stop="apply">
            {{ _t('Apply') }}
          </ChoyButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { ChoyButton } from '@/web';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createStoreByModel } from '@/web/web/stores/registry';
import type Company from '@/base/service/models/company';
import { createTranslate } from '@/web/web/i18n';
import { syncCompanyDraftsFromJwt } from './o_switch_company_draft';

defineOptions({ name: 'SwitchCompany' });

const { _t } = createTranslate('auth', { scope: 'web/components/layout/SwitchCompany' });

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
const rootRef = ref<HTMLElement | null>(null);
/** Guards async open-sync so a late refresh cannot reset an in-progress selection. */
let panelOpenGeneration = 0;

function togglePanel() {
  visible.value = !visible.value;
}

function onDocumentClick(event: MouseEvent) {
  if (!visible.value) return;
  const root = rootRef.value;
  if (root && !root.contains(event.target as Node)) {
    visible.value = false;
  }
}

function onDocumentKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && visible.value) {
    visible.value = false;
  }
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('keydown', onDocumentKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('keydown', onDocumentKeydown);
});

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
    syncCompanyDraftsFromJwt({
      panelVisible: visible.value,
      activeCompanyId: active,
      enabledCompanyIds: enabled,
      apply: (nextActive, nextEnabled) => {
        draftActiveCompanyId.value = nextActive;
        draftEnabledCompanyIds.value = uniq(nextEnabled);
        ensureActiveInEnabled();
      },
    });
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
 * Keep the current company in available companies after select changes.
 */
function onEnabledChange(): void {
  ensureActiveInEnabled();
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
 * Seed select options from allowed ids immediately so selects do not drop draft values
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
    if (fetchedSig.value !== sig) return;

    const out: CompanyRow[] = (rows || [])
      .map(r => ({ Id: String((r as any)?.Id ?? '').trim(), DisplayName: String((r as any)?.DisplayName ?? '').trim() }))
      .filter(r => !!r.Id);

    const map = new Map(out.map(r => [r.Id, r] as const));
    companies.value = allowedCompanyIds.value.map(id => map.get(id) ?? { Id: id, DisplayName: '' });
    labelsReady.value = true;
  } catch {
    if (fetchedSig.value !== sig) return;
    fetchedSig.value = '';
    seedCompanyOptions(allowedCompanyIds.value);
    labelsReady.value = true;
  }
}

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
}
</style>
