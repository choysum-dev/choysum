<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Sidebar: identity button + Popover when multiple companies are allowed.
    Collapsed tip hosts outside the as-child chain so TooltipRoot cannot steal
    the panel position. Toolbar: Popover (no tip needed).
  -->
  <ChoyCollapsedIdentityTipHost
    v-if="variant === 'sidebar'"
    :label="companyIdentityTooltip"
  >
    <ChoyPopover v-if="canSwitchCompanies" v-model:open="visible">
      <ChoyPopoverTrigger as-child>
        <ChoySidebarMenuButton
          size="lg"
          type="button"
          class="w-full min-w-0 data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
          :aria-expanded="visible"
          :aria-label="_t('Switch company')"
          data-testid="company-switch-trigger"
        >
          <span
            class="bg-background text-foreground ring-sidebar-border/60 flex aspect-square size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg ring-1"
            data-testid="company-switch-logo-wrap"
          >
            <img
              :src="companyLogoUrl"
              alt=""
              class="size-8 object-contain p-0.5"
              width="32"
              height="32"
              data-testid="company-switch-logo"
            />
          </span>
          <span
            class="grid min-w-0 flex-1 text-start text-sm leading-tight"
            data-testid="company-switch-label"
          >
            <span class="truncate font-medium">{{ currentCompanyLabel }}</span>
            <span
              class="text-muted-foreground flex min-w-0 items-center gap-1 truncate text-xs"
              data-testid="company-switch-meta"
            >
              <span class="truncate">{{ currentCompanyMeta }}</span>
              <ChoyBadge
                v-if="enabledCompanyCount > 1"
                variant="outline"
                class="border-sidebar-border bg-background text-sidebar-foreground h-4 shrink-0 px-1 text-[0.65rem] leading-none"
                data-testid="company-switch-enabled-count"
              >
                {{ enabledCompanyCount }}
              </ChoyBadge>
            </span>
          </span>
          <span
            class="ms-auto inline-flex shrink-0 opacity-50"
            aria-hidden="true"
            data-testid="company-switch-chevrons"
          >
            <ChevronsUpDown class="size-4" />
          </span>
        </ChoySidebarMenuButton>
      </ChoyPopoverTrigger>
      <ChoyPopoverContent
        side="right"
        align="start"
        :side-offset="4"
        class="w-72 min-w-56 rounded-lg p-0"
        data-testid="company-switch-panel"
      >
        <SwitchCompanyPanel
          v-model:search-filter="searchFilter"
          :select-all-state="selectAllState"
          :allowed-count="allowedCompanyIds.length"
          :entries="panelEntries"
          :active-company-id="draftActiveCompanyId"
          :dirty="isDirty"
          :disabled="submitting || !labelsReady"
          @toggle-select-all="onToggleSelectAll"
          @toggle-branch="onToggleCompanyBranch"
          @loginto="onLogIntoCompany"
          @reset="onResetDraft"
          @confirm="onConfirmDraft"
        />
      </ChoyPopoverContent>
    </ChoyPopover>

    <ChoySidebarMenuButton
      v-else
      size="lg"
      type="button"
      class="w-full min-w-0"
      :aria-label="currentCompanyLabel"
      data-testid="company-switch-trigger"
      @click="onIdentityActivate"
    >
      <span
        class="bg-background text-foreground ring-sidebar-border/60 flex aspect-square size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg ring-1"
        data-testid="company-switch-logo-wrap"
      >
        <img
          :src="companyLogoUrl"
          alt=""
          class="size-8 object-contain p-0.5"
          width="32"
          height="32"
          data-testid="company-switch-logo"
        />
      </span>
      <span
        class="grid min-w-0 flex-1 text-start text-sm leading-tight"
        data-testid="company-switch-label"
      >
        <span class="truncate font-medium">{{ currentCompanyLabel }}</span>
        <span
          class="text-muted-foreground truncate text-xs"
          data-testid="company-switch-meta"
        >
          {{ currentCompanyMeta }}
        </span>
      </span>
    </ChoySidebarMenuButton>
  </ChoyCollapsedIdentityTipHost>

  <ChoyPopover v-else-if="canSwitchCompanies" v-model:open="visible">
    <ChoyPopoverTrigger as-child>
      <ChoyButton
        variant="ghost"
        size="sm"
        :class="triggerClass"
        :aria-expanded="visible"
        :aria-label="_t('Switch company')"
        data-testid="company-switch-trigger"
      >
        <span
          class="inline-flex min-w-0 items-center gap-1 truncate text-sm font-medium"
          data-testid="company-switch-label"
        >
          <span class="truncate">{{ currentCompanyLabel }}</span>
          <ChoyBadge
            v-if="enabledCompanyCount > 1"
            variant="outline"
            class="bg-background h-4 shrink-0 px-1 text-[0.65rem] leading-none"
            data-testid="company-switch-enabled-count"
          >
            {{ enabledCompanyCount }}
          </ChoyBadge>
        </span>
      </ChoyButton>
    </ChoyPopoverTrigger>
    <ChoyPopoverContent
      class="w-72 min-w-56 rounded-lg p-0"
      align="end"
      side="bottom"
      :side-offset="4"
      data-testid="company-switch-panel"
    >
      <SwitchCompanyPanel
        v-model:search-filter="searchFilter"
        :select-all-state="selectAllState"
        :allowed-count="allowedCompanyIds.length"
        :entries="panelEntries"
        :active-company-id="draftActiveCompanyId"
        :dirty="isDirty"
        :disabled="submitting || !labelsReady"
        @toggle-select-all="onToggleSelectAll"
        @toggle-branch="onToggleCompanyBranch"
        @loginto="onLogIntoCompany"
        @reset="onResetDraft"
        @confirm="onConfirmDraft"
      />
    </ChoyPopoverContent>
  </ChoyPopover>

  <ChoyButton
    v-else
    variant="ghost"
    size="sm"
    :class="triggerClass"
    :aria-label="currentCompanyLabel"
    data-testid="company-switch-trigger"
    @click="onIdentityActivate"
  >
    <span class="truncate text-sm font-medium" data-testid="company-switch-label">
      {{ currentCompanyLabel }}
    </span>
  </ChoyButton>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { ChevronsUpDown } from 'lucide-vue-next';
