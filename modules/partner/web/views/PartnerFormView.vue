<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{ create: partnerActions.create, edit: partnerActions.edit, copy: partnerActions.copy, delete: partnerActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-region="partner-detail-root">
      <div data-region="partner-primary-form">
        <el-card shadow="never" class="pfv-card" data-region="partner-section-basic">
          <template #header>
            <div class="pfv-card__header"><span>基础信息</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Name" label="名称" :rules="[{ required: true, message: '必填' }]" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Code" label="编码" :rules="[{ required: true, message: '必填' }]" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField
                :store="store"
                prop="CompanyId"
                label="所属公司"
                :searchView="CompanyListView"
                search-view-title="选择公司"
                :rules="[{ required: true, message: '必填' }]"
                @value-click="onCompanyValueClick"
              />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OBooleanField :store="store" prop="IsCompany" label="组织主体" widget="switch" />
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OBooleanField :store="store" prop="IsActive" label="启用" widget="switch" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <ODateTimeField :store="store" prop="CreatedAt" label="创建时间" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <ODateTimeField :store="store" prop="UpdatedAt" label="更新时间" />
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="pfv-card" data-region="partner-section-commercial-basics">
          <template #header>
            <div class="pfv-card__header"><span>商业基础</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OIntField :store="store" prop="CustomerRank" label="客户等级" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OIntField :store="store" prop="SupplierRank" label="供应商等级" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField :store="store" prop="LanguageId" label="默认语言" :searchView="LanguageListView" search-view-title="选择语言" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField :store="store" prop="CurrencyId" label="默认币种" :searchView="CurrencyListView" search-view-title="选择币种" />
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField :store="store" prop="CountryId" label="国家" :searchView="CountryListView" search-view-title="选择国家" />
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="pfv-card" data-region="partner-section-contact-channel">
          <template #header>
            <div class="pfv-card__header"><span>联系方式</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Email" label="邮箱" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Phone" label="电话" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Mobile" label="手机" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Reference" label="外部参考" />
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="pfv-card" data-region="partner-section-default-entry">
          <template #header>
            <div class="pfv-card__header"><span>默认入口</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="8">
              <OManyToOneField :store="store" prop="DefaultContactId" label="默认联系人" :readonly="true" @value-click="onDefaultContactValueClick">
                <OVarCharField :store="store" prop="DefaultContactId.Name" label="名称" />
              </OManyToOneField>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="8">
              <OManyToOneField
                :store="store"
                prop="DefaultBillingAddressId"
                label="默认开票地址"
                :readonly="true"
                @value-click="onDefaultBillingAddressValueClick"
              >
                <OVarCharField :store="store" prop="DefaultBillingAddressId.Name" label="名称" />
              </OManyToOneField>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="8">
              <OManyToOneField
                :store="store"
                prop="DefaultShippingAddressId"
                label="默认收货地址"
                :readonly="true"
                @value-click="onDefaultShippingAddressValueClick"
              >
                <OVarCharField :store="store" prop="DefaultShippingAddressId.Name" label="名称" />
              </OManyToOneField>
            </el-col>
          </el-row>
        </el-card>
      </div>

      <el-card shadow="never" class="pfv-card" data-region="partner-detail-tabs">
        <template #header>
          <div class="pfv-card__header">
            <span>关联数据</span>
            <div data-slot="partner-detail-tab-list" style="display: none"></div>
          </div>
        </template>
        <el-tabs v-model="activeTab" type="card" class="pfv-tabs" data-slot="partner-detail-tab-panels">
          <el-tab-pane label="联系人与地址" name="contacts" data-region="partner-tab-contacts">
            <div data-region="partner-panel-contacts">
              <OOneToManyKanbanField
                :store="store"
                prop="Contacts"
                label=""
                :default-record="defaultContactRecord"
                :editable="canEditContacts()"
                :removable="canEditContacts()"
                :add-button-text="'添加联系人'"
                :form-view="PartnerContactFormView"
                :create-dialog-title="'新增联系人'"
                :edit-dialog-title="'编辑联系人'"
                :display-dialog-title="'查看联系人'"
              >
                <template #card="{ item, editable, removable, edit, remove }">
                  <div class="pfv-contact-card">
                    <div class="pfv-contact-card__title-row">
                      <div class="pfv-contact-card__title">{{ item?.Name || '未命名联系人' }}</div>
                      <div class="pfv-contact-card__flags">
                        <el-tag v-if="item?.IsDefault" size="small" type="success">默认</el-tag>
                        <el-tag v-if="item?.IsActive === false" size="small" type="info">停用</el-tag>
                      </div>
                    </div>
                    <div class="pfv-contact-card__meta">角色：{{ getContactRoleLabel(item?.ContactRole) }}</div>
                    <div class="pfv-contact-card__meta">地址类型：{{ getAddressTypeLabel(item?.AddressType) }}</div>
                    <div v-if="item?.Email" class="pfv-contact-card__line">邮箱：{{ item.Email }}</div>
                    <div v-if="item?.Phone" class="pfv-contact-card__line">电话：{{ item.Phone }}</div>
                    <div v-if="editable || removable" class="pfv-contact-card__actions">
                      <el-button v-if="editable" type="primary" text size="small" @click.stop="edit">编辑</el-button>
                      <el-button v-if="removable" type="danger" text size="small" @click.stop="remove">删除</el-button>
                    </div>
                  </div>
                </template>
              </OOneToManyKanbanField>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </div>
  </OFormView>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Partner from '@/partner/service/models/partner';
import type Company from '@/base/service/models/company';
import type Address from '@/base/service/models/address';
import { ElCard, ElRow, ElCol, ElTabs, ElTabPane } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/OManyToOneField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/OManyToOneRefField.vue';
import OOneToManyKanbanField from '@/web/web/components/field/OOneToManyKanbanField.vue';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import LanguageListView from '@/base/web/views/LanguageListView.vue';
import CurrencyListView from '@/base/web/views/CurrencyListView.vue';
import CountryListView from '@/base/web/views/CountryListView.vue';
import PartnerContactFormView from '@/partner/web/views/PartnerContactFormView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ElButton, ElTag } from 'element-plus';

defineOptions({ name: 'PartnerFormView', inheritAttrs: true });

/**
 * Props consumed by the partner form view.
 */
const props = withDefaults(
  defineProps<{
    store: WebModelStore<Partner>;
    recordId?: string;
    initialValues?: Partial<Partner>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const { store, recordId, initialValues, viewMode, showHeader, createAction } = props;
const partnerActions = defineModelActions('partner.Partner', { entityTitle: '伙伴' });
const { hasAction } = usePermission();
const router = useRouter();
const activeTab = ref('contacts');

/**
 * Display labels for partner contact address categories.
 */
const addressTypeLabels: Record<string, string> = {
  billing: '开票地址',
  shipping: '收货地址',
  office: '办公地址',
  registered: '注册地址',
  other: '其他',
};

/**
 * Display labels for partner contact business roles.
 */
const contactRoleLabels: Record<string, string> = {
  general: '普通联系人',
  billing: '财务联系人',
  shipping: '收货联系人',
  procurement: '采购联系人',
  sales: '销售联系人',
  finance: '财务负责人',
  legal: '法务联系人',
};

/**
 * Builds the default contact row used when a new related contact is added.
 */
function defaultContactRecord() {
  return {
    IsActive: true,
    IsDefault: false,
    Sequence: 10,
  };
}

/**
 * Reports whether the current actor can edit related contact rows.
 */
function canEditContacts() {
  return hasAction(partnerActions.edit);
}

/**
 * Maps a contact address type to its display label.
 */
function getAddressTypeLabel(value?: string) {
  if (!value) {
    return '-';
  }
  return addressTypeLabels[value] || value;
}

/**
 * Maps a contact role to its display label.
 */
function getContactRoleLabel(value?: string) {
  if (!value) {
    return '-';
  }
  return contactRoleLabels[value] || value;
}

/**
 * Opens the selected company record from the partner form.
 */
function onCompanyValueClick(payload: ManyToOneRefValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}

/**
 * Opens the derived default contact record.
 */
function onDefaultContactValueClick(payload: ManyToOneValueClickPayload<Partner>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'PartnerDetail', params: { id } });
}

/**
 * Opens the derived default billing address record.
 */
function onDefaultBillingAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
}

/**
 * Opens the derived default shipping address record.
 */
function onDefaultShippingAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
}
</script>

<style scoped>
.pfv-card {
  margin-bottom: 14px;
}

.pfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.pfv-tabs {
  --el-tabs-header-height: 42px;
}

.pfv-contact-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  height: 100%;
  min-height: 0;
}

.pfv-contact-card__title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.pfv-contact-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.pfv-contact-card__flags {
  display: inline-flex;
  gap: 6px;
}

.pfv-contact-card__meta,
.pfv-contact-card__line {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pfv-contact-card__actions {
  margin-top: auto;
  padding-top: 4px;
  display: inline-flex;
  gap: 4px;
}
</style>
