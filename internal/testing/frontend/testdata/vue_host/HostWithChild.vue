<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->
<script lang="ts">
import { defineComponent } from 'vue';
import ChildWidget from './ChildWidget.vue';

// Options API so choysumMount stubs can replace `components` lookups (script-setup
// local imports are closed over and cannot be stubbed without a compiler plugin).
export default defineComponent({
  name: 'HostWithChild',
  components: { ChildWidget },
  setup() {
    return { label: 'parent' };
  },
});
</script>

<template>
  <div class="parent">
    <span class="label">{{ label }}</span>
    <ChildWidget class="child" />
  </div>
</template>
