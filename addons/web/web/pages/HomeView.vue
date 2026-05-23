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
import { ref, onMounted, computed, shallowRef } from 'vue';
import { useI18nStore } from '@/web/web/stores';
import OPage from '@/web/web/components/page/OPage.vue';
import { ElCard, ElCol, ElRow, ElStatistic, ElIcon, ElProgress, ElButton, ElEmpty, ElDivider } from 'element-plus';

import { Key, Plus, Edit, User, TrendCharts, Connection, Finished } from '@element-plus/icons-vue';

// Activity item type definition.
interface ActivityItem {
  id: number;
  type: string;
  user: string;
  time: Date;
}

// I18n store.
const i18nStore = useI18nStore();
// Keep this computed for potential logic, but do not use it to drive RTL classes.
const isRtlMode = computed(() => i18nStore.currentLocale.textDirection === 'rtl');

// Statistics data kept in shallow refs to reduce reactivity overhead.
const statistics = shallowRef({
  activeUsers: 128,
  totalProjects: 24,
  completedTasks: 342,
  systemHealth: 98.5,
});

// System status kept in shallow refs to reduce reactivity overhead.
const systemStatus = shallowRef({
  cpuUsage: 24,
  memoryUsage: 38,
  diskUsage: 45,
  networkUsage: 12,
});

// Recent activities kept in shallow refs to reduce reactivity overhead.
const recentActivities = shallowRef<ActivityItem[]>([
  { id: 1, type: 'login', user: 'admin', time: new Date(Date.now() - 3600000) },
  { id: 2, type: 'create', user: 'user1', time: new Date(Date.now() - 7200000) },
  { id: 3, type: 'update', user: 'user2', time: new Date(Date.now() - 10800000) },
]);

// Load dashboard data.
async function loadDashboardData() {
  try {
    // This will load real data from the API in the future.
    statistics.value = {
      activeUsers: 128,
      totalProjects: 24,
      completedTasks: 342,
      systemHealth: 98.5,
    };

    recentActivities.value = [
      { id: 1, type: 'login', user: 'admin', time: new Date(Date.now() - 3600000) },
      { id: 2, type: 'create', user: 'user1', time: new Date(Date.now() - 7200000) },
      { id: 3, type: 'update', user: 'user2', time: new Date(Date.now() - 10800000) },
    ];
  } catch (error) {
    console.error('Failed to load dashboard data:', error);
  }
}

// Format activity time.
function formatActivityTime(time: Date): string {
  return i18nStore.formatDateTime(time, { type: 'relative' });
}

const formattedActivities = computed(() => {
  return recentActivities.value.map(activity => ({
    id: activity.id,
    // Precompute everything to avoid branching in the template.
    iconComponent: activity.type === 'login' ? Key : activity.type === 'create' ? Plus : Edit,
    title: `${activity.user} ${activity.type === 'login' ? '登录了系统' : activity.type === 'create' ? '创建了新项目' : '更新了设置'}`,
    time: formatActivityTime(activity.time),
  }));
});

const progressColors = computed(() => ({
  cpu: systemStatus.value.cpuUsage > 80 ? '#F56C6C' : '#67C23A',
  memory: systemStatus.value.memoryUsage > 80 ? '#F56C6C' : '#67C23A',
  disk: systemStatus.value.diskUsage > 80 ? '#F56C6C' : '#67C23A',
  network: systemStatus.value.networkUsage > 80 ? '#F56C6C' : '#67C23A',
}));

const systemMetrics = computed(() => [
  {
    key: 'cpu',
    label: 'CPU使用率',
    value: systemStatus.value.cpuUsage,
    color: progressColors.value.cpu,
  },
  {
    key: 'memory',
    label: '内存使用率',
    value: systemStatus.value.memoryUsage,
    color: progressColors.value.memory,
  },
  {
    key: 'disk',
    label: '磁盘使用率',
    value: systemStatus.value.diskUsage,
    color: progressColors.value.disk,
  },
  {
    key: 'network',
    label: '网络使用率',
    value: systemStatus.value.networkUsage,
    color: progressColors.value.network,
  },
]);

// Load data after the component mounts.
onMounted(loadDashboardData);
</script>

