<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts" _name="App">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import App from '@/web/web/App.vue';

/**
 * Extend the root application component with auth-side XPath hooks.
 */
export default defineComponent({
  extends: App,
  components: {
    Xpath,
  },
  setup(props, ctx) {
    const baseSetup = App?.setup?.(props, ctx) || {};

    return {
      ...baseSetup,
    };
  },
});
</script>
