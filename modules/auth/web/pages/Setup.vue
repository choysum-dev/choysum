<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="submitting" fill-height class="flex min-h-0 w-full flex-1 flex-col">
    <div class="setup-panel flex min-h-0 w-full flex-1 flex-col overflow-y-auto" data-testid="setup-panel">
      <div class="setup-shell w-full px-4 pb-8">
        <header class="setup-header mx-auto w-full text-center">
          <h1 class="text-xl font-semibold leading-tight">{{ _t('Initial Setup') }}</h1>
          <div
            v-if="error"
            class="setup-error mt-2 flex items-start justify-between gap-2 text-sm text-destructive"
            role="alert"
          >
            <span class="min-w-0 leading-5">{{ error }}</span>
            <button
              type="button"
              class="inline-flex size-5 shrink-0 items-center justify-center rounded-sm border-0 bg-transparent p-0 text-destructive/70 appearance-none shadow-none outline-none hover:text-destructive focus-visible:ring-2 focus-visible:ring-destructive/40"
              :aria-label="_t('Close')"
              @click="error = ''"
            >
              <X class="size-3.5" />
            </button>
          </div>
          <p v-else class="setup-step-hint mt-2 text-sm text-foreground/60">{{ stepHint }}</p>
          <ChoyStepper v-model="step" :steps="stepperSteps" :readonly="submitting" class="setup-stepper mt-8" />
        </header>
        <div class="setup-content mx-auto mt-10 w-full">
          <div v-if="optionsLoading" class="setup-options-loading text-sm text-foreground/60">
            {{ _t('Loading setup options…') }}
          </div>
          <div v-else-if="optionsError" class="setup-options-error flex flex-col gap-3 text-sm" role="alert">
            <span class="text-destructive">{{ optionsError }}</span>
            <ChoyButton type="button" variant="outline" class="setup-options-retry self-start" @click="loadOptions">
              {{ _t('Retry') }}
            </ChoyButton>
          </div>
          <div v-else class="setup-form">
            <ChoyFormView
              ref="formRef"
              :store="formStore"
              view-mode="create"
              embedded
              :show-header="false"
              :show-actions="false"
              :show-messages="false"
              :resolve-record-id-from-route="false"
              :initial-values="initialValues"
              :submit-handler="onFormSubmit"
            >
              <template #default="{ formData }">
                <SetupFormWatch :form-data="formData" @ready="onFormReady" @change="onFormChange" />
                <SetupAdminStep
                  v-if="step === 'admin'"
                  :store="formStore"
                  :form-data="formData"
                  :server-errors="serverErrors"
                />
                <SetupOrganizationStep
                  v-else-if="step === 'organization'"
                  :store="formStore"
                  :form-data="formData"
                  :server-errors="serverErrors"
                  :country-currencies="countryCurrencies"
                  :currency-touched="currencyTouched"
                  @currency-touched="currencyTouched = true"
                />
                <SetupReviewStep v-else :store="formStore" :form-data="formData" :selections="selections" />
                <div class="setup-actions mt-6 flex justify-between gap-2">
                  <ChoyButton
                    v-if="step !== 'admin'"
                    type="button"
                    variant="outline"
                    size="lg"
                    class="setup-back"
                    :disabled="submitting"
                    @click="goBack"
                  >
                    {{ _t('Back') }}
                  </ChoyButton>
                  <ChoyButton
                    id="setup-submit"
                    type="submit"
                    size="lg"
                    class="submit-button ml-auto"
                    :disabled="submitting"
                  >
                    <template v-if="step === 'review'">
                      {{ submitting ? _t('Complete Setup') + '…' : _t('Complete Setup') }}
                    </template>
                    <template v-else>{{ _t('Next') }}</template>
                  </ChoyButton>
                </div>
              </template>
            </ChoyFormView>
          </div>
        </div>
      </div>
    </div>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, shallowRef, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { X } from 'lucide-vue-next';