<template>
  <OPage title="仪表板" padding width="full" elevated>
    <template #toolbar>
      <div class="dashboard-toolbar">
        <span class="welcome-text">欢迎回来，管理员</span>
        <el-button type="primary" @click="loadDashboardData">刷新数据</el-button>
      </div>
    </template>

    <!-- Statistics cards section using Element Plus statistics components. -->
    <el-row :gutter="20" class="dashboard-stats">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card class="stat-card" shadow="hover">
          <el-statistic title="活跃用户" :value="statistics.activeUsers" :precision="0">
            <template #prefix>
              <el-icon class="stat-icon">
                <User />
              </el-icon>
            </template>
            <template #suffix>人</template>
          </el-statistic>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <el-card class="stat-card" shadow="hover">
          <el-statistic title="项目总数" :value="statistics.totalProjects" :precision="0">
            <template #prefix>
              <el-icon class="stat-icon">
                <TrendCharts />
              </el-icon>
            </template>
            <template #suffix>个</template>
          </el-statistic>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <el-card class="stat-card" shadow="hover">
          <el-statistic title="已完成任务" :value="statistics.completedTasks" :precision="0">
            <template #prefix>
              <el-icon class="stat-icon">
                <Finished />
              </el-icon>
            </template>
            <template #suffix>个</template>
          </el-statistic>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6">
        <el-card class="stat-card" shadow="hover">
          <el-statistic title="系统健康度" :value="statistics.systemHealth" :precision="1">
            <template #prefix>
              <el-icon class="stat-icon">
                <Connection />
              </el-icon>
            </template>
            <template #suffix>%</template>
          </el-statistic>
          <el-progress
            class="health-progress"
            :percentage="statistics.systemHealth"
            :color="statistics.systemHealth > 90 ? '#67C23A' : '#E6A23C'"
            :stroke-width="6"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- System status and performance section. -->
    <el-row :gutter="20" class="dashboard-performance">
      <el-col :span="24">
        <el-card class="performance-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <h3>系统性能</h3>
            </div>
          </template>
          <div class="performance-metrics">
            <div v-for="metric in systemMetrics" :key="metric.key" class="metric" v-memo="[metric.value, metric.color]">
              <div class="metric-header">
                <span class="metric-label" v-text="metric.label"></span>
                <span class="metric-value">{{ metric.value }}%</span>
              </div>
              <el-progress :percentage="metric.value" :stroke-width="15" :color="metric.color" />
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Recent activity and quick links section. -->
    <el-row :gutter="20" class="dashboard-activity">
      <el-col :xs="24" :md="16">
        <el-card class="activity-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <h3>最近活动</h3>
              <el-button text>查看全部</el-button>
            </div>
          </template>

          <div class="activity-list">
            <div v-for="item in formattedActivities" :key="item.id" class="activity-item" v-memo="[item.id, item.time]">
              <div class="activity-icon">
                <el-icon>
                  <component :is="item.iconComponent" />
                </el-icon>
              </div>
              <div class="activity-content">
                <div class="activity-title" v-text="item.title"></div>
                <div class="activity-time" v-text="item.time"></div>
              </div>
            </div>

            <el-empty v-if="formattedActivities.length === 0" description="暂无活动记录" />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="8">
        <el-card class="links-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <h3>快速访问</h3>
            </div>
          </template>

          <div class="quick-links">
            <el-button type="primary" text icon="User">用户管理</el-button>
            <el-button type="primary" text icon="Setting">系统设置</el-button>
            <el-button type="primary" text icon="Folder">项目列表</el-button>
            <el-button type="primary" text icon="Document">查看报表</el-button>
            <el-button type="primary" text icon="QuestionFilled">帮助中心</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </OPage>
</template>

<style lang="scss" scoped>
.dashboard-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.welcome-text {
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
}

.dashboard-stats {
  margin-block-end: var(--el-margin-large, 24px);
}

.stat-card {
  height: 100%;
  display: flex;
  flex-direction: column;

  :deep(.el-card__body) {
    padding: var(--el-padding-medium, 16px);
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  :deep(.el-statistic) {
    text-align: center;
  }

  :deep(.el-statistic__head) {
    margin-block-end: 8px;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
  }

  :deep(.el-statistic__content) {
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: var(--el-font-weight-bold);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large, 24px);
  }

  .stat-icon {
    margin-inline-end: 4px;
    font-size: 20px;
    color: var(--el-color-primary);
  }

  .health-progress {
    margin-block-start: 8px;
  }
}

.dashboard-performance,
.dashboard-activity {
  margin-block-end: var(--el-margin-large, 20px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  h3 {
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: var(--el-font-weight-bold);
    color: var(--el-text-color-primary);
  }
}

.performance-metrics {
  display: flex;
  flex-direction: column;
  gap: var(--el-gap-large, 16px);
}

.metric {
  margin-block-end: 8px;
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-block-end: 4px;
}

.metric-label {
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
}

.metric-value {
  font-size: var(--el-font-size-small);
  color: var(--el-text-color-regular);
  font-weight: var(--el-font-weight-bold);
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: var(--el-gap-base, 16px);
}

.activity-item {
  display: flex;
  align-items: flex-start;
  padding-block-end: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
    padding-block-end: 0;
  }
}

.activity-icon {
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 50%;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-inline-end: 12px;
  flex-shrink: 0;
}

.activity-content {
  flex: 1;
  min-width: 0;
}

.activity-title {
  font-weight: var(--el-font-weight-bold);
  margin-block-end: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.activity-time {
  font-size: var(--el-font-size-small);
  color: var(--el-text-color-secondary);
}

.quick-links {
  display: flex;
  flex-direction: column;
  gap: var(--el-gap-small, 10px);
}

.quick-links .el-button {
  justify-content: flex-start;
  height: 40px;
  width: 100%;

  :deep(.el-icon) {
    margin-inline-end: 8px;
  }
}

/* Responsive adjustments. */
@media (max-width: 768px) {
  .stat-card {
    margin-block-end: var(--el-margin-small, 16px);
  }

  .activity-card,
  .links-card {
    margin-block-end: var(--el-margin-small, 16px);
  }

  .stat-card :deep(.el-statistic__content) {
    font-size: var(--el-font-size-large, 18px);
  }
}
</style>
