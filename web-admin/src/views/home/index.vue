<script setup lang="ts">
defineOptions({ name: 'dashboard' })
import { computed } from 'vue'
import { useTheme } from '@/stores/theme'

const { currentPreset } = useTheme()

const stats = [
  {
    label: '总用户数',
    value: '28,451',
    trend: '+12.5%',
    up: true,
    icon: 'M16 11c1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 3-1.34 3-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z',
    sub: '本月新增 +1,234',
  },
  {
    label: '今日睡眠记录',
    value: '8,923',
    trend: '+5.2%',
    up: true,
    icon: 'M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z',
    sub: '较昨日 +442 条',
  },
  {
    label: '平均睡眠质量',
    value: '86%',
    trend: '+3.1%',
    up: true,
    icon: 'M16 6l2.29 2.29-4.88 4.88-4-4L2 16.59 3.41 18l6-6 4 4 6.3-6.29L22 12V6z',
    sub: '优秀：23,481 人',
  },
  {
    label: '活跃设备',
    value: '12,340',
    trend: '-1.2%',
    up: false,
    icon: 'M17 1.01L7 1c-1.1 0-2 .9-2 2v18c0 1.1.9 2 2 2h10c1.1 0 2-.9 2-2V3c0-1.1-.9-1.99-2-1.99zM17 19H7V5h10v14z',
    sub: '在线 11,820 台',
  },
]

const weekBars = [
  { day: '周一', quality: 72, duration: 6.5 },
  { day: '周二', quality: 85, duration: 7.2 },
  { day: '周三', quality: 78, duration: 6.8 },
  { day: '周四', quality: 91, duration: 8.0 },
  { day: '周五', quality: 68, duration: 6.2 },
  { day: '周六', quality: 94, duration: 8.5 },
  { day: '周日', quality: 88, duration: 7.8 },
]

const sleepStages = [
  { label: '深度睡眠', pct: 22, color: 'var(--color-primary)' },
  { label: '浅度睡眠', pct: 45, color: 'var(--color-primary-light)' },
  { label: 'REM 睡眠', pct: 20, color: '#34D399' },
  { label: '清醒时段', pct: 13, color: 'var(--text-muted)' },
]

const recentRecords = [
  { user: '张 **', time: '22:48', duration: '7h 32m', quality: 92, status: '优秀' },
  { user: '李 **', time: '23:15', duration: '6h 55m', quality: 78, status: '良好' },
  { user: '王 **', time: '00:02', duration: '5h 48m', quality: 61, status: '一般' },
  { user: '赵 **', time: '22:30', duration: '8h 10m', quality: 96, status: '优秀' },
  { user: '陈 **', time: '23:50', duration: '7h 05m', quality: 84, status: '良好' },
]

function qualityColor(score: number) {
  if (score >= 90) return '#10B981'
  if (score >= 75) return '#3B82F6'
  if (score >= 60) return '#F59E0B'
  return '#EF4444'
}

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '深夜好'
  if (h < 12) return '早上好'
  if (h < 18) return '下午好'
  return '晚上好'
})
</script>

