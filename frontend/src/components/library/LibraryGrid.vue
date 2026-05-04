<script setup>
import LibraryCard from './LibraryCard.vue'

defineProps({
  items: {
    type: Array,
    default: () => [],
  },
  showProgress: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['select'])
</script>

<template>
  <div class="library-grid">
    <LibraryCard
      v-for="item in items"
      :key="item.id"
      :item="item"
      :show-progress="showProgress"
      @select="$emit('select', $event)"
    />
  </div>
</template>

<style scoped>
.library-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(118px, 1fr));
  gap: 18px 14px;
  align-items: start;
}
</style>
