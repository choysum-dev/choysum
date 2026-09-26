<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-dogfood-partner min-h-screen bg-background text-foreground">
    <ChoyPage title="Dogfood Partner" width="wide">
      <template #title-actions>
        <ChoyButton variant="outline" size="sm" @click="goGallery">Gallery</ChoyButton>
      </template>

      <p class="mb-4 text-sm text-muted-foreground">
        Partner-style form with
        <code class="text-xs">data-anchor="partner.detail.tab-panels"</code>,
        xpath-injected Bank tab, and isolation Chatter.
      </p>

      <div data-region="partner-detail-root" class="flex flex-col gap-4">
        <ChoyCard title="Partner" data-region="partner-detail-header">
          <ChoyGrid :cols="12">
            <ChoyCol :span="6">
              <ChoyVarcharField v-model="name" label="Name" name="Name" required />
            </ChoyCol>
            <ChoyCol :span="6">
              <ChoyVarcharField v-model="email" label="Email" name="Email" />
            </ChoyCol>
          </ChoyGrid>
        </ChoyCard>

        <ChoyCard title="Detail" data-region="partner-detail-tabs">
          <ChoyTabs
            v-model="activeTab"
            :data-anchor="PARTNER_DETAIL_TAB_PANELS_ANCHOR"
            default-value="general"
          >
            <ChoyTab value="general" label="General" data-region="partner-general-tab">
              <p class="text-sm text-muted-foreground">
                Base tab. Extension modules target
                <code class="text-xs">//*[@data-anchor='partner.detail.tab-panels']</code>
                with position=inside to inject more ChoyTab panes.
              </p>
              <ChoyTextField v-model="notes" label="Notes" name="Notes" class="mt-3" />
            </ChoyTab>
            <ChoyTab value="contacts" label="Contacts" data-region="partner-contacts-tab">
              <p class="text-sm text-muted-foreground">Contacts placeholder (isolation).</p>
            </ChoyTab>
          </ChoyTabs>
        </ChoyCard>

        <ChoyChatter
          model="partner.Partner"
          :res-id="recordId"
          :entries="entries"
          :loading="loading"
          :error="error"
          :following="following"
          :follower-count="followerCount"
          :followers-loading="followersLoading"
          :posting="posting"
          :post-error="postError"
          :current-user-id="currentUserId"
          :current-user-name="currentUserName"
          @post="onPost"
          @follow="onFollow"
          @unfollow="onUnfollow"
        />
      </div>
    </ChoyPage>
    <Toaster />
  </div>
</template>

<script lang="ts">
import { defineComponent, onBeforeUnmount, ref } from 'vue';
import { useRouter } from 'vue-router';
import '../styles/tokens.css';
import '../styles/theme.override.css';
import '../styles/preflight-policy.css';
import '../styles/choy-tailwind.generated.css';
import ChoyButton from '../components/layout/ChoyButton.vue';
import ChoyCard from '../components/layout/ChoyCard.vue';
import ChoyCol from '../components/layout/ChoyCol.vue';
import ChoyGrid from '../components/layout/ChoyGrid.vue';
import ChoyPage from '../components/layout/ChoyPage.vue';
import ChoyTab from '../components/layout/ChoyTab.vue';
import ChoyTabs from '../components/layout/ChoyTabs.vue';
import ChoyTextField from '../components/field/ChoyTextField.vue';
import ChoyVarcharField from '../components/field/ChoyVarcharField.vue';
import ChoyChatter from '../components/chatter/ChoyChatter.vue';
import type { ChatterTimelineEntry } from '../components/chatter/chatterTypes';
import { mergeChatterTimeline } from '../components/chatter/mergeChatterTimeline';
import Toaster from '../components/vendor/ui/toast/Toaster.vue';
import {
  applyChoyThemePreference,
  readChoyThemePreference,
} from '../composables/applyChoyThemePreference';
import { PARTNER_DETAIL_TAB_PANELS_ANCHOR } from './partnerDetailXpath';

/**
 * Partner-style dogfood base view.
 * Public IMD anchor: partner.detail.tab-panels (see DogfoodPartnerFormXpath).
 */
export default defineComponent({
  name: 'DogfoodPartnerForm',
  components: {
    ChoyButton,
    ChoyCard,
    ChoyCol,
    ChoyGrid,
    ChoyPage,
    ChoyTab,
    ChoyTabs,
    ChoyTextField,
    ChoyVarcharField,
    ChoyChatter,
    Toaster,
  },
  setup() {
    applyChoyThemePreference(readChoyThemePreference(), { persist: false });

    const router = useRouter();
    const name = ref('Acme Partner');
    const email = ref('partner@example.com');
    const notes = ref('Isolation partner notes.');
    const activeTab = ref('general');
    const recordId = ref('res_partner_dogfood');
    const currentUserId = ref('usr_dogfood');
    const currentUserName = ref('Dogfood User');

    const loading = ref(false);
    const error = ref<string | null>(null);
    const posting = ref(false);
    const postError = ref<string | null>(null);
    const following = ref(false);
    const followerCount = ref(1);
    const followersLoading = ref(false);

    const entries = ref<ChatterTimelineEntry[]>(
      mergeChatterTimeline(
        [
          {
            Id: 'm1',
            Type: 'comment',
            Body: 'Welcome to the partner dogfood chatter.',
            AuthorUid: 'usr_other',
            CreatedAt: '2024-06-01T10:00:00.000Z',
          },
        ],
        [
          {
            Id: 'f1',
            Field: 'Name',
            Kind: 'field',
            OldValue: 'Old Co',
            NewValue: 'Acme Partner',
            ActorUid: 'usr_dogfood',
            At: '2024-06-01T09:00:00.000Z',
          },
        ],
      ),
    );

    function goGallery(): void {
      void router.push({ name: 'ChoyUiGallery' });
    }

    const timerIds: number[] = [];

    function schedule(fn: () => void, ms: number): void {
      const id = window.setTimeout(() => {
        const index = timerIds.indexOf(id);
        if (index !== -1) timerIds.splice(index, 1);
        fn();
      }, ms);
      timerIds.push(id);
    }

    onBeforeUnmount(() => {
      for (const id of timerIds) {
        window.clearTimeout(id);
      }
      timerIds.length = 0;
    });

    function onPost(body: string): void {
      posting.value = true;
      postError.value = null;
      schedule(() => {
        entries.value = [
          ...entries.value,
          {
            kind: 'message',
            id: `m_${Date.now()}`,
            at: Date.now(),
            type: 'comment',
            body,
            authorUid: currentUserId.value,
          },
        ];
        posting.value = false;
      }, 120);
    }

    function onFollow(): void {
      followersLoading.value = true;
      schedule(() => {
        following.value = true;
        followerCount.value += 1;
        followersLoading.value = false;
      }, 80);
    }

    function onUnfollow(): void {
      followersLoading.value = true;
      schedule(() => {
        following.value = false;
        followerCount.value = Math.max(0, followerCount.value - 1);
        followersLoading.value = false;
      }, 80);
    }

    return {
      PARTNER_DETAIL_TAB_PANELS_ANCHOR,
      name,
      email,
      notes,
      activeTab,
      recordId,
      currentUserId,
      currentUserName,
      loading,
      error,
      posting,
      postError,
      following,
      followerCount,
      followersLoading,
      entries,
      goGallery,
      onPost,
      onFollow,
      onUnfollow,
    };
  },
});
</script>
