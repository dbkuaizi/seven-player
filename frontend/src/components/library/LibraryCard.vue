<script setup>
import { buildLibraryImageStyle } from '../../utils/libraryAssets'

defineProps({
  item: {
    type: Object,
    required: true,
  },
  showProgress: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['select'])
</script>

<template>
  <article
    class="library-card"
    role="button"
    tabindex="0"
    @click="$emit('select', item)"
    @keydown.enter.prevent="$emit('select', item)"
    @keydown.space.prevent="$emit('select', item)"
  >
    <div
      class="library-poster"
      :class="`library-poster--${item.posterTone}`"
      :style="buildLibraryImageStyle(item.posterUrl)"
    >
      <div class="library-poster-vignette" />
      <div class="library-poster-hover">
        <v-icon icon="mdi-information-outline" size="22" />
      </div>
      <div class="library-poster-rating">{{ item.rating.toFixed(1) }}</div>
    </div>

    <div class="library-card-body">
      <div class="library-card-title">{{ item.title }}</div>
      <div class="library-card-meta">{{ item.duration }}</div>
      <v-progress-linear
        v-if="showProgress && item.progress > 0"
        class="library-progress"
        :model-value="item.progress"
        color="primary"
        height="3"
        rounded
      />
    </div>
  </article>
</template>

<style scoped>
.library-card {
  min-width: 0;
  cursor: pointer;
  outline: none;
  border-radius: 10px;
  transition:
    transform 0.16s ease,
    filter 0.16s ease;
}

.library-card:hover,
.library-card:focus-visible {
  transform: translateY(-3px);
}

.library-card:focus-visible {
  box-shadow: 0 0 0 3px rgba(var(--v-theme-primary), 0.24);
}

.library-poster {
  position: relative;
  aspect-ratio: 2 / 3;
  overflow: hidden;
  border-radius: 8px;
  background-size: cover;
  background-position: center;
  background:
    radial-gradient(circle at 22% 12%, rgba(255, 255, 255, 0.38), transparent 24%),
    linear-gradient(145deg, #334155, #111827);
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.18);
  transition:
    box-shadow 0.16s ease,
    transform 0.16s ease;
}

.library-card:hover .library-poster,
.library-card:focus-visible .library-poster {
  box-shadow: 0 16px 34px rgba(15, 23, 42, 0.28);
}

.library-poster--amber { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.38), transparent 24%), linear-gradient(145deg, #f59e0b, #78350f); }
.library-poster--steel { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.32), transparent 24%), linear-gradient(145deg, #64748b, #020617); }
.library-poster--red { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.34), transparent 24%), linear-gradient(145deg, #ef4444, #450a0a); }
.library-poster--rose { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.34), transparent 24%), linear-gradient(145deg, #fb7185, #4c0519); }
.library-poster--green { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.30), transparent 24%), linear-gradient(145deg, #16a34a, #052e16); }
.library-poster--violet { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.34), transparent 24%), linear-gradient(145deg, #8b5cf6, #2e1065); }
.library-poster--mint { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.38), transparent 24%), linear-gradient(145deg, #5eead4, #134e4a); }
.library-poster--sky { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.38), transparent 24%), linear-gradient(145deg, #38bdf8, #082f49); }
.library-poster--ink { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.22), transparent 24%), linear-gradient(145deg, #475569, #0f172a); }
.library-poster--ocean { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.36), transparent 24%), linear-gradient(145deg, #0ea5e9, #064e3b); }
.library-poster--cyan { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.36), transparent 24%), linear-gradient(145deg, #22d3ee, #164e63); }
.library-poster--paper { background: radial-gradient(circle at 20% 12%, rgba(255,255,255,.42), transparent 24%), linear-gradient(145deg, #d6d3d1, #57534e); }

.library-poster[style] {
  background-size: cover;
  background-position: center;
}

.library-poster-vignette {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 68% 22%, rgba(255, 255, 255, 0.16), transparent 22%),
    linear-gradient(to top, rgba(0, 0, 0, 0.78), transparent 45%),
    linear-gradient(160deg, rgba(255, 255, 255, 0.12), transparent 38%);
}

.library-poster-rating {
  position: absolute;
  z-index: 2;
  right: 9px;
  bottom: 8px;
  color: #fff;
  font-size: 16px;
  line-height: 1;
  font-weight: 800;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.78);
}

.library-poster-hover {
  position: absolute;
  inset: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: rgba(0, 0, 0, 0.18);
  opacity: 0;
  transition: opacity 0.16s ease;
}

.library-card:hover .library-poster-hover,
.library-card:focus-visible .library-poster-hover {
  opacity: 1;
}

.library-card-body {
  padding-top: 9px;
  min-width: 0;
}

.library-card-title {
  font-size: 14px;
  line-height: 1.18;
  font-weight: 750;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.library-card-meta {
  margin-top: 5px;
  color: rgba(var(--v-theme-on-surface), 0.58);
  font-size: 12px;
  line-height: 1.15;
  white-space: nowrap;
}

.library-progress {
  margin-top: 7px;
}
</style>
