<script setup lang="ts">
import MainLayout from './layouts/MainLayout.vue';
import AppToaster from './components/AppToaster.vue';
import './styles/global.css';
</script>

<template>
  <MainLayout />
  <!-- 全局轻提示（vue-sonner）：浮层定位、不参与布局；入口是 utils/toast.ts -->
  <AppToaster />
</template>

<style>
/* 全局设置文字不能选中 */
* {
  user-select: none;
  -webkit-user-select: none;
}

input,
textarea,
[contenteditable="true"],
.selectable,
.selectable * {
  user-select: text !important;
  -webkit-user-select: text !important;
}
</style>
