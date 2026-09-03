<template>
  <div class="max-w-[700px] mx-auto w-full select-none">
    <h1 class="text-2xl font-semibold text-primary mb-5 tracking-[0.5px]">
      <slot name="title">TODO</slot>
    </h1>
    <div class="flex flex-col gap-5">
      <slot />
    </div>
  </div>
</template>