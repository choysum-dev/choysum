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
import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useI18nStore } from '@/web/web/stores';
import OPage from '@/web/web/components/page/OPage.vue';
import { ElSteps, ElStep, ElButton, ElDivider, ElCard, ElRow, ElCol, ElTag, ElIcon, ElDescriptions, ElDescriptionsItem } from 'element-plus';
import { Document, VideoPlay, ChatDotRound, SetUp, User, Folder, Compass } from '@element-plus/icons-vue';

// Use the i18n store to detect text direction.
const i18nStore = useI18nStore();
const isRtlMode = computed(() => i18nStore.currentLocale.textDirection === 'rtl');

// Router instance.
const router = useRouter();

// Active onboarding step.
const activeStep = ref(0);

// System version information.
const systemInfo = ref({
  version: '1.0.0',
  buildDate: '2025-04-01',
  environment: 'Production',
  uptime: '2天 14小时 35分钟',
});

// Resource card definitions.
const resourceCards = [
  {
    title: '文档中心',
    description: '查阅详细文档、API参考和开发指南',
    icon: Document,
    action: goToDocumentation,
    buttonText: '浏览文档',
  },
  {
    title: '视频教程',
    description: '通过视频快速学习系统的核心功能',
    icon: VideoPlay,
    action: () => window.open('https://example.com/videos', '_blank'),
    buttonText: '观看教程',
  },
  {
    title: '社区支持',
    description: '加入我们活跃的用户社区获取帮助',
    icon: ChatDotRound,
    action: () => window.open('https://example.com/community', '_blank'),
    buttonText: '访问论坛',
  },
];

// Step content definitions.
const steps = [
  {
    title: '设置账户',
    description: '完成您的个人信息和偏好设置',
    content: '设置您的个人信息、通知偏好和界面设置，打造专属的工作环境。',
    icon: SetUp,
  },
  {
    title: '添加团队成员',
    description: '邀请您的团队加入平台',
    content: '邀请您的团队成员加入平台，分配适当的权限和角色。',
    icon: User,
  },
  {
    title: '创建第一个项目',
    description: '建立并配置您的工作空间',
    content: '创建项目工作空间，设置项目结构和目标，添加团队成员并分配任务。',
    icon: Folder,
  },
  {
    title: '探索功能',
    description: '了解Choysum的强大功能',
    content: '了解仪表板、报表、任务管理等核心功能，充分发挥平台的强大能力。',
    icon: Compass,
  },
];

// Navigate to the dashboard.
function goToDashboard() {
  router.push('/dashboard');
}

// Advance to the next step.
function nextStep() {
  if (activeStep.value < 3) {
    activeStep.value++;
  }
}

// Return to the previous step.
function prevStep() {
  if (activeStep.value > 0) {
    activeStep.value--;
  }
}

// Open the documentation.
function goToDocumentation() {
  window.open('https://docs.example.com', '_blank');
}
</script>

