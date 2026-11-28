<!-- frontend/src/components/meta/MetaWorkHero.vue
     作品详情页顶部的大图区（/works 新排版，用户 2026-11-23 按 Steam 商店页布局定稿）：
     背景大图铺满 → 压一层到页面底色的渐变遮罩 → 左下是来源/社团/发售日/形式/评分的小字 +
     作品名 + 标签 chips，右下是竖版封面（**用游戏库卡片那张 cover 图**，未入库时才退回远程封面）。
     2026-11-24 用户要求：原来左信息栏里的「社团（开发商）」「评分」并进这行小字，
     左信息栏整块删除 —— 所以这里用 kickerParts 统一拼「源 · 社团 · 发售日 · 形式 · 评分」，
     不再各自写死分隔符（少一项就少一个点）。
     背景图（2026-11-27 起）：**后端缓存的本地图**（HeroImage，covers/ 里那张）——
       本地没有才由后端联网补一张并落盘，页面本身不再直接挂远程 URL，
       所以不会出现「先远程预览、再换成本地图」的两次换 src（用户反馈的抖动）。
       Steam 源在本机 Steam 客户端缓存里有 hero 时优先用那张；DLsite 源用画廊第二张预览图。
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

    <!-- 左上角返回游戏库（用户 2026-11-28：「hero图左上角添加退出按钮回到library界面」）。
         用 naive 的 n-button circle 而不是自己写 <button>（用户 2026-11-28：「使用naive ui的圆形样式」）：
         尺寸/圆形/波纹/hover/焦点圈/禁用态都由 naive 给，这里只把颜色掰成"压在图片上"的口径。
         配色仍不跟主题走：白色图标 + 半透明黑底（--color-hero-veil），因为品牌色压在
         任意画面上不一定有对比度。
         ⚠️ naive 把 --n-* 以内联样式写在按钮元素上，外部 class 覆盖不掉；
         必须在同一元素用 :style 覆盖同名变量才生效（见 styles/naiveTheme.ts 的同类注释）。 -->
    <n-button
      class="work-hero-back"
      quaternary
      circle
      size="small"
      aria-label="返回游戏库"
      title="返回游戏库"
      :style="backButtonStyle"
      @click="emit('back')"
    >
      <template #icon>
        <span class="i-lucide-arrow-left icon-sm" aria-hidden="true" />
      </template>
    </n-button>

    <div class="work-hero-body">
      <div class="flex-1 min-w-0 flex flex-col gap-2.5">
        <p v-if="kickerParts.length" class="work-hero-kicker">
          <template v-for="(part, index) in kickerParts" :key="part">
            <span v-if="index > 0" class="work-hero-dot" aria-hidden="true">·</span>
            <span>{{ part }}</span>
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
import { NButton } from 'naive-ui'
import type { WorkMeta } from '../../services/metaService'

/** hero 里最多显示几个标签（其余折成「+N」，避免把标题挤爆） */
const HERO_TAG_LIMIT = 10

const props = withDefaults(
  defineProps<{
    work: WorkMeta
    /** 背景大图（后端缓存的 hero 本地图，data URL） */
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

/** 返回游戏库：只把点击抛给 WorksPage（它持有 router） */
const emit = defineEmits<{ back: [] }>()

/**
 * 返回按钮的配色：压在 hero 图片上，所以固定用「白色图标 + 半透明黑底」，
 * 不跟主题走（浅色主题下用浅底/品牌色都会糊）。
 * 走 :style 覆盖 naive 的 --n-* 内联变量 —— quaternary 的静息底是 transparent，
 * 这里把它换成半透明黑，hover/pressed 再加深一档。
 *
 * ⚠️ 不要试图用 --n-border* 画描边：naive 的边框在 `.n-button__border` /
 * `.n-button__state-border` 两个子元素上，跟 --n-border 变量不是一条链路
 * （实测 --n-border 传了也不生效，border 仍是 0px）。所以描边交给 naive，
 * 这里只把「环形边界」用静息底色的半透明度做出来。
 */
const backButtonStyle = {
  '--n-color': 'transparent',
  '--n-color-hover': 'color-mix(in srgb, var(--color-hero-veil) 88%, transparent)',
  '--n-color-pressed': 'var(--color-hero-veil)',
  '--n-color-focus': 'color-mix(in srgb, var(--color-hero-veil) 88%, transparent)',
  '--n-text-color': 'var(--color-text-on-image)',
  '--n-text-color-hover': 'var(--color-text-on-image)',
  '--n-text-color-pressed': 'var(--color-text-on-image)',
  '--n-text-color-focus': 'var(--color-text-on-image)',
  '--n-ripple-color': 'var(--color-hero-veil)',
} as Record<string, string>

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

/**
 * 标题上方那行小字（用户 2026-11-24 定稿）：源 · 社团/开发商 · 发售日 · 作品形式 · 评分。
 * 「社团」取 work.maker —— DLsite 是社团名、Steam 是开发商（getFromSteam 里做的映射）；
 * 评分带评价条数（与左栏原来的「评分 4.5（123 条）」口径一致），没有条数就只显示分数。
 */
const kickerParts = computed(() => {
  const parts: string[] = []
  if (props.sourceLabel) parts.push(props.sourceLabel)
  if (props.work.maker) parts.push(props.work.maker)
  if (props.work.releaseDate) parts.push(props.work.releaseDate)
  if (props.work.workType) parts.push(props.work.workType)
  const rating = props.work.rating?.trim()
  if (rating) {
    const count = props.work.ratingCount?.trim()
    parts.push(count ? `评分 ${rating}（${count} 条）` : `评分 ${rating}`)
  }
  return parts
})
</script>

<style scoped>
/* ===== 大图区 =====
   高度用 min-height：窗口宽时图铺得开，窄时也不至于把标题挤出卡片。
   2026-11-25 全站卡片改方形：大图区与竖版封面都去掉圆角（描边保留）。 */
.work-hero {
  position: relative;
  display: flex;
  align-items: flex-end;
  min-height: 340px;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: 0;
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
  /* 用户 2026-11-25 澄清：hero 画面本身不模糊、也不加反光带。
     模糊拷贝只用来给周边 UI 做遮罩（见 WorksPage 的 .work-ambient 与动作栏的 ::after）。 */
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

/* 左上角返回按钮：现在是 naive 的 n-button.circle（尺寸/圆形/波纹/焦点圈都由 naive 给），
   这里只定位 + 给一块静息半透明黑底 —— 按钮压在图片上，完全透明会看不出边界。
   底色等动态颜色走 :style 覆盖 --n-*（见 script 的 backButtonStyle）；
   1px 白描边用 outline 画（naive 的 border 在子元素上，,--n-border 那条路不通）。 */
.work-hero-back {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 3;
  background-color: color-mix(in srgb, var(--color-hero-veil) 55%, transparent);
  outline: 1px solid color-mix(in srgb, var(--color-text-on-image) 26%, transparent);
  outline-offset: -1px;
}

.work-hero-back:hover {
  background-color: color-mix(in srgb, var(--color-hero-veil) 88%, transparent);
  outline-color: color-mix(in srgb, var(--color-text-on-image) 45%, transparent);
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
  border-radius: 0;
  background: var(--color-bg-secondary);
  color: var(--color-text-muted);
  box-shadow: var(--color-shadow-heavy);
}
</style>
