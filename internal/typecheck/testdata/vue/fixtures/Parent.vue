<script setup lang="ts">
import Child from "./Child.vue";

const child: typeof Child = Child;
void child;
</script>

<template>
  <div>parent</div>
</template>