import { beginNavigationLoading, runWithLoading } from '@/core/web/client/loadingRegistry';
import { useAuthStore, type InitialSetupOptions } from '../stores/auth';
import SetupFormWatch from './setup/SetupFormWatch.vue';
import SetupAdminStep from './setup/SetupAdminStep.vue';
import SetupOrganizationStep from './setup/SetupOrganizationStep.vue';
import SetupReviewStep from './setup/SetupReviewStep.vue';
import {
  ChoyPage,
  ChoyButton,
  ChoyFormView,
  ChoyStepper,
  createLocalFormStore,
  type ChoyStepperStep,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import { ChoysumError } from '../error';
import { resolveLoginRedirect, runHandledAuthSubmit } from './login_form';
import {
  SETUP_STEPS,
  buildCompleteInitialSetupReq,
  requestSetupLoadDemo,
  runSetupLoadDemo,
  runSetupSubmit,
  setupLocalFieldForServerField,
  setupLoginFallbackTarget,
  setupStepForServerField,
  setupWantsDemo,
  type SetupStep,
} from './setup_form';
import { clearSetupDraft, loadSetupDraft, saveSetupDraft } from './setup_draft';
import {
  browserTimezone,
  buildSetupSelections,
  countryDefaultCurrencies,
  resolveAdminLanguageCode,
  resolveSetupInitialValues,
  type SetupInitialValues,
  type SetupSelectionItem,
} from './setup_options';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';

const { _t } = createTranslate('auth', { scope: 'web/pages/Setup' });
void [
  _t('Failed to load demo data. You can retry from Modules later.'),
];

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
/** Page busy for Complete→Login→(demo); independent of store.loading flicker. */
const submitting = ref(false);

const formStore = createLocalFormStore({
  fields: [
    { name: 'Username', label: 'Username', type: 'varchar' },
    { name: 'Password', label: 'Password', type: 'varchar' },
    { name: 'ConfirmPassword', label: 'Confirm Password', type: 'varchar' },
    { name: 'Email', label: 'Email', type: 'varchar' },
    { name: 'OrgName', label: 'Organization Name', type: 'varchar' },
    { name: 'CountryCode', label: 'Country / Region', type: 'selection', selection: [] },
    { name: 'CurrencyCode', label: 'Base Currency', type: 'selection', selection: [] },
    { name: 'Timezone', label: 'Business Timezone', type: 'selection', selection: [] },
    { name: 'WithDemo', label: 'Load demo data', type: 'boolean' },
  ],
  initialValues: { WithDemo: false },
  storeId: 'auth.setup',
});

const step = ref<SetupStep>('admin');
const stepperSteps = computed<ChoyStepperStep[]>(() => [
  { value: 'admin', label: _t('Administrator Info') },
  { value: 'organization', label: _t('Organization Info') },
  { value: 'review', label: _t('Review') },
]);
const stepHint = computed(() => {
  if (step.value === 'organization') return _t('Set up your organization basics');
  if (step.value === 'review') return _t('Review the settings and finish setup');
  return _t('Create the administrator account');
});

const error = ref('');
const serverErrors = reactive<Record<string, string>>({});
const formRef = ref<{ validate: () => Promise<boolean> } | null>(null);
const formData = shallowRef<Record<string, unknown> | null>(null);

const optionsLoading = ref(true);
const optionsError = ref('');
const options = shallowRef<InitialSetupOptions | null>(null);
const initialValues = ref<SetupInitialValues | Record<string, never>>({});
const currencyTouched = ref(false);

const emptySelections = (): Record<'CountryCode' | 'CurrencyCode' | 'Timezone', SetupSelectionItem[]> => ({
  CountryCode: [],
  CurrencyCode: [],
  Timezone: [],
});
const selections = shallowRef(emptySelections());
const countryCurrencies = computed(() => (options.value ? countryDefaultCurrencies(options.value) : {}));

function setPageError(message: string) {
  error.value = message;
}

const i18nStore = useI18nStore();

function uiLanguageCode(): string {
  return String(i18nStore.terminologyLang || '');
}

async function leaveIfSetupReady(): Promise<boolean> {
  try {
    if (await authStore.ensureSetupStateReady()) {
      await router.replace(setupLoginFallbackTarget(route.query));
      return true;
    }
  } catch {
    // Keep the current page when the probe is unavailable.
  }
  return false;
}

function applyOptionSelections(resp: InitialSetupOptions) {
  options.value = resp;
  const next = buildSetupSelections(resp);
  for (const key of Object.keys(next) as Array<keyof typeof next>) {
    formStore.setFieldSelection(key, next[key]);
  }
  selections.value = next;
}

async function loadOptions() {
  optionsLoading.value = true;
  optionsError.value = '';
  try {
    await runWithLoading({ silent: true }, async () => {
      const resp = await authStore.getInitialSetupOptions();
      applyOptionSelections(resp);
      const values = resolveSetupInitialValues({
        options: resp,
        draft: loadSetupDraft(),
        browserTimezone: browserTimezone(),
      });
      const countryCurrency = values.CountryCode ? countryCurrencies.value[values.CountryCode] : undefined;
      currencyTouched.value = !!values.CurrencyCode && !!countryCurrency && values.CurrencyCode !== countryCurrency;
      initialValues.value = values;
    });
  } catch (err) {
    if (await leaveIfSetupReady()) return;
    optionsError.value =
      err instanceof ChoysumError && err.message ? err.message : _t('Failed to load setup options. Please try again.');
  } finally {
    optionsLoading.value = false;
  }
}

/** Option labels are localized by the server; refetch them when the UI language changes. */
let labelReloadSeq = 0;
async function reloadOptionLabels() {
  if (!options.value) return;
  const seq = ++labelReloadSeq;
  try {
    await runWithLoading({ silent: true }, async () => {
      const resp = await authStore.getInitialSetupOptions();
      if (seq === labelReloadSeq) applyOptionSelections(resp);
    });
  } catch {
    // Keep the current labels; values stay valid because codes do not change.
  }
}

watch(uiLanguageCode, (next, prev) => {
  if (next && next !== prev) void reloadOptionLabels();
});

function onFormReady(data: Record<string, unknown>) {
  const first = !formData.value;
  formData.value = data;
  // First paint only — later ready emissions must not steal focus mid-edit.
  if (first) focusPrimaryControlForStep();
}

function onFormChange(fields: string[]) {
  if (error.value) error.value = '';
  for (const field of fields) {
    if (serverErrors[field]) delete serverErrors[field];
  }
  if (formData.value) saveSetupDraft(formData.value);
}

function setupWizardForm(): HTMLFormElement | null {
  const root = document.querySelector('.setup-form') as HTMLElement | null;
  if (!root || typeof root.querySelector !== 'function') return null;
  return root.querySelector('form') as HTMLFormElement | null;
}

function inputIn(rootSel: string): HTMLElement | null {
  const root = document.querySelector(rootSel) as HTMLElement | null;
  if (!root || typeof root.querySelector !== 'function') return null;
  return root.querySelector('input') as HTMLElement | null;
}

/** Prefer the first empty required control so Enter/typing match the remaining work. */
function resolvePrimaryControl(): HTMLElement | null {
  if (step.value === 'review') {
    return document.querySelector('#setup-submit') as HTMLElement | null;
  }
  if (step.value === 'organization') {
    return inputIn('.setup-org-name');
  }
  const username = inputIn('.setup-username');
  const password = inputIn('.setup-password');
  const usernameFilled = !!String((username as HTMLInputElement | null)?.value ?? '').trim();
  if (usernameFilled && password) return password;
  return username || password;
}

/** Invalidates in-flight focus retries on step change / unmount. */
let focusGen = 0;
let focusTimer: ReturnType<typeof setTimeout> | null = null;

function clearFocusRetry() {
  if (focusTimer != null) {
    clearTimeout(focusTimer);
    focusTimer = null;
  }
}

/**
 * Focus the step's primary control. Retries while FormView fields finish mounting
 * after the options gate flips (a single nextTick is often too early).
 */
function focusPrimaryControlForStep(attempt = 0) {
  const gen = attempt === 0 ? ++focusGen : focusGen;
  if (attempt === 0) clearFocusRetry();
  void nextTick().then(() => {
    if (gen !== focusGen) return;
    if (optionsLoading.value || optionsError.value) return;
    const el = resolvePrimaryControl();
    if (el && typeof el.focus === 'function') {
      el.focus();
      return;
    }
    if (attempt < 12) {
      focusTimer = setTimeout(() => focusPrimaryControlForStep(attempt + 1), 32);
    }
  });
}

watch(step, () => focusPrimaryControlForStep());
watch(optionsLoading, (loading, wasLoading) => {
  if (wasLoading && !loading && !optionsError.value) focusPrimaryControlForStep();
});

/**
 * Enter advances / completes the wizard. A single body capture listener covers
 * field controls and the post-load case where focus never left `body`. Skips
 * IME, modified keys, open comboboxes, chrome buttons, and Enter outside setup.
 */
function onEnterKey(e: KeyboardEvent) {
  if (e.defaultPrevented) return;
  if (e.isComposing || e.keyCode === 229) return;
  if (e.key && e.key !== 'Enter') return;
  if (e.shiftKey || e.altKey || e.ctrlKey || e.metaKey) return;
  if (optionsLoading.value || optionsError.value || submitting.value) return;
  const target = e.target as HTMLElement | null;
  if (!target) return;

  const tag = String(target.tagName || '').toLowerCase();
  const inSetup = !!target.closest('[data-testid="setup-panel"]');
  const onPageChrome = tag === 'body' || tag === 'html';
  // Ignore Enter from dialogs / menus outside the setup panel.
  if (!inSetup && !onPageChrome) return;

  if (tag === 'textarea') return;
  if (target.closest('[data-testid="choy-selection-clear"]')) return;

  const trigger = target.closest('[data-slot="combobox-trigger"]') as HTMLElement | null;
  const triggerOpen =
    !!trigger &&
    (trigger.getAttribute('aria-expanded') === 'true' || trigger.getAttribute('data-state') === 'open');
  if (triggerOpen) return;
  // Portaled Selection search input (role=combobox) while the list is open.
  if (tag === 'input' && target.getAttribute('role') === 'combobox' && target.getAttribute('aria-expanded') === 'true') {
    return;
  }

  const fromCheckbox = !!target.closest('[data-slot="checkbox"]');
  const button = (tag === 'button' ? target : target.closest('button')) as HTMLElement | null;
  // Back / Submit / header chrome: native activation.
  if (button && !trigger && !fromCheckbox) return;

  const form = (target.closest('form') as HTMLFormElement | null) || setupWizardForm();
  if (!form) return;

  e.preventDefault();
  e.stopPropagation();
  if (typeof form.requestSubmit === 'function') form.requestSubmit();
  else form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

function onDocumentEnterKey(e: KeyboardEvent) {
  if (e.key !== 'Enter' && e.keyCode !== 13) return;
  onEnterKey(e);
}

onMounted(() => {
  void loadOptions();
  document.body.addEventListener('keydown', onDocumentEnterKey, true);
});

onUnmounted(() => {
  focusGen++;
  clearFocusRetry();
  document.body.removeEventListener('keydown', onDocumentEnterKey, true);
});

function goBack() {
  const idx = SETUP_STEPS.indexOf(step.value);
  if (idx > 0) step.value = SETUP_STEPS[idx - 1];
}

function goNext() {
  const idx = SETUP_STEPS.indexOf(step.value);
  if (idx >= 0 && idx < SETUP_STEPS.length - 1) step.value = SETUP_STEPS[idx + 1];
}

/** Show a server field error on its step, then reveal it through FormView validation. */
function showServerFieldError(field: string, message: string): boolean {
  const local = setupLocalFieldForServerField(field);
  const target = setupStepForServerField(field);
  if (!local || !target) return false;
  serverErrors[local] = message;
  step.value = target;
  void nextTick()
    .then(() => nextTick())
    .then(() => formRef.value?.validate());
  return true;
}

async function onFormSubmit(ctx: { formData: Record<string, unknown> }) {
  if (step.value !== 'review') {
    goNext();
    return { handled: true as const, skipSuccessMessage: true as const };
  }
  const req = buildCompleteInitialSetupReq(ctx.formData, resolveAdminLanguageCode(options.value, uiLanguageCode()));
  const wantsDemo = setupWantsDemo(ctx.formData);
  return runHandledAuthSubmit({
    submit: async () => {
      if (submitting.value) return false;
      submitting.value = true;
      let ok = false;
      try {
        // Page mask owns feedback; silent so nested RPCs do not open chrome bar/chef.
        ok = await runWithLoading({ silent: true }, async () => {
          const done = await runSetupSubmit({
            loading: false,
            req,
            setupFailedMessage: _t('Initial setup failed. Please try again later.'),
            completeInitialSetup: input => authStore.completeInitialSetupCore(input),
            login: (username, password) => authStore.loginCore(username, password),
            setError: setPageError,
            onFieldError: showServerFieldError,
            onSetupSucceeded: () => clearSetupDraft(),
            // Marker already written; leave /setup so Ready guard does not bounce the user.
            // Await replace so submitting stays true until the fallback route settles.
            onSetupCompleteLoginFailed: async () => {
              try {
                await router.replace(setupLoginFallbackTarget(route.query, { setupDone: true }));
              } catch {
                // Setup already persisted; do not leave a blank enabled form.
                setPageError(
                  _t('Setup is complete, but the login page could not open. Please open login manually.'),
                );
              }
            },
            // Another tab may have finished setup; leave /setup when the probe says Ready.
            onSetupFailed: async () => {
              await leaveIfSetupReady();
            },
          });
          // Setup + login already succeeded; demo failure must not undo that.
          if (done && wantsDemo) {
            await runSetupLoadDemo({
              requestLoadDemo: requestSetupLoadDemo,
              failedMessage: _t('Failed to load demo data. You can retry from Modules later.'),
              onFailure: message => {
                ChoyMessage.warning(message);
              },
            });
          }
          return done;
        });
        return ok;
      } finally {
        // Keep mask until onSuccess hands off the nav bar; clear on failure only.
        if (!ok) submitting.value = false;
      }
    },
    fallbackMessage: _t('Initial setup failed. Please try again later.'),
    setError: setPageError,
    onSuccess: async () => {
      // Open nav chrome first; keep setup mask until replace finishes so the page
      // never looks idle/clickable between submit end and Guest→App handoff.
      beginNavigationLoading();
      try {
        await router.replace(resolveLoginRedirect(route.query.redirect?.toString()));
      } catch {
        // Setup already persisted; do not re-enable a blank completable form.
        setPageError(
          _t('Setup is complete, but the page could not open. Please open login manually.'),
        );
      } finally {
        submitting.value = false;
      }
    },
  });
}
</script>

<style scoped>
/* Top-anchored so the header and stepper stay put when step content changes height. */
.setup-shell {
  padding-top: clamp(2rem, 12vh, 7.5rem);
}

/* Own width, independent of step content, so the stepper never stretches with wider steps. */
.setup-header {
  max-width: 32rem;
}

.setup-content {
  max-width: 28rem;
}

.setup-actions .submit-button {
  min-width: 6rem;
}
</style>
