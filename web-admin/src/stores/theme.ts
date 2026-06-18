import { reactive, computed } from 'vue'

const STORAGE_KEY = 'panda-sleep-admin:theme'

export interface ThemePreset {
  id: string
  name: string
  color: string
}

export const THEMES: ThemePreset[] = [
  { id: 'purple', name: '星夜紫', color: '#c3f65c' },
  { id: 'blue',   name: '月光蓝', color: '#3B82F6' },
  { id: 'indigo', name: '星河靛', color: '#6366F1' },
  { id: 'teal',   name: '极光绿', color: '#14B8A6' },
  { id: 'pink',   name: '流星粉', color: '#EC4899' },
]

// ─── Color math helpers ────────────────────────────────────────────────────────

function parseHex(hex: string): [number, number, number] {
  return [parseInt(hex.slice(1, 3), 16), parseInt(hex.slice(3, 5), 16), parseInt(hex.slice(5, 7), 16)]
}

function toHexStr(r: number, g: number, b: number): string {
  return '#' + [r, g, b]
    .map(v => Math.min(255, Math.max(0, Math.round(v))).toString(16).padStart(2, '0'))
    .join('')
}

/** El Plus light-N: channel + (255 - channel) * N * 0.1 */
function elLight(hex: string, n: number): string {
  const [r, g, b] = parseHex(hex)
  return toHexStr(r + (255 - r) * n * 0.1, g + (255 - g) * n * 0.1, b + (255 - b) * n * 0.1)
}

/** El Plus dark-N: channel * (1 - N * 0.1) */
function elDark(hex: string, n: number): string {
  const [r, g, b] = parseHex(hex)
  return toHexStr(r * (1 - n * 0.1), g * (1 - n * 0.1), b * (1 - n * 0.1))
}

function mixWithWhite(hex: string, ratio: number): string {
  const [r, g, b] = parseHex(hex)
  return toHexStr(r + (255 - r) * ratio, g + (255 - g) * ratio, b + (255 - b) * ratio)
}

function hexToRgba(hex: string, alpha: number): string {
  const [r, g, b] = parseHex(hex)
  return `rgba(${r},${g},${b},${alpha})`
}

function computeGradient(hex: string, dark: boolean): string {
  const [r, g, b] = parseHex(hex)
  const rgba = (a: number) => `rgba(${r},${g},${b},${a})`

  if (dark) {
    // 更深的夜空底色，叠加主色星云，整体更接近星空环境
    const s0 = toHexStr(r * 0.04, g * 0.05, b * 0.10)
    const s1 = toHexStr(r * 0.09, g * 0.10, b * 0.19)
    const s2 = toHexStr(r * 0.05, g * 0.06, b * 0.12)
    const s3 = '#050814'
    return [
      `radial-gradient(ellipse 78% 70% at 88% 10%, ${rgba(0.24)} 0%, transparent 58%)`,
      `radial-gradient(ellipse 62% 78% at 8% 92%, ${rgba(0.18)} 0%, transparent 56%)`,
      `radial-gradient(ellipse 44% 50% at 52% 48%, ${rgba(0.10)} 0%, transparent 52%)`,
      `linear-gradient(180deg, ${s3} 0%, ${s0} 18%, ${s1} 58%, ${s2} 100%)`,
    ].join(', ')
  }

  // Light: 中心柔白，四周由主题色做云朵状渐变环绕
  // 构图：对角线主色云 + 次对角线辅色云 + 上下小云气，让整体更自然、不对称但保持平衡
  const centerTint = mixWithWhite(hex, 0.97)
  const midTint = mixWithWhite(hex, 0.92)
  const edgeTint = mixWithWhite(hex, 0.82)
  return [
    // 主云：左上 + 右下（构图主轴）
    `radial-gradient(ellipse 58% 46% at 10% 14%, ${rgba(0.22)} 0%, transparent 75%)`,
    `radial-gradient(ellipse 58% 46% at 90% 86%, ${rgba(0.22)} 0%, transparent 75%)`,
    // 辅云：右上 + 左下（平衡构图）
    `radial-gradient(ellipse 40% 34% at 94% 18%, ${rgba(0.14)} 0%, transparent 78%)`,
    `radial-gradient(ellipse 40% 34% at 6% 82%, ${rgba(0.14)} 0%, transparent 78%)`,
    // 顶/底边云气：拉长边缘柔和过渡
    `radial-gradient(ellipse 42% 18% at 50% -4%, ${rgba(0.10)} 0%, transparent 82%)`,
    `radial-gradient(ellipse 42% 18% at 50% 104%, ${rgba(0.10)} 0%, transparent 82%)`,
    // 底色：中心略偏上（48%）带"天空感"，三段径向渐变平滑过渡
    `radial-gradient(ellipse 140% 130% at 50% 48%, ${centerTint} 0%, ${midTint} 55%, ${edgeTint} 100%)`,
  ].join(', ')
}

