<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="rr-audience-hints">
    <el-alert
      class="rr-audience-hints__info"
      type="info"
      :closable="false"
      show-icon
      :title="_t('Audience and scope are separate')"
      :description="
        _t(
          'Empty Role means all users (audience). Empty Application and Model means scope-global (all models). Do not treat empty Role as the same as scope-global.'
        )
      "
    />
    <el-alert
      v-if="showGrantEveryoneWarning"
      class="rr-audience-hints__warn"
      type="warning"
      :closable="false"
      show-icon
      :title="_t('Wide-open grant for all users')"
      :description="
        _t(
          'Kind=grant with an empty Role applies to everyone and can open a large domain. Prefer attaching grants to a concrete role, or use Kind=restrict for all-users rows.'
        )
      "
    />
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue';
import { ElAlert } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';
import { isGrantEveryoneWarning } from '@/auth/web/views/role_record_rule_audience';

defineOptions({ name: 'RoleRecordRuleAudienceHints' });
const { _t } = createTranslate('auth', { scope: 'web/views/RoleRecordRuleAudienceHints' });

/** Injected by OFormView via formController.provideToChildren(). */
const formRoot = inject<{ draft?: Record<string, any> } | null>('form-root', null);

const showGrantEveryoneWarning = computed(() => isGrantEveryoneWarning(formRoot?.draft as Record<string, any> | null | undefined));
</script>

<style scoped>
.rr-audience-hints__info {
  margin-bottom: 12px;
}
.rr-audience-hints__warn {
  margin-bottom: 12px;
}
</style>
