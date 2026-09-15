<template>
  <button
    type="button"
    class="theme-toggle"
    :title="buttonTitle"
    :aria-label="buttonTitle"
    @click="themeStore.toggleTheme()"
  >
    <span :class="isDark ? 'i-lucide-moon icon-md' : 'i-lucide-sun icon-md'" aria-hidden="true" />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useThemeStore } from '../../stores/Theme'

const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDark)

const buttonTitle = computed(() =>
  isDark.value ? '切换到浅色模式' : '切换到深色模式'
)
</script>