// ─── localStorage ─────────────────────────────────────────────────────────────

export type ThemeMode = 'dark' | 'light'
export type ThemePreference = ThemeMode | 'system'

interface StoredTheme {
  preference: ThemePreference
  primaryColor: string
}

function isPreference(value: unknown): value is ThemePreference {
  return value === 'system' || value === 'dark' || value === 'light'
}

function loadStored(): StoredTheme | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<StoredTheme> & { mode?: unknown }
    // 兼容旧版本只写 mode 字段的记录
    const pref: unknown = parsed.preference ?? parsed.mode
    if (!isPreference(pref)) return null
    return {
      preference: pref,
      primaryColor: typeof parsed.primaryColor === 'string' ? parsed.primaryColor : THEMES[0].color,
    }
  } catch {
    return null
  }
}

function saveStored(preference: ThemePreference, primaryColor: string) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ preference, primaryColor }))
  } catch {
    /* noop */
  }
}

// ─── System preference watcher ────────────────────────────────────────────────

function systemPrefersDark(): boolean {
  return typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-color-scheme: dark)').matches
}

function resolveMode(pref: ThemePreference): ThemeMode {
  if (pref === 'system') return systemPrefersDark() ? 'dark' : 'light'
  return pref
}

let mediaQuery: MediaQueryList | null = null
let mediaHandler: ((e: MediaQueryListEvent) => void) | null = null

function bindSystemWatcher() {
  if (mediaQuery || typeof window === 'undefined' || !window.matchMedia) return
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaHandler = (e) => {
    if (state.preference !== 'system') return
    const next: ThemeMode = e.matches ? 'dark' : 'light'
    if (state.mode !== next) {
      state.mode = next
      applyTheme()
    }
  }
  mediaQuery.addEventListener('change', mediaHandler)
}

function unbindSystemWatcher() {
  if (mediaQuery && mediaHandler) mediaQuery.removeEventListener('change', mediaHandler)
  mediaQuery = null
  mediaHandler = null
}

// ─── Reactive state ────────────────────────────────────────────────────────────

const stored = loadStored()
const initialPreference: ThemePreference = stored?.preference ?? 'dark'

const state = reactive({
  preference: initialPreference,
  mode: resolveMode(initialPreference) as ThemeMode,
  primaryColor: stored?.primaryColor ?? THEMES[0].color,
  collapsed: false,
  menuTitle: '数据概览',
})

if (state.preference === 'system') bindSystemWatcher()

// ─── applyTheme ────────────────────────────────────────────────────────────────

