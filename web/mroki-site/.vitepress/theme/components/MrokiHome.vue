<script setup lang="ts">
import { computed, ref } from 'vue'
import { withBase } from 'vitepress'
import {
  hero,
  heroDiff,
  features,
  featuresIntro,
  showcase,
  closing,
  footer,
  iconExternal,
  type HomeAction,
} from '../home-content'

// Resolve an action's href: external links pass through; internal links are
// base-prepended so the page keeps working under a non-root base.
function href(action: HomeAction): string {
  return action.external ? action.link : withBase(action.link)
}

// External anchors open in a new tab; announce that to assistive tech.
function ariaLabel(action: HomeAction): string | undefined {
  return action.external ? `${action.text} (opens in new tab)` : undefined
}

// Decorative hero backdrop: a real (dark-only) hub screenshot, dimmed and masked
// in CSS so it reads as "real app" texture behind the crisp diff mockup without
// competing with it or hurting copy contrast. Base-safe for non-root deploys.
const heroBgUrl = computed(() => withBase('/screenshots/hero-diff.png'))

// Showcase view switcher (mirrors the hub's Unified/Split/Patch control).
const activeView = ref(showcase.defaultView)
const currentView = computed(
  () => showcase.views.find((view) => view.id === activeView.value) ?? showcase.views[0],
)

// The one example, rendered three ways to mirror the hub's viewer. Kinds map to
// the hero diff's line classes; the derived shapes below drive each view.
type MockKind = 'context' | 'removed' | 'added'
interface MockLine {
  kind: MockKind
  text: string
}
interface SplitRow {
  left: MockLine | null
  right: MockLine | null
}
type PatchOpKind = 'replace' | 'add' | 'remove'
interface PatchOp {
  op: PatchOpKind
  path: string
  from?: string
  to?: string
}

const example = showcase.example
const lastFieldIndex = example.length - 1

// One `"key": value` line at the object's single nesting level.
function fieldLine(key: string, value: string, comma: boolean): string {
  return `  "${key}": ${value}${comma ? ',' : ''}`
}

// Unified: a single git-style column. Changed fields expand to a removed +
// added pair; context/removed/added render one line each, wrapped in braces.
const unifiedLines = computed<MockLine[]>(() => {
  const lines: MockLine[] = [{ kind: 'context', text: '{' }]
  example.forEach((field, i) => {
    const comma = i !== lastFieldIndex
    if (field.change === 'changed') {
      lines.push({ kind: 'removed', text: fieldLine(field.key, field.live ?? '', comma) })
      lines.push({ kind: 'added', text: fieldLine(field.key, field.shadow ?? '', comma) })
    } else if (field.change === 'removed') {
      lines.push({ kind: 'removed', text: fieldLine(field.key, field.live ?? '', comma) })
    } else if (field.change === 'added') {
      lines.push({ kind: 'added', text: fieldLine(field.key, field.shadow ?? '', comma) })
    } else {
      lines.push({ kind: 'context', text: fieldLine(field.key, field.live ?? '', comma) })
    }
  })
  lines.push({ kind: 'context', text: '}' })
  return lines
})

// Split: live on the left, shadow on the right, aligned row for row. Added
// fields leave the left blank; removed fields leave the right blank.
const splitRows = computed<SplitRow[]>(() => {
  const brace = (text: string): SplitRow => ({
    left: { kind: 'context', text },
    right: { kind: 'context', text },
  })
  const rows: SplitRow[] = [brace('{')]
  example.forEach((field, i) => {
    const comma = i !== lastFieldIndex
    if (field.change === 'changed') {
      rows.push({
        left: { kind: 'removed', text: fieldLine(field.key, field.live ?? '', comma) },
        right: { kind: 'added', text: fieldLine(field.key, field.shadow ?? '', comma) },
      })
    } else if (field.change === 'removed') {
      rows.push({
        left: { kind: 'removed', text: fieldLine(field.key, field.live ?? '', comma) },
        right: null,
      })
    } else if (field.change === 'added') {
      rows.push({
        left: null,
        right: { kind: 'added', text: fieldLine(field.key, field.shadow ?? '', comma) },
      })
    } else {
      rows.push({
        left: { kind: 'context', text: fieldLine(field.key, field.live ?? '', comma) },
        right: { kind: 'context', text: fieldLine(field.key, field.shadow ?? '', comma) },
      })
    }
  })
  rows.push(brace('}'))
  return rows
})

