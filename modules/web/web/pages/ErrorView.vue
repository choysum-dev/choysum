<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
// Copyright 2025 The Choysum Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useI18nStore } from '@/web/web/stores';
import OPage from '@/web/web/components/page/OPage.vue';
import { ElButton, ElResult, ElIcon, ElDivider } from 'element-plus';
import { WarningFilled, CircleCloseFilled, QuestionFilled } from '@element-plus/icons-vue';

// Define local types.
type ResultStatus = 'success' | 'warning' | 'info' | 'error' | '404' | '403' | '500';
type ButtonType = '' | 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'text' | 'default';

// Action button definition.
interface ActionItem {
  text: string;
  action: () => void;
  type: ButtonType;
  icon?: string;
}

// Error result configuration.
interface ErrorConfig {
  title: string;
  subtitle: string;
  icon: any;
  status: ResultStatus;
  message: string;
  actions: ActionItem[];
}

// Router instances.
const router = useRouter();
const route = useRoute();

// Use the i18n store to detect text direction.
const i18nStore = useI18nStore();
const isRtlMode = computed(() => i18nStore.currentLocale.textDirection === 'rtl');

// Resolve the error type from the path or query.
const errorType = computed(() => {
  // Get the error type from the path.
  const pathMatch = route.path.match(/\/error\/(\d+)/);
  if (pathMatch && pathMatch[1]) {
    return pathMatch[1];
  }

  // Get the error type from query parameters.
  return route.query.code || '404';
});

// Error type mapping.
const errorConfig = computed<ErrorConfig>(() => {
  switch (errorType.value) {
    case '403':
      // Get authorization error details from query parameters.
      const reason = route.query.reason as string;
      const message = route.query.message as string;
      const fromPath = route.query.from as string;

      // Build a reason-specific subtitle.
      let subtitle = '您没有权限访问此页面';
      if (reason === 'role') {
        subtitle = '您缺少所需角色权限';
      } else if (reason === 'permission') {
        subtitle = '您缺少所需访问权限';
      }

      // Build custom action buttons.
      const actions: ActionItem[] = [{ text: '返回首页', action: goHome, type: 'primary' }];

      // Add a return button when a source path is provided.
      if (fromPath) {
        actions.push({ text: '返回上一页', action: () => goToPath(fromPath), type: 'default' });
      }

      // Add a contact administrator button.
      actions.push({ text: '联系管理员', action: contactAdmin, type: 'default' });

      return {
        title: '权限不足',
        subtitle,
        icon: WarningFilled,
        status: '403',
        message: message || '请确认您拥有访问此资源的权限，或联系系统管理员获取帮助。',
        actions,
      };
    case '500':
      return {
        title: '服务器错误',
        subtitle: '服务器遇到了一个错误',
        icon: CircleCloseFilled,
        status: '500',
        message: (route.query.message as string) || '系统发生了内部错误，请稍后重试或联系技术支持。',
        actions: [
          { text: '返回首页', action: goHome, type: 'primary' },
          { text: '重试', action: retry, type: 'default' },
          { text: '报告问题', action: reportIssue, type: 'warning' },
        ],
      };
    default: // 404
      return {
        title: '页面未找到',
        subtitle: '您访问的页面不存在',
        icon: QuestionFilled,
        status: '404',
        message: '请检查URL是否正确，或者该页面可能已被移动或删除。',
        actions: [
          { text: '返回首页', action: goHome, type: 'primary' },
          { text: '返回上一页', action: goBack, type: 'default' },
        ],
      };
  }
});

// Navigate home.
function goHome() {
  router.push('/');
}

// Navigate back.
function goBack() {
  router.back();
}

// Navigate to a specific path.
function goToPath(path: string) {
  router.push(path);
}

// Retry the current page.
function retry() {
  window.location.reload();
}

// Contact the administrator.
function contactAdmin() {
  // This can open a contact form or send an email.
  window.open('mailto:admin@example.com', '_blank');
}

// Report an issue.
function reportIssue() {
  // This can open an issue report form.
  window.open('https://example.com/support', '_blank');
}
</script>

<template>
  <OPage width="medium" padding>
    <div class="error-container" :dir="isRtlMode ? 'rtl' : 'ltr'">
      <!-- Render the Element Plus Result component. -->
      <el-result :status="errorConfig.status" :title="errorConfig.title" :sub-title="errorConfig.subtitle">
        <!-- Error details. -->
        <template #extra>
          <p class="error-message" v-html="errorConfig.message"></p>

          <el-divider />

          <!-- Action buttons. -->
          <div class="error-actions">
            <el-button v-for="(action, index) in errorConfig.actions" :key="index" :type="action.type" @click="action.action">
              {{ action.text }}
            </el-button>
          </div>
        </template>
      </el-result>
    </div>
  </OPage>
</template>

<style lang="scss" scoped>
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--el-padding-large, 24px) var(--el-padding-medium, 16px);
  text-align: center;
  min-height: 60vh;
}

.error-message {
  max-width: 600px;
  margin: 0 auto var(--el-margin-medium, 16px);
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-base);
  line-height: var(--el-line-height-base);
  word-break: break-word; /* Ensure long messages wrap correctly. */
}

.error-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--el-gap-base, 12px);
  justify-content: center;
  margin-top: var(--el-margin-medium, 16px);
}

.error-illustration {
  margin-top: var(--el-margin-large, 32px);
  max-width: 320px;
  width: 100%;
}

.error-image {
  width: 100%;
  height: auto;
  filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.1));
}

/* Responsive adjustments. */
@media (max-width: 768px) {
  .error-container {
    padding: var(--el-padding-medium, 16px) var(--el-padding-small, 8px);
  }

  .error-message {
    font-size: var(--el-font-size-small);
  }

  .error-illustration {
    max-width: 240px;
    margin-top: var(--el-margin-medium, 16px);
  }
}
</style>