import {
  ChoyBadge,
  ChoyButton,
  ChoyCollapsedIdentityTipHost,
  ChoyPopover,
  ChoyPopoverContent,
  ChoyPopoverTrigger,
  ChoySidebarMenuButton,
  resolveCompanyLogoUrl,
} from '@/web';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createStoreByModel } from '@/web/web/stores/registry';
import type Company from '@/base/service/models/company';
import { createTranslate } from '@/web/web/i18n';
import { collapsedIdentityTooltip } from './collapsed_identity_tooltip';
import { syncCompanyDraftsFromJwt } from './o_switch_company_draft';
import SwitchCompanyPanel from './SwitchCompanyPanel.vue';
import {
  branchCheckState,
  buildCompanyChildrenMap,
  buildCompanyTreeEntries,
  ensureActiveInEnabledScope,
  filterCompanyTreeBySearch,
  isCompanyBranchToggleDisabled,
  logIntoCompany,
  selectAllCheckState,
  setEqCompanyIds,
  toggleCompanyBranch,
  toggleSelectAllCompanies,
  uniqCompanyIds,
  type SwitchCompanyRow,
} from './o_switch_company_selection';

defineOptions({ name: 'SwitchCompany' });

withDefaults(
  defineProps<{
    /** toolbar = inset header actions; sidebar = SidebarHeader workspace switcher */
    variant?: 'toolbar' | 'sidebar';
  }>(),
  { variant: 'toolbar' },
);

const { _t } = createTranslate('auth', { scope: 'web/components/layout/SwitchCompany' });

