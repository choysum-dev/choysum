/// <reference path="/choysum-vue-virtual/types/template-helpers.d.ts" />
/// <reference path="/choysum-vue-virtual/types/props-fallback.d.ts" />

import Child from "./Child.vue";

const wrong: number = Child as any as number;
void wrong;
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
const __VLS_ctx = {} as import('vue').ComponentPublicInstance;
type __VLS_LocalComponents = {};
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = {};
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
const __VLS_0 = Child;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
}));
const __VLS_2 = __VLS_1({
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5!: Parameters<NonNullable<typeof __VLS_3['expose']>>[0];
var __VLS_3!: __VLS_ExtractComponentContext<typeof __VLS_0, typeof __VLS_2>;
type __VLS_RootEl = 
| NonNullable<typeof __VLS_5>['$el'];
const __VLS_export = (await import('vue')).defineComponent({
});
export default {} as typeof __VLS_export;
