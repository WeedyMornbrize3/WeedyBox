<!-- frontend/src/components/meta/MetaWorkHero.vue
     作品详情页顶部的大图区（/works 新排版，用户 2026-11-23 按 Steam 商店页布局定稿）：
     背景大图铺满 → 压一层到页面底色的渐变遮罩 → 左下是来源/发售日/类型的小字 + 作品名 +
     标签 chips，右下是竖版封面（**用游戏库卡片那张 cover 图**，未入库时才退回远程封面）。
     背景图的优先级在 stores/metaStore.ts 的 backgroundUrl 里算好了：
       Steam 源 = 本机 librarycache 的 library_hero*，没有才用商店接口的预览图；
       DLsite 源 = 画廊第二张预览图（第一张是游戏库卡片封面）。
     背景图加载失败时只保留渐变底，不影响文字与操作。 -->
<template>
  <section class="work-hero" :class="{ 'work-hero-plain': !showBackground }">
    <img
      v-if="showBackground"
      :src="backgroundSrc"
      class="work-hero-bg"
      alt=""
      aria-hidden="true"
      referrerpolicy="no-referrer"
      @error="backgroundOk = false"
    />
    <div class="work-hero-shade" aria-hidden="true" />

    <div class="work-hero-body">
      <div class="flex-1 min-w-0 flex flex-col gap-2.5">
        <p class="work-hero-kicker">
          <span v-if="sourceLabel">{{ sourceLabel }}</span>
          <template v-if="work.releaseDate">
            <span v-if="sourceLabel" class="work-hero-dot" aria-hidden="true">·</span>
            <span>{{ work.releaseDate }}</span>
          </template>
          <template v-if="work.workType">
            <span class="work-hero-dot" aria-hidden="true">·</span>
            <span>{{ work.workType }}</span>
          </template>
          <template v-if="work.ageRating">
            <span class="work-hero-dot" aria-hidden="true">·</span>
            <span>{{ work.ageRating }}</span>
          </template>
        </p>

        <h1 class="work-hero-title">{{ work.name }}</h1>

        <!-- 标签预览：前 12 个 + 「+N」；真实的增删编辑在右栏的 MetaTagEditor -->
        <ul v-if="tags.length" class="work-hero-tags">
          <li v-for="tag in visibleTags" :key="tag" class="work-hero-tag">{{ tag }}</li>
          <li v-if="hiddenTagCount > 0" class="work-hero-tag">+{{ hiddenTagCount }}</li>
        </ul>
      </div>

      <!-- 竖版小图：2:3，与游戏库卡片同一张封面 -->
      <div class="work-hero-cover">
        <img
          v-if="coverSrc && coverOk"
          :src="coverSrc"
          :alt="`${work.name} 封面`"
          class="w-full h-full object-cover"
          referrerpolicy="no-referrer"
          @error="coverOk = false"
        />
        <span v-else class="i-lucide-image-off icon-xl" aria-hidden="true" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { WorkMeta } from '../../services/metaService'

/** hero 里最多显示几个标签（其余折成「+N」，避免把标题挤爆） */
const HERO_TAG_LIMIT = 12

const props = withDefaults(
  defineProps<{
    work: WorkMeta
    /** 背景大图（可能是本地 hero 的 data URL，也可能是远程预览图） */
    backgroundSrc?: string
    /** 竖版封面（游戏库卡片的 cover 优先） */
    coverSrc?: string
    /** 标签列表（作品标签编辑稿；为空时退回 work.tags） */
    tags?: string[]
    /** 来源名称（DLsite / Steam），显示在标题上方 */
    sourceLabel?: string
  }>(),
  { backgroundSrc: '', coverSrc: '', tags: () => [], sourceLabel: '' },
)

/** 背景图加载失败标记：换作品（src 变化）时复位 */
const backgroundOk = ref(true)
watch(
  () => props.backgroundSrc,
  () => {
    backgroundOk.value = true
  },
)