const triggerClass = 'choy-header__action-item h-control max-w-[12rem] truncate px-2.5';

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
const searchFilter = ref('');
const submitting = ref(false);

const meta = computed(() => ((authStore.identity as any)?.metadata ?? {}) as any);
const currentActiveCompanyId = computed(() => String(meta.value?.activeCompanyId ?? '').trim());
const currentEnabledCompanyIds = computed(() =>
  Array.isArray(meta.value?.enabledCompanyIds)
    ? meta.value.enabledCompanyIds.map((x: any) => String(x ?? '').trim()).filter(Boolean)
    : ([] as string[]),
);
/** Latest allowlist from User.CompanyId/CompanyIds after a successful Browse. */
const liveAllowedCompanyIds = ref<string[]>([]);
/** True after a successful user-row allowlist Browse; then live list is authoritative. */
const liveAllowlistReady = ref(false);
/** Monotonic generation so only the latest Browse success/failure updates live state. */
let allowlistBrowseGeneration = 0;
const allowedCompanyIds = computed(() => {
  if (liveAllowlistReady.value) {
    return uniqCompanyIds(liveAllowedCompanyIds.value);
  }
  const xs = Array.isArray(meta.value?.allowedCompanyIds) ? meta.value.allowedCompanyIds : [];
  const ids = xs.map((x: any) => String(x ?? '').trim()).filter(Boolean);
  return uniqCompanyIds([
    ...ids,
    ...currentEnabledCompanyIds.value,
    currentActiveCompanyId.value,
  ]);
});

const draftActiveCompanyId = ref('');
const draftEnabledCompanyIds = ref<string[]>([]);

watch(
  [currentActiveCompanyId, currentEnabledCompanyIds],
  ([active, enabled]) => {
    syncCompanyDraftsFromJwt({
      panelVisible: visible.value,
      activeCompanyId: active,
      enabledCompanyIds: enabled,
      apply: (nextActive, nextEnabled) => {
        const next = ensureActiveInEnabledScope(nextActive, nextEnabled);
        draftActiveCompanyId.value = next.activeCompanyId;
        draftEnabledCompanyIds.value = next.enabledCompanyIds;
      },
    });
  },
  { immediate: true },
);

watch(
  allowedCompanyIds,
  allowed => {
    if (!allowed.length) return;
    const filtered = uniqCompanyIds(draftEnabledCompanyIds.value.filter(id => allowed.includes(id)));
    let active = draftActiveCompanyId.value;
    if (active && !allowed.includes(active)) active = allowed[0];
    const next = ensureActiveInEnabledScope(active, filtered);
    draftActiveCompanyId.value = next.activeCompanyId;
    draftEnabledCompanyIds.value = next.enabledCompanyIds;
  },
  { flush: 'sync' },
);

const companies = ref<SwitchCompanyRow[]>([]);
const fetchedSig = ref('');
/** True after a company-label fetch finishes (success or fail-soft). */
const labelsReady = ref(false);

/**
 * Seed list rows from allowed ids while DisplayName rows are still loading.
 */
function seedCompanyOptions(ids: string[]): void {
  const prev = new Map(companies.value.map(c => [c.Id, c] as const));
  companies.value = ids.map(id => prev.get(id) ?? { Id: id, DisplayName: '' });
}

/**
 * Load company labels and ParentId edges for the currently allowed company set.
 */
async function ensureCompanies(): Promise<void> {
  const run = async () => {
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
      const rows = (await companyStore.Search(['Id', 'in', ids] as any, {
        fields: ['Id', 'DisplayName', 'Code', 'Logo', 'ParentId'],
        limit: 1000,
      } as any)) as any[];
      if (fetchedSig.value !== sig) return;

      const out: SwitchCompanyRow[] = (rows || [])
        .map(r => ({
          Id: String((r as any)?.Id ?? '').trim(),
          DisplayName: String((r as any)?.DisplayName ?? '').trim(),
          Code: String((r as any)?.Code ?? '').trim(),
          Logo: (r as any)?.Logo,
          ParentId: (r as any)?.ParentId,
        }))
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
  };

  if (companiesInFlight) {
    await companiesInFlight;
    if (fetchedSig.value === allowedCompanyIds.value.slice().sort().join(',')) return;
  }
  companiesInFlight = run().finally(() => {
    companiesInFlight = null;
  });
  await companiesInFlight;
}

