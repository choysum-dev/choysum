<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Xpath expr="//el-tabs[@data-slot='partner-detail-tab-panels']" position="inside">
    <el-tab-pane :label="_t('Identifiers and Commercial Extensions')" name="commercial_identifiers" data-region="partner-commercial-tab">
      <div data-region="partner-commercial-panel">
        <div data-region="partner-identifier-section">
          <OOneToManyKanbanField
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
              <div class="pcmv-identifier-card">
                <div class="pcmv-identifier-card__title-row">
                  <div class="pcmv-identifier-card__title">{{ item?.IdentifierType || _t('Unnamed Type') }}</div>
                  <div class="pcmv-identifier-card__flags">
                    <el-tag v-if="item?.IsPrimary" size="small" type="success">{{ _t('Primary') }}</el-tag>
                    <el-tag v-if="item?.IsActive === false" size="small" type="info">{{ _t('Inactive') }}</el-tag>
                  </div>
                </div>
                <div class="pcmv-identifier-card__line">{{ _t('Value') }}: {{ item?.Value || '-' }}</div>
                <div class="pcmv-identifier-card__meta">{{ _t('Country') }}: {{ resolveCountryLabel(item) }}</div>
                <div class="pcmv-identifier-card__meta">{{ _t('Valid From') }}: {{ formatDateTime(item?.ValidFrom) || '-' }}</div>
                <div class="pcmv-identifier-card__meta">{{ _t('Valid To') }}: {{ formatDateTime(item?.ValidTo) || '-' }}</div>
                <div v-if="editable || removable" class="pcmv-identifier-card__actions">
                  <el-button v-if="editable" type="primary" text size="small" @click.stop="edit">{{ _t('Edit') }}</el-button>
                  <el-button v-if="removable" type="danger" text size="small" @click.stop="remove">{{ _t('Delete') }}</el-button>
                </div>
              </div>
            </template>
          </OOneToManyKanbanField>
        </div>
      </div>
    </el-tab-pane>
  </Xpath>
</template>

<script lang="ts" _name="PartnerFormView">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import PartnerFormView from '@/partner/web/views/PartnerFormView.vue';
import { ElButton, ElTabPane, ElTag } from 'element-plus';
import OOneToManyKanbanField from '@/web/web/components/field/OOneToManyKanbanField.vue';
import PartnerIdentifierFormView from '@/partner_commercial/web/views/PartnerIdentifierFormView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

/**
 * Extends the base partner form with commercial identifier management UI.
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
    PartnerIdentifierFormView,
  },
  /**
   * Builds partner-commercial specific view state on top of the base partner form setup.
   */
  setup(props, ctx) {
    const baseSetup = PartnerFormView?.setup?.(props, ctx) || {};
    const store = (baseSetup as any)?.store as any;
    const { _t, _lt } = createTranslate('partner_commercial', { scope: 'web/views/PartnerFormView' });
    const partnerIdentifierActions = defineModelActions('partner.PartnerIdentifier', { entityTitle: _lt('Identifier') });
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
  color: var(--el-text-color-primary);
}

.pcmv-identifier-card__flags {
  display: inline-flex;
  gap: 6px;
}

.pcmv-identifier-card__meta,
.pcmv-identifier-card__line {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pcmv-identifier-card__actions {
  margin-top: auto;
  padding-top: 4px;
  display: inline-flex;
  gap: 4px;
}
</style>
