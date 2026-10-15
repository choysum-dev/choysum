import { h, nextTick, reactive, toRef, watch } from 'vue';
import { ChoyMessage } from '../../../composables/useChoyMessage';
import {
  resolveConfirmChoy,
  useConfirmChoyStore,
} from '../../../composables/confirmChoyAction';

import { UseUserFiltersKey } from '@/web/web/composables/search/useUserFilters';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import Search from './Search.vue';
import SearchFilter from './SearchFilter.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import Popover from '@/web/web/components/vendor/ui/popover/Popover.vue';
import PopoverContent from '@/web/web/components/vendor/ui/popover/PopoverContent.vue';

const savedFiltersApi = {
  state: null as null | {
    favoriteMenuItems: any[];
    loading: boolean;
    loadError: string | null;
    defaultsForOpen: any[];
  },
  load: fnRecorder(async () => {}),
  apply: fnRecorder(),
  saveCurrent: fnRecorder(async () => ({ Id: '1' })),
  updateMeta: fnRecorder(async () => {}),
  remove: fnRecorder(async () => {}),
  lastCodeDefaults: undefined as unknown,
  lastScopeKey: undefined as unknown,
};

const msgWarning = fnRecorder();
const msgSuccess = fnRecorder();
const msgError = fnRecorder();
const origWarning = ChoyMessage.warning;
const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

/** Auto-resolve confirm dialogs opened by Search (confirmChoyAction). */
let autoConfirmChoice: 'confirm' | 'dismiss' = 'confirm';
let confirmCallCount = 0;
let stopConfirmWatch: (() => void) | undefined;

const stubbed = [
  ChoyButton,
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogTitle,
  Popover,
  PopoverContent,
  SearchFilter,
];

function installConfirmAutoResolve() {
  stopConfirmWatch?.();
  const store = useConfirmChoyStore();
  stopConfirmWatch = watch(
    () => store.open,
    open => {
      if (!open) return;
      confirmCallCount += 1;
      const choice = autoConfirmChoice;
      queueMicrotask(() => resolveConfirmChoy(choice));
    },
  );
}

function installEpStubs() {
  ChoyMessage.warning = msgWarning as typeof ChoyMessage.warning;
  ChoyMessage.success = msgSuccess as typeof ChoyMessage.success;
  ChoyMessage.error = msgError as typeof ChoyMessage.error;

  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    props: {
      variant: { type: String, default: 'default' },
      size: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
      type: { type: String, default: 'button' },
    },
    emits: ['click'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: props.type || 'button',
            ...attrs,
            class: ['choy-btn', attrs.class],
            disabled: props.disabled || undefined,
            onClick: (e: any) => emit('click', e),
          },
          slots.default?.(),
        );
    },
  });
  stubSfc(ChoyDialog as any, {
    name: 'Dialog',
    props: { open: { type: Boolean, default: false } },
    emits: ['update:open'],
    setup(props: any, { slots }: any) {
      return () =>
        props.open
          ? h('div', { class: 'choy-dialog', role: 'dialog' }, slots.default?.())
          : null;
    },
  });
  stubSfc(ChoyDialogContent as any, {
    name: 'DialogContent',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'dialog-content' }, slots.default?.());
    },
  });
  stubSfc(ChoyDialogTitle as any, {
    name: 'DialogTitle',
    setup(_: any, { slots }: any) {
      return () => {
        const kids = slots.default?.() || [];
        const text = kids
          .map((c: any) => (typeof c?.children === 'string' ? c.children : ''))
          .join('');
        return h('div', { class: 'dialog-title', 'data-title': text }, kids);
      };
    },
  });
  stubSfc(Popover as any, {
    name: 'Popover',
    props: { open: { type: Boolean, default: false } },
    emits: ['update:open'],
    setup(_: any, { slots }: any) {
      // Always show trigger + content so menu actions are clickable in unit tests.
      return () => h('div', { class: 'el-popover' }, slots.default?.());
    },
  });
  stubSfc(PopoverContent as any, {
    name: 'PopoverContent',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'pop' }, slots.default?.());
    },
  });
  stubSfc(SearchFilter as any, {
    name: 'SearchFilter',
    props: ['store', 'draft', 'fields'],
    emits: ['cancel', 'confirm'],
    setup(_: any, { emit }: any) {
      return () =>
        h('div', { class: 'filter-editor' }, [
          h('button', { type: 'button', class: 'confirm', onClick: () => emit('confirm') }),
          h('button', { type: 'button', class: 'cancel', onClick: () => emit('cancel') }),
        ]);
    },
  });
}

function restoreEpStubs() {
  for (const Comp of stubbed) restoreSfc(Comp as any);
  ChoyMessage.warning = origWarning;
  ChoyMessage.success = origSuccess;
  ChoyMessage.error = origError;
  stopConfirmWatch?.();
  stopConfirmWatch = undefined;
}