/** 封面加载失败标记：同样跟随 src 复位 */
const coverOk = ref(true)
watch(
  () => props.coverSrc,
  () => {
    coverOk.value = true
  },
)

const showBackground = computed(() => Boolean(props.backgroundSrc) && backgroundOk.value)
const visibleTags = computed(() => props.tags.slice(0, HERO_TAG_LIMIT))
const hiddenTagCount = computed(() => Math.max(0, props.tags.length - HERO_TAG_LIMIT))
</script>

<style scoped>
/* ===== 大图区 =====
   高度用 min-height：窗口宽时图铺得开，窄时也不至于把标题挤出卡片。
   圆角 + 描边与全站卡片一致（card-flat 的长相），图片不溢出圆角。 */
.work-hero {
  position: relative;
  display: flex;
  align-items: flex-end;
  min-height: 340px;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-bg-secondary);
  isolation: isolate;
}

/* 没有背景图（或加载失败）时的兜底底纹：用主题色做出轻微的纵向层次 */
.work-hero-plain {
  background:
    linear-gradient(160deg, var(--color-bg-secondary), var(--color-bg-tertiary));
}

.work-hero-bg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
}

/* 压两层渐变（都只做「压暗」，不往页面底色过渡 —— 早期版本用
   `var(--color-bg-primary)` 做渐变终点，浅色主题下颜色是米色，把整张图盖掉了 76%，
   看起来就像"背景图没生效"（2026-11-23 用户反馈）。现在：
   ① 底部黑色渐变压暗，保证白色标题/标签可读（到 42% 处只剩两成强度）；
   ② 左侧暗角，让长标题压在亮画面上也读得清。
   图片顶部与右侧保持原样，大图能真正看出来。 */
.work-hero-shade {
  position: absolute;
  inset: 0;
  z-index: 1;
  background:
    linear-gradient(
      to top,
      color-mix(in srgb, var(--color-hero-veil) 92%, transparent) 0%,
      color-mix(in srgb, var(--color-hero-veil) 45%, transparent) 42%,
      transparent 78%
    ),
    linear-gradient(
      to right,
      color-mix(in srgb, var(--color-hero-veil) 70%, transparent) 0%,
      color-mix(in srgb, var(--color-hero-veil) 22%, transparent) 46%,
      transparent 72%
    );
}

.work-hero-body {
  position: relative;
  z-index: 2;
  display: flex;
  align-items: flex-end;
  gap: 24px;
  width: 100%;
  padding: 110px 28px 24px;
}

/* 来源 / 发售日 / 类型：压在图片上的小字，两套主题都用浅色（图片本身是深色画面） */
.work-hero-kicker {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 12px;
  line-height: 1.4;
  color: var(--color-text-on-image);
  opacity: 0.82;
}

.work-hero-dot {
  opacity: 0.6;
}

.work-hero-title {
  margin: 0;
  font-size: 28px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: 0.2px;
  color: var(--color-text-on-image);
  overflow-wrap: anywhere;
  /* 画面亮的时候白字容易糊，补一层很轻的黑投影（颜色复用遮罩 token） */
  text-shadow: 0 2px 12px var(--color-hero-veil);
}

.work-hero-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

/* 标签 chip：半透明深色底 + 白字，压在图片上（与界面主题无关，所以色值走全局 token） */
.work-hero-tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  line-height: 1.5;
  white-space: nowrap;
  color: var(--color-text-on-image);
  background: var(--color-hero-veil);
}

/* 竖版封面：2:3（与游戏库卡片同比例），比标题略高，作为 hero 的视觉锚点 */
.work-hero-cover {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 140px;
  aspect-ratio: 2 / 3;
  overflow: hidden;
  border: 1px solid var(--color-border-dark);
  border-radius: 8px;
  background: var(--color-bg-secondary);
  color: var(--color-text-muted);
  box-shadow: var(--color-shadow-heavy);
}
</style>
