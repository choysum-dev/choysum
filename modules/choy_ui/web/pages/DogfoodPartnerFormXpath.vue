<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    IMD xpath fixture: inject a ChoyTab into the partner detail tab panels.
    Expr matches the cutover contract (pointed data-anchor, not el-tabs/data-slot).
  -->
  <Xpath expr="//*[@data-anchor='partner.detail.tab-panels']" position="inside">
    <ChoyTab
      value="bank_accounts"
      label="Bank Accounts"
      data-region="partner-bank-tab"
    >
      <div data-region="partner-bank-panel" class="flex flex-col gap-2 text-sm">
        <p class="text-muted-foreground">
          Xpath-injected tab (dogfood stand-in for partner_bank).
        </p>
        <ul class="list-disc pl-5 text-foreground">
          <li>Checking · ****1234 · Default inbound</li>
          <li>Savings · ****5678</li>
        </ul>
      </div>
    </ChoyTab>
  </Xpath>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import ChoyTab from '../components/layout/ChoyTab.vue';
import DogfoodPartnerForm from './DogfoodPartnerForm.vue';

/**
 * Extends the partner dogfood form and inserts a Bank Accounts tab via xpath.
 * Build-time IMD merge replaces Xpath with the ChoyTab pane inside the anchor.
 */
export default defineComponent({
  name: 'DogfoodPartnerForm',
  extends: DogfoodPartnerForm,
  // IMD merge keeps only this script; re-bind base setup so reactive state survives.
  setup: DogfoodPartnerForm.setup,
  components: {
    Xpath,
    ChoyTab,
  },
});
</script>
