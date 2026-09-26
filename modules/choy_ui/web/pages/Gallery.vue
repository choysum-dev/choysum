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
          Isolation kit shell: tokens, L1 shells, fields, Form/List/Search/Kanban, Html/Json/Properties,
          O2M·M2M, Chatter, L2 controls, L3 engines, persisted density / dark. No Element Plus on this page.
        </p>
        <div class="choy-gallery-controls">
          <Button variant="outline" size="sm" @click="toggleDark">{{ isDark ? 'Light' : 'Dark' }}</Button>
          <Button variant="outline" size="sm" @click="toggleDensity">Density: {{ density }}</Button>
          <Button variant="outline" size="sm" @click="goDogfoodCompany">
            Dogfood Company
          </Button>
          <Button variant="outline" size="sm" @click="goDogfoodPartner">
            Dogfood Partner
          </Button>
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
        <h2>L1 shells</h2>
        <div class="choy-gallery-l2-grid">
          <Card>
            <CardHeader>
              <CardTitle>Layout + Page</CardTitle>
              <CardDescription>App shell vs page chrome (nested)</CardDescription>
            </CardHeader>
            <CardContent>
              <ChoyLayout class="min-h-40 overflow-hidden rounded-md border border-border text-sm">
                <template #header>
                  <div class="flex items-center justify-between gap-2 px-3 py-2">
                    <span class="font-medium">Header</span>
                    <ChoyNotificationBell :count="3" />
                  </div>
                </template>
                <template #aside>
                  <nav class="p-3 text-foreground/70">Aside</nav>
                </template>
                <ChoyPage title="Sample page" :padding="true" :action-export="true" class="!p-3">
                  <p class="text-sm text-foreground/80">Page body inside layout main.</p>
                </ChoyPage>
                <template #footer>
                  <div class="px-3 py-2 text-xs text-foreground/60">Footer</div>
                </template>
              </ChoyLayout>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Card / Grid / Tabs / Button</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-4">
              <ChoyGrid :cols="2" gap="sm">
                <ChoyCol :span="1">
                  <ChoyCard title="Col A">
                    <p class="text-sm">Grid column A</p>
                  </ChoyCard>
                </ChoyCol>
                <ChoyCol :span="1">
                  <ChoyCard title="Col B">
                    <p class="text-sm">Grid column B</p>
                  </ChoyCard>
                </ChoyCol>
              </ChoyGrid>
              <ChoyTabs v-model="l1Tab" default-value="overview">
                <ChoyTab value="overview" label="Overview">
                  <p class="text-sm text-foreground/80">L1 tabs overview panel.</p>
                </ChoyTab>
                <ChoyTab value="details" label="Details">
                  <p class="text-sm text-foreground/80">L1 tabs details panel.</p>
                </ChoyTab>
              </ChoyTabs>
              <div class="flex flex-wrap gap-2">
                <ChoyButton @click="showChoyMessage('success')">ChoyButton + Message</ChoyButton>
                <ChoyButton variant="outline" @click="showChoyMessage('info')">Info</ChoyButton>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>FormView + fields</CardTitle>
              <CardDescription>Main-path chrome with Varchar / Date / Boolean</CardDescription>
            </CardHeader>
            <CardContent>
              <ChoyFormView title="Demo record" :show-actions="true">
                <template #breadcrumb>
                  <ChoyBreadcrumb :items="[{ label: 'Gallery' }, { label: 'Demo record' }]" />
                </template>
                <template #system-actions>
                  <ChoyButton size="sm" variant="outline">Save</ChoyButton>
                </template>
                <template #button-box>
                  <ChoyButton size="sm" variant="ghost">Action</ChoyButton>
                </template>
                <div class="flex flex-col gap-3">
                  <ChoyVarcharField v-model="galleryName" label="Name" name="name" />
                  <ChoyDateField v-model="galleryDate" label="Date" name="date" />
                  <ChoyBooleanField v-model="galleryActive" label="Active" widget="switch" />
                </div>
              </ChoyFormView>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>List + Search</CardTitle>
              <CardDescription>
                ChoyListView / SearchView; selected {{ galleryListSelection.length }}
              </CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col gap-3">
              <ChoyListView
                v-model:row-selection="galleryListSelection"
                :columns="galleryListColumns"
                :data="galleryListRows"
                :row-id="demoRowId"
                :height="200"
              >
                <template #search>
                  <ChoySearchView
                    v-model:keyword="galleryListKeyword"
                    placeholder="Filter names…"
                    @query-update="onGalleryListSearch"
                  />
                </template>
              </ChoyListView>
              <ChoyPagination
                v-model:page="galleryListPage"
                v-model:page-size="galleryListPageSize"
                :total="galleryListFiltered.length"
              />
            </CardContent>
          </Card>
        </div>
      </section>

      <section class="choy-gallery-section">
        <h2>PR6 — Kanban / relations / Html / Json / Properties</h2>
        <div class="choy-gallery-l2-grid">
          <Card class="sm:col-span-2">
            <CardHeader>
              <CardTitle>Kanban</CardTitle>
              <CardDescription>Drag cards across lanes (local state)</CardDescription>
            </CardHeader>
            <CardContent>
              <ChoyKanbanView v-model:lanes="galleryKanbanLanes" @card-move="onGalleryKanbanMove" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Html / Json</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-4">
              <ChoyHtmlField v-model="galleryHtml" label="Notes (HTML)" name="html" />
              <ChoyJsonField v-model="galleryJson" label="Meta (JSON)" name="json" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Properties</CardTitle>
            </CardHeader>
            <CardContent class="flex flex-col gap-4">
              <ChoyPropertiesField
                v-model="galleryPropsMap"
                label="Dynamic props"
                :items="galleryPropItems"
              />
              <ChoyPropertiesDefinitionEditor
                :items="galleryPropDefs"
                @saved="onGalleryPropDefsSaved"
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>One2Many</CardTitle>
              <CardDescription>list + kanban widgets</CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col gap-4">
              <ChoyOneToManyField
                v-model="galleryO2MRows"
                label="Lines (list)"
                widget="list"
                :columns="galleryO2MColumns"
                title-field="Title"
              />
              <ChoyOneToManyField
                v-model="galleryO2MRows"
                label="Lines (kanban)"
                widget="kanban"
                title-field="Title"
                subtitle-field="Note"
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Many2Many</CardTitle>
              <CardDescription>tags / list / tree</CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col gap-4">
              <ChoyManyToManyField
                v-model="galleryM2MIds"
                label="Tags"
                widget="tags"
                search-key="gallery.m2m"
                :search="searchPartners"
                :options="partnerCatalog"
              />
              <ChoyManyToManyField
                v-model="galleryM2MIds"
                label="List"
                widget="list"
                search-key="gallery.m2m"
                :search="searchPartners"
                :options="partnerCatalog"
                :height="160"
              />
              <ChoyManyToManyField
                v-model="galleryM2MTreeIds"
                label="Tree"
                widget="tree"
                :tree-nodes="galleryTreeNodes"
              />
            </CardContent>
          </Card>
        </div>
      </section>

      <section class="choy-gallery-section">
        <h2>L3 engines (internal)</h2>
        <p class="mb-4 text-sm text-foreground/70">
          Gallery dogfood only — not exported from the public barrel.
        </p>
        <div class="choy-gallery-l2-grid">
          <Card>
            <CardHeader>
              <CardTitle>DataTable</CardTitle>
              <CardDescription>TanStack Table + virtualizer; selected {{ tableSelection.length }}</CardDescription>
            </CardHeader>
            <CardContent>
              <DataTable
                v-model:row-selection="tableSelection"
                :columns="tableColumns"
                :data="tableRows"
                :height="220"
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>DatePicker</CardTitle>
              <CardDescription>@internationalized/date + Reka Calendar</CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col gap-2">
              <DatePicker v-model="pickedDate" />
              <p class="text-sm text-foreground/70">Value: {{ pickedDate ?? '(null)' }}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>RelationCombobox</CardTitle>
              <CardDescription>NameSearch mock + virtual list</CardDescription>
            </CardHeader>
            <CardContent class="flex flex-col gap-2">
              <RelationCombobox
                v-model="relationId"
                search-key="gallery.partners"
                :search="searchPartners"
                @search-more="onRelationSearchMore"
              />
              <p class="text-sm text-foreground/70">
                Selected: {{ relationId ?? '(null)' }}
                <span v-if="relationSearchMoreHint"> · {{ relationSearchMoreHint }}</span>
              </p>
            </CardContent>
          </Card>
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

      <section class="choy-gallery-section">
        <h2>Chatter</h2>
        <ChoyChatter
          model="choy.GalleryDemo"
          res-id="gallery_1"
          :entries="galleryChatterEntries"
          :following="galleryFollowing"
          :follower-count="galleryFollowerCount"
          :posting="galleryPosting"
          current-user-id="usr_gallery"
          current-user-name="Gallery User"
          @post="onGalleryChatterPost"
          @follow="galleryFollowing = true; galleryFollowerCount += 1"
          @unfollow="galleryFollowing = false; galleryFollowerCount = Math.max(0, galleryFollowerCount - 1)"
        />
      </section>

      <Toaster />
    </div>
  </TooltipProvider>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import '../styles/tokens.css';
