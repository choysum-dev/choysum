<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="sandbox-record-properties-definition flex w-full flex-col gap-3"
    data-testid="sandbox-record-properties-definition"
  >
    <p class="m-0 max-w-3xl text-sm text-muted-foreground">
      {{
        _t(
          'Edit the app-level schema for SandboxRecord.Properties. Labels support multiple languages; use Translate next to each label (and selection options) before Save.',
        )
      }}
    </p>
    <ChoyPropertiesDefinitionEditor
      application="sandbox"
      target-model="SandboxRecord"
      properties-field="Properties"
    />
  </div>
</template>

<script setup lang="ts">
import { ChoyPropertiesDefinitionEditor } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SandboxRecordPropertiesDefinitionView' });

const { _t } = createTranslate('sandbox', {
  scope: 'web/views/SandboxRecordPropertiesDefinitionView',
});
</script>