// Patch: the changed fields only, as RFC 6902 operations.
const patchOps = computed<PatchOp[]>(() =>
  example
    .filter((field) => field.change !== 'context')
    .map((field): PatchOp => {
      if (field.change === 'changed') {
        return { op: 'replace', path: field.path, from: field.live, to: field.shadow }
      }
      if (field.change === 'added') {
        return { op: 'add', path: field.path, to: field.shadow }
      }
      return { op: 'remove', path: field.path, from: field.live }
    }),
)

// Collect the tab buttons so arrow-key navigation can move focus with the
// roving-tabindex pattern.
const tabEls = ref<(HTMLButtonElement | null)[]>([])
function setTabEl(el: unknown, index: number): void {
  tabEls.value[index] = el as HTMLButtonElement | null
}

function selectView(id: string): void {
  activeView.value = id
}

// Left/right (and Home/End) move between tabs and carry focus, as expected of a
// tablist; other keys fall through to default behavior.
function onTabKeydown(event: KeyboardEvent, index: number): void {
  const { views } = showcase
  let next = index
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    next = (index + 1) % views.length
  } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
    next = (index - 1 + views.length) % views.length
  } else if (event.key === 'Home') {
    next = 0
  } else if (event.key === 'End') {
    next = views.length - 1
  } else {
    return
  }
  event.preventDefault()
  activeView.value = views[next].id
  tabEls.value[next]?.focus()
}
</script>

