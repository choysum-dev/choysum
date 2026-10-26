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
          <ChoyOneToManyKanbanField
            :store="store"
            :prop="'PartnerIdentifiers' as any"
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
              <div class="flex h-full min-h-0 flex-col gap-1.5">
                <div class="flex items-center justify-between gap-2">
                  <div class="truncate text-sm font-medium text-foreground">{{ item?.IdentifierType || _t('Unnamed Type') }}</div>
                  <div class="inline-flex shrink-0 gap-1">
                    <ChoyBadge
                      v-if="item?.IsPrimary"
                      variant="secondary"
                      class="px-1.5 py-0 text-[10px] font-normal"
                    >
                      {{ _t('Primary') }}
                    </ChoyBadge>
                    <ChoyBadge
                      v-if="item?.IsActive === false"
                      variant="outline"
                      class="px-1.5 py-0 text-[10px] font-normal"
                    >
                      {{ _t('Inactive') }}
                    </ChoyBadge>
                  </div>
                </div>
                <div class="truncate text-xs text-muted-foreground">{{ _t('Value') }} {{ item?.Value || '-' }}</div>
                <div class="truncate text-xs text-muted-foreground">{{ _t('Country') }} {{ resolveCountryLabel(item) }}</div>
                <div class="truncate text-xs text-muted-foreground">
                  {{ _t('Valid From') }} {{ formatDateTime(item?.ValidFrom) || '-' }}
                  <span class="text-muted-foreground/60"> · </span>
                  {{ _t('Valid To') }} {{ formatDateTime(item?.ValidTo) || '-' }}
                </div>
                <div v-if="editable || removable" class="mt-auto inline-flex gap-1 pt-1">
                  <ChoyButton v-if="editable" variant="ghost" size="sm" @click.stop="edit">{{ _t('Edit') }}</ChoyButton>
                  <ChoyButton
                    v-if="removable"
                    variant="ghost"
                    size="sm"
                    class="text-destructive hover:bg-destructive/10 hover:text-destructive"
                    @click.stop="remove"
                  >
                    {{ _t('Delete') }}
                  </ChoyButton>
                </div>
              </div>
            </template>
          </ChoyOneToManyKanbanField>
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
  ChoyBadge,
  ChoyButton,
  ChoyOneToManyField,
  ChoyOneToManyKanbanField,
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
    ChoyBadge,
    ChoyButton,
    ChoyTab,
    // Generic SFCs are not Options-API component-typed; cast for registration.
    ChoyOneToManyField: ChoyOneToManyField as any,
    ChoyOneToManyKanbanField: ChoyOneToManyKanbanField as any,
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

