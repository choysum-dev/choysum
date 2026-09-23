<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <TooltipProvider>
    <div class="choy-gallery-root" :class="{ dark: isDark }" :data-density="density">
      <header class="choy-gallery-header">
        <h1 class="choy-gallery-title">Choy UI Gallery</h1>
        <p class="choy-gallery-lede">
          Isolation kit shell: tokens, L2 controls, density / dark toggles. No Element Plus on this page.
        </p>
        <div class="choy-gallery-controls">
          <Button variant="outline" size="sm" @click="toggleDark">{{ isDark ? 'Light' : 'Dark' }}</Button>
          <Button variant="outline" size="sm" @click="toggleDensity">Density: {{ density }}</Button>
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

      <section class="choy-gallery-section">
        <h2>L2 controls</h2>

        <div class="choy-gallery-l2-grid">
          <Card>
            <CardHeader>
              <CardTitle>Button</CardTitle>
              <CardDescription>Token-backed variants</CardDescription>
            </CardHeader>
            <CardContent class="flex flex-wrap gap-2">
              <Button>Default</Button>
              <Button variant="secondary">Secondary</Button>
              <Button variant="outline">Outline</Button>
              <Button variant="ghost">Ghost</Button>
              <Button variant="destructive">Destructive</Button>
              <Button variant="link">Link</Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Input &amp; Textarea</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-3">
              <Input v-model="sampleInput" placeholder="Type here…" />
              <Textarea v-model="sampleTextarea" placeholder="Multiline…" rows="3" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Checkbox &amp; Switch</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-4">
              <label class="flex items-center gap-2 text-sm">
                <Checkbox v-model="checkboxOn" />
                Accept terms ({{ checkboxOn ? 'on' : 'off' }})
              </label>
              <label class="flex items-center gap-2 text-sm">
                <Switch v-model="switchOn" />
                Notifications ({{ switchOn ? 'on' : 'off' }})
              </label>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Tabs</CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs v-model="activeTab" default-value="one">
                <TabsList>
                  <TabsTrigger value="one">Overview</TabsTrigger>
                  <TabsTrigger value="two">Details</TabsTrigger>
                </TabsList>
                <TabsContent value="one" class="text-sm text-foreground/80">First tab panel.</TabsContent>
                <TabsContent value="two" class="text-sm text-foreground/80">Second tab panel.</TabsContent>
              </Tabs>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Dialog</CardTitle>
              <CardDescription>Trigger, Esc, and overlay close</CardDescription>
            </CardHeader>
            <CardContent>
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child>
                  <Button variant="outline">Open dialog</Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogTitle>Sample dialog</DialogTitle>
                  <DialogDescription>Press Esc or use the close control.</DialogDescription>
                  <p class="text-sm">Dialog open: {{ dialogOpen }}</p>
                </DialogContent>
              </Dialog>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Select</CardTitle>
            </CardHeader>
            <CardContent>
              <Select v-model="selectValue">
                <SelectTrigger class="w-[12rem]" placeholder="Pick a fruit" />
                <SelectContent>
                  <SelectItem value="apple">Apple</SelectItem>
                  <SelectItem value="banana">Banana</SelectItem>
                  <SelectItem value="cherry">Cherry</SelectItem>
                </SelectContent>
              </Select>
              <p class="mt-2 text-xs text-foreground/70">Selected: {{ selectValue || '—' }}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Combobox</CardTitle>
            </CardHeader>
            <CardContent>
              <Combobox v-model="comboboxValue" class="w-[14rem]">
                <ComboboxAnchor>
                  <ComboboxInput placeholder="Search city…" />
                </ComboboxAnchor>
                <ComboboxContent>
                  <ComboboxItem value="shanghai">Shanghai</ComboboxItem>
                  <ComboboxItem value="beijing">Beijing</ComboboxItem>
                  <ComboboxItem value="shenzhen">Shenzhen</ComboboxItem>
                </ComboboxContent>
              </Combobox>
              <p class="mt-2 text-xs text-foreground/70">Value: {{ comboboxValue || '—' }}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Menu, Popover, Tooltip</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-wrap items-center gap-2">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="sm">Menu</Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  <DropdownMenuItem @select="onMenuAction('profile')">Profile</DropdownMenuItem>
                  <DropdownMenuItem @select="onMenuAction('settings')">Settings</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              <Popover>
                <PopoverTrigger as-child>
                  <Button variant="outline" size="sm">Popover</Button>
                </PopoverTrigger>
                <PopoverContent>
                  <p class="text-sm">Floating panel with token borders.</p>
                </PopoverContent>
              </Popover>

              <Tooltip>
                <TooltipTrigger as-child>
                  <Button variant="ghost" size="sm">Tooltip</Button>
                </TooltipTrigger>
                <TooltipContent>Helpful hint</TooltipContent>
              </Tooltip>
            </CardContent>
            <CardFooter v-if="lastMenuAction" class="text-xs text-foreground/70">
              Last menu: {{ lastMenuAction }}
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Badge &amp; Separator</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-3">
              <div class="flex flex-wrap gap-2">
                <Badge>Default</Badge>
                <Badge variant="secondary">Secondary</Badge>
                <Badge variant="outline">Outline</Badge>
                <Badge variant="destructive">Destructive</Badge>
              </div>
              <Separator />
              <p class="text-sm text-foreground/70">Content below the separator.</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Skeleton &amp; Scroll area</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-3">
              <div class="flex gap-3">
                <Skeleton class="size-10 rounded-full" />
                <div class="flex flex-1 flex-col gap-2">
                  <Skeleton class="h-3 w-3/4" />
                  <Skeleton class="h-3 w-1/2" />
                </div>
              </div>
              <ScrollArea class="h-24 w-full rounded-md border border-border">
                <div class="space-y-2 p-3 text-sm">
                  <p v-for="n in 8" :key="n">Scroll row {{ n }}</p>
                </div>
              </ScrollArea>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Toast</CardTitle>
            </CardHeader>
            <CardContent>
              <Button variant="secondary" @click="showSampleToast">Show toast</Button>
            </CardContent>
          </Card>
        </div>
      </section>

      <Toaster />
    </div>
  </TooltipProvider>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import '../styles/tokens.css';