<template>
  <div class="mh">
    <!-- Hero -->
    <section class="mh-hero">
      <div
        class="mh-hero-bg"
        aria-hidden="true"
        :style="{ backgroundImage: `url(${heroBgUrl})` }"
      ></div>
      <div class="mh-container mh-hero-inner">
        <div class="mh-hero-copy">
          <h1 class="mh-hero-heading">
            <span class="mh-hero-name">{{ hero.name }}</span>
            <span class="mh-hero-tagline">{{ hero.tagline }}</span>
          </h1>
          <p class="mh-hero-lede">{{ hero.lede }}</p>
          <div class="mh-actions">
            <a
              v-for="action in hero.actions"
              :key="action.text"
              class="mh-cta"
              :class="`mh-cta--${action.theme}`"
              :href="href(action)"
              :target="action.external ? '_blank' : undefined"
              :rel="action.external ? 'noopener noreferrer' : undefined"
              :aria-label="ariaLabel(action)"
            >
              <span>{{ action.text }}</span>
              <span
                v-if="action.external"
                class="mh-cta-ext"
                aria-hidden="true"
                v-html="iconExternal"
              />
            </a>
          </div>
        </div>
        <figure class="mh-herodiff" role="img" :aria-label="heroDiff.caption">
          <div class="mh-herodiff-bar" aria-hidden="true">
            <span class="mh-herodiff-label">{{ heroDiff.label }}</span>
            <span class="mh-herodiff-badge">{{ heroDiff.badge }}</span>
          </div>
          <div class="mh-herodiff-code" aria-hidden="true">
            <span
              v-for="(line, i) in heroDiff.lines"
              :key="i"
              class="mh-diff-line"
              :class="`mh-diff-line--${line.kind}`"
            >
              <span class="mh-diff-gutter">{{
                line.kind === 'added' ? '+' : line.kind === 'removed' ? '-' : ''
              }}</span>
              <span class="mh-diff-text">{{ line.text }}</span>
            </span>
          </div>
        </figure>
      </div>
    </section>

    <!-- Features -->
    <section class="mh-features">
      <div class="mh-container">
        <div class="mh-section-head">
          <h2 class="mh-section-title">{{ featuresIntro.title }}</h2>
          <p class="mh-section-lede">{{ featuresIntro.lede }}</p>
        </div>
        <ul class="mh-feature-grid">
          <li v-for="feature in features" :key="feature.title" class="mh-feature">
            <span class="mh-feature-icon" aria-hidden="true" v-html="feature.icon" />
            <h3 class="mh-feature-title">{{ feature.title }}</h3>
            <p class="mh-feature-detail">{{ feature.detail }}</p>
          </li>
        </ul>
      </div>
    </section>

    <!-- Product showcase -->
    <section class="mh-showcase">
      <div class="mh-container mh-showcase-inner">
        <div class="mh-showcase-copy">
          <h2 class="mh-showcase-title">{{ showcase.title }}</h2>
          <p class="mh-showcase-lede">{{ showcase.lede }}</p>
          <div class="mh-actions">
            <a
              class="mh-cta"
              :class="`mh-cta--${showcase.action.theme}`"
              :href="href(showcase.action)"
              :target="showcase.action.external ? '_blank' : undefined"
              :rel="showcase.action.external ? 'noopener noreferrer' : undefined"
              >{{ showcase.action.text }}</a
            >
          </div>
        </div>
        <div class="mh-showcase-media">
          <div class="mh-viewtabs" role="tablist" aria-label="Diff view">
            <button
              v-for="(view, index) in showcase.views"
              :id="`mh-viewtab-${view.id}`"
              :key="view.id"
              :ref="(el) => setTabEl(el, index)"
              class="mh-viewtab"
              type="button"
              role="tab"
              :aria-selected="activeView === view.id"
              aria-controls="mh-viewpanel"
              :tabindex="activeView === view.id ? 0 : -1"
              @click="selectView(view.id)"
              @keydown="onTabKeydown($event, index)"
            >
              {{ view.label }}
            </button>
          </div>

          <figure class="mh-frame">
            <div class="mh-frame-bar" aria-hidden="true">
              <span></span><span></span><span></span>
            </div>
            <div
              id="mh-viewpanel"
              class="mh-frame-body"
              role="tabpanel"
              :aria-labelledby="`mh-viewtab-${currentView.id}`"
              tabindex="0"
            >
              <Transition name="mh-fade" mode="out-in">
                <div :key="currentView.id" class="mh-mock">
                  <!-- Unified: one git-style column -->
                  <div v-if="activeView === 'unified'" class="mh-mock-code">
                    <span
                      v-for="(line, i) in unifiedLines"
                      :key="i"
                      class="mh-diff-line"
                      :class="`mh-diff-line--${line.kind}`"
                    >
                      <span class="mh-diff-gutter">{{
                        line.kind === 'added' ? '+' : line.kind === 'removed' ? '-' : ''
                      }}</span>
                      <span class="mh-diff-text">{{ line.text }}</span>
                    </span>
                  </div>

                  <!-- Split: live vs shadow, side by side -->
                  <div v-else-if="activeView === 'split'" class="mh-split">
                    <div class="mh-split-col">
                      <div class="mh-split-head">
                        <span class="mh-split-dot mh-split-dot--live" aria-hidden="true"></span>Live
                      </div>
                      <div class="mh-mock-code">
                        <span
                          v-for="(row, i) in splitRows"
                          :key="i"
                          class="mh-diff-line"
                          :class="
                            row.left ? `mh-diff-line--${row.left.kind}` : 'mh-diff-line--blank'
                          "
                        >
                          <span class="mh-diff-text">{{ row.left ? row.left.text : '' }}</span>
                        </span>
                      </div>
                    </div>
                    <div class="mh-split-col">
                      <div class="mh-split-head">
                        <span
                          class="mh-split-dot mh-split-dot--shadow"
                          aria-hidden="true"
                        ></span
                        >Shadow
                      </div>
                      <div class="mh-mock-code">
                        <span
                          v-for="(row, i) in splitRows"
                          :key="i"
                          class="mh-diff-line"
                          :class="
                            row.right ? `mh-diff-line--${row.right.kind}` : 'mh-diff-line--blank'
                          "
                        >
                          <span class="mh-diff-text">{{ row.right ? row.right.text : '' }}</span>
                        </span>
                      </div>
                    </div>
                  </div>

                  <!-- Patch: RFC 6902 operations -->
                  <ul v-else class="mh-patch">
                    <li v-for="(op, i) in patchOps" :key="i" class="mh-patch-row">
                      <span class="mh-patch-op" :class="`mh-patch-op--${op.op}`">{{ op.op }}</span>
                      <span class="mh-patch-path">{{ op.path }}</span>
                      <span class="mh-patch-val">
                        <template v-if="op.op === 'replace'"
                          ><span class="mh-patch-from">{{ op.from }}</span
                          ><span class="mh-patch-arrow" aria-hidden="true"> → </span
                          ><span class="mh-patch-to">{{ op.to }}</span></template
                        >
                        <template v-else-if="op.op === 'add'"
                          ><span class="mh-patch-to">{{ op.to }}</span></template
                        >
                        <template v-else
                          ><span class="mh-patch-from">{{ op.from }}</span></template
                        >
                      </span>
                    </li>
                  </ul>
                </div>
              </Transition>
            </div>
          </figure>

          <p class="mh-frame-caption">{{ currentView.caption }}</p>
        </div>
      </div>
    </section>

    <!-- Closing CTA band -->
    <section class="mh-closing">
      <div class="mh-container mh-closing-inner">
        <h2 class="mh-closing-title">{{ closing.title }}</h2>
        <p class="mh-closing-lede">{{ closing.lede }}</p>
        <div class="mh-actions">
          <a
            v-for="action in closing.actions"
            :key="action.text"
            class="mh-cta"
            :class="`mh-cta--${action.theme}`"
            :href="href(action)"
            :target="action.external ? '_blank' : undefined"
            :rel="action.external ? 'noopener noreferrer' : undefined"
            :aria-label="ariaLabel(action)"
          >
            <span>{{ action.text }}</span>
            <span
              v-if="action.external"
              class="mh-cta-ext"
              aria-hidden="true"
              v-html="iconExternal"
            />
          </a>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="mh-footer">
      <div class="mh-container mh-footer-inner">
        <nav class="mh-footer-links" aria-label="Footer">
          <a
            v-for="link in footer.links"
            :key="link.text"
            :href="link.external ? link.link : withBase(link.link)"
            :target="link.external ? '_blank' : undefined"
            :rel="link.external ? 'noopener noreferrer' : undefined"
            :aria-label="link.external ? `${link.text} (opens in new tab)` : undefined"
          >
            <span>{{ link.text }}</span>
            <span
              v-if="link.external"
              class="mh-footer-ext"
              aria-hidden="true"
              v-html="iconExternal"
            />
          </a>
        </nav>
        <p class="mh-footer-copyright">{{ footer.copyright }}</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
