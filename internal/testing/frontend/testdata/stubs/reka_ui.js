// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `reka-ui`.
 * Real Reka primitives pull in browser-only APIs that throw in QuickJS
 * (`TypeError: not a function`). Passthrough slot hosts keep Dialog / Toast /
 * Combobox / Calendar / Sidebar mounts usable in FE unit tests.
 */
import { computed, defineComponent, h, inject, provide, ref } from 'vue';

function stub(name) {
  return defineComponent({
    name,
    inheritAttrs: false,
    setup(_, { slots, attrs }) {
      return () =>
        h(
          'div',
          { 'data-reka-stub': name, ...attrs },
          [
            slots.default ? slots.default() : null,
            slots.trigger ? slots.trigger() : null,
            slots.content ? slots.content() : null,
          ],
        );
    },
  });
}

/** Minimal createContext used by Sidebar / Command providers. */
export function createContext(name) {
  const key = Symbol(String(name || 'reka-context'));
  function useContext(fallible) {
    const value = inject(key, undefined);
    if (value === undefined && !fallible) {
      throw new Error(`reka-ui stub: missing context ${String(name)}`);
    }
    return value;
  }
  function provideContext(value) {
    provide(key, value);
  }
  return [useContext, provideContext];
}

export function useForwardPropsEmits(props) {
  return props;
}

export function useForwardProps(props) {
  return props;
}

export function useId() {
  return `reka-stub-${Math.random().toString(36).slice(2, 9)}`;
}

export function useFilter() {
  return {
    contains: (value, search) =>
      !search || String(value || '').toLowerCase().includes(String(search).toLowerCase()),
  };
}

