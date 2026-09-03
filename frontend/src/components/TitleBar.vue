<template>
  <header 
    class="titlebar"
    style="--wails-draggable: drag;
      -webkit-app-region: drag;
      border-bottom: 1px solid var(--color-border);
      border-color: var(--color-border);   
    "
  >
    <div class="titlebar-left"></div>

    <div class="flex items-center gap-1" style="--wails-draggable: no-drag; -webkit-app-region: no-drag">
      <button class="titlebar-btn" @click="minimize" title="最小化">─</button>
      <button class="titlebar-btn" @click="toggleMaximize" title="最大化/还原">
        {{ isMaximised ? '⧉' : '☐' }}
      </button>
      <button class="titlebar-btn-close" @click="close" title="关闭">✕</button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { 
  MinimizeWindow, 
  ToggleMaximizeWindow, 
  CloseWindow, 
  IsWindowMaximised 
} from '../../bindings/WeedyBox/internal/service/windowservice'

const isMaximised = ref(false)

const minimize = async () => {
  try {
    await MinimizeWindow()
  } catch (e) {
    console.error('最小化失败', e)
  }
}

const toggleMaximize = async () => {
  try {
    await ToggleMaximizeWindow()
    isMaximised.value = await IsWindowMaximised()
  } catch (e) {
    console.error('切换最大化失败', e)
  }
}

const close = async () => {
  try {
    await CloseWindow()
  } catch (e) {
    console.error('关闭失败', e)
  }
}

onMounted(async () => {
  try {
    isMaximised.value = await IsWindowMaximised()
  } catch (e) {
    console.error('获取窗口状态失败', e)
  }
})
</script>