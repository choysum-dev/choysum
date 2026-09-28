<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath :expr="PARTNER_DETAIL_TAB_PANELS_XPATH" position="inside">
    <ChoyTab
      value="commercial_identifiers"
      :label="_t('Identifiers and Commercial Extensions')"
      data-region="partner-commercial-tab"
    >
      <div data-region="partner-commercial-panel">
        <div data-region="partner-identifier-section">
          <ChoyOneToManyField
            :store="store"
            :prop="'PartnerIdentifiers' as any"
            widget="kanban"
            label=""
            :default-record="defaultIdentifierRecord"
            :editable="canEditIdentifiers()"
            :removable="canEditIdentifiers()"
            :add-button-text="_t('Add Identifier')"
            :form-view="PartnerIdentifierFormView"
            :create-dialog-title="_t('New Identifier')"
            :edit-dialog-title="_t('Edit Identifier')"
            :display-dialog-title="_t('View Identifier')"
          >
            <template #card="{ item, editable, removable, edit, remove }">
              <div class="pcmv-identifier-card">
                <div class="pcmv-identifier-card__title-row">
                  <div class="pcmv-identifier-card__title">{{ item?.IdentifierType || _t('Unnamed Type') }}</div>
                  <div class="pcmv-identifier-card__flags">
                    <span v-if="item?.IsPrimary" class="pcmv-flag pcmv-flag--success">{{ _t('Primary') }}</span>
                    <span v-if="item?.IsActive === false" class="pcmv-flag pcmv-flag--muted">{{ _t('Inactive') }}</span>
                  </div>
                </div>
                <div class="pcmv-identifier-card__line">{{ _t('Value') }}: {{ item?.Value || '-' }}</div>
                <div class="pcmv-identifier-card__meta">{{ _t('Country') }}: {{ resolveCountryLabel(item) }}</div>
                <div class="pcmv-identifier-card__meta">{{ _t('Valid From') }}: {{ formatDateTime(item?.ValidFrom) || '-' }}</div>
                <div class="pcmv-identifier-card__meta">{{ _t('Valid To') }}: {{ formatDateTime(item?.ValidTo) || '-' }}</div>
                <div v-if="editable || removable" class="pcmv-identifier-card__actions">
                  <ChoyButton v-if="editable" variant="ghost" size="sm" @click.stop="edit">{{ _t('Edit') }}</ChoyButton>
                  <ChoyButton v-if="removable" variant="ghost" size="sm" @click.stop="remove">{{ _t('Delete') }}</ChoyButton>
                </div>
              </div>
            </template>
          </ChoyOneToManyField>
        </div>
      </div>
    </ChoyTab>
  </Xpath>
</template>

<script lang="ts" _name="PartnerFormView">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import PartnerFormView from '@/partner/web/views/PartnerFormView.vue';
import {
  ChoyButton,
  ChoyOneToManyField,
  ChoyTab,
  PARTNER_DETAIL_TAB_PANELS_XPATH,
} from '@/web';
import PartnerIdentifierFormView from '@/partner_commercial/web/views/PartnerIdentifierFormView.vue';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { partnerIdentifierActions } from './partner_identifier_actions';

/**
 * Extends the base partner form with commercial identifier management UI.
 */
export default defineComponent({
  name: 'PartnerFormView',
  extends: PartnerFormView,
  components: {
    Xpath,
    ChoyButton,
    ChoyTab,
    ChoyOneToManyField,
    PartnerIdentifierFormView,
  },
  /**
   * Builds partner-commercial specific view state on top of the base partner form setup.
   */
  setup(props, ctx) {
    const baseSetup = PartnerFormView?.setup?.(props, ctx) || {};
    const store = (baseSetup as any)?.store as any;
    const { _t } = createTranslate('partner_commercial', { scope: 'web/views/PartnerFormView' });
    const { hasAction } = usePermission();

    /**
     * Builds the default related identifier row.
     */
    function defaultIdentifierRecord() {
      return {
        IsPrimary: false,
        IsActive: true,
      };
    }

    /**
     * Reports whether the current actor can edit identifier rows.
     */
    function canEditIdentifiers() {
      return hasAction(partnerIdentifierActions.edit);
    }

    /**
     * Resolves a display label for the related country field.
     */
    function resolveCountryLabel(item: any): string {
      const country = item?.CountryId;
      if (country && typeof country === 'object') {
        return String(country.Name || country.DisplayName || country.Id || '-');
      }
      return country ? String(country) : '-';
    }

    /**
     * Formats an identifier validity timestamp for card display.
     */
    function formatDateTime(value: any): string {
      if (!value) return '';
      const dt = value instanceof Date ? value : new Date(String(value));
      if (Number.isNaN(dt.getTime())) return String(value);
      return `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, '0')}-${String(dt.getDate()).padStart(2, '0')} ${String(dt.getHours()).padStart(
        2,
        '0'
      )}:${String(dt.getMinutes()).padStart(2, '0')}`;
    }

    return {
      ...baseSetup,
      store,
      _t,
      hasAction,
      partnerIdentifierActions,
      PartnerIdentifierFormView,
      PARTNER_DETAIL_TAB_PANELS_XPATH,
      defaultIdentifierRecord,
      canEditIdentifiers,
      resolveCountryLabel,
      formatDateTime,
    };
  },
});
</script>

<style scoped>
.pcmv-identifier-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  height: 100%;
  min-height: 0;
}

.pcmv-identifier-card__title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.pcmv-identifier-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--choy-foreground, inherit);
}

.pcmv-identifier-card__flags {
  display: inline-flex;
  gap: 6px;
}

.pcmv-flag {
  display: inline-flex;
  align-items: center;
  border-radius: 0.375rem;
  border: 1px solid transparent;
  padding: 0.125rem 0.5rem;
  font-size: 12px;
  font-weight: 600;
}

.pcmv-flag--success {
  background: color-mix(in oklab, var(--choy-success, #16a34a) 18%, transparent);
  color: var(--choy-success, #16a34a);
}

.pcmv-flag--muted {
  background: color-mix(in oklab, var(--choy-muted, #64748b) 18%, transparent);
  color: var(--choy-muted-foreground, #64748b);
}

.pcmv-identifier-card__meta,
.pcmv-identifier-card__line {
  font-size: 12px;
  color: var(--choy-muted-foreground, #64748b);
}

.pcmv-identifier-card__actions {
  margin-top: auto;
  padding-top: 4px;
  display: inline-flex;
  gap: 4px;
}
</style>
