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
            <div class="flex h-full min-h-0 flex-col gap-1.5">
              <div class="flex items-center justify-between gap-2">
                <div class="text-sm font-semibold text-foreground">{{ item?.AccountName || _t('Unnamed Account') }}</div>
                <div class="inline-flex gap-1.5">
                  <span v-if="item?.IsDefaultInbound" class="inline-flex items-center rounded-md border border-transparent px-2 py-0.5 text-xs font-semibold bg-success/20 text-success">{{ _t('Default Inbound') }}</span>
                  <span v-if="item?.IsDefaultOutbound" class="inline-flex items-center rounded-md border border-transparent px-2 py-0.5 text-xs font-semibold bg-warning/20 text-warning">{{ _t('Default Outbound') }}</span>
                  <span v-if="item?.IsActive === false" class="inline-flex items-center rounded-md border border-transparent px-2 py-0.5 text-xs font-semibold bg-muted text-muted-foreground">{{ _t('Inactive') }}</span>
                </div>
              </div>
              <div class="text-xs text-muted-foreground">{{ _t('Bank') }}: {{ item?.BankNameSnapshot || '-' }}</div>
              <div class="text-xs text-muted-foreground">{{ _t('Type') }}: {{ getAccountTypeLabel(item?.AccountType) }}</div>
              <div class="text-xs text-muted-foreground">{{ _t('Account Number') }}: {{ item?.AccountNoMasked || '-' }}</div>
              <div class="text-xs text-muted-foreground">
                {{ _t('Inbound/Outbound') }}: {{ item?.AllowInbound ? _t('Yes') : _t('No') }}/{{ item?.AllowOutbound ? _t('Yes') : _t('No') }}
              </div>
              <div v-if="editable || removable" class="mt-auto inline-flex gap-1 pt-1">
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

