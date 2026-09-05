/// <reference path="/choysum-vue-virtual/types/template-helpers.d.ts" />
/// <reference path="/choysum-vue-virtual/types/props-fallback.d.ts" />

import { defineComponent } from "vue";

export default {} as typeof __VLS_export;;
const __VLS_ctx = {} as InstanceType<__VLS_PickNotAny<typeof __VLS_export, new () => {}>>;
type __VLS_LocalComponents = {};
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = {};
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (__VLS_ctx.inc)},
});
( __VLS_ctx.count );
type __VLS_RootEl = 
| __VLS_Elements['button'];
// @ts-ignore
[inc,count,];
const __VLS_export = defineComponent({
  name: "ScriptLangTs",
  data() {
    return { count: 0 as number };
  },
  methods: {
    inc(): void {
      this.count++;
    },
  },
});

