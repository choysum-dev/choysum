<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
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
</script>

<template>
  <button @click="inc">{{ count }}</button>
</template>