watch(
  allowedCompanyIds,
  ids => {
    seedCompanyOptions(ids);
    void ensureCompanies();
  },
  { immediate: true },
);

/**
 * Load the current user's CompanyId/CompanyIds so the switcher does not stay stuck
 * on stale JWT allowedCompanyIds after the user record was edited.
 */
async function syncAllowedCompaniesFromUser(): Promise<void> {
  const userId = String((authStore.identity as any)?.userId || '').trim();
  if (!userId) {
    allowlistBrowseGeneration += 1;
    liveAllowlistReady.value = false;
    liveAllowedCompanyIds.value = [];
    return;
  }
  const gen = ++allowlistBrowseGeneration;
  try {
    const userStore = createStoreByModel('auth.User');
    const user = (await userStore.Browse(userId, ['Id', 'CompanyId', 'CompanyIds'] as any)) as any;
    // Discard superseded or post-logout Browse results.
    if (gen !== allowlistBrowseGeneration) return;
    if (String((authStore.identity as any)?.userId || '').trim() !== userId) return;
    liveAllowedCompanyIds.value = uniqCompanyIds([
      String(user?.CompanyId ?? '').trim(),
      ...(Array.isArray(user?.CompanyIds) ? user.CompanyIds.map((x: any) => String(x ?? '').trim()) : []),
    ]);
    liveAllowlistReady.value = true;
  } catch {
    if (gen !== allowlistBrowseGeneration) return;
    // Fail soft: keep metadata-derived allowlist.
    liveAllowlistReady.value = false;
  }
}

/** Skip one visible-open allowlist Browse when identity activate already refreshed. */
let skipNextOpenAllowlistRefresh = false;
/** Skip open-time draft reset after a failed switch so the user can retry Confirm. */
let preserveDraftOnNextOpen = false;
/** In-flight company label/hierarchy Search promise. */
let companiesInFlight: Promise<void> | null = null;

/**
 * Refresh allowlist from the user row so the trigger can unlock after CompanyIds
 * change. Does not RefreshTokens: that forces loadUser and an unrelated i18n cascade.
 * Only re-fetch company labels when the allowlist signature changes.
 */
async function refreshAllowlist(): Promise<void> {
  const beforeSig = allowedCompanyIds.value.slice().sort().join(',');
  await syncAllowedCompaniesFromUser();
  const afterSig = allowedCompanyIds.value.slice().sort().join(',');
  if (beforeSig !== afterSig) {
    fetchedSig.value = '';
  }
  await ensureCompanies();
}

/**
 * Identity-only trigger: re-read allowlist; open the switcher once ≥2 companies exist.
 */
async function onIdentityActivate(): Promise<void> {
  if (submitting.value || visible.value) return;
  await refreshAllowlist();
  // Panel may have opened concurrently while allowlist was loading.
  if (visible.value) return;
  if (allowedCompanyIds.value.length >= 2) {
    skipNextOpenAllowlistRefresh = true;
    visible.value = true;
  }
}

// JWT allowlist can lag behind User.CompanyIds after form edits; sync on login.
watch(
  () => String((authStore.identity as any)?.userId || '').trim(),
  userId => {
    if (!userId) {
      liveAllowedCompanyIds.value = [];
      liveAllowlistReady.value = false;
      return;
    }
    void refreshAllowlist();
  },
  { immediate: true },
);

/**
 * Re-sync drafts each time the panel opens, then refresh CompanyIds from the user row.
 */
