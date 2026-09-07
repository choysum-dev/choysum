<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
// Distinctive script lines for Codecov / lcov spike assertions (no template → no render helpers).
const SPIKE_MARKER = 42;

function spikeLabel(n: number): string {
  return `spike-${n}`;
}

const label = spikeLabel(SPIKE_MARKER);
void label;
</script>
