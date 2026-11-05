<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :loading="loading" fill-height class="flex min-h-0 w-full flex-1 flex-col">
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
          <ChoyStepper v-model="step" :steps="stepperSteps" :readonly="!!loading" class="setup-stepper mt-8" />
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
          <div v-else class="setup-form" @keydown.enter="onEnterKey">
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
                <SetupReviewStep v-else :form-data="formData" :selections="selections" />
                <div class="setup-actions mt-6 flex justify-between gap-2">
                  <ChoyButton
                    v-if="step !== 'admin'"
                    type="button"
                    variant="outline"
                    size="lg"
                    class="setup-back"
                    :disabled="loading"
                    @click="goBack"
                  >
                    {{ _t('Back') }}
                  </ChoyButton>
                  <ChoyButton type="submit" size="lg" class="submit-button ml-auto" :disabled="loading">
                    <template v-if="step === 'review'">
                      {{ loading ? _t('Complete Setup') + '…' : _t('Complete Setup') }}
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
import { computed, nextTick, onMounted, reactive, ref, shallowRef, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { X } from 'lucide-vue-next';
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
  runSetupSubmit,
  setupLocalFieldForServerField,
  setupLoginFallbackTarget,
  setupStepForServerField,
  type SetupStep,
} from './setup_form';
import { clearSetupDraft, loadSetupDraft, saveSetupDraft } from './setup_draft';
import {
  browserTimezone,
  buildSetupSelections,
  countryDefaultCurrencies,
  resolveAdminLanguageCode,
  resolveSetupInitialValues,
  type SetupSelectionItem,
} from './setup_options';

const { _t } = createTranslate('auth', { scope: 'web/pages/Setup' });

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

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
  ],
  initialValues: {},
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
const initialValues = ref<Record<string, string | null>>({});
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
  try {
    return String(i18nStore.terminologyLang || '');
  } catch {
    return '';
  }
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
    const resp = await authStore.getInitialSetupOptions();
    if (seq === labelReloadSeq) applyOptionSelections(resp);
  } catch {
    // Keep the current labels; values stay valid because codes do not change.
  }
}

watch(uiLanguageCode, (next, prev) => {
  if (next && next !== prev) void reloadOptionLabels();
});

onMounted(() => {
  void loadOptions();
});

function onFormReady(data: Record<string, unknown>) {
  formData.value = data;
}

function onFormChange(fields: string[]) {
  if (error.value) error.value = '';
  for (const field of fields) {
    if (serverErrors[field]) delete serverErrors[field];
  }
  if (formData.value) saveSetupDraft(formData.value);
}

/**
 * Enter in a text input advances the wizard (or completes it on the last step).
 * Skips IME composition, modified keys, and keys a control already handled.
 */
function onEnterKey(e: KeyboardEvent) {
  if (e.defaultPrevented || e.isComposing || e.keyCode === 229) return;
  if (e.shiftKey || e.altKey || e.ctrlKey || e.metaKey) return;
  const target = e.target as HTMLElement | null;
  if (!target || String(target.tagName || '').toLowerCase() !== 'input') return;
  if (target.getAttribute('role') === 'combobox') return;
  const form = target.closest('form');
  if (!form || loading.value) return;
  e.preventDefault();
  if (typeof form.requestSubmit === 'function') form.requestSubmit();
  else form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

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
  return runHandledAuthSubmit({
    submit: () =>
      runSetupSubmit({
        loading: !!loading.value,
        req,
        setupFailedMessage: _t('Initial setup failed. Please try again later.'),
        completeInitialSetup: input => authStore.completeInitialSetup(input),
        login: (username, password) => authStore.login(username, password),
        setError: setPageError,
        onFieldError: showServerFieldError,
        onSetupSucceeded: () => clearSetupDraft(),
        // Marker already written; leave /setup so Ready guard does not bounce the user.
        onSetupCompleteLoginFailed: () => {
          void router.replace(setupLoginFallbackTarget(route.query, { setupDone: true }));
        },
        // Another tab may have finished setup; leave /setup when the probe says Ready.
        onSetupFailed: async () => {
          await leaveIfSetupReady();
        },
      }),
    fallbackMessage: _t('Initial setup failed. Please try again later.'),
    setError: setPageError,
    onSuccess: () => router.replace(resolveLoginRedirect(route.query.redirect?.toString())),
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
