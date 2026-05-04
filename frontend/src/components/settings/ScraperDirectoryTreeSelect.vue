<script setup>
import { computed, ref, watch } from 'vue'
import { PreviewDirectory } from '../../../bindings/panplayer/app'
import {
  createDirectoryTarget,
  formatDirectoryTargetPath,
  normalizeDirectoryTargets,
  normalizeDirectoryTargetValue,
  rootBreadcrumb,
} from '../../utils/directoryTarget'
import { sanitizeErrorMessage } from '../../utils/error'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
})

const emit = defineEmits(['save'])

const menuOpen = ref(false)
const nodes = ref([createTreeNode(createDirectoryTarget('0', [rootBreadcrumb()]), 0)])
const expandedIds = ref([])
const selectedTargets = ref([])

const expandedSet = computed(() => new Set(expandedIds.value))
const selectedIds = computed(() => new Set(selectedTargets.value.map((target) => target.id)))
const selectedSummary = computed(() => {
  const count = selectedTargets.value.length
  return count ? `已选择 ${count} 个目录` : '默认扫描所有目录'
})

const visibleNodes = computed(() => {
  const result = []
  const walk = (list) => {
    for (const node of list) {
      result.push(node)
      if (expandedSet.value.has(node.id)) {
        walk(node.children)
      }
    }
  }
  walk(nodes.value)
  return result
})

watch(
  () => props.modelValue,
  (value) => {
    selectedTargets.value = normalizeDirectoryTargets(value, 50)
    seedSelectedBranches()
  },
  { deep: true, immediate: true },
)

watch(menuOpen, (open) => {
  if (!open) {
    return
  }
  expandNode(nodes.value[0])
})

function createTreeNode(target, level) {
  return {
    id: target.id,
    title: target.name,
    subtitle: formatDirectoryTargetPath(target.path),
    target,
    level,
    children: [],
    loaded: false,
    loading: false,
    error: '',
    hasChildren: true,
  }
}

function seedSelectedBranches() {
  for (const target of selectedTargets.value) {
    const path = target.path || []
    let siblings = nodes.value
    let currentPath = []

    for (const crumb of path) {
      currentPath = [...currentPath, crumb]
      const normalizedTarget = createDirectoryTarget(crumb.id, currentPath)
      let node = siblings.find((item) => item.id === normalizedTarget.id)
      if (!node) {
        node = createTreeNode(normalizedTarget, Math.max(0, currentPath.length - 1))
        siblings.push(node)
      }
      siblings = node.children
    }
  }
}

async function expandNode(node) {
  if (!node) {
    return
  }

  if (!expandedSet.value.has(node.id)) {
    expandedIds.value = [...expandedIds.value, node.id]
  }

  if (!node.loaded && !node.loading) {
    await loadNodeChildren(node)
  }
}

function toggleNodeExpanded(node) {
  if (!node) {
    return
  }

  if (expandedSet.value.has(node.id)) {
    expandedIds.value = expandedIds.value.filter((id) => id !== node.id)
    return
  }

  expandNode(node)
}

async function loadNodeChildren(node) {
  node.loading = true
  node.error = ''

  try {
    const data = await PreviewDirectory(node.id)
    const resolvedTarget = normalizeDirectoryTargetValue(data?.dirId || node.id, data?.path) || node.target
    node.target = resolvedTarget
    node.title = resolvedTarget.name
    node.subtitle = formatDirectoryTargetPath(resolvedTarget.path)

    const existing = new Map(node.children.map((child) => [child.id, child]))
    const children = []
    for (const item of data?.items || []) {
      if (!item?.isDirectory || !item.fileId) {
        continue
      }

      const childTarget = createDirectoryTarget(item.fileId, [
        ...resolvedTarget.path,
        { id: item.fileId, name: item.originalName || item.name || '未命名文件夹' },
      ])
      const child = existing.get(childTarget.id) || createTreeNode(childTarget, node.level + 1)
      child.target = childTarget
      child.title = childTarget.name
      child.subtitle = formatDirectoryTargetPath(childTarget.path)
      children.push(child)
    }

    node.children = children
    node.loaded = true
    node.hasChildren = children.length > 0
  } catch (error) {
    node.error = sanitizeErrorMessage(error?.message || error) || '目录加载失败'
  } finally {
    node.loading = false
  }
}

