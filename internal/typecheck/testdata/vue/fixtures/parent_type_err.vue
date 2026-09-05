<script setup lang="ts">
import Child from "./Child.vue";

const wrong: number = Child as any as number;
void wrong;
</script>

<template>
  <Child />
</template>