export const CalendarCell = stub('CalendarCell');
export const CalendarCellTrigger = stub('CalendarCellTrigger');
export const CalendarGrid = stub('CalendarGrid');
export const CalendarGridBody = stub('CalendarGridBody');
export const CalendarGridHead = stub('CalendarGridHead');
export const CalendarGridRow = stub('CalendarGridRow');
export const CalendarHeadCell = stub('CalendarHeadCell');
export const CalendarHeader = stub('CalendarHeader');
export const CalendarHeading = stub('CalendarHeading');
export const CalendarNext = stub('CalendarNext');
export const CalendarPrev = stub('CalendarPrev');
export const CalendarRoot = stub('CalendarRoot');
export const CheckboxIndicator = stub('CheckboxIndicator');
/** Toggleable checkbox host so kit Checkbox v-model works under FE-QJS. */
export const CheckboxRoot = defineComponent({
  name: 'CheckboxRoot',
  inheritAttrs: false,
  props: {
    modelValue: { type: [Boolean, String], default: false },
    checked: { type: [Boolean, String], default: undefined },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:modelValue', 'update:checked'],
  setup(props, { slots, attrs, emit }) {
    const on = computed(() => {
      const v = props.checked !== undefined ? props.checked : props.modelValue;
      return v === true || v === 'indeterminate';
    });
    function toggle() {
      if (props.disabled) return;
      const next = !on.value;
      emit('update:modelValue', next);
      emit('update:checked', next);
    }
    return () =>
      h(
        'button',
        {
          type: 'button',
          ...attrs,
          'data-reka-stub': 'CheckboxRoot',
          'data-state': on.value ? 'checked' : 'unchecked',
          'aria-checked': on.value ? 'true' : 'false',
          disabled: props.disabled || undefined,
          onClick: (e) => {
            const prev = attrs.onClick;
            if (typeof prev === 'function') prev(e);
            toggle();
          },
        },
        slots.default ? slots.default() : null,
      );
  },
});
export const ComboboxAnchor = stub('ComboboxAnchor');
export const ComboboxContent = stub('ComboboxContent');
export const ComboboxEmpty = stub('ComboboxEmpty');
export const ComboboxInput = stub('ComboboxInput');
export const ComboboxItem = stub('ComboboxItem');
export const ComboboxPortal = stub('ComboboxPortal');
export const ComboboxRoot = stub('ComboboxRoot');
export const ComboboxViewport = stub('ComboboxViewport');
export const DialogClose = stub('DialogClose');
export const DialogContent = stub('DialogContent');
export const DialogDescription = stub('DialogDescription');
export const DialogOverlay = stub('DialogOverlay');
export const DialogPortal = stub('DialogPortal');
export const DialogRoot = defineComponent({
  name: 'DialogRoot',
  inheritAttrs: false,
  props: {
    open: { type: Boolean, default: undefined },
  },
  emits: ['update:open'],
  setup(props, { slots, attrs, emit }) {
    return () =>
      h(
        'div',
        {
          'data-reka-stub': 'DialogRoot',
          // Treat undefined as open so legacy mounts that omit v-model still render.
          'data-state': props.open === false ? 'closed' : 'open',
          ...attrs,
        },
        slots.default
          ? slots.default({
              open: props.open !== false,
              close: () => emit('update:open', false),
            })
          : null,
      );
  },
});
export const DialogTitle = stub('DialogTitle');
export const DialogTrigger = stub('DialogTrigger');
export const DropdownMenuContent = stub('DropdownMenuContent');
export const DropdownMenuGroup = stub('DropdownMenuGroup');
export const DropdownMenuItem = stub('DropdownMenuItem');
export const DropdownMenuLabel = stub('DropdownMenuLabel');
export const DropdownMenuPortal = stub('DropdownMenuPortal');
export const DropdownMenuRoot = stub('DropdownMenuRoot');
export const DropdownMenuSeparator = stub('DropdownMenuSeparator');
export const DropdownMenuSub = stub('DropdownMenuSub');
export const DropdownMenuSubContent = stub('DropdownMenuSubContent');
export const DropdownMenuSubTrigger = stub('DropdownMenuSubTrigger');
export const DropdownMenuTrigger = stub('DropdownMenuTrigger');
export const NavigationMenuContent = stub('NavigationMenuContent');
export const NavigationMenuIndicator = stub('NavigationMenuIndicator');
export const NavigationMenuItem = stub('NavigationMenuItem');
export const NavigationMenuLink = defineComponent({
  name: 'NavigationMenuLink',
  inheritAttrs: false,
  props: {
    href: { type: String, default: undefined },
    active: { type: Boolean, default: undefined },
    asChild: { type: Boolean, default: false },
  },
  setup(props, { slots, attrs }) {
    return () =>
      h(
        props.href ? 'a' : 'div',
        {
          'data-reka-stub': 'NavigationMenuLink',
          href: props.href,
          ...attrs,
        },
        slots.default ? slots.default() : null,
      );
  },
});
export const NavigationMenuList = stub('NavigationMenuList');
export const NavigationMenuRoot = stub('NavigationMenuRoot');
export const NavigationMenuTrigger = stub('NavigationMenuTrigger');
export const NavigationMenuViewport = stub('NavigationMenuViewport');
export const PopoverAnchor = stub('PopoverAnchor');
export const PopoverContent = stub('PopoverContent');
export const PopoverPortal = stub('PopoverPortal');
export const PopoverRoot = defineComponent({
  name: 'PopoverRoot',
  inheritAttrs: false,
  props: {
    open: { type: Boolean, default: undefined },
  },
  emits: ['update:open'],
  setup(props, { slots, attrs }) {
    return () =>
      h(
        'div',
        {
          'data-reka-stub': 'PopoverRoot',
          'data-state': props.open ? 'open' : 'closed',
          ...attrs,
        },
        slots.default ? slots.default() : null,
      );
  },
});
export const PopoverTrigger = stub('PopoverTrigger');
/** Single/multiple toggle group host used by SearchFilter AND/OR. */
export const ToggleGroupRoot = defineComponent({
  name: 'ToggleGroupRoot',
  inheritAttrs: false,
  props: {
    modelValue: { default: undefined },
    type: { type: String, default: 'single' },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { slots, emit, attrs }) {
    return () =>
      h(
        'div',
        {
          'data-reka-stub': 'ToggleGroupRoot',
          'data-state': props.modelValue != null ? 'on' : 'off',
          ...attrs,
        },
        slots.default ? slots.default() : null,
      );
  },
});
export const ToggleGroupItem = defineComponent({
  name: 'ToggleGroupItem',
  inheritAttrs: false,
  props: {
    value: { type: String, required: true },
    disabled: { type: Boolean, default: false },
  },
  setup(props, { slots, attrs }) {
    return () =>
      h(
        'button',
        {
          type: 'button',
          'data-reka-stub': 'ToggleGroupItem',
          'data-value': props.value,
          disabled: props.disabled || undefined,
          ...attrs,
        },
        slots.default ? slots.default() : null,
      );
  },
});
export const ScrollAreaRoot = stub('ScrollAreaRoot');
export const ScrollAreaScrollbar = stub('ScrollAreaScrollbar');
export const ScrollAreaThumb = stub('ScrollAreaThumb');
export const ScrollAreaViewport = stub('ScrollAreaViewport');
export const SelectContent = stub('SelectContent');
export const SelectIcon = stub('SelectIcon');
export const SelectItem = stub('SelectItem');
export const SelectItemIndicator = stub('SelectItemIndicator');
export const SelectItemText = stub('SelectItemText');
export const SelectPortal = stub('SelectPortal');
export const SelectRoot = stub('SelectRoot');
export const SelectTrigger = stub('SelectTrigger');
export const SelectValue = stub('SelectValue');
export const SelectViewport = stub('SelectViewport');
export const SwitchRoot = stub('SwitchRoot');
export const SwitchThumb = stub('SwitchThumb');
export const TabsContent = stub('TabsContent');
export const TabsList = stub('TabsList');
export const TabsRoot = stub('TabsRoot');
export const TabsTrigger = stub('TabsTrigger');
export const ToastClose = stub('ToastClose');
export const ToastDescription = stub('ToastDescription');
export const ToastProvider = stub('ToastProvider');
export const ToastRoot = stub('ToastRoot');
export const ToastTitle = stub('ToastTitle');
export const ToastViewport = stub('ToastViewport');
export const TooltipContent = stub('TooltipContent');
export const TooltipPortal = stub('TooltipPortal');
export const TooltipProvider = stub('TooltipProvider');
export const TooltipRoot = stub('TooltipRoot');
export const TooltipTrigger = stub('TooltipTrigger');
export const AvatarRoot = stub('AvatarRoot');
export const AvatarFallback = stub('AvatarFallback');
export const AvatarImage = stub('AvatarImage');
// Vendor Avatar.vue imports AvatarRoot; keep a plain Avatar alias for other importers.
export const Avatar = AvatarRoot;

const COLLAPSIBLE_OPEN = Symbol('reka-collapsible-open');

export const CollapsibleRoot = defineComponent({
  name: 'CollapsibleRoot',
  inheritAttrs: false,
  props: {
    open: { type: Boolean, default: undefined },
    defaultOpen: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
  },
  emits: ['update:open'],
  setup(props, { slots, attrs, emit }) {
    const uncontrolled = ref(!!props.defaultOpen);
    const open = computed({
      get: () => (props.open === undefined ? uncontrolled.value : !!props.open),
      set: (value) => {
        uncontrolled.value = value;
        emit('update:open', value);
      },
    });
    provide(COLLAPSIBLE_OPEN, open);
    return () =>
      h(
        'div',
        {
          'data-reka-stub': 'CollapsibleRoot',
          'data-state': open.value ? 'open' : 'closed',
          ...attrs,
        },
        slots.default ? slots.default({ open: open.value }) : null,
      );
  },
});

export const CollapsibleContent = defineComponent({
  name: 'CollapsibleContent',
  inheritAttrs: false,
  setup(_, { slots, attrs }) {
    const open = inject(COLLAPSIBLE_OPEN, ref(true));
    return () =>
      open.value
        ? h('div', { 'data-reka-stub': 'CollapsibleContent', ...attrs }, slots.default ? slots.default() : null)
        : null;
  },
});

export const CollapsibleTrigger = defineComponent({
  name: 'CollapsibleTrigger',
  inheritAttrs: false,
  props: {
    asChild: { type: Boolean, default: false },
  },
  setup(props, { slots, attrs }) {
    const open = inject(COLLAPSIBLE_OPEN, ref(false));
    function toggle() {
      open.value = !open.value;
    }
    return () => {
      if (props.asChild) {
        return h(
          'div',
          {
            'data-reka-stub': 'CollapsibleTrigger',
            onClick: toggle,
          },
          slots.default ? slots.default() : null,
        );
      }
      return h(
        'button',
        {
          type: 'button',
          'data-reka-stub': 'CollapsibleTrigger',
          'aria-expanded': open.value ? 'true' : 'false',
          ...attrs,
          onClick: toggle,
        },
        slots.default ? slots.default() : null,
      );
    };
  },
});

export const Primitive = defineComponent({
  name: 'Primitive',
  inheritAttrs: false,
  props: {
    as: { type: [String, Object], default: 'div' },
    asChild: { type: Boolean, default: false },
  },
  setup(props, { slots, attrs }) {
    return () => {
      if (props.asChild) {
        return slots.default ? slots.default() : null;
      }
      const tag = typeof props.as === 'string' ? props.as : 'div';
      return h(tag, { 'data-reka-stub': 'Primitive', ...attrs }, slots.default ? slots.default() : null);
    };
  },
});
export const Separator = stub('Separator');
export const ListboxRoot = stub('ListboxRoot');
export const ListboxContent = stub('ListboxContent');
export const ListboxFilter = stub('ListboxFilter');
export const ListboxGroup = stub('ListboxGroup');
export const ListboxGroupLabel = stub('ListboxGroupLabel');
export const ListboxItem = stub('ListboxItem');

/** Native <label> so FormItem `for` association works in FE-QJS (generic stub is a div). */
export const Label = defineComponent({
  name: 'Label',
  inheritAttrs: false,
  setup(_, { slots, attrs }) {
    return () =>
      h('label', { 'data-reka-stub': 'Label', ...attrs }, [
        slots.default ? slots.default() : null,
      ]);
  },
});
export const AspectRatio = stub('AspectRatio');

// AlertDialog: mirror DialogRoot open/closed so ConfirmHost mounts in FE unit.
export const AlertDialogRoot = DialogRoot;
export const AlertDialogPortal = DialogPortal;
export const AlertDialogOverlay = DialogOverlay;
export const AlertDialogContent = DialogContent;
export const AlertDialogTitle = DialogTitle;
export const AlertDialogDescription = DialogDescription;
export const AlertDialogTrigger = DialogTrigger;
export const AlertDialogAction = stub('AlertDialogAction');
export const AlertDialogCancel = stub('AlertDialogCancel');
