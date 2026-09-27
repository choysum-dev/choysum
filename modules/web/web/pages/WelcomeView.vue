<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage width="medium" padding :title="_t('Welcome to Choysum')">
    <ChoyCard :title="_t('Get started')">
      <p class="mb-4 text-sm text-foreground/80">
        {{ _t('Next-generation open-source ERP platform.') }}
      </p>
      <ol class="mb-4 list-decimal space-y-2 pl-5 text-sm">
        <li v-for="(step, index) in steps" :key="index" :class="index === activeStep ? 'font-medium' : 'text-foreground/70'">
          <span>{{ step.title }}</span>
          <span class="block text-foreground/60">{{ step.description }}</span>
        </li>
      </ol>
      <div class="flex flex-wrap gap-2">
        <ChoyButton v-if="activeStep > 0" type="button" variant="outline" @click="activeStep -= 1">
          {{ _t('Back') }}
        </ChoyButton>
        <ChoyButton
          v-if="activeStep < steps.length - 1"
          type="button"
          @click="activeStep += 1"
        >
          {{ _t('Continue') }}
        </ChoyButton>
        <ChoyButton v-else type="button" @click="goHome">
          {{ _t('Go to home') }}
        </ChoyButton>
      </div>
    </ChoyCard>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/pages/WelcomeView' });
const router = useRouter();
const activeStep = ref(0);

const steps = computed(() => [
  {
    title: _t('Sign in'),
    description: _t('Use your account to access the workspace.'),
  },
  {
    title: _t('Explore'),
    description: _t('Open menus and records through the Choy shell.'),
  },
  {
    title: _t('Build'),
    description: _t('Domain modules import Choy* from @/web.'),
  },
]);

function goHome() {
  router.push('/home');
}
</script>