function toggleSelection(node, checked) {
  const next = new Map(selectedTargets.value.map((target) => [target.id, target]))
  if (checked) {
    next.set(node.id, node.target)
  } else {
    next.delete(node.id)
  }
  saveTargets([...next.values()])
}

function removeTarget(target) {
  saveTargets(selectedTargets.value.filter((item) => item.id !== target.id))
}

function clearTargets() {
  saveTargets([])
}

function saveTargets(targets) {
  selectedTargets.value = normalizeDirectoryTargets(targets, 50)
  emit('save', selectedTargets.value)
}
</script>

<template>
  <v-menu
    v-model="menuOpen"
    :close-on-content-click="false"
    location="bottom start"
    max-height="420"
    offset="6"
  >
    <template #activator="{ props: menuProps }">
      <v-text-field
        v-bind="menuProps"
        class="scraper-tree-activator mb-3"
        clearable
        density="compact"
        hide-details="auto"
        label="刮削目录"
        :model-value="selectedSummary"
        persistent-placeholder
        placeholder="不选择则默认扫描所有目录"
        prepend-inner-icon="mdi-folder-search-outline"
        readonly
        variant="outlined"
        @click:clear.stop="clearTargets"
      />
    </template>

    <v-card class="scraper-tree-menu" elevation="8">
      <div class="scraper-tree-menu-header">
        <div>
          <div class="text-body-2 font-weight-medium">选择参与刮削的目录</div>
          <div class="text-caption text-medium-emphasis">展开目录时会按需读取子文件夹。</div>
        </div>
        <v-btn
          :disabled="!selectedTargets.length"
          size="small"
          variant="text"
          @click="clearTargets"
        >
          清空
        </v-btn>
      </div>

      <div
        v-if="selectedTargets.length"
        class="scraper-tree-selected"
      >
        <v-chip
          v-for="target in selectedTargets"
          :key="target.id"
          closable
          size="x-small"
          variant="tonal"
          @click:close="removeTarget(target)"
        >
          {{ formatDirectoryTargetPath(target.path) }}
        </v-chip>
      </div>

      <v-divider />

      <div class="scraper-tree-body">
        <div
          v-for="node in visibleNodes"
          :key="node.id"
          class="scraper-tree-node"
          :class="{ 'scraper-tree-node--selected': selectedIds.has(node.id) }"
          :style="{ '--tree-level': node.level }"
          @click="toggleNodeExpanded(node)"
        >
          <v-btn
            class="scraper-tree-expand"
            :disabled="node.loaded && !node.hasChildren"
            :icon="expandedSet.has(node.id) ? 'mdi-chevron-down' : 'mdi-chevron-right'"
            :loading="node.loading"
            size="x-small"
            variant="text"
            @click.stop="toggleNodeExpanded(node)"
          />

          <v-checkbox-btn
            class="scraper-tree-check"
            color="primary"
            density="compact"
            :model-value="selectedIds.has(node.id)"
            @click.stop
            @update:model-value="toggleSelection(node, $event)"
          />

          <v-icon
            class="scraper-tree-folder"
            size="18"
          >
            {{ expandedSet.has(node.id) ? 'mdi-folder-open-outline' : 'mdi-folder-outline' }}
          </v-icon>

          <div class="scraper-tree-node-text">
            <div class="scraper-tree-node-title">{{ node.title }}</div>
            <div class="scraper-tree-node-subtitle">{{ node.subtitle }}</div>
            <div
              v-if="node.error"
              class="scraper-tree-node-error"
            >
              {{ node.error }}
            </div>
          </div>
        </div>
      </div>
    </v-card>
  </v-menu>
</template>