watch(visible, async isOpen => {
  if (!isOpen) {
    searchFilter.value = '';
    return;
  }
  const openGen = ++panelOpenGeneration;
  if (preserveDraftOnNextOpen) {
    preserveDraftOnNextOpen = false;
  } else {
    const synced = ensureActiveInEnabledScope(currentActiveCompanyId.value, currentEnabledCompanyIds.value);
    draftActiveCompanyId.value = synced.activeCompanyId;
    draftEnabledCompanyIds.value = synced.enabledCompanyIds;
  }
  searchFilter.value = '';

  if (skipNextOpenAllowlistRefresh) {
    skipNextOpenAllowlistRefresh = false;
    return;
  }
  await refreshAllowlist();
  if (openGen !== panelOpenGeneration || !visible.value) return;
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

/** Collapsed tip label; omitted while the switcher panel is open. */
const companyIdentityTooltip = computed(() =>
  collapsedIdentityTooltip(visible.value, currentCompanyLabel.value),
);

/** Muted second line under the company name (Code, else a workspace fallback). */
const currentCompanyMeta = computed(() => {
  const id = currentActiveCompanyId.value;
  const row = companies.value.find(c => c.Id === id);
  const code = String(row?.Code || '').trim();
  if (code) return code;
  return _t('Workspace');
});

const companyLogoUrl = computed(() => {
  const id = currentActiveCompanyId.value;
  const row = companies.value.find(c => c.Id === id);
  return resolveCompanyLogoUrl(row?.Logo);
});

/** Normalize enabled scope the same way drafts do (active is always included). */
const effectiveCurrentEnabledCompanyIds = computed(() => {
  return ensureActiveInEnabledScope(currentActiveCompanyId.value, currentEnabledCompanyIds.value)
    .enabledCompanyIds;
});

const enabledCompanyCount = computed(() => effectiveCurrentEnabledCompanyIds.value.length);

const canSwitchCompanies = computed(() => allowedCompanyIds.value.length >= 2);

const childrenMap = computed(() => buildCompanyChildrenMap(companies.value));

const treeEntries = computed(() => buildCompanyTreeEntries(companies.value));

const visibleTreeEntries = computed(() =>
  filterCompanyTreeBySearch(treeEntries.value, searchFilter.value),
);

const panelEntries = computed(() =>
  visibleTreeEntries.value.map(entry => ({
    id: entry.id,
    level: entry.level,
    name: entry.name,
    checked: branchCheckState(entry.id, draftEnabledCompanyIds.value, childrenMap.value),
    toggleDisabled: isCompanyBranchToggleDisabled(
      entry.id,
      draftEnabledCompanyIds.value,
      childrenMap.value,
    ),
  })),
);

const isDirty = computed(() => {
  if (draftActiveCompanyId.value !== currentActiveCompanyId.value) return true;
  if (!setEqCompanyIds(draftEnabledCompanyIds.value, effectiveCurrentEnabledCompanyIds.value)) return true;
  return false;
});

const selectAllState = computed(() =>
  selectAllCheckState(draftEnabledCompanyIds.value, allowedCompanyIds.value),
);

/**
 * Persist a company scope. Close the panel first so it is not left open over the
 * page loading mask while SwitchCompanyScope refreshes PermissionState / remounts views.
 * On failure, reopen with the pending draft so the user can retry.
 */
async function commitScope(activeCompanyId: string, enabledCompanyIds: string[]): Promise<void> {
  const allowed = new Set(allowedCompanyIds.value);
  const next = ensureActiveInEnabledScope(
    activeCompanyId,
    enabledCompanyIds.filter(id => allowed.has(id)),
  );
  if (!next.activeCompanyId || !next.enabledCompanyIds.length) return;
  submitting.value = true;
  visible.value = false;
  try {
    await authStore.switchCompanyScope(next.activeCompanyId, next.enabledCompanyIds);
  } catch {
    draftActiveCompanyId.value = next.activeCompanyId;
    draftEnabledCompanyIds.value = next.enabledCompanyIds;
    preserveDraftOnNextOpen = true;
    visible.value = true;
  } finally {
    submitting.value = false;
  }
}

/**
 * Click a company name: enter immediately and close.
 * Uses the committed enabled scope (not pending checkbox drafts). Mark submitting
 * before mutating drafts so the confirm footer cannot flash.
 */
async function onLogIntoCompany(companyId: string): Promise<void> {
  if (submitting.value) return;
  if (!labelsReady.value) {
    await ensureCompanies();
  }
  if (submitting.value) return;
  const target = String(companyId ?? '').trim();
  if (!target || !allowedCompanyIds.value.includes(target)) return;

  const next = logIntoCompany({
    companyId: target,
    draftActiveCompanyId: currentActiveCompanyId.value,
    draftEnabledCompanyIds: effectiveCurrentEnabledCompanyIds.value,
    children: childrenMap.value,
  });
  const allowed = new Set(allowedCompanyIds.value);
  const scope = ensureActiveInEnabledScope(
    next.activeCompanyId,
    next.enabledCompanyIds.filter(id => allowed.has(id)),
  );
  if (!scope.activeCompanyId || !scope.enabledCompanyIds.length) return;

  // Already on this active + enabled set: close without a no-op SwitchCompanyScope.
  if (
    scope.activeCompanyId === currentActiveCompanyId.value
    && setEqCompanyIds(scope.enabledCompanyIds, effectiveCurrentEnabledCompanyIds.value)
  ) {
    visible.value = false;
    return;
  }

  submitting.value = true;
  draftActiveCompanyId.value = scope.activeCompanyId;
  draftEnabledCompanyIds.value = scope.enabledCompanyIds;
  try {
    // Close before the token update remounts the page under a loading mask.
    visible.value = false;
    await authStore.switchCompanyScope(scope.activeCompanyId, scope.enabledCompanyIds);
  } catch {
    preserveDraftOnNextOpen = true;
    visible.value = true;
  } finally {
    submitting.value = false;
  }
}

/**
 * Toggle one company branch in the draft without closing.
 */
function onToggleCompanyBranch(companyId: string): void {
  if (submitting.value) return;
  if (isCompanyBranchToggleDisabled(companyId, draftEnabledCompanyIds.value, childrenMap.value)) {
    return;
  }
  const next = toggleCompanyBranch({
    companyId,
    draftActiveCompanyId: draftActiveCompanyId.value,
    draftEnabledCompanyIds: draftEnabledCompanyIds.value,
    children: childrenMap.value,
  });
  draftActiveCompanyId.value = next.activeCompanyId;
  draftEnabledCompanyIds.value = next.enabledCompanyIds;
}

/**
 * Toggle the select-all control against the full allowlist.
 */
function onToggleSelectAll(): void {
  if (submitting.value) return;
  const next = toggleSelectAllCompanies({
    draftActiveCompanyId: draftActiveCompanyId.value,
    draftEnabledCompanyIds: draftEnabledCompanyIds.value,
    allowedCompanyIds: allowedCompanyIds.value,
  });
  draftActiveCompanyId.value = next.activeCompanyId;
  draftEnabledCompanyIds.value = next.enabledCompanyIds;
}

/**
 * Discard draft changes back to the committed JWT scope.
 */
function onResetDraft(): void {
  const synced = ensureActiveInEnabledScope(currentActiveCompanyId.value, currentEnabledCompanyIds.value);
  draftActiveCompanyId.value = synced.activeCompanyId;
  draftEnabledCompanyIds.value = synced.enabledCompanyIds;
}

/**
 * Confirm draft enabled/active changes.
 */
async function onConfirmDraft(): Promise<void> {
  if (!isDirty.value || submitting.value) return;
  await commitScope(draftActiveCompanyId.value, draftEnabledCompanyIds.value);
}
</script>
