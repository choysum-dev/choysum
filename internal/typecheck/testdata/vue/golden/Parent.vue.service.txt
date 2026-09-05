/// <reference path="/choysum-vue-virtual/types/template-helpers.d.ts" />
/// <reference path="/choysum-vue-virtual/types/props-fallback.d.ts" />

import Child from "./Child.vue";

const child: typeof Child = Child;
void child;
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
const __VLS_ctx = {} as import('vue').ComponentPublicInstance;
type __VLS_LocalComponents = {};
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = {};
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
});
type __VLS_RootEl = 
| __VLS_Elements['div'];
const __VLS_export = (await import('vue')).defineComponent({
});
export default {} as typeof __VLS_export;
