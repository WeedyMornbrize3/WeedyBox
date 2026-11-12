<!-- frontend/src/components/settings/TagLimitInput.vue
     设置页里的紧凑数字控件：[-] 数字 [+]。
     按设置页规范做小（约 26px 高、12px 字），不套 btn-brand 那 35.2px 的大按钮；
     边界（4–64）由 stores/App.ts 的常量统一约束，这里只负责交互与显示。 -->
<template>
  <div class="tag-limit" role="group" aria-label="作品标签显示数量">
    <button
      type="button"
      class="tag-limit__step"
      :disabled="modelValue <= TAG_LIMIT_MIN"
      aria-label="减少标签数量"
      @click="step(-1)"
    >
      −
    </button>
    <input
      class="tag-limit__input"
      type="number"
      inputmode="numeric"
      :min="TAG_LIMIT_MIN"
      :max="TAG_LIMIT_MAX"
      step="1"
      :value="modelValue"
      aria-label="作品标签显示数量"
      @change="commit"
    />
    <button
      type="button"
      class="tag-limit__step"
      :disabled="modelValue >= TAG_LIMIT_MAX"
      aria-label="增加标签数量"
      @click="step(1)"
    >
      +
    </button>
  </div>
</template>

<script setup lang="ts">
import { TAG_LIMIT_MAX, TAG_LIMIT_MIN } from '../../stores/App'

const props = defineProps<{ modelValue: number }>()
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

function clamp(value: number): number {
  return Math.min(TAG_LIMIT_MAX, Math.max(TAG_LIMIT_MIN, Math.round(value)))
}

function step(delta: number) {
  emit('update:modelValue', clamp(props.modelValue + delta))
}

/** 失焦/回车提交时把非法输入拉回边界（输入框里的值也一起纠正） */
function commit(event: Event) {
  const input = event.target as HTMLInputElement
  const next = clamp(Number(input.value) || props.modelValue)
  input.value = String(next)
  emit('update:modelValue', next)
}
</script>

<style scoped>
.tag-limit {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px 4px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background-color: var(--color-bg-secondary);
}

.tag-limit__input {
  width: 44px;
  height: 22px;
  padding: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text-primary);
  font-size: 12px;
  text-align: center;
}

.tag-limit__step {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
  transition: background-color var(--motion-fast) var(--ease-enter),
    color var(--motion-fast) var(--ease-enter);
}

.tag-limit__step:hover:not(:disabled) {
  background-color: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.tag-limit__step:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* 隐藏 number 输入的上下箭头，避免和自定义 +/- 打架 */
.tag-limit__input::-webkit-outer-spin-button,
.tag-limit__input::-webkit-inner-spin-button {
  appearance: none;
  margin: 0;
}
</style>