/* Centered content shell, shared by every section. */
.mh-container {
  max-width: 1152px;
  margin-inline: auto;
  padding-inline: clamp(24px, 5vw, 40px);
}

/* ---- Hero -------------------------------------------------------- */
.mh-hero {
  position: relative;
  padding-block: clamp(72px, 12vh, 140px) clamp(48px, 8vw, 88px);
}

/* Decorative backdrop: a real (dark-only) hub screenshot, dimmed and masked so
   it adds "real app" texture behind the crisp mockup without competing with it
   or hurting copy contrast. Sits below .mh-hero-inner (z-index: 1). Anchored
   top-right, behind the mockup; faded to nothing before it reaches the copy.
   Faint in light mode; a touch stronger in dark mode, where the dark shot
   belongs. */
.mh-hero-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background-repeat: no-repeat;
  background-position: top right;
  background-size: 62% auto;
  opacity: 0.05;
  -webkit-mask-image: linear-gradient(to left, #000 30%, transparent 66%);
  mask-image: linear-gradient(to left, #000 30%, transparent 66%);
}

.dark .mh-hero-bg {
  opacity: 0.16;
}

.mh-hero-inner {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: clamp(32px, 5vw, 72px);
  align-items: center;
  text-align: left;
}

.mh-hero-heading {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin: 0;
}

.mh-hero-name {
  font-size: clamp(2.5rem, 5vw, 4rem);
  font-weight: 700;
  line-height: 1.05;
  letter-spacing: -0.03em;
  color: var(--vp-c-text-1);
}

.mh-hero-tagline {
  font-size: clamp(1.25rem, 2.6vw, 1.75rem);
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
}

.mh-hero-lede {
  margin: 1.25rem 0 0;
  max-width: 52ch;
  font-size: 1.125rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

/* ---- Buttons (styled from the theme's own CTA tokens) ------------ */
.mh-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
}

.mh-hero .mh-actions {
  justify-content: flex-start;
  margin-top: 1.75rem;
}

.mh-cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 46px;
  padding-inline: 22px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 600;
  text-decoration: none;
  transition: background-color 0.25s, border-color 0.25s, color 0.25s;
}