<template>
  <OPage width="medium" elevated padding :useCard="false">
    <div class="welcome-container" :dir="isRtlMode ? 'rtl' : 'ltr'">
      <!-- Welcome header section. -->
      <div class="welcome-header">
        <img src="@/web/web/assets/logo.svg" alt="Choysum Logo" class="welcome-logo" />
        <h1 class="welcome-title">欢迎使用 Choysum</h1>
        <p class="welcome-subtitle">下一代企业级开源管理平台</p>
        <el-tag type="success" effect="plain" class="version-tag">v{{ systemInfo.version }}</el-tag>
      </div>

      <el-divider />

      <!-- Onboarding steps section. -->
      <div class="welcome-steps">
        <h2 class="section-title">快速开始</h2>

        <el-steps :active="activeStep" finish-status="success" simple class="welcome-step-list">
          <el-step v-for="(step, index) in steps" :key="index" :title="step.title" :description="step.description">
            <template #icon>
              <el-icon><component :is="step.icon" /></el-icon>
            </template>
          </el-step>
        </el-steps>

        <!-- Step content section. -->
        <el-card class="step-content" shadow="hover">
          <div class="step-panel">
            <div class="step-icon">
              <el-icon><component :is="steps[activeStep].icon" /></el-icon>
            </div>
            <h3 class="step-title">{{ steps[activeStep].title }}</h3>
            <p class="step-description">{{ steps[activeStep].content }}</p>

            <div class="step-actions">
              <el-button v-if="activeStep > 0" @click="prevStep" icon="ArrowLeft">上一步</el-button>

              <template v-if="activeStep < steps.length - 1">
                <el-button type="primary" @click="nextStep" icon="ArrowRight" class="next-button">继续</el-button>
              </template>
              <template v-else>
                <el-button type="primary" @click="goToDashboard" icon="Right" class="next-button">进入仪表板</el-button>
              </template>
            </div>
          </div>
        </el-card>
      </div>

      <el-divider />

      <!-- Resource cards section. -->
      <el-row :gutter="20" class="resource-cards">
        <el-col v-for="(card, index) in resourceCards" :key="index" :sm="24" :md="8">
          <el-card shadow="hover" class="resource-card">
            <template #header>
              <div class="card-header">
                <el-icon class="card-icon">
                  <component :is="card.icon" />
                </el-icon>
                <h3>{{ card.title }}</h3>
              </div>
            </template>
            <p>{{ card.description }}</p>
            <el-button type="primary" text @click="card.action">
              {{ card.buttonText }}
            </el-button>
          </el-card>
        </el-col>
      </el-row>

      <!-- System information section. -->
      <el-card class="system-info" shadow="hover">
        <template #header>
          <div class="card-header">
            <h3>系统信息</h3>
          </div>
        </template>

        <el-descriptions :column="isRtlMode ? 1 : 2" border>
          <el-descriptions-item label="版本">
            {{ systemInfo.version }}
          </el-descriptions-item>
          <el-descriptions-item label="构建日期">
            {{ systemInfo.buildDate }}
          </el-descriptions-item>
          <el-descriptions-item label="环境">
            {{ systemInfo.environment }}
          </el-descriptions-item>
          <el-descriptions-item label="运行时间">
            {{ systemInfo.uptime }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- Footer actions section. -->
      <div class="welcome-footer">
        <el-button type="primary" size="large" @click="goToDashboard">
          <el-icon class="el-icon--left"><Compass /></el-icon>
          进入系统
        </el-button>
      </div>
    </div>
  </OPage>
</template>

<style lang="scss" scoped>
.welcome-container {
  padding-block: var(--el-padding-large, 24px);
}

.welcome-header {
  text-align: center;
  margin-block-end: var(--el-margin-large, 24px);
  position: relative;
}

.welcome-logo {
  height: 80px;
  margin-block-end: var(--el-margin-medium, 16px);
}

.welcome-title {
  font-size: var(--el-font-size-extra-large);
  font-weight: var(--el-font-weight-bold);
  margin: 0 0 var(--el-margin-small, 12px);
  color: var(--el-color-primary);
}

.welcome-subtitle {
  font-size: var(--el-font-size-medium);
  color: var(--el-text-color-secondary);
  margin: 0;
}

.version-tag {
  position: absolute;
  top: 0;
  inset-inline-end: 0;
}

.section-title {
  margin-block-end: var(--el-margin-large, 24px);
  font-size: var(--el-font-size-large);
  font-weight: var(--el-font-weight-bold);
  color: var(--el-text-color-primary);
}

.welcome-steps {
  margin-block: var(--el-margin-extra-large, 30px);
}

.welcome-step-list {
  margin-block-end: var(--el-margin-medium, 16px);
}

.step-content {
  margin-block: var(--el-margin-medium, 20px);
  transition: all var(--el-transition-duration) var(--el-transition-function-ease-in-out);

  :deep(.el-card__body) {
    padding: var(--el-padding-large, 24px);
  }
}

.step-panel {
  text-align: center;
}

.step-icon {
  display: inline-flex;
  justify-content: center;
  align-items: center;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  margin-block-end: var(--el-margin-medium, 16px);

  .el-icon {
    font-size: 24px;
  }
}

.step-title {
  margin-top: 0;
  margin-block-end: var(--el-margin-medium, 16px);
  font-size: var(--el-font-size-large);
  font-weight: var(--el-font-weight-bold);
  color: var(--el-text-color-primary);
}

.step-description {
  margin-block-end: var(--el-margin-large, 24px);
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-base);
  line-height: var(--el-line-height-base);
  max-width: 540px;
  margin-inline: auto;
}

.step-actions {
  display: flex;
  gap: var(--el-gap-base, 12px);
  justify-content: center;
  margin-top: var(--el-margin-large, 24px);

  .next-button {
    margin-inline-start: auto;
  }
}

.resource-cards {
  margin-block-end: var(--el-margin-extra-large, 30px);
}

.resource-card {
  height: 100%;
  transition: transform var(--el-transition-duration) var(--el-transition-function-fast-out-slow-in);

  &:hover {
    transform: translateY(-5px);
  }

  .el-card__body {
    display: flex;
    flex-direction: column;
    height: 100%;

    p {
      flex: 1;
      margin-block-end: var(--el-margin-medium, 16px);
      color: var(--el-text-color-secondary);
    }

    .el-button {
      align-self: flex-start;
    }
  }
}

.card-header {
  display: flex;
  align-items: center;

  .card-icon {
    margin-inline-end: var(--el-margin-small, 8px);
    font-size: 18px;
    color: var(--el-color-primary);
  }

  h3 {
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: var(--el-font-weight-bold);
  }
}

.system-info {
  margin-block: var(--el-margin-large, 24px);

  :deep(.el-descriptions__label) {
    width: 120px;
  }

  :deep(.el-descriptions__cell) {
    padding: var(--el-padding-small, 8px) var(--el-padding-medium, 16px);
  }
}

.welcome-footer {
  margin-block-start: var(--el-margin-extra-large, 30px);
  text-align: center;

  .el-button {
    padding: var(--el-padding-small, 12px) var(--el-padding-large, 24px);
    font-size: var(--el-font-size-medium);
  }
}

/* Responsive adjustments. */
@media (max-width: 768px) {
  .welcome-title {
    font-size: var(--el-font-size-large);
  }

  .welcome-subtitle {
    font-size: var(--el-font-size-base);
  }

  .resource-card {
    margin-block-end: var(--el-margin-medium, 16px);
  }

  .step-content {
    :deep(.el-card__body) {
      padding: var(--el-padding-medium, 16px);
    }
  }

  .step-actions {
    flex-direction: column;
    gap: var(--el-gap-small, 8px);

    .el-button {
      width: 100%;
    }

    .next-button {
      margin-inline-start: 0;
    }
  }
}
</style>
