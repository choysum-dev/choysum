<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage width="medium" padding class="error-page">
    <ChoyCard :title="errorConfig.title">
      <p class="mb-2 text-sm text-foreground/70">{{ errorConfig.subtitle }}</p>
      <p class="mb-4 break-words text-sm">{{ errorConfig.message }}</p>
      <div class="flex flex-wrap justify-center gap-2">
        <ChoyButton
          v-for="(action, index) in errorConfig.actions"
          :key="index"
          :variant="action.variant"
          type="button"
          @click="action.action"
        >
          {{ action.text }}
        </ChoyButton>
      </div>
    </ChoyCard>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/pages/ErrorView' });

type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive';

interface ActionItem {
  text: string;
  action: () => void;
  variant: ButtonVariant;
}

interface ErrorConfig {
  title: string;
  subtitle: string;
  message: string;
  actions: ActionItem[];
}

const router = useRouter();
const route = useRoute();

/** vue-router may yield string[] for repeated keys — take the first entry. */
function pickQueryString(value: unknown): string | undefined {
  const first = Array.isArray(value) ? value[0] : value;
  return first == null || first === '' ? undefined : String(first);
}

/** Same-app absolute path only; rejects protocol-relative `//host` and non-paths. */
function sameAppPath(value: unknown): string | undefined {
  const raw = pickQueryString(value);
  if (!raw || !/^\/(?!\/)/.test(raw)) return undefined;
  return raw;
}

const errorConfig = computed<ErrorConfig>(() => {
  // Auth/web redirects use static paths (/error/403) rather than :code params.
  const pathMatch = route.path.match(/\/error\/(\d+)/);
  // Prefer path (/error/403) over a stray ?code=; params.code wins for :code routes.
  const paramCode = pickQueryString(route.params.code);
  const queryCode = pickQueryString(route.query.code);
  const code = String(paramCode || pathMatch?.[1] || queryCode || '404');
  switch (code) {
    case '403': {
      const reason = pickQueryString(route.query.reason);
      const message = pickQueryString(route.query.message);
      const fromPath = sameAppPath(route.query.from);
      let subtitle = _t('You do not have permission to access this page');
      if (reason === 'role') {
        subtitle = _t('You are missing the required role');
      } else if (reason === 'permission') {
        subtitle = _t('You are missing the required permission');
      }
      const actions: ActionItem[] = [{ text: _t('Back to home'), action: goHome, variant: 'default' }];
      if (fromPath) {
        actions.push({ text: _t('Go back'), action: () => goToPath(fromPath), variant: 'outline' });
      }
      actions.push({ text: _t('Contact administrator'), action: contactAdmin, variant: 'outline' });
      return {
        title: _t('Access denied'),
        subtitle,
        message: message || _t('Confirm you have access to this resource, or contact an administrator.'),
        actions,
      };
    }
    case '500':
      return {
        title: _t('Server error'),
        subtitle: _t('The server encountered an error'),
        message:
          pickQueryString(route.query.message) ||
          _t('An internal error occurred. Try again later or contact support.'),
        actions: [
          { text: _t('Back to home'), action: goHome, variant: 'default' },
          { text: _t('Retry'), action: retry, variant: 'outline' },
          { text: _t('Report a problem'), action: reportIssue, variant: 'destructive' },
        ],
      };
    default:
      return {
        title: _t('Page not found'),
        subtitle: _t('The page you requested does not exist'),
        message: _t('Check that the URL is correct, or the page may have been moved or deleted.'),
        actions: [
          { text: _t('Back to home'), action: goHome, variant: 'default' },
          { text: _t('Go back'), action: goBack, variant: 'outline' },
        ],
      };
  }
});

function goHome() {
  router.push('/');
}

function goBack() {
  router.back();
}

function goToPath(path: string) {
  router.push(path);
}

function retry() {
  window.location.reload();
}

function contactAdmin() {
  window.open('mailto:admin@example.com', '_blank');
}

function reportIssue() {
  window.open('https://example.com/support', '_blank');
}
</script>
