<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//el-tabs[@data-slot='partner-detail-tab-panels']" position="inside">
    <el-tab-pane :label="_t('Bank Accounts')" name="bank_accounts" data-region="partner-bank-tab">
      <div data-region="partner-bank-panel">
        <OOneToManyKanbanField
          :store="store"
          prop="BankAccounts"
          label=""
          :default-record="defaultBankAccountRecord"
          :editable="canEditBankAccounts()"
          :removable="canEditBankAccounts()"
          :add-button-text="_t('Add Bank Account')"
          :form-view="PartnerBankAccountFormView"
          :create-dialog-title="_t('New Bank Account')"
          :edit-dialog-title="_t('Edit Bank Account')"
          :display-dialog-title="_t('View Bank Account')"
          data-region="partner-bank-section"
        >
          <template #card="{ item, editable, removable, edit, remove }">
            <div class="pbfv-bank-card">
              <div class="pbfv-bank-card__title-row">
                <div class="pbfv-bank-card__title">{{ item?.AccountName || _t('Unnamed Account') }}</div>
                <div class="pbfv-bank-card__flags">
                  <el-tag v-if="item?.IsDefaultInbound" size="small" type="success">{{ _t('Default Inbound') }}</el-tag>
                  <el-tag v-if="item?.IsDefaultOutbound" size="small" type="warning">{{ _t('Default Outbound') }}</el-tag>
                  <el-tag v-if="item?.IsActive === false" size="small" type="info">{{ _t('Inactive') }}</el-tag>
                </div>
              </div>
              <div class="pbfv-bank-card__meta">{{ _t('Bank') }}: {{ item?.BankNameSnapshot || '-' }}</div>
              <div class="pbfv-bank-card__meta">{{ _t('Type') }}: {{ getAccountTypeLabel(item?.AccountType) }}</div>
              <div class="pbfv-bank-card__line">{{ _t('Account Number') }}: {{ item?.AccountNoMasked || '-' }}</div>
              <div class="pbfv-bank-card__meta">
                {{ _t('Inbound/Outbound') }}: {{ item?.AllowInbound ? _t('Yes') : _t('No') }}/{{ item?.AllowOutbound ? _t('Yes') : _t('No') }}
              </div>
              <div v-if="editable || removable" class="pbfv-bank-card__actions">
                <el-button v-if="editable" type="primary" text size="small" @click.stop="edit">{{ _t('Edit') }}</el-button>
                <el-button v-if="removable" type="danger" text size="small" @click.stop="remove">{{ _t('Delete') }}</el-button>
              </div>
            </div>
          </template>
        </OOneToManyKanbanField>
      </div>
    </el-tab-pane>
  </Xpath>
</template>

<script lang="ts" _name="PartnerFormView">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import PartnerFormView from '@/partner/web/views/PartnerFormView.vue';
import type Partner from '@/partner_bank/service/models/partner';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElButton, ElTabPane, ElTag } from 'element-plus';
import OOneToManyKanbanField from '@/web/web/components/field/OOneToManyKanbanField.vue';
import PartnerBankAccountFormView from '@/partner_bank/web/views/PartnerBankAccountFormView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

/**
 * Extends the base partner form with partner bank account management UI.
 */
export default defineComponent({
  name: 'PartnerFormView',
  extends: PartnerFormView,
  components: {
    Xpath,
    ElButton,
    ElTabPane,
    ElTag,
    OOneToManyKanbanField,
    PartnerBankAccountFormView,
  },
  /**
   * Builds partner-bank specific view state on top of the base partner form setup.
   */
  setup(props, ctx) {
    const baseSetup = PartnerFormView?.setup?.(props, ctx) || {};
    const store = (baseSetup as { store: WebModelStore<Partner> }).store;
    const { _t, _lt } = createTranslate('partner_bank', { scope: 'web/views/PartnerFormView' });
    const bankAccountActions = defineModelActions('partner.BankAccount', { entityTitle: _lt('Bank Account') });
    const { hasAction } = usePermission();

    /**
     * Builds the default related bank account row.
     */
    function defaultBankAccountRecord() {
      return {
        AllowInbound: true,
        AllowOutbound: true,
        IsDefaultInbound: false,
        IsDefaultOutbound: false,
        IsActive: true,
      };
    }

    /**
     * Reports whether the current actor can edit bank account rows.
     */
    function canEditBankAccounts() {
      return hasAction(bankAccountActions.edit);
    }

    /**
     * Maps a bank account type to its display label.
     */
    function getAccountTypeLabel(value?: string) {
      switch (value) {
        case 'checking':
          return _t('Checking Account');
        case 'savings':
          return _t('Savings Account');
        case 'corporate':
          return _t('Corporate Account');
        case 'other':
          return _t('Other');
        default:
          return value || '-';
      }
    }

    return {
      ...baseSetup,
      store,
      _t,
      hasAction,
      bankAccountActions,
      PartnerBankAccountFormView,
      defaultBankAccountRecord,
      canEditBankAccounts,
      getAccountTypeLabel,
    };
  },
});
</script>

<style scoped>
.pbfv-bank-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  height: 100%;
  min-height: 0;
}

.pbfv-bank-card__title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.pbfv-bank-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.pbfv-bank-card__flags {
  display: inline-flex;
  gap: 6px;
}

.pbfv-bank-card__meta,
.pbfv-bank-card__line {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pbfv-bank-card__actions {
  margin-top: auto;
  padding-top: 4px;
  display: inline-flex;
  gap: 4px;
}
</style>
