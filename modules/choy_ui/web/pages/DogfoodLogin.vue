<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ref } from 'vue';
import '../styles/tokens.css';
import '../styles/preflight-policy.css';
// Produced by web build (EnsureChoyTailwindCSS); not committed.
import '../styles/choy-tailwind.generated.css';
import ChoyButton from '../components/layout/ChoyButton.vue';
import ChoyCard from '../components/layout/ChoyCard.vue';
import ChoyPage from '../components/layout/ChoyPage.vue';
import Input from '../components/vendor/ui/input/Input.vue';
import Toaster from '../components/vendor/ui/toast/Toaster.vue';
import { ChoyMessage } from '../composables/useChoyMessage';

/**
 * Dogfood login page: exercises ChoyButton + L2 Input + ChoyMessage.
 * Fake submit only — no auth RPC. Toaster clears its own toasts on unmount.
 */
const login = ref('');
const password = ref('');
const submitting = ref(false);

function onSubmit(): void {
  if (submitting.value) {
    return;
  }
  submitting.value = true;
  try {
    if (!login.value.trim() || !password.value) {
      ChoyMessage.error('Login failed', { description: 'Login and password are required.' });
      return;
    }
    ChoyMessage.success('Signed in', { description: `Welcome, ${login.value.trim()}.` });
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="choy-dogfood-login choy-gallery-token-scope min-h-full bg-background text-foreground">
    <ChoyPage title="Dogfood Login" width="narrow" class="max-w-md">
      <ChoyCard title="Sign in">
        <form class="flex flex-col gap-3" @submit.prevent="onSubmit">
          <label class="flex flex-col gap-1 text-sm">
            <span>Login</span>
            <Input v-model="login" name="login" autocomplete="username" placeholder="admin" />
          </label>
          <label class="flex flex-col gap-1 text-sm">
            <span>Password</span>
            <Input
              v-model="password"
              name="password"
              type="password"
              autocomplete="current-password"
              placeholder="••••••••"
            />
          </label>
          <div class="flex justify-end gap-2 pt-2">
            <ChoyButton type="submit" :disabled="submitting">Sign in</ChoyButton>
          </div>
        </form>
      </ChoyCard>
      <template #footer>
        <p class="text-sm text-foreground/70">
          Isolation dogfood — uses ChoyButton / ChoyMessage; no Element Plus.
        </p>
      </template>
    </ChoyPage>
    <Toaster />
  </div>
</template>