import '../styles/preflight-policy.css';
// Produced by web build (EnsureChoyTailwindCSS); not committed.
import '../styles/choy-tailwind.generated.css';
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Checkbox,
  Combobox,
  ComboboxAnchor,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Input,
  Popover,
  PopoverContent,
  PopoverTrigger,
  ScrollArea,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  Separator,
  Skeleton,
  Switch,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
  Toaster,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
  toast,
} from '../components/vendor/ui';

type Density = 'comfortable' | 'compact';

const isDark = ref(false);
const density = ref<Density>('comfortable');

const sampleInput = ref('');
const sampleTextarea = ref('');
const checkboxOn = ref<boolean | 'indeterminate'>(false);
const switchOn = ref(true);
const activeTab = ref('one');
const dialogOpen = ref(false);
const selectValue = ref('');
const comboboxValue = ref('');
const lastMenuAction = ref('');

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

const galleryTokenScopeClass = 'choy-gallery-token-scope';

/** Host documentElement theme captured once on mount; restored on unmount. */
let hostHadDark = false;
let hostHadTokenScope = false;
let hostDensity: string | null = null;
let hostThemeCaptured = false;

/**
 * Captures host theme attributes before the gallery mutates documentElement.
 */
function captureHostThemeOnce(): void {
  if (hostThemeCaptured) {
    return;
  }
  const el = document.documentElement;
  hostHadDark = el.classList.contains('dark');
  hostHadTokenScope = el.classList.contains(galleryTokenScopeClass);
  hostDensity = el.getAttribute('data-density');
  hostThemeCaptured = true;
}

/**
 * Mirrors gallery tokens onto documentElement so Reka portals under body inherit --choy-*.
 */
function syncGalleryTokenScope(): void {
  captureHostThemeOnce();
  const el = document.documentElement;
  el.classList.add(galleryTokenScopeClass);
  el.classList.toggle('dark', isDark.value);
  el.setAttribute('data-density', density.value);
}

/**
 * Restores documentElement theme state the gallery did not own.
 */
function clearGalleryTokenScope(): void {
  // onUnmounted may run without onMounted (SSR / suspended / HMR unmount);
  // without a capture the zero defaults would strip the host's own theme.
  if (!hostThemeCaptured) {
    return;
  }
  const el = document.documentElement;
  if (!hostHadTokenScope) {
    el.classList.remove(galleryTokenScopeClass);
  }
  // Only revert attributes this gallery still owns; another writer may
  // have changed them while the gallery was mounted.
  if (el.classList.contains('dark') === isDark.value) {
    el.classList.toggle('dark', hostHadDark);
  }
  if (el.getAttribute('data-density') === density.value) {
    if (hostDensity === null) {
      el.removeAttribute('data-density');
    } else {
      el.setAttribute('data-density', hostDensity);
    }
  }
  hostThemeCaptured = false;
}

onMounted(syncGalleryTokenScope);
watch([isDark, density], syncGalleryTokenScope);
onUnmounted(clearGalleryTokenScope);

/**
 * Toggles light / dark token sets on the gallery root.
 */
function toggleDark(): void {
  isDark.value = !isDark.value;
}

/**
 * Toggles comfortable / compact density on the gallery root.
 */
function toggleDensity(): void {
  density.value = density.value === 'comfortable' ? 'compact' : 'comfortable';
}

function onMenuAction(action: string): void {
  lastMenuAction.value = action;
}

function showSampleToast(): void {
  toast({
    title: 'Saved',
    description: 'Gallery toast via Reka ToastProvider.',
  });
}
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

.choy-gallery-l2-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
  gap: var(--choy-space-4);
}
</style>