<template>
  <div class="dashboard">
    <!-- Welcome banner -->
    <div class="welcome-banner glass-card">
      <div class="welcome-text">
        <p class="welcome-greeting">{{ greeting }}，管理员 👋</p>
        <p class="welcome-sub">今天是个好日子，查看用户的睡眠健康数据吧</p>
      </div>
      <div class="welcome-orb" :style="{ background: `radial-gradient(circle at 30% 50%, ${currentPreset.color}55 0%, transparent 70%)` }" aria-hidden="true" />
    </div>

    <!-- Stat cards -->
    <div class="stats-grid">
      <div v-for="stat in stats" :key="stat.label" class="stat-card glass-card">
        <div class="stat-icon" :style="{ background: `linear-gradient(135deg, ${currentPreset.color}22, ${currentPreset.light}33)`, color: currentPreset.color }">
          <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20" aria-hidden="true">
            <path :d="stat.icon" />
          </svg>
        </div>
        <div class="stat-body">
          <p class="stat-label">{{ stat.label }}</p>
          <p class="stat-value">{{ stat.value }}</p>
          <div class="stat-footer">
            <span class="stat-trend" :class="stat.up ? 'up' : 'down'">
              <svg viewBox="0 0 24 24" fill="currentColor" width="10" height="10" aria-hidden="true">
                <path :d="stat.up ? 'M7 14l5-5 5 5z' : 'M7 10l5 5 5-5z'" />
              </svg>
              {{ stat.trend }}
            </span>
            <span class="stat-sub">{{ stat.sub }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Charts row -->
    <div class="charts-row">
      <!-- Weekly sleep quality bar chart -->
      <div class="chart-card glass-card">
        <div class="card-header">
          <h3 class="card-title">近7日睡眠质量</h3>
          <span class="card-badge">本周</span>
        </div>
        <div class="bar-chart" role="img" aria-label="近7日睡眠质量柱状图">
          <div v-for="bar in weekBars" :key="bar.day" class="bar-group">
            <div class="bar-value">{{ bar.quality }}%</div>
            <div class="bar-wrap">
              <div
                class="bar-fill"
                :style="{
                  height: bar.quality + '%',
                  background: `linear-gradient(180deg, ${currentPreset.light} 0%, ${currentPreset.color} 100%)`,
                  boxShadow: `0 4px 16px ${currentPreset.glow}`,
                }"
              />
            </div>
            <div class="bar-label">{{ bar.day }}</div>
            <div class="bar-duration">{{ bar.duration }}h</div>
          </div>
        </div>
      </div>

      <!-- Sleep stage distribution -->
      <div class="chart-card glass-card">
        <div class="card-header">
          <h3 class="card-title">睡眠阶段分布</h3>
          <span class="card-badge">今日均值</span>
        </div>
        <div class="stage-chart">
          <!-- Segmented bar -->
          <div class="stage-bar" role="img" aria-label="睡眠阶段分布">
            <div
              v-for="stage in sleepStages"
              :key="stage.label"
              class="stage-segment"
              :style="{ width: stage.pct + '%', background: stage.color }"
              :title="`${stage.label}: ${stage.pct}%`"
            />
          </div>
          <!-- Legend -->
          <div class="stage-legend">
            <div v-for="stage in sleepStages" :key="stage.label" class="legend-item">
              <span class="legend-dot" :style="{ background: stage.color }" />
              <span class="legend-label">{{ stage.label }}</span>
              <span class="legend-pct">{{ stage.pct }}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Recent records table -->
    <div class="table-card glass-card">
      <div class="card-header">
        <h3 class="card-title">最近睡眠记录</h3>
        <button class="view-all-btn" style="cursor:pointer">查看全部</button>
      </div>
      <el-table :data="recentRecords" style="width: 100%; background: transparent">
        <el-table-column label="用户" prop="user" />
        <el-table-column label="入睡时间" prop="time" />
        <el-table-column label="睡眠时长" prop="duration" />
        <el-table-column label="质量评分">
          <template #default="{ row }">
            <div class="quality-cell">
              <div class="quality-bar-bg">
                <div
                  class="quality-bar-fill"
                  :style="{ width: row.quality + '%', background: qualityColor(row.quality) }"
                />
              </div>
              <span class="quality-score" :style="{ color: qualityColor(row.quality) }">{{ row.quality }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态">
          <template #default="{ row }">
            <span class="status-tag" :style="{ color: qualityColor(row.quality), background: qualityColor(row.quality) + '22' }">
              {{ row.status }}
            </span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Welcome banner */
.welcome-banner {
  position: relative;
  overflow: hidden;
  padding: 28px 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.welcome-greeting {
  font-family: 'Raleway', sans-serif;
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.welcome-sub {
  font-size: 14px;
  color: var(--text-secondary);
}

.welcome-orb {
  position: absolute;
  right: -40px;
  top: -40px;
  width: 200px;
  height: 200px;
  border-radius: 50%;
  pointer-events: none;
}

/* Stats grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  padding: 20px;
  display: flex;
  align-items: flex-start;
  gap: 14px;
  cursor: default;
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.4s ease;
}

.stat-body { flex: 1; min-width: 0; }

.stat-label {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 4px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-value {
  font-family: 'Raleway', sans-serif;
  font-size: 26px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
  margin-bottom: 8px;
}

.stat-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.stat-trend {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
}

.stat-trend.up { color: #10B981; background: rgba(16,185,129,0.12); }
.stat-trend.down { color: #EF4444; background: rgba(239,68,68,0.12); }

.stat-sub {
  font-size: 11.5px;
  color: var(--text-muted);
}

/* Charts row */
.charts-row {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 16px;
}

.chart-card, .table-card {
  padding: 20px 24px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.card-title {
  font-family: 'Raleway', sans-serif;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.card-badge {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-primary-light);
  background: var(--glass-active);
  padding: 3px 8px;
  border-radius: 10px;
  border: 1px solid var(--glass-border);
}

/* Bar chart */
.bar-chart {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  height: 160px;
  padding: 0 4px;
}

.bar-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  height: 100%;
}

.bar-value {
  font-size: 10.5px;
  color: var(--text-muted);
  font-weight: 500;
}

.bar-wrap {
  flex: 1;
  width: 100%;
  background: var(--glass-bg);
  border-radius: 6px;
  display: flex;
  align-items: flex-end;
  overflow: hidden;
  border: 1px solid var(--glass-border);
}

.bar-fill {
  width: 100%;
  border-radius: 5px;
  transition: height 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  min-height: 4px;
}

.bar-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
}

.bar-duration {
  font-size: 10px;
  color: var(--text-muted);
  opacity: 0.7;
}

/* Stage chart */
.stage-chart { display: flex; flex-direction: column; gap: 20px; }

.stage-bar {
  display: flex;
  height: 12px;
  border-radius: 8px;
  overflow: hidden;
  gap: 2px;
}

.stage-segment {
  border-radius: 4px;
  transition: width 0.6s ease, background 0.4s ease;
}

.stage-legend {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.legend-label {
  flex: 1;
  font-size: 13px;
  color: var(--text-secondary);
}

.legend-pct {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

/* Table */
.quality-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quality-bar-bg {
  flex: 1;
  height: 6px;
  background: var(--glass-bg);
  border-radius: 3px;
  overflow: hidden;
  border: 1px solid var(--glass-border);
}

.quality-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.5s ease, background 0.4s ease;
}

.quality-score {
  font-size: 13px;
  font-weight: 600;
  width: 28px;
  text-align: right;
}

.status-tag {
  font-size: 11.5px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 10px;
}

.view-all-btn {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--color-primary-light);
  background: transparent;
  border: 1px solid var(--glass-border);
  padding: 4px 12px;
  border-radius: 8px;
  transition: all 0.18s ease;
  outline: none;
}

.view-all-btn:hover {
  background: var(--glass-hover);
  color: var(--text-primary);
}

.view-all-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--color-primary);
}
</style>