.mh-cta--brand {
  background: var(--vp-button-brand-bg);
  color: var(--vp-button-brand-text);
  border-color: var(--vp-button-brand-border);
}

.mh-cta--brand:hover {
  background: var(--vp-button-brand-hover-bg);
  color: var(--vp-button-brand-hover-text);
  border-color: var(--vp-button-brand-hover-border);
}

.mh-cta--alt {
  background: var(--vp-button-alt-bg);
  color: var(--vp-button-alt-text);
  border-color: var(--vp-button-alt-border);
}

.mh-cta--alt:hover {
  background: var(--vp-button-alt-hover-bg);
  color: var(--vp-button-alt-hover-text);
  border-color: var(--vp-button-alt-hover-border);
}

.mh-cta:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

/* External-link glyph inside a CTA. */
.mh-cta-ext {
  display: inline-grid;
  place-items: center;
  margin-left: 6px;
}

.mh-cta-ext :deep(svg) {
  width: 15px;
  height: 15px;
}

/* ---- Hero diff motif (mroki's core job, shown once) ------------- */
.mh-herodiff {
  margin: 0;
  width: 100%;
  max-width: 520px;
  justify-self: end;
  border: 1px solid var(--vp-c-border);
  border-radius: 12px;
  overflow: hidden;
  background: var(--vp-c-bg-soft);
  text-align: left;
  box-shadow: 0 20px 48px -24px rgba(9, 9, 11, 0.4);
}

.mh-herodiff-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg);
}

.mh-herodiff-label {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--vp-c-text-2);
}

.mh-herodiff-badge {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--vp-c-green-1);
  background: var(--vp-c-green-soft);
}

.mh-herodiff-code {
  display: flex;
  flex-direction: column;
  padding-block: 12px;
  font-family: var(--vp-font-family-mono);
  font-size: 0.875rem;
  line-height: 1.7;
}

.mh-diff-line {
  display: flex;
  gap: 10px;
  padding-inline: 14px;
  color: var(--vp-c-text-2);
}

.mh-diff-gutter {
  flex: none;
  width: 0.75ch;
  text-align: center;
  color: var(--vp-c-text-3);
}

.mh-diff-text {
  white-space: pre;
}

.mh-diff-line--removed {
  color: var(--vp-c-danger-1);
  background: var(--vp-c-danger-soft);
}

.mh-diff-line--removed .mh-diff-gutter {
  color: var(--vp-c-danger-1);
}

.mh-diff-line--added {
  color: var(--vp-c-green-1);
  background: var(--vp-c-green-soft);
}

.mh-diff-line--added .mh-diff-gutter {
  color: var(--vp-c-green-1);
}

/* ---- Section heading (features + any titled section) ------------ */
.mh-section-head {
  max-width: 46ch;
  margin: 0 auto clamp(32px, 5vw, 48px);
  text-align: center;
}

.mh-section-title {
  margin: 0;
  font-size: clamp(1.5rem, 3vw, 2.125rem);
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
  border: none;
  padding: 0;
}

.mh-section-lede {
  margin: 0.75rem 0 0;
  font-size: 1.0625rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

/* ---- Features (card grid, hub card-token parity) --------------- */
.mh-features {
  padding-block: clamp(24px, 4vw, 48px) clamp(64px, 10vw, 112px);
}

.mh-feature-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: clamp(16px, 2vw, 24px);
  margin: 0;
  padding: 0;
  list-style: none;
}

.mh-feature {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.875rem;
  padding: clamp(24px, 3vw, 32px);
  border: 1px solid var(--vp-c-border);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
  transition: border-color 0.2s;
}

.mh-feature:hover {
  border-color: var(--vp-c-text-3);
}

.mh-feature-icon {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border: 1px solid var(--vp-c-border);
  border-radius: 10px;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
}

.mh-feature-icon :deep(svg) {
  width: 24px;
  height: 24px;
}

.mh-feature-title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--vp-c-text-1);
  border: none;
  padding: 0;
}

