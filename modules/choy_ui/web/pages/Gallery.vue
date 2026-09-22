<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-gallery-root" :class="{ dark: isDark }" :data-density="density">
    <header class="choy-gallery-header">
      <h1 class="choy-gallery-title">Choy UI Gallery</h1>
      <p class="choy-gallery-lede">Isolation kit shell: tokens + density / dark toggles. No Element Plus on this page.</p>
      <div class="choy-gallery-controls">
        <button type="button" class="choy-gallery-btn" @click="toggleDark">{{ isDark ? 'Light' : 'Dark' }}</button>
        <button type="button" class="choy-gallery-btn" @click="toggleDensity">
          Density: {{ density }}
        </button>
      </div>
    </header>

    <section class="choy-gallery-section">
      <h2>Color tokens</h2>
      <div class="choy-gallery-swatches">
        <div v-for="swatch in colorSwatches" :key="swatch.name" class="choy-gallery-swatch">
          <div class="choy-gallery-swatch__chip" :style="{ background: `var(${swatch.name})` }" />
          <code>{{ swatch.name }}</code>
        </div>
      </div>
    </section>

    <section class="choy-gallery-section">
      <h2>Utility smoke (generated Tailwind)</h2>
      <div class="flex items-center gap-4 p-4 rounded-md bg-muted text-foreground border border-border">
        <span class="text-sm text-primary">bg-primary / text-sm / flex</span>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import '../styles/tokens.css';
import '../styles/preflight-policy.css';
import '../styles/choy-tailwind.generated.css';

type Density = 'comfortable' | 'compact';

const isDark = ref(false);
const density = ref<Density>('comfortable');

const colorSwatches = [
  { name: '--choy-color-primary' },
  { name: '--choy-color-success' },
  { name: '--choy-color-warning' },
  { name: '--choy-color-danger' },
  { name: '--choy-color-info' },
  { name: '--choy-color-foreground' },
  { name: '--choy-color-background' },
  { name: '--choy-color-muted' },
  { name: '--choy-color-border' },
  { name: '--choy-color-ring' },
];

/**
 * Applies the current theme preference to the document root for token overrides.
 */
function applyThemeToDocument(): void {
  const root = document.documentElement;
  root.classList.toggle('dark', isDark.value);
  root.dataset.density = density.value;
}

/**
 * Toggles light / dark token sets.
 */
function toggleDark(): void {
  isDark.value = !isDark.value;
}

/**
 * Toggles comfortable / compact density.
 */
function toggleDensity(): void {
  density.value = density.value === 'comfortable' ? 'compact' : 'comfortable';
}

onMounted(() => {
  applyThemeToDocument();
});

watch([isDark, density], () => {
  applyThemeToDocument();
});
</script>

<style scoped>
.choy-gallery-header {
  padding: var(--choy-layout-content-padding);
  border-bottom: 1px solid var(--choy-color-border);
  background: var(--choy-color-background);
}

.choy-gallery-title {
  margin: 0 0 var(--choy-space-2);
  font-size: var(--choy-font-size-lg);
  font-weight: 600;
}

.choy-gallery-lede {
  margin: 0 0 var(--choy-space-4);
  color: var(--choy-color-foreground);
  opacity: 0.8;
  font-size: var(--choy-font-size-sm);
}

.choy-gallery-controls {
  display: flex;
  gap: var(--choy-space-2);
  flex-wrap: wrap;
}

.choy-gallery-btn {
  appearance: none;
  border: 1px solid var(--choy-color-border);
  border-radius: var(--choy-radius-md);
  background: var(--choy-color-muted);
  color: var(--choy-color-foreground);
  padding: var(--choy-space-2) var(--choy-space-3);
  font: inherit;
  cursor: pointer;
}

.choy-gallery-btn:focus-visible {
  outline: 2px solid var(--choy-color-ring);
  outline-offset: 2px;
}

.choy-gallery-section {
  padding: var(--choy-layout-content-padding);
}

.choy-gallery-section h2 {
  margin: 0 0 var(--choy-space-3);
  font-size: var(--choy-font-size-base);
}

.choy-gallery-swatches {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
  gap: var(--choy-space-3);
}

.choy-gallery-swatch {
  display: flex;
  flex-direction: column;
  gap: var(--choy-space-2);
}

.choy-gallery-swatch__chip {
  height: 2.5rem;
  border-radius: var(--choy-radius-md);
  border: 1px solid var(--choy-color-border);
}

.choy-gallery-swatch code {
  font-size: var(--choy-font-size-sm);
  word-break: break-all;
}
</style>