import '../styles/theme.override.css';
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
import ChoyButton from '../components/layout/ChoyButton.vue';
import ChoyCard from '../components/layout/ChoyCard.vue';
import ChoyCol from '../components/layout/ChoyCol.vue';
import ChoyGrid from '../components/layout/ChoyGrid.vue';
import ChoyLayout from '../components/layout/ChoyLayout.vue';
import ChoyNotificationBell from '../components/layout/ChoyNotificationBell.vue';
import ChoyPage from '../components/layout/ChoyPage.vue';
import ChoyTab from '../components/layout/ChoyTab.vue';
import ChoyTabs from '../components/layout/ChoyTabs.vue';
import ChoyFormView from '../components/view/ChoyFormView.vue';
import ChoyListView from '../components/view/ChoyListView.vue';
import ChoyKanbanView from '../components/view/ChoyKanbanView.vue';
import ChoySearchView from '../components/view/ChoySearchView.vue';
import ChoyPagination from '../components/view/ChoyPagination.vue';
import ChoyBreadcrumb from '../components/view/ChoyBreadcrumb.vue';
import ChoyVarcharField from '../components/field/ChoyVarcharField.vue';
import ChoyDateField from '../components/field/ChoyDateField.vue';
import ChoyBooleanField from '../components/field/ChoyBooleanField.vue';
import ChoyHtmlField from '../components/field/ChoyHtmlField.vue';
import ChoyJsonField from '../components/field/ChoyJsonField.vue';
import ChoyPropertiesField from '../components/field/ChoyPropertiesField.vue';
import ChoyPropertiesDefinitionEditor from '../components/field/ChoyPropertiesDefinitionEditor.vue';
import ChoyOneToManyField from '../components/field/ChoyOneToManyField.vue';
import ChoyManyToManyField from '../components/field/ChoyManyToManyField.vue';
import {
  groupRowsIntoChoyKanbanLanes,
  type ChoyKanbanLane,
  type ChoyKanbanMove,
} from '../components/view/kanbanViewHelpers';
import type { ChoyJsonValue } from '../components/field/jsonFieldHelpers';
import type { PropertiesMap } from '../components/field/propertiesHelpers';
import type { PropertyItemDefinition, ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import type { ChoyManyToManyTreeNode } from '../components/field/ChoyManyToManyField.vue';
import {
  filterRowsByKeyword,
  type ChoySearchQuery,
} from '../components/view/searchViewHelpers';
import { choyPageOffset, clampChoyPage, choyTotalPages } from '../components/view/paginationHelpers';
import DataTable from '../components/internal/DataTable.vue';
import DatePicker from '../components/internal/DatePicker.vue';
import RelationCombobox from '../components/internal/RelationCombobox.vue';
import { ChoyMessage } from '../composables/useChoyMessage';
import {
  persistChoyThemePreference,
  readChoyThemePreference,
  resolveChoyThemePreference,
} from '../composables/applyChoyThemePreference';
import ChoyChatter from '../components/chatter/ChoyChatter.vue';
import type { ChatterTimelineEntry } from '../components/chatter/chatterTypes';
import type { ColumnDef } from '@tanstack/vue-table';
import type { RelationOption } from '../components/internal/relationComboboxHelpers';

type Density = 'comfortable' | 'compact';
type DemoRow = { Id: string; name: string; role: string };

const router = useRouter();
const initialTheme = resolveChoyThemePreference(readChoyThemePreference());
const isDark = ref(initialTheme.dark);
const density = ref<Density>(initialTheme.density);

const sampleInput = ref('');
const sampleTextarea = ref('');
const checkboxOn = ref<boolean | 'indeterminate'>(false);
const switchOn = ref(true);
const activeTab = ref('one');
const l1Tab = ref('overview');
const dialogOpen = ref(false);
const selectValue = ref<string | null>(null);
const comboboxValue = ref('');
const lastMenuAction = ref('');

const galleryName = ref('Demo Partner');
const galleryDate = ref<string | null>('2026-09-23');
const galleryActive = ref(true);
const galleryListKeyword = ref('');
const galleryListApplied = ref('');
const galleryListSelection = ref<Array<string | number>>([]);
const galleryListPage = ref(1);
const galleryListPageSize = ref(8);

const tableSelection = ref<Array<string | number>>([]);
const pickedDate = ref<string | null>(null);
const relationId = ref<string | null>(null);
const relationSearchMoreHint = ref('');

const galleryKanbanLanes = ref<ChoyKanbanLane[]>(
  groupRowsIntoChoyKanbanLanes(
    [
      { Id: 'k1', Title: 'Draft invoice', State: 'draft', Note: 'Acme' },
      { Id: 'k2', Title: 'Review contract', State: 'review', Note: 'Legal' },
      { Id: 'k3', Title: 'Ship order', State: 'done', Note: 'Warehouse' },
      { Id: 'k4', Title: 'Write proposal', State: 'draft', Note: 'Sales' },
    ],
    {
      laneField: 'State',
      laneDefs: [
        { key: 'draft', label: 'Draft' },
        { key: 'review', label: 'Review' },
        { key: 'done', label: 'Done' },
      ],
      titleField: 'Title',
      subtitleField: 'Note',
    },
  ),
);

const galleryHtml = ref<string | null>('<p>Hello <strong>Choy</strong> HTML</p>');
const galleryJson = ref<ChoyJsonValue>({ region: 'APAC', tier: 1 });
const galleryPropDefs = ref<PropertyItemDefinition[]>([
  { name: 'color', type: 'char', string: 'Color', default: 'blue' },
  { name: 'priority', type: 'selection', string: 'Priority', selection: [['low', 'Low'], ['high', 'High']] },
  { name: 'active', type: 'boolean', string: 'Active', default: true },
]);
const galleryPropItems = ref<ResolvedPropertyItem[]>([
  { name: 'color', type: 'char', string: 'Color', value: 'blue' },
  {
    name: 'priority',
    type: 'selection',
    string: 'Priority',
    selection: [['low', 'Low'], ['high', 'High']],
    value: 'low',
  },
  { name: 'active', type: 'boolean', string: 'Active', value: true },
]);
const galleryPropsMap = ref<PropertiesMap>(
  Object.assign(Object.create(null), { color: 'blue', priority: 'low', active: true }),
);

type O2MRow = { Id: string; Title: string; Note?: string };
const galleryO2MRows = ref<O2MRow[]>([
  { Id: 'l1', Title: 'Line A', Note: 'first' },
  { Id: 'l2', Title: 'Line B', Note: 'second' },
]);
const galleryO2MColumns: ColumnDef<O2MRow, unknown>[] = [
  { accessorKey: 'Title', header: 'Title', size: 140 },
  { accessorKey: 'Note', header: 'Note', size: 120 },
];

const galleryM2MIds = ref<string[]>(['p1', 'p2']);
const galleryM2MTreeIds = ref<string[]>(['n1']);
const galleryTreeNodes: ChoyManyToManyTreeNode[] = [
  {
    id: 'n1',
    label: 'Root A',
    children: [
      { id: 'n1a', label: 'Child A1' },
      { id: 'n1b', label: 'Child A2' },
    ],
  },
  { id: 'n2', label: 'Root B' },
];

const galleryChatterEntries = ref<ChatterTimelineEntry[]>([
  {
    kind: 'fieldChange',
    id: 'gf1',
    at: Date.parse('2024-06-01T09:00:00.000Z'),
    field: 'Status',
    changeKind: 'field',
    oldValue: 'draft',
    newValue: 'open',
    actorUid: 'usr_gallery',
  },
  {
    kind: 'message',
    id: 'gm1',
    at: Date.parse('2024-06-01T10:00:00.000Z'),
    type: 'comment',
    body: 'Gallery chatter smoke comment.',
    authorUid: 'usr_other',
  },
]);
const galleryFollowing = ref(false);
const galleryFollowerCount = ref(0);
const galleryPosting = ref(false);
let galleryPostTimer: number | undefined;

function onGalleryChatterPost(body: string): void {
  galleryPosting.value = true;
  window.clearTimeout(galleryPostTimer);
  galleryPostTimer = window.setTimeout(() => {
    galleryPostTimer = undefined;
    galleryChatterEntries.value = [
      ...galleryChatterEntries.value,
      {
        kind: 'message',
        id: `gm_${Date.now()}`,
        at: Date.now(),
        type: 'comment',
        body,
        authorUid: 'usr_gallery',
      },
    ];
    galleryPosting.value = false;
  }, 80);
}

function onGalleryKanbanMove(move: ChoyKanbanMove): void {
  ChoyMessage.info('Kanban move', {
    description: `${move.cardId}: ${move.fromLaneKey} → ${move.toLaneKey}`,
  });
}

function onGalleryPropDefsSaved(items: PropertyItemDefinition[]): void {
  galleryPropDefs.value = items;
  galleryPropItems.value = items.map(item => ({
    ...item,
    value: galleryPropsMap.value[item.name] ?? item.default,
  }));
  ChoyMessage.success('Property definition saved (gallery)');
}

const tableColumns: ColumnDef<DemoRow, unknown>[] = [
  { accessorKey: 'name', header: 'Name', size: 160 },
  { accessorKey: 'role', header: 'Role', size: 140 },
];

const galleryListColumns = tableColumns;

function demoRowId(row: DemoRow): string {
  return row.Id;
}

const tableRows: DemoRow[] = Array.from({ length: 40 }, (_, i) => ({
  Id: `r${i + 1}`,
  name: `Partner ${i + 1}`,
  role: i % 2 === 0 ? 'Customer' : 'Vendor',
}));

const galleryListFiltered = computed(() =>
  filterRowsByKeyword(tableRows, galleryListApplied.value, ['name', 'role']),
);

const galleryListRows = computed(() => {
  const size = Math.max(1, Math.floor(galleryListPageSize.value) || 1);
  const pages = choyTotalPages(galleryListFiltered.value.length, size);
  const safePage = clampChoyPage(galleryListPage.value, pages);
  const offset = choyPageOffset(safePage, size);
  return galleryListFiltered.value.slice(offset, offset + size);
});

function onGalleryListSearch(query: ChoySearchQuery): void {
  galleryListApplied.value = query.keyword;
  galleryListPage.value = 1;
  // The visible row set changes, so ids selected on the previous result set must not linger.
  galleryListSelection.value = [];
}

const partnerCatalog: RelationOption[] = Array.from({ length: 80 }, (_, i) => ({
  id: `p${i + 1}`,
  label: `Partner ${i + 1}`,
}));

async function searchPartners(query: string, opts: { limit: number }): Promise<RelationOption[]> {
  const q = query.toLowerCase();
  const filtered = partnerCatalog.filter(
    (row) => !q || row.label.toLowerCase().includes(q) || row.id.includes(q),
  );
  return filtered.slice(0, opts.limit);
}

function onRelationSearchMore(query: string): void {
  relationSearchMoreHint.value = `Search more requested for "${query || '*'}"`;
}

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
onUnmounted(() => {
  window.clearTimeout(galleryPostTimer);
  galleryPostTimer = undefined;
  clearGalleryTokenScope();
});

/**
 * Toggles light / dark token sets on the gallery root.
 */
function goDogfoodCompany(): void {
  void router.push({ name: 'ChoyUiDogfoodCompany' });
}

function goDogfoodPartner(): void {
  void router.push({ name: 'ChoyUiDogfoodPartner' });
}

function persistGalleryTheme(): void {
  persistChoyThemePreference({
    theme: isDark.value ? 'dark' : 'light',
    density: density.value,
  });
}

function toggleDark(): void {
  isDark.value = !isDark.value;
  persistGalleryTheme();
}

/**
 * Toggles comfortable / compact density on the gallery root.
 */
function toggleDensity(): void {
  density.value = density.value === 'comfortable' ? 'compact' : 'comfortable';
  persistGalleryTheme();
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

function showChoyMessage(level: 'success' | 'info'): void {
  if (level === 'success') {
    ChoyMessage.success('Gallery action');
    return;
  }
  ChoyMessage.info('L1 shell tip', { description: 'ChoyMessage wraps the L2 toast store.' });
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