.mh-feature-detail {
  margin: 0;
  font-size: 0.9375rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

/* ---- Responsive ------------------------------------------------- */
@media (max-width: 860px) {
  .mh-hero-inner {
    grid-template-columns: 1fr;
    text-align: center;
  }

  .mh-hero .mh-actions {
    justify-content: center;
  }

  .mh-herodiff {
    justify-self: center;
    margin-inline: auto;
    max-width: 480px;
  }

  .mh-showcase-inner {
    grid-template-columns: 1fr;
  }

  .mh-showcase-copy {
    max-width: none;
  }
}

@media (max-width: 640px) {
  .mh-feature-grid {
    grid-template-columns: 1fr;
  }

  /* Single-column layout: drop the backdrop so it never sits under stacked copy. */
  .mh-hero-bg {
    display: none;
  }
}

/* ---- Product showcase (framed hub screenshot) ------------------- */
.mh-showcase {
  border-top: 1px solid var(--vp-c-divider);
  padding-block: clamp(56px, 9vw, 96px);
  background: var(--vp-c-bg-alt);
}

.mh-showcase-inner {
  display: grid;
  grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
  gap: clamp(32px, 5vw, 64px);
  align-items: center;
}

.mh-showcase-copy {
  max-width: 42ch;
}

.mh-showcase-title {
  margin: 0;
  font-size: clamp(1.5rem, 3vw, 2.125rem);
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
  border: none;
  padding: 0;
}

.mh-showcase-lede {
  margin: 1rem 0 0;
  font-size: 1.0625rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

.mh-showcase .mh-actions {
  margin-top: 1.75rem;
}

/* Stack the view switcher, framed screenshot, and caption in the media column. */
.mh-showcase-media {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* Segmented control mirroring the hub's own Unified/Split/Patch switcher. */
.mh-viewtabs {
  display: inline-flex;
  align-self: flex-start;
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--vp-c-bg);
}

.mh-viewtab {
  appearance: none;
  border: 0;
  background: transparent;
  color: var(--vp-c-text-2);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 500;
  line-height: 1;
  padding: 7px 14px;
  cursor: pointer;
  transition:
    color 0.15s ease,
    background-color 0.15s ease;
}

.mh-viewtab + .mh-viewtab {
  border-left: 1px solid var(--vp-c-border);
}

.mh-viewtab:hover {
  color: var(--vp-c-text-1);
}

.mh-viewtab[aria-selected='true'] {
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
}

.mh-viewtab:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: -2px;
}

/* Window chrome so the rendered diff reads as "the app" on both site themes.
   Theme-aware surfaces (not the dark-only hub palette) keep the mockup crisp. */
.mh-frame {
  margin: 0;
  border: 1px solid var(--vp-c-border);
  border-radius: 12px;
  overflow: hidden;
  background: var(--vp-c-bg-soft);
  box-shadow: 0 12px 32px -12px rgba(9, 9, 11, 0.35);
}

.mh-frame-bar {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 12px 16px;
  background: var(--vp-c-bg);
  border-bottom: 1px solid var(--vp-c-divider);
}

.mh-frame-bar span {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--vp-c-divider);
}

/* Holds whichever view is active. A min-height near the default (Split) view,
   plus vertical centering, keeps the frame steady as views of different heights
   crossfade — the shorter Patch view sits centered instead of jumping. */
.mh-frame-body {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 232px;
}

.mh-mock {
  width: 100%;
}

/* Shared monospace canvas for the unified and split renders, matching the hero
   diff's type treatment. */
.mh-mock-code {
  display: flex;
  flex-direction: column;
  padding-block: 12px;
  font-family: var(--vp-font-family-mono);
  font-size: 0.8125rem;
  line-height: 1.7;
  overflow-x: auto;
}

/* Empty half of a split row (a field present on only one side): hold the line's
   height so both columns stay aligned. */
.mh-diff-line--blank {
  min-height: 1.7em;
}

/* Split view: live vs shadow columns with a divider between. */
.mh-split {
  display: grid;
  grid-template-columns: 1fr 1fr;
}

.mh-split-col {
  min-width: 0;
}

.mh-split-col + .mh-split-col {
  border-left: 1px solid var(--vp-c-divider);
}

.mh-split-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--vp-c-divider);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--vp-c-text-2);
}

