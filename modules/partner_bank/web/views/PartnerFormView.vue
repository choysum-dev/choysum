<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath :expr="PARTNER_DETAIL_TAB_PANELS_XPATH" position="inside">
    <ChoyTab value="bank_accounts" :label="_t('Bank Accounts')" data-region="partner-bank-tab">
      <div data-region="partner-bank-panel">
        <ChoyOneToManyKanbanField
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
                  <span v-if="item?.IsDefaultInbound" class="pbfv-flag pbfv-flag--success">{{ _t('Default Inbound') }}</span>
                  <span v-if="item?.IsDefaultOutbound" class="pbfv-flag pbfv-flag--warning">{{ _t('Default Outbound') }}</span>
                  <span v-if="item?.IsActive === false" class="pbfv-flag pbfv-flag--muted">{{ _t('Inactive') }}</span>
                </div>
              </div>
              <div class="pbfv-bank-card__meta">{{ _t('Bank') }}: {{ item?.BankNameSnapshot || '-' }}</div>
              <div class="pbfv-bank-card__meta">{{ _t('Type') }}: {{ getAccountTypeLabel(item?.AccountType) }}</div>
              <div class="pbfv-bank-card__line">{{ _t('Account Number') }}: {{ item?.AccountNoMasked || '-' }}</div>
              <div class="pbfv-bank-card__meta">
                {{ _t('Inbound/Outbound') }}: {{ item?.AllowInbound ? _t('Yes') : _t('No') }}/{{ item?.AllowOutbound ? _t('Yes') : _t('No') }}
              </div>
              <div v-if="editable || removable" class="pbfv-bank-card__actions">
                <ChoyButton v-if="editable" variant="ghost" size="sm" @click.stop="edit">{{ _t('Edit') }}</ChoyButton>
                <ChoyButton v-if="removable" variant="ghost" size="sm" @click.stop="remove">{{ _t('Delete') }}</ChoyButton>
              </div>
            </div>
          </template>
        </ChoyOneToManyKanbanField>
      </div>
    </ChoyTab>
  </Xpath>
</template>

<script lang="ts" _name="PartnerFormView">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import PartnerFormView from '@/partner/web/views/PartnerFormView.vue';
import type Partner from '@/partner_bank/service/models/partner';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  ChoyButton,
  ChoyOneToManyField,
  ChoyTab,
  PARTNER_DETAIL_TAB_PANELS_XPATH,  ChoyOneToManyKanbanField} from '@/web';
import PartnerBankAccountFormView from '@/partner_bank/web/views/PartnerBankAccountFormView.vue';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { bankAccountActions } from './bank_account_actions';

/**
 * Extends the base partner form with partner bank account management UI.
 */
export default defineComponent({
  name: 'PartnerFormView',
  extends: PartnerFormView,
  components: {
    Xpath,
    ChoyButton,
    ChoyTab,
    ChoyOneToManyField,
    PartnerBankAccountFormView,
  },
  /**
   * Builds partner-bank specific view state on top of the base partner form setup.
   */
  setup(props, ctx) {
    const baseSetup = PartnerFormView?.setup?.(props, ctx) || {};
    const store = (baseSetup as { store: WebModelStore<Partner> }).store;
    const { _t } = createTranslate('partner_bank', { scope: 'web/views/PartnerFormView' });
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
      PARTNER_DETAIL_TAB_PANELS_XPATH,
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
  color: var(--choy-foreground, inherit);
}

.pbfv-bank-card__flags {
  display: inline-flex;
  gap: 6px;
}

.pbfv-flag {
  display: inline-flex;
  align-items: center;
  border-radius: 0.375rem;
  border: 1px solid transparent;
  padding: 0.125rem 0.5rem;
  font-size: 12px;
  font-weight: 600;
}

.pbfv-flag--success {
  background: color-mix(in oklab, var(--choy-success, #16a34a) 18%, transparent);
  color: var(--choy-success, #16a34a);
}

.pbfv-flag--warning {
  background: color-mix(in oklab, var(--choy-warning, #d97706) 18%, transparent);
  color: var(--choy-warning, #d97706);
}

.pbfv-flag--muted {
  background: color-mix(in oklab, var(--choy-muted, #64748b) 18%, transparent);
  color: var(--choy-muted-foreground, #64748b);
}

.pbfv-bank-card__meta,
.pbfv-bank-card__line {
  font-size: 12px;
  color: var(--choy-muted-foreground, #64748b);
}

.pbfv-bank-card__actions {
  margin-top: auto;
  padding-top: 4px;
  display: inline-flex;
  gap: 4px;
}
</style>
