<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="rr-audience-hints">
    <div
      class="rr-audience-hints__info mb-3 rounded-md border border-info/40 bg-info/10 px-3 py-2 text-sm"
      role="alert"
    >
      <p class="font-medium text-foreground">{{ _t('Audience and scope are separate') }}</p>
      <p class="mt-1 text-foreground/80">
        {{
          _t(
            'Empty Role means all users (audience). Empty Application and Model means scope-global (all models). Do not treat empty Role as the same as scope-global.'
          )
        }}
      </p>
    </div>
    <div
      v-if="showGrantEveryoneWarning"
      class="rr-audience-hints__warn mb-3 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-sm"
      role="alert"
    >
      <p class="font-medium text-foreground">{{ _t('Wide-open grant for all users') }}</p>
      <p class="mt-1 text-foreground/80">
        {{
          _t(
            'Kind=grant with an empty Role applies to everyone and can open a large domain. Prefer attaching grants to a concrete role, or use Kind=restrict for all-users rows.'
          )
        }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import { isGrantEveryoneWarning } from '@/auth/web/views/role_record_rule_audience';

defineOptions({ name: 'RoleRecordRuleAudienceHints' });
const { _t } = createTranslate('auth', { scope: 'web/views/RoleRecordRuleAudienceHints' });

/** Injected by ChoyFormView via formController.provideToChildren(). */
const formRoot = inject<{ draft?: Record<string, any> } | null>('form-root', null);

const showGrantEveryoneWarning = computed(() => isGrantEveryoneWarning(formRoot?.draft as Record<string, any> | null | undefined));
</script>