.mh-split-dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.mh-split-dot--live {
  background: var(--vp-c-green-1);
}

.mh-split-dot--shadow {
  background: var(--vp-c-brand-1);
}

/* Patch view: RFC 6902 operations, one row each. */
.mh-patch {
  list-style: none;
  margin: 0;
  padding: 16px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.mh-patch-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  font-family: var(--vp-font-family-mono);
  font-size: 0.8125rem;
  line-height: 1.5;
}

.mh-patch-op {
  flex: none;
  width: 5.5em;
  text-align: center;
  padding: 2px 0;
  border-radius: 6px;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.mh-patch-op--replace {
  color: var(--vp-c-yellow-1);
  background: var(--vp-c-yellow-soft);
}

.mh-patch-op--add {
  color: var(--vp-c-green-1);
  background: var(--vp-c-green-soft);
}

.mh-patch-op--remove {
  color: var(--vp-c-danger-1);
  background: var(--vp-c-danger-soft);
}

.mh-patch-path {
  flex: none;
  color: var(--vp-c-text-1);
  font-weight: 500;
}

.mh-patch-val {
  min-width: 0;
  color: var(--vp-c-text-2);
  word-break: break-all;
}

.mh-patch-from {
  color: var(--vp-c-danger-1);
}

.mh-patch-to {
  color: var(--vp-c-green-1);
}

.mh-patch-arrow {
  color: var(--vp-c-text-3);
}

.mh-frame-caption {
  margin: 0;
  color: var(--vp-c-text-2);
  font-size: 0.875rem;
  line-height: 1.5;
}

/* Crossfade between views; stilled for reduced-motion users. */
.mh-fade-enter-active,
.mh-fade-leave-active {
  transition: opacity 0.22s ease;
}

.mh-fade-enter-from,
.mh-fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .mh-fade-enter-active,
  .mh-fade-leave-active {
    transition: none;
  }
}

/* ---- Closing CTA band ------------------------------------------- */
.mh-closing {
  border-top: 1px solid var(--vp-c-divider);
  padding-block: clamp(64px, 10vw, 112px);
  text-align: center;
}

.mh-closing-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.mh-closing-title {
  margin: 0;
  max-width: 22ch;
  font-size: clamp(1.625rem, 3.4vw, 2.375rem);
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: -0.02em;
  color: var(--vp-c-text-1);
  border: none;
  padding: 0;
}

.mh-closing-lede {
  margin: 1rem auto 0;
  max-width: 54ch;
  font-size: 1.0625rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

.mh-closing .mh-actions {
  justify-content: center;
  margin-top: 2rem;
}

/* ---- Footer (full-width rule, hub-neutral) ---------------------- */
.mh-footer {
  border-top: 1px solid var(--vp-c-divider);
  padding-block: clamp(28px, 5vw, 40px);
  background: var(--vp-c-bg);
}

.mh-footer-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
}

.mh-footer-links {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px 24px;
}

.mh-footer-links a {
  display: inline-flex;
  align-items: center;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--vp-c-text-2);
  text-decoration: underline;
  text-decoration-color: transparent;
  text-underline-offset: 3px;
  transition: color 0.2s, text-decoration-color 0.2s;
}

.mh-footer-links a:hover {
  color: var(--vp-c-text-1);
  text-decoration-color: currentColor;
}

.mh-footer-ext {
  display: inline-grid;
  place-items: center;
  margin-left: 5px;
}

.mh-footer-ext :deep(svg) {
  width: 13px;
  height: 13px;
}

.mh-footer-links a:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
  border-radius: 4px;
}

.mh-footer-copyright {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--vp-c-text-2);
}

/* ---- Motion: one authored moment, on the hero only -------------- */
@keyframes mh-rise {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

.mh-hero-heading {
  animation: mh-rise 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
}

.mh-hero-lede {
  animation: mh-rise 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.08s both;
}

.mh-hero .mh-actions {
  animation: mh-rise 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.24s both;
}

.mh-herodiff {
  animation: mh-rise 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.32s both;
}

@media (prefers-reduced-motion: reduce) {
  .mh-hero-heading,
  .mh-hero-lede,
  .mh-hero .mh-actions,
  .mh-herodiff {
    animation: none;
  }
}
</style>