function makeUserFiltersFactory() {
  const prev = savedFiltersApi.state;
  savedFiltersApi.state = reactive({
    favoriteMenuItems: (prev?.favoriteMenuItems || []).slice(),
    loading: false,
    loadError: prev?.loadError ?? null,
    defaultsForOpen: (prev?.defaultsForOpen || [{ name: 'CodeDefault', query: ['A', '=', 1] }]).slice(),
  });
  return (params: {
    applyNamedFilter: (nf: any) => void;
    codeDefaults?: () => any;
    scopeKey?: () => string;
  }) => {
    savedFiltersApi.lastScopeKey = params.scopeKey?.();
    savedFiltersApi.lastCodeDefaults = params.codeDefaults?.();
    savedFiltersApi.apply.mockImplementation((fav: { name: string; filter: any }) => {
      params.applyNamedFilter({ name: fav.name, query: fav.filter });
    });
    return {
      favoriteMenuItems: toRef(savedFiltersApi.state!, 'favoriteMenuItems'),
      loading: toRef(savedFiltersApi.state!, 'loading'),
      loadError: toRef(savedFiltersApi.state!, 'loadError'),
      defaultsForOpen: toRef(savedFiltersApi.state!, 'defaultsForOpen'),
      load: savedFiltersApi.load,
      apply: savedFiltersApi.apply,
      saveCurrent: savedFiltersApi.saveCurrent,
      updateMeta: savedFiltersApi.updateMeta,
      remove: savedFiltersApi.remove,
    };
  };
}

function makeStore() {
  return {
    storeId: 'demo.Widget',
    fieldsMetadata: {
      Name: { id: '1', type: 'varchar', string: 'Name' },
      CreatedAt: { id: '2', type: 'datetime', string: 'Created At' },
      Status: { id: '3', type: 'selection', string: 'Status' },
    },
    state: { queryState: { defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }] } },
    getFieldsGetTranslatedString: () => undefined,
  } as any;
}

function btnByText(m: { qa: (s: string) => Element[] }, substr: string) {
  return m.qa('button').find(b => (b.textContent || '').includes(substr));
}

