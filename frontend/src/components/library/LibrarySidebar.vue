<script setup>
defineProps({
  sections: {
    type: Array,
    default: () => [],
  },
  activeSection: {
    type: String,
    default: '',
  },
  activeChild: {
    type: String,
    default: '',
  },
  expandedSections: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits(['select-section', 'select-child', 'toggle-section'])
</script>

<template>
  <aside class="library-sidebar">
    <div
      v-for="section in sections"
      :key="section.id"
      class="library-nav-group"
    >
      <button
        type="button"
        class="library-nav-primary"
        :class="{ 'library-nav-primary--active': activeSection === section.id }"
        @click="emit('select-section', section.id)"
      >
        <span class="library-nav-icon" :style="{ color: section.color }">
          <v-icon size="18">{{ section.icon }}</v-icon>
        </span>
        <span class="library-nav-label">{{ section.label }}</span>
        <v-btn
          class="library-nav-expand"
          :icon="expandedSections[section.id] ? 'mdi-chevron-up' : 'mdi-chevron-down'"
          size="x-small"
          variant="text"
          @click.stop="emit('toggle-section', section.id)"
        />
      </button>

      <div v-if="expandedSections[section.id]" class="library-nav-children">
        <button
          v-for="child in section.children"
          :key="`${section.id}-${child}`"
          type="button"
          class="library-nav-child"
          :class="{ 'library-nav-child--active': activeSection === section.id && activeChild === child }"
          @click="emit('select-child', section.id, child)"
        >
          {{ child }}
        </button>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.library-sidebar {
  width: 178px;
  flex: 0 0 178px;
  min-height: 0;
  padding: 10px 8px;
  border-right: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  overflow-y: auto;
}

.library-nav-group + .library-nav-group {
  margin-top: 4px;
}

.library-nav-primary {
  width: 100%;
  min-height: 36px;
  padding: 0 4px 0 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: rgba(var(--v-theme-on-surface), 0.82);
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  text-align: left;
}

.library-nav-primary:hover,
.library-nav-primary--active {
  background: rgba(var(--v-theme-primary), 0.08);
  color: rgba(var(--v-theme-on-surface), 0.96);
}

.library-nav-icon {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
}

.library-nav-label {
  min-width: 0;
  flex: 1 1 auto;
  font-size: 13px;
  font-weight: 600;
}

.library-nav-expand {
  flex: 0 0 auto;
  color: rgba(var(--v-theme-on-surface), 0.58);
}

.library-nav-children {
  margin: 3px 0 7px 31px;
  display: grid;
  gap: 2px;
}

.library-nav-child {
  min-height: 28px;
  padding: 0 10px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: rgba(var(--v-theme-on-surface), 0.64);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
}

.library-nav-child:hover,
.library-nav-child--active {
  color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.08);
}
</style>