export function applyTheme() {
  const hex = state.primaryColor
  const isDark = state.mode === 'dark'
  const root = document.documentElement

  // El Plus primary color system — all derived vars so El Plus components follow the theme
  root.style.setProperty('--el-color-primary',         hex)
  root.style.setProperty('--el-color-primary-dark-2',  elDark(hex, 2))
  root.style.setProperty('--el-color-primary-light-3', elLight(hex, 3))
  root.style.setProperty('--el-color-primary-light-5', elLight(hex, 5))
  root.style.setProperty('--el-color-primary-light-7', elLight(hex, 7))
  root.style.setProperty('--el-color-primary-light-8', elLight(hex, 8))
  root.style.setProperty('--el-color-primary-light-9', elLight(hex, 9))

  // Custom vars used by our components
  root.style.setProperty('--color-primary',       hex)
  root.style.setProperty('--color-primary-light', mixWithWhite(hex, 0.3))
  root.style.setProperty('--color-glow',          hexToRgba(hex, 0.35))
  root.style.setProperty('--gradient-bg',         computeGradient(hex, isDark))

  if (isDark) {
    root.style.setProperty('--glass-bg',     'rgba(255,255,255,0.055)')
    root.style.setProperty('--glass-border', 'rgba(255,255,255,0.10)')
    root.style.setProperty('--glass-hover',  'rgba(255,255,255,0.09)')
    root.style.setProperty('--glass-active', 'rgba(255,255,255,0.14)')
    root.style.setProperty('--glass-card',   'rgba(255,255,255,0.05)')
    root.style.setProperty('--text-primary',   '#F1F5F9')
    root.style.setProperty('--text-secondary', '#94A3B8')
    root.style.setProperty('--text-muted',     '#64748B')
    root.style.setProperty('--divider', 'rgba(255,255,255,0.08)')
    // 星空层：暗色模式下增强星点、星雾和冷色夜空氛围
    root.style.setProperty('--star-color',        'rgba(255,255,255,0.92)')
    root.style.setProperty('--star-soft-color',   'rgba(186, 210, 255, 0.55)')
    root.style.setProperty('--star-opacity-main', '0.85')
    root.style.setProperty('--star-opacity-soft', '0.50')
    root.style.setProperty('--aurora-opacity',    '0.30')
    root.style.setProperty('--night-haze',        hexToRgba(hex, 0.18))
    // Clouds: white wisps visible on dark; nebulae brighter
    root.style.setProperty('--cloud-nebula-op',   '0.24')
    root.style.setProperty('--cloud-wisp-color',  'rgba(255,255,255,0.08)')
    root.classList.add('dark')
    root.classList.remove('light')
  } else {
    root.style.setProperty('--glass-bg',     'rgba(255,255,255,0.60)')
    root.style.setProperty('--glass-border', 'rgba(255,255,255,0.80)')
    root.style.setProperty('--glass-hover',  'rgba(255,255,255,0.78)')
    root.style.setProperty('--glass-active', 'rgba(255,255,255,0.88)')
    root.style.setProperty('--glass-card',   'rgba(255,255,255,0.25)')
    root.style.setProperty('--text-primary',   '#1E1B4B')
    root.style.setProperty('--text-secondary', '#4C4B7A')
    root.style.setProperty('--text-muted',     '#6B7280')
    root.style.setProperty('--divider', 'rgba(0,0,0,0.07)')
    root.style.setProperty('--star-color',        'rgba(255,255,255,0)')
    root.style.setProperty('--star-soft-color',   'rgba(255,255,255,0)')
    root.style.setProperty('--star-opacity-main', '0')
    root.style.setProperty('--star-opacity-soft', '0')
    root.style.setProperty('--aurora-opacity',    '0')
    root.style.setProperty('--night-haze',        'rgba(255,255,255,0)')
    // Clouds: color-tinted wisps on light; nebulae softer
    root.style.setProperty('--cloud-nebula-op',   '0.13')
    root.style.setProperty('--cloud-wisp-color',  hexToRgba(hex, 0.07))
    root.classList.add('light')
    root.classList.remove('dark')
  }

  saveStored(state.preference, state.primaryColor)
}

// ─── Public API ────────────────────────────────────────────────────────────────

export function setCustomPrimary(hex: string) {
  state.primaryColor = hex
  applyTheme()
}

export function useTheme() {
  const currentPreset = computed(() => ({
    color: state.primaryColor,
    light: mixWithWhite(state.primaryColor, 0.3),
    glow:  hexToRgba(state.primaryColor, 0.35),
  }))

  // 保留显式 dark/light 入口，同时新增跟随系统的偏好入口
  function setPreference(pref: ThemePreference) {
    state.preference = pref
    state.mode = resolveMode(pref)
    if (pref === 'system') {
      bindSystemWatcher()
    } else {
      unbindSystemWatcher()
    }
    applyTheme()
  }

  function setMode(mode: ThemeMode) {
    setPreference(mode)
  }

  function setTheme(id: string) {
    const preset = THEMES.find(t => t.id === id)
    if (preset) { state.primaryColor = preset.color; applyTheme() }
  }

  function toggleCollapse() { state.collapsed = !state.collapsed }

  function setMenuTitle(title: string) { state.menuTitle = title }

  return { state, currentPreset, THEMES, setMode, setPreference, setTheme, toggleCollapse, setMenuTitle }
}
