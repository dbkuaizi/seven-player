<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: Number, default: 20 },
  items: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue'])

const selectedTitle = computed(() => {
  const selected = props.items.find((item) => Number(item.value) === Number(props.modelValue))
  return selected?.title || `${props.modelValue} / 页`
})

function selectValue(value) {
  emit('update:modelValue', Number(value))
}
</script>

<template>
  <v-menu location="top" offset="4">
    <template #activator="{ props: menuProps }">
      <v-btn
        v-bind="menuProps"
        class="page-size-menu-button"
        variant="text"
        size="small"
        aria-label="选择每页数量"
      >
        <span class="page-size-menu-content">
          <span class="page-size-menu-text">{{ selectedTitle }}</span>
          <v-icon size="18" icon="mdi-chevron-down" />
        </span>
      </v-btn>
    </template>

    <v-list class="page-size-menu-list" density="compact" min-width="112">
      <v-list-item
        v-for="item in props.items"
        :key="item.value"
        :active="Number(item.value) === Number(props.modelValue)"
        @click="selectValue(item.value)"
      >
        <v-list-item-title>{{ item.title }}</v-list-item-title>
      </v-list-item>
    </v-list>
  </v-menu>
</template>