function setInputValue(el: Element | null, value: string) {
  const input = el as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('OSearch behavior', () => {
  beforeEach(() => {
    const st = savedFiltersApi.state || {
      favoriteMenuItems: [],
      loading: false,
      loadError: null,
      defaultsForOpen: [],
    };
    savedFiltersApi.state = st as any;
    st.favoriteMenuItems = [];
    st.loading = false;
    st.loadError = null;
    st.defaultsForOpen = [{ name: 'CodeDefault', query: ['A', '=', 1] }];
    savedFiltersApi.load.mockReset();
    savedFiltersApi.apply.mockReset();
    savedFiltersApi.saveCurrent.mockReset();
    savedFiltersApi.updateMeta.mockReset();
    savedFiltersApi.remove.mockReset();
    savedFiltersApi.load.mockImplementation(async () => {});
    savedFiltersApi.saveCurrent.mockImplementation(async () => ({ Id: '1' }));
    savedFiltersApi.updateMeta.mockImplementation(async () => {});
    savedFiltersApi.remove.mockImplementation(async () => {});
    savedFiltersApi.lastScopeKey = undefined;
    savedFiltersApi.lastCodeDefaults = undefined;
    msgWarning.mockClear();
    msgSuccess.mockClear();
    msgError.mockClear();
    autoConfirmChoice = 'confirm';
    confirmCallCount = 0;
    installConfirmAutoResolve();
  });

  afterEach(() => {
    restoreEpStubs();
  });

  function mountSearch(props: Record<string, any> = {}) {
    installEpStubs();
    const emitted: Record<string, any[][]> = {};
    const track = (name: string) => (...args: any[]) => {
      (emitted[name] ||= []).push(args);
    };
    const m = mountApp(Search as any, {
      reactiveProps: true,
      props: {
        store: makeStore(),
        placeholder: 'Find…',
        defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }],
        ...props,
      },
      on: {
        onQueryUpdate: track('query-update'),
        onDefaultsReady: track('defaults-ready'),
      },
      provide: {
        [UseUserFiltersKey as symbol]: makeUserFiltersFactory(),
      },
    });
    return { ...m, emitted };
  }

  test('wires empty scopeKey when route inject is unavailable', async () => {
    const m = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.lastScopeKey).toBe('');
    m.unmount();
  });

  test('emits query-update on enter / search icon and syncs controlled keyword', async () => {
    const m = mountSearch({ currentKeyword: 'hello' });
    await flushPromises();
    const input = m.q('input.choy-search__input') as HTMLInputElement;
    expect(input.value).toBe('hello');

    m.setupState().onEnter();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);

    const before = (m.emitted['query-update'] || []).length;
    m.qa('.choy-search__leading-btn')[0]!.dispatchEvent(new Event('click', { bubbles: true }));
    await nextTick();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(before);
    m.unmount();
  });

  function isMenuOpen(m: { setupState: () => any }): boolean {
    const st = m.setupState() as { menuVisible?: { value: boolean } | boolean };
    return typeof st.menuVisible === 'boolean' ? st.menuVisible : !!st.menuVisible?.value;
  }

  test('opens search menu on input focus and shell click', async () => {
    const m = mountSearch();
    await flushPromises();
    expect(isMenuOpen(m)).toBe(false);

    m.setupState().onInputFocus();
    await nextTick();
    expect(isMenuOpen(m)).toBe(true);

    // Close via toggle, then shell click re-opens even if input stays focused.
    (m.q('[data-testid=choy-search-menu-toggle]') as HTMLElement).click();
    await nextTick();
    expect(isMenuOpen(m)).toBe(false);

    m.setupState().focusInput();
    await nextTick();
    expect(isMenuOpen(m)).toBe(true);
    m.unmount();
  });

  test('keeps menu open when dismiss-outside fires right after focus open', async () => {
    const m = mountSearch();
    await flushPromises();
    const st = m.setupState() as {
      onInputFocus: () => void;
      onMenuDismissOutside: (e: any) => void;
      searchRootRef?: HTMLElement | { value: HTMLElement | null };
    };
    st.onInputFocus();
    await nextTick();
    expect(isMenuOpen(m)).toBe(true);

    // Same-gesture focusOutside/pointerDownOutside must not close the menu.
    let prevented = false;
    st.onMenuDismissOutside({
      preventDefault() {
        prevented = true;
      },
      detail: { originalEvent: { target: document.body } },
    });
    expect(prevented).toBe(true);
    expect(isMenuOpen(m)).toBe(true);

    const root = m.q('.choy-search') as HTMLElement;
    if (st.searchRootRef && typeof st.searchRootRef === 'object' && 'value' in st.searchRootRef) {
      st.searchRootRef.value = root;
    }
    // After suppress window, shell-targeted dismiss still prevented; true outside is not.
    await new Promise<void>(resolve => setTimeout(resolve, 350));
    prevented = false;
    st.onMenuDismissOutside({
      preventDefault() {
        prevented = true;
      },
      detail: { originalEvent: { target: root, closest: (s: string) => (s.includes('choy.search') ? root : null) } },
    });
    expect(prevented).toBe(true);

    prevented = false;
    st.onMenuDismissOutside({
      preventDefault() {
        prevented = true;
      },
      detail: { originalEvent: { target: document.body } },
    });
    expect(prevented).toBe(false);
    m.unmount();
  });

  test('chevron toggles search menu and Escape closes it', async () => {
    const m = mountSearch();
    await flushPromises();
    const toggle = m.q('[data-testid=choy-search-menu-toggle]') as HTMLElement;

    toggle.click();
    await nextTick();
    expect(isMenuOpen(m)).toBe(true);
    expect(toggle.getAttribute('aria-expanded')).toBe('true');

    toggle.click();
    await nextTick();
    expect(isMenuOpen(m)).toBe(false);
    expect(toggle.getAttribute('aria-expanded')).toBe('false');

    toggle.click();
    await nextTick();
    expect(isMenuOpen(m)).toBe(true);

    m.setupState().onInputKeydown({
      key: 'Escape',
      preventDefault() {},
    });
    await nextTick();
    expect(isMenuOpen(m)).toBe(false);
    m.unmount();
  });

  test('shows grouping tag and clears grouping', async () => {
    const m = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    expect(m.q('.choy-search__grouptag')).toBeTruthy();
    (m.q('.choy-search__tag-close') as HTMLElement).click();
    await flushPromises();
    const queryUpdates = m.emitted['query-update']!;
    const payload = queryUpdates[queryUpdates.length - 1]![0] as any;
    expect(payload.appliedGroups).toEqual([]);
    m.unmount();
  });

  test('opens custom filter editor, warns on incomplete save, and cancels', async () => {
    const m = mountSearch();
    await nextTick();
    const custom = btnByText(m, 'Custom filter');
    expect(custom).toBeTruthy();
    (custom as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();

    (m.q('.confirm') as HTMLElement).click();
    await nextTick();
    expect(msgWarning.calls.length).toBeGreaterThan(0);

    (m.q('.cancel') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeFalsy();
    m.unmount();
  });

  test('applies controlled filters and supports tag close / backspace delete', async () => {
    const filters = [
      {
        id: 'f1',
        name: 'Active',
        logic: 'And' as const,
        children: [{ id: 'c1', field: 'Active', operator: '=', value: true }],
      },
    ];
    const m = mountSearch({ currentAppliedFilters: filters });
    await flushPromises();
    expect(m.q('.choy-search__tag')).toBeTruthy();

    (m.q('.choy-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();
    (m.q('.cancel') as HTMLElement).click();
    await nextTick();

    (m.q('.choy-search__tag-close') as HTMLElement).click();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);

    m.props.currentAppliedFilters = [];
    await flushPromises();
    m.props.currentAppliedFilters = [
      {
        id: 'f2',
        name: 'X',
        logic: 'And',
        children: [{ id: 'c2', field: 'Name', operator: '=', value: 'a' }],
      },
    ];
    await flushPromises();
    expect(m.q('.choy-search__tag')?.textContent || '').toContain('X');
    (m.q('.choy-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();
    (m.q('.cancel') as HTMLElement).click();
    m.unmount();
  });

  test('toggles default filter from menu', async () => {
    const m = mountSearch();
    await nextTick();
    const item = btnByText(m, 'Active');
    expect(item).toBeTruthy();
    (item as HTMLElement).click();
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);
    m.unmount();
  });

  test('applies tree select change for grouping', async () => {
    const m = mountSearch();
    await nextTick();
    const state = m.setupState() as { onGroupFieldSelect?: (id: string | null) => void };
    expect(typeof state.onGroupFieldSelect).toBe('function');
    state.onGroupFieldSelect!('f:Status');
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);
    const queryUpdates = m.emitted['query-update']!;
    const payload = queryUpdates[queryUpdates.length - 1]![0] as any;
    expect(payload.appliedGroups?.some((g: any) => g.field === 'Status' || g === 'Status')).toBe(true);
    m.unmount();
  });

  test('saves a complete edited draft and closes the editor', async () => {
    const m = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f1',
          name: 'Named',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'ok' }],
        },
      ],
    });
    await flushPromises();
    (m.q('.choy-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();
    const before = (m.emitted['query-update'] || []).length;
    (m.q('.confirm') as HTMLElement).click();
    await flushPromises();
    expect(m.q('.filter-editor')).toBeFalsy();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(before);
    m.unmount();
  });

  test('closes quietly when edited filter disappears before save', async () => {
    msgWarning.mockClear();
    const m = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f-gone',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    (m.q('.choy-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();

    m.props.currentAppliedFilters = [];
    await flushPromises();
    (m.q('.confirm') as HTMLElement).click();
    await flushPromises();
    expect(m.q('.filter-editor')).toBeFalsy();
    expect(msgWarning.calls.length).toBe(0);
    m.unmount();
  });

  test('opens grouping menu from group tag and toggles applied menu items', async () => {
    const m = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    (m.q('.choy-search__grouptag') as HTMLElement).click();
    await nextTick();
    const statusItem = btnByText(m, 'Status');
    expect(statusItem).toBeTruthy();
    (statusItem as HTMLElement).click();
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);
    m.unmount();
  });

  test('pending-deletes last filter tag via backspace then removes it', async () => {
    const m = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f1',
          name: 'One',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    const fakeEvt = {
      key: 'Backspace',
      target: { selectionStart: 0, selectionEnd: 0 },
      preventDefault() {},
    };
    m.setupState().onInputKeydown(fakeEvt);
    await nextTick();
    expect(m.q('.choy-search__tag--pending-delete')).toBeTruthy();

    m.setupState().onInputKeydown(fakeEvt);
    await flushPromises();
    expect(m.q('.choy-search__tag')).toBeFalsy();

    m.props.currentAppliedFilters = [
      {
        id: 'f2',
        name: 'Two',
        logic: 'And',
        children: [{ id: 'c2', field: 'Name', operator: '=', value: 'b' }],
      },
    ];
    await flushPromises();
    m.setupState().onInputKeydown(fakeEvt);
    await nextTick();
    m.setupState().onInputKeydown({ key: 'a', target: { selectionStart: 0, selectionEnd: 0 }, preventDefault() {} });
    await nextTick();
    expect(m.q('.choy-search__tag--pending-delete')).toBeFalsy();
    m.unmount();
  });

  test('focuses the keyword input when the shell is clicked', async () => {
    const m = mountSearch();
    const input = m.q('input.choy-search__input') as HTMLInputElement;
    const focus = fnRecorder();
    input.focus = focus as any;
    (m.q('.choy-search__main') as HTMLElement).click();
    expect(focus.calls.length).toBeGreaterThan(0);
    m.unmount();
  });

  test('loads favorites on mount and emits defaults-ready for SearchView first frame', async () => {
    const m = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.load.calls.length).toBeGreaterThan(0);
    expect(m.emitted['defaults-ready']?.[0]?.[0]).toEqual([{ name: 'CodeDefault', query: ['A', '=', 1] }]);
    expect(m.text()).toContain('No favorites yet');

    savedFiltersApi.state!.loadError = 'boom';
    await nextTick();
    expect(m.text()).toContain('Failed to load favorites');
    const before = savedFiltersApi.load.calls.length;
    const retry = btnByText(m, 'Retry');
    expect(retry).toBeTruthy();
    (retry as HTMLElement).click();
    expect(savedFiltersApi.load.calls.length).toBeGreaterThan(before);
    m.unmount();
  });

  test('search menu exposes Filters, Group by, and Favorites columns', async () => {
    const m = mountSearch();
    await flushPromises();
    expect(m.q('[data-slot=search-menu]')).toBeTruthy();
    expect(m.q('.choy-search-menu')).toBeTruthy();
    expect(m.q('[data-slot=search-filters]')).toBeTruthy();
    expect(m.q('[data-slot=search-groupby]')).toBeTruthy();
    expect(m.q('[data-slot=search-favorites]')).toBeTruthy();
    expect(m.q('.choy-search-menu__col--groupby')).toBeTruthy();
    expect(m.q('.choy-search-menu__col--favorites')).toBeTruthy();
    expect(m.q('[data-slot=search-filters]')?.textContent || '').toContain('Custom filter');
    expect(m.q('[data-slot=search-favorites]')?.textContent || '').toContain('Save current filters');
    expect(m.q('[data-slot=search-groupby]')?.textContent || '').toContain('Custom group by');
    expect(m.q('[data-testid=choy-search-groupby-select]')).toBeTruthy();
    m.unmount();
  });

  test('group-by field Combobox applies a field and closes the menu', async () => {
    const m = mountSearch();
    await flushPromises();
    const state = m.setupState() as {
      flatGroupOptions?: { value: Array<{ id: string; label: string }> } | Array<{ id: string; label: string }>;
      filteredGroupOptions?: { value: Array<{ id: string; label: string }> } | Array<{ id: string; label: string }>;
      onGroupFieldSelect?: (id: string | null | undefined) => void;
      menuVisible?: { value: boolean } | boolean;
      groupFieldOpen?: { value: boolean } | boolean;
    };
    const input = m.q('[data-testid=choy-search-groupby-select]') as HTMLElement | null;
    expect(input).toBeTruthy();
    input!.dispatchEvent(new Event('focus', { bubbles: true }));
    await flushPromises();
    const opened =
      typeof state.groupFieldOpen === 'boolean' ? state.groupFieldOpen : state.groupFieldOpen?.value;
    expect(opened).toBe(true);
    const options = Array.isArray(state.flatGroupOptions)
      ? state.flatGroupOptions
      : state.flatGroupOptions?.value || [];
    expect(options.length).toBeGreaterThan(0);
    const filtered = Array.isArray(state.filteredGroupOptions)
      ? state.filteredGroupOptions
      : state.filteredGroupOptions?.value || options;
    expect(filtered.length).toBe(options.length);
    const pick = options[0]!;
    expect(typeof state.onGroupFieldSelect).toBe('function');
    state.onGroupFieldSelect!(pick.id);
    await flushPromises();
    const updates = m.emitted['query-update'] || [];
    expect(updates.length).toBeGreaterThan(0);
    const last = updates[updates.length - 1]?.[0] as { appliedGroups?: unknown[] };
    expect(Array.isArray(last?.appliedGroups) && last.appliedGroups!.length > 0).toBe(true);
    const open = typeof state.menuVisible === 'boolean' ? state.menuVisible : state.menuVisible?.value;
    expect(open).toBe(false);
    m.unmount();
  });

  test('covers codeDefaults singleton/undefined branches', async () => {
    const a = mountSearch({ defaultFilters: undefined as any });
    await flushPromises();
    expect(savedFiltersApi.lastCodeDefaults).toBeUndefined();
    a.unmount();

    const b = mountSearch({ defaultFilters: { name: 'Solo', query: ['X', '=', 1] } as any });
    await flushPromises();
    expect(savedFiltersApi.lastCodeDefaults).toEqual([{ name: 'Solo', query: ['X', '=', 1] }]);
    b.unmount();
  });

  test('shows Check icon for the active favorite id', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-check',
        name: 'Mine',
        shared: true,
        isDefault: false,
        canDelete: false,
        filter: {},
        groups: [],
      },
    ];
    const m = mountSearch();
    await flushPromises();
    expect(m.q('.choy-search__menu-icon--applied')).toBeFalsy();
    (btnByText(m, 'Mine') as HTMLElement).click();
    await flushPromises();
    // Re-open menu after apply closes it.
    const st = m.setupState() as { menuVisible?: { value: boolean } | boolean };
    if (typeof st.menuVisible === 'boolean') st.menuVisible = true;
    else if (st.menuVisible) st.menuVisible.value = true;
    await flushPromises();
    expect(m.q('.choy-search__menu-icon--applied')).toBeTruthy();
    expect(m.text()).toContain('Shared');
    expect(m.q('[data-testid=choy-search-favorite-tag]')?.textContent || '').toContain('Mine');
    m.unmount();
  });

  test('applies and removes favorites (confirm), and saves with empty-name warning', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-1',
        name: 'Mine',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: { And: [['Active', '=', true]] },
        groups: [{ field: 'CreatedAt', granularity: 'year' }],
      },
    ];
    const m = mountSearch();
    await flushPromises();

    const applyBtn = btnByText(m, 'Mine');
    expect(applyBtn).toBeTruthy();
    const beforeEmit = (m.emitted['query-update'] || []).length;
    (applyBtn as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.apply.calls[0]![0]).toMatchObject({
      name: 'Mine',
      filter: { And: [['Active', '=', true]] },
    });
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(beforeEmit);
    const applyPayload = (m.emitted['query-update'] || []).at(-1)?.[0] as {
      appliedGroups?: Array<{ field?: string; granularity?: string }>;
    };
    expect(applyPayload?.appliedGroups?.[0]).toMatchObject({
      field: 'CreatedAt',
      granularity: 'year',
    });

    // Re-open the search menu so delete starts with menuVisible=true (apply closed it).
    const st = m.setupState() as { menuVisible?: { value: boolean } | boolean };
    if (typeof st.menuVisible === 'boolean') st.menuVisible = true;
    else if (st.menuVisible) st.menuVisible.value = true;
    await flushPromises();

    (m.q('.choy-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(confirmCallCount).toBeGreaterThan(0);
    expect(savedFiltersApi.remove.calls[0]).toEqual(['fav-1']);
    expect(msgSuccess.calls.length).toBeGreaterThan(0);
    const menuAfterDelete =
      typeof st.menuVisible === 'boolean' ? st.menuVisible : st.menuVisible?.value;
    expect(menuAfterDelete).toBe(true);

    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    expect(m.q('[role="dialog"]')).toBeTruthy();

    const saveBtn = m.qa('button').find(b => (b.textContent || '').trim() === 'Save');
    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), '');
    await nextTick();
    (saveBtn as HTMLElement).click();
    await nextTick();
    expect(msgWarning.calls.length).toBeGreaterThan(0);
    expect(savedFiltersApi.saveCurrent.calls.length).toBe(0);

    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'NewFav');
    await nextTick();
    (saveBtn as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.saveCurrent.calls[0]![0]).toEqual({
      name: 'NewFav',
      isDefault: false,
      shared: false,
    });
    expect((m.emitted['defaults-ready'] || []).length).toBe(3);
    m.unmount();
  });

  test('cancels favorite delete when confirm dialog dismisses', async () => {
    autoConfirmChoice = 'dismiss';
    msgError.mockClear();
    msgSuccess.mockClear();
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-cancel',
        name: 'KeepMe',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
      },
    ];
    const m = mountSearch();
    await flushPromises();
    const st = m.setupState() as { menuVisible?: { value: boolean } | boolean };
    if (typeof st.menuVisible === 'boolean') st.menuVisible = true;
    else if (st.menuVisible) st.menuVisible.value = true;
    await flushPromises();
    (m.q('.choy-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.remove.calls.length).toBe(0);
    expect(msgSuccess.calls.length).toBe(0);
    expect(msgError.calls.length).toBe(0);
    const menuAfterCancel =
      typeof st.menuVisible === 'boolean' ? st.menuVisible : st.menuVisible?.value;
    expect(menuAfterCancel).toBe(true);
    m.unmount();
  });

  test('shows ChoyMessage.error when remove or save fails', async () => {
    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {
      throw new Error('delete failed');
    });
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-err',
        name: 'Bad',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
      },
    ];
    const m = mountSearch();
    await flushPromises();
    (m.q('.choy-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['delete failed']);

    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {});
    savedFiltersApi.saveCurrent.mockImplementation(async () => {
      throw 'save blew up';
    });
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'FailFav');
    await nextTick();
    const saveBtn = m.qa('button').find(b => (b.textContent || '').trim() === 'Save');
    (saveBtn as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['save blew up']);
    m.unmount();
  });

  test('stringifies non-Error remove failures and Error save failures', async () => {
    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {
      throw 'delete-string';
    });
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-str',
        name: 'Str',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
      },
    ];
    const m = mountSearch();
    await flushPromises();
    (m.q('.choy-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['delete-string']);

    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {});
    savedFiltersApi.saveCurrent.mockImplementation(async () => {
      throw new Error('save failed');
    });
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'ErrFav');
    await nextTick();
    const saveBtn = m.qa('button').find(b => (b.textContent || '').trim() === 'Save');
    (saveBtn as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['save failed']);
    m.unmount();
  });

  test('guards re-entrant save while saveFavoriteSaving is true', async () => {
    let resolveSave!: (v: any) => void;
    savedFiltersApi.saveCurrent.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveSave = resolve;
        })
    );
    const m = mountSearch();
    await flushPromises();
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'Once');
    await nextTick();
    const saveBtn = m.qa('button').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement;
    saveBtn.click();
    saveBtn.click();
    await nextTick();
    expect(savedFiltersApi.saveCurrent.calls.length).toBe(1);
    resolveSave!({ Id: '1' });
    await flushPromises();
    m.unmount();
  });

  test('emits query-update for groupby-only favorites and shows star facet', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-groupby',
        name: 'ByYear',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
        groups: [{ field: 'CreatedAt', granularity: 'year' }],
      },
    ];
    const m = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }],
    });
    await flushPromises();
    // Override after mount: factory wires applyNamedFilter during setup.
    savedFiltersApi.apply.mockImplementation(() => {
      /* no-op: filters length stays the same (groupby-only favorite) */
    });
    const beforeEmit = (m.emitted['query-update'] || []).length;
    const applyBtn = btnByText(m, 'ByYear');
    (applyBtn as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.apply.calls.length).toBeGreaterThan(0);
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(beforeEmit);
    const last = (m.emitted['query-update'] || []).at(-1)?.[0] as {
      appliedGroups?: Array<{ field?: string; granularity?: string }>;
    };
    // Merge: keep Status, add/overwrite CreatedAt from favorite.
    const groupFields = (last?.appliedGroups || []).map(g =>
      g.granularity ? `${g.field}:${g.granularity}` : String(g.field)
    );
    expect(groupFields).toContain('Status');
    expect(groupFields).toContain('CreatedAt:year');
    expect(m.q('[data-testid=choy-search-favorite-tag]')?.textContent || '').toContain('ByYear');
    // Star folds favorite groups; only user-stacked Status remains as a group tag.
    const groupTagText = m.q('.choy-search__grouptag')?.textContent || '';
    expect(groupTagText).toContain('Status');
    expect(groupTagText).not.toContain('CreatedAt');
    m.unmount();
  });

  test('favorite alone shows only the star tag (Odoo-style)', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-only',
        name: 'Module List',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: { And: [['Active', '=', true]] },
        groups: [{ field: 'CreatedAt', granularity: 'year' }],
      },
    ];
    const m = mountSearch({ currentAppliedGroups: [] });
    await flushPromises();
    savedFiltersApi.apply.mockImplementation((fav: { name: string; filter: any }) => {
      const st = m.setupState() as { filters?: any[] | { value: any[] }; applyNamedFilter?: (nf: any) => void };
      if (typeof st.applyNamedFilter === 'function') {
        st.applyNamedFilter({ name: fav.name, query: fav.filter });
      } else {
        const list = Array.isArray(st.filters) ? st.filters : st.filters?.value;
        list?.push({
          id: 'fav-cond',
          name: fav.name,
          logic: 'And',
          children: [{ id: 'c1', field: 'Active', operator: '=', value: true }],
        });
      }
    });
    (btnByText(m, 'Module List') as HTMLElement).click();
    await flushPromises();
    // Parent would pass applied groups back; simulate controlled prop update.
    (m.props as any).currentAppliedGroups = [{ field: 'CreatedAt', granularity: 'year' }];
    await nextTick();
    expect(m.q('[data-testid=choy-search-favorite-tag]')?.textContent || '').toContain('Module List');
    expect(m.q('.choy-search__grouptag')).toBeFalsy();
    // Favorite condition chip is folded into the star (not a separate funnel tag).
    const funnelTags = m.qa('.choy-search__tag').filter(
      el =>
        !(el as HTMLElement).classList.contains('choy-search__grouptag') &&
        !(el as HTMLElement).classList.contains('choy-search__favtag')
    );
    expect(funnelTags.some(el => (el.textContent || '').includes('Module List'))).toBe(false);
    m.unmount();
  });

  test('stacks filters after favorite and clears favorite without wiping extras', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-a',
        name: 'ByYear',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: { And: [['Active', '=', true]] },
        groups: [{ field: 'CreatedAt', granularity: 'year' }],
      },
      {
        id: 'fav-b',
        name: 'ByName',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
        groups: [{ field: 'Name' }],
      },
    ];
    const m = mountSearch({ currentAppliedGroups: [{ field: 'Status' }] });
    await flushPromises();

    (btnByText(m, 'ByYear') as HTMLElement).click();
    await flushPromises();
    expect(m.q('[data-testid=choy-search-favorite-tag]')?.textContent || '').toContain('ByYear');

    // Extra funnel chip after favorite (simulates custom filter / named filter).
    const st = m.setupState() as {
      filters?: any[] | { value: any[] };
      applyNamedFilter?: (nf: any) => void;
    };
    const filterList = (): any[] => {
      const f = st.filters as any;
      return Array.isArray(f) ? f : f?.value || [];
    };
    if (typeof st.applyNamedFilter === 'function') {
      st.applyNamedFilter({ name: 'Extra', query: ['Code', '=', 'x'] });
    } else {
      filterList().push({
        id: 'extra-1',
        name: 'Extra',
        logic: 'And',
        children: [{ id: 'c-x', field: 'Code', operator: '=', value: 'x' }],
      });
    }
    await nextTick();
    expect(filterList().some((f: any) => f.name === 'Extra')).toBe(true);
    expect(filterList().some((f: any) => f.name === 'ByYear')).toBe(true);

    // Switch favorite: replace star + its groups/condition; keep Extra.
    const stMenu = m.setupState() as { menuVisible?: { value: boolean } | boolean };
    if (typeof stMenu.menuVisible === 'boolean') stMenu.menuVisible = true;
    else if (stMenu.menuVisible) stMenu.menuVisible.value = true;
    await flushPromises();
    (btnByText(m, 'ByName') as HTMLElement).click();
    await flushPromises();
    expect(m.q('[data-testid=choy-search-favorite-tag]')?.textContent || '').toContain('ByName');
    expect(filterList().some((f: any) => f.name === 'Extra')).toBe(true);
    expect(filterList().some((f: any) => f.name === 'ByYear')).toBe(false);
    const afterSwitch = (m.emitted['query-update'] || []).at(-1)?.[0] as {
      appliedGroups?: Array<{ field?: string; granularity?: string }>;
    };
    const switchFields = (afterSwitch?.appliedGroups || []).map(g => String(g.field));
    expect(switchFields).toContain('Status');
    expect(switchFields).toContain('Name');
    expect(switchFields).not.toContain('CreatedAt');

    // Clear star: drop ByName contribution; keep Extra + Status.
    (m.q('[data-testid=choy-search-favorite-clear]') as HTMLElement).click();
    await flushPromises();
    expect(m.q('[data-testid=choy-search-favorite-tag]')).toBeFalsy();
    expect(filterList().some((f: any) => f.name === 'Extra')).toBe(true);
    const afterClear = (m.emitted['query-update'] || []).at(-1)?.[0] as {
      appliedGroups?: Array<{ field?: string; granularity?: string }>;
    };
    const clearFields = (afterClear?.appliedGroups || []).map(g => String(g.field));
    expect(clearFields).toContain('Status');
    expect(clearFields).not.toContain('Name');
    m.unmount();
  });

  function favCheckboxRoot(labelEl: Element): HTMLElement {
    return labelEl.querySelector('[data-slot="checkbox"]') as HTMLElement;
  }

  function favCheckboxOn(labelEl: Element): boolean {
    const el = favCheckboxRoot(labelEl);
    return (
      el?.getAttribute('data-state') === 'checked' ||
      el?.getAttribute('aria-checked') === 'true'
    );
  }

  async function setFavCheckbox(labelEl: Element, on: boolean): Promise<void> {
    if (favCheckboxOn(labelEl) === on) return;
    favCheckboxRoot(labelEl).click();
    await nextTick();
  }

  test('saves with isDefault and shared checkboxes enabled', async () => {
    const m = mountSearch();
    await flushPromises();
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'DefaultOnly');
    await nextTick();
    const defaultCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Use by default'));
    expect(defaultCheck).toBeTruthy();
    await setFavCheckbox(defaultCheck!, true);
    const saveBtn = m.qa('button').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement;
    saveBtn.click();
    await flushPromises();
    expect(savedFiltersApi.saveCurrent.calls[0]![0]).toEqual({
      name: 'DefaultOnly',
      isDefault: true,
      shared: false,
    });

    savedFiltersApi.saveCurrent.mockClear();
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'SharedOnly');
    await nextTick();
    const sharedCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Share with all users'));
    expect(sharedCheck).toBeTruthy();
    await setFavCheckbox(sharedCheck!, true);
    (m.qa('button').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.saveCurrent.calls[0]![0]).toEqual({
      name: 'SharedOnly',
      isDefault: false,
      shared: true,
    });
    m.unmount();
  });

  test('edits favorite metadata via dialog without create', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-edit',
        name: 'Company Management',
        shared: true,
        isDefault: true,
        canDelete: true,
        filter: { And: [['X', '=', 1]] },
      },
      {
        id: 'fav-readonly',
        name: 'Others',
        shared: true,
        isDefault: false,
        canDelete: false,
        filter: {},
      },
    ];
    const m = mountSearch();
    await flushPromises();
    expect(m.qa('.choy-search__menu-item-edit').length).toBe(1);
    expect(m.qa('.choy-search__menu-item-delete').length).toBe(1);

    (m.q('.choy-search__menu-item-edit') as HTMLElement).click();
    await nextTick();
    const dialog = m.q('[role="dialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog!.querySelector('[data-title]')?.getAttribute('data-title')).toBe('Edit favorite');
    expect((((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null) as HTMLInputElement).value).toBe('Company Management');
    const defaultCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Use by default'));
    const sharedCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Share with all users'));
    expect(favCheckboxOn(defaultCheck!)).toBe(true);
    expect(favCheckboxOn(sharedCheck!)).toBe(true);

    setInputValue(((m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null), 'Renamed Fav');
    await nextTick();
    await setFavCheckbox(sharedCheck!, false);
    const beforeReady = (m.emitted['defaults-ready'] || []).length;
    (m.qa('button').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.updateMeta.calls[0]).toEqual([
      'fav-edit',
      { name: 'Renamed Fav', isDefault: true, shared: false },
    ]);
    expect(savedFiltersApi.saveCurrent.calls.length).toBe(0);
    expect(msgSuccess.calls[0]).toEqual(['Favorite updated']);
    expect((m.emitted['defaults-ready'] || []).length).toBe(beforeReady + 1);
    m.unmount();
  });

  test('Clear all resets keyword, filters, and grouping', async () => {
    const m = mountSearch({
      currentKeyword: 'alpha',
      currentAppliedFilters: [
        {
          id: 'f1',
          name: 'Active',
          logic: 'AND',
          conditions: [{ field: 'Active', operator: '=', value: true }],
        },
      ] as any,
      currentAppliedGroups: [{ field: 'Name' }] as any,
    });
    await flushPromises();
    const clearBtn = m.q('[data-testid=choy-search-clear-all]') as HTMLElement | null;
    expect(clearBtn).not.toBeNull();
    const before = (m.emitted['query-update'] || []).length;
    clearBtn!.click();
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(before);
    const updates = m.emitted['query-update'] || [];
    const last = updates[updates.length - 1]?.[0] as any;
    expect(last?.keyword == null || last?.keyword === '').toBe(true);
    expect(last?.appliedFilters).toEqual([]);
    expect(last?.appliedGroups).toEqual([]);
    m.unmount();
  });
});
