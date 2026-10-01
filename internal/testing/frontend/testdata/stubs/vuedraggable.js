// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * FE unit stub for `vuedraggable` (Vue 3 / Sortable wrapper).
 * Renders the `item` scoped slot for each entry in `list` (or `modelValue`).
 */
import { defineComponent, h } from 'vue';

export default defineComponent({
  name: 'DraggableStub',
  props: {
    list: { type: Array, default: undefined },
    modelValue: { type: Array, default: undefined },
    itemKey: { type: [String, Function], default: 'id' },
    group: { type: [String, Object], default: undefined },
    animation: { type: Number, default: 0 },
    disabled: { type: Boolean, default: false },
    ghostClass: { type: String, default: undefined },
  },
  emits: ['change', 'update:modelValue'],
  setup(props, ctx) {
    return () => {
      const items = Array.isArray(props.list)
        ? props.list
        : Array.isArray(props.modelValue)
          ? props.modelValue
          : [];
      const children = [];
      const itemSlot = ctx.slots.item;
      if (itemSlot) {
        for (let i = 0; i < items.length; i++) {
          const element = items[i];
          const key =
            typeof props.itemKey === 'function'
              ? props.itemKey(element)
              : element != null
                ? element[props.itemKey]
                : i;
          children.push(
            h(
              'div',
              { key: key != null ? String(key) : String(i), class: 'fe-stub-draggable-item' },
              itemSlot({ element, index: i }),
            ),
          );
        }
      } else if (ctx.slots.default) {
        children.push(...ctx.slots.default());
      }
      return h('div', { class: 'fe-stub-draggable', 'data-disabled': props.disabled ? 'true' : 'false' }, children);
    };
  },
});
