<script setup>
const props = defineProps({
  notice: { type: Object, required: true },
})

const emit = defineEmits(['close'])
</script>

<template>
  <v-snackbar
    :key="props.notice.key"
    :model-value="props.notice.show"
    class="notice-snackbar"
    :color="props.notice.color"
    :timeout="props.notice.timeout"
    location="bottom end"
    timer="white"
    variant="flat"
    absolute
    contained
    rounded="lg"
    max-width="420"
    @update:model-value="(value) => !value && emit('close')"
  >
    {{ props.notice.text }}

    <template #actions>
      <v-btn
        icon="mdi-close"
        size="x-small"
        variant="text"
        @click="emit('close')"
      />
    </template>
  </v-snackbar>
</template>
