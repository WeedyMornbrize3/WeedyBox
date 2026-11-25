<template>
  <n-button quaternary circle size="small" :title="buttonTitle" :aria-label="buttonTitle" @click="themeStore.toggleTheme()">
    <template #icon>
      <span :class="isDark ? 'i-lucide-moon' : 'i-lucide-sun'" class="icon-md" aria-hidden="true" />
    </template>
  </n-button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton } from 'naive-ui'
import { useThemeStore } from '../../stores/Theme'

const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDark)

const buttonTitle = computed(() => (isDark.value ? '切换到浅色模式' : '切换到深色模式'))
</script>
