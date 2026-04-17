<template>
  <div class="top-bar-wrapper" style="--wails-draggable:drag">
    <div class="top-bar">
      <div class="control-group" :class="{ active: activeButtons.toggle || isStealthMode }" style="--wails-draggable:no-drag">
        <span class="key-hint">{{ shortcuts.toggle?.keyName || (isMacOS ? '⌘2' : 'F9') }}</span>
        <span class="label">隐藏/展示</span>
      </div>
      <div class="control-group" :class="{ active: activeButtons.solve }" style="--wails-draggable:no-drag">
        <span class="key-hint">{{ shortcuts.solve?.keyName || (isMacOS ? '⌘1' : 'F8') }}</span>
        <span class="label">一键解题</span>
      </div>
      <div class="control-group" :class="{ active: activeButtons.clickthrough || isClickThrough }" style="--wails-draggable:no-drag">
        <span class="key-hint">{{ shortcuts.clickthrough?.keyName || (isMacOS ? '⌘3' : 'F10') }}</span>
        <span class="label">鼠标穿透</span>
      </div>
      <div class="control-group" style="cursor: default;">
        <span class="key-hint">{{ isMacOS ? '⌘⌥+Move' : 'Alt+Move' }}</span>
        <span class="label">移动/滚动</span>
      </div>
      <div class="divider"></div>
      <div class="control-group" @click="$emit('openSettings')" style="cursor: pointer; --wails-draggable:no-drag"
        @mouseenter="showSettingsTooltip" @mouseleave="hideSettingsTooltip" ref="settingsBtnRef">
        <span class="label">⚙️ 设置</span>
      </div>
      <div class="divider"></div>
      <div class="status-group" ref="statusGroupRef" @click="toggleStatusPanel" style="--wails-draggable:no-drag">
        <span class="status-dot" :class="statusClass"></span>
        <span class="status-label">状态</span>
      </div>
      <div class="divider"></div>
      <div class="control-group" style="cursor: pointer; --wails-draggable:no-drag" @click="$emit('quit')">
        <span class="label">❌ 退出</span>
      </div>
    </div>
  </div>

  <Teleport to="body">
    <!-- 状态面板 - 点击显示 -->
    <Transition name="panel-fade">
      <div class="status-panel" v-if="showStatusPanel" :style="panelStyle" @click.stop>
        <div class="panel-header">
          <span class="panel-title">运行状态</span>
          <button class="panel-close" @click="showStatusPanel = false">✕</button>
        </div>
        <div class="panel-body">
          <div class="status-row">
            <span class="row-label">当前状态</span>
            <span class="row-value" :class="statusValueClass">{{ statusText }}</span>
          </div>
          <div class="status-row">
            <span class="row-label">API 连接</span>
            <span class="row-value" :class="apiStatusClass">
              {{ apiStatusText }}
            </span>
          </div>
          <div class="status-row">
            <span class="row-label">使用模型</span>
            <span class="row-value model">{{ settings.model || '未设置' }}</span>
          </div>
          <div class="status-row">
            <span class="row-label">隐身模式</span>
            <span class="row-value" :class="isStealthMode ? 'success' : 'error'">
              {{ isStealthMode ? '已开启' : '已关闭' }}
            </span>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 设置按钮 tooltip -->
    <div class="settings-tooltip" v-if="showSettingsTip" :style="settingsTooltipStyle">
      <div class="tooltip-warning">
        ⚠️ 注意：打开设置将获取焦点<br>录屏期间请勿操作
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  shortcuts: Object,
  activeButtons: Object,
  isClickThrough: Boolean,
  statusIcon: String,
  statusText: String,

  settings: Object,
  isStealthMode: Boolean,
  isMacOS: Boolean
})

defineEmits(['openSettings', 'quit'])

// 根据状态文本计算状态类名
const statusClass = computed(() => {
  const text = props.statusText || ''
  if (text === '已连接' || text === '就绪' || text === '解题完成') return 'connected'
  if (text.includes('未配置')) return 'unconfigured'
  if (text.includes('无效') || text.includes('Key')) return 'invalid-key'
  if (text.includes('失败') || text.includes('出错')) return 'disconnected'
  if (text.includes('思考') || text.includes('复制')) return 'connected'
  return 'unconfigured'
})

// 状态值样式
const statusValueClass = computed(() => {
  const text = props.statusText || ''
  if (text === '已连接' || text === '就绪' || text === '解题完成' || text.includes('思考') || text.includes('复制')) return 'success'
  if (text.includes('未配置')) return 'warning'
  return 'error'
})

// API 状态文本
const apiStatusText = computed(() => {
  const text = props.statusText || ''
  if (text === '已连接' || text === '就绪' || text === '解题完成' || text.includes('思考') || text.includes('复制')) return '连接正常'
  if (text.includes('Key') || text.includes('无效')) return 'Key 无效'
  if (text.includes('失败')) return '连接失败'
  return '未配置'
})

// API 状态样式
const apiStatusClass = computed(() => {
  const text = props.statusText || ''
  if (text === '已连接' || text === '就绪' || text === '解题完成' || text.includes('思考') || text.includes('复制')) return 'success'
  if (text.includes('未配置')) return 'warning'
  return 'error'
})

// 状态面板
const showStatusPanel = ref(false)
const statusGroupRef = ref(null)
const panelStyle = reactive({ top: '0px', left: '0px' })

function toggleStatusPanel() {
  if (showStatusPanel.value) {
    showStatusPanel.value = false
    return
  }
  
  if (statusGroupRef.value) {
    const rect = statusGroupRef.value.getBoundingClientRect()
    panelStyle.top = `${rect.bottom + 8}px`
    // 确保面板不会超出右边界
    const panelWidth = 220
    let left = rect.left + rect.width / 2 - panelWidth / 2
    if (left + panelWidth > window.innerWidth - 10) {
      left = window.innerWidth - panelWidth - 10
    }
    if (left < 10) left = 10
    panelStyle.left = `${left}px`
    showStatusPanel.value = true
  }
}

// 点击外部关闭面板
function handleClickOutside(e) {
  if (showStatusPanel.value && statusGroupRef.value && !statusGroupRef.value.contains(e.target)) {
    showStatusPanel.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// 设置按钮 tooltip
const showSettingsTip = ref(false)
const settingsBtnRef = ref(null)
const settingsTooltipStyle = reactive({ top: '0px', left: '0px' })

function showSettingsTooltip() {
  if (settingsBtnRef.value) {
    const rect = settingsBtnRef.value.getBoundingClientRect()
    settingsTooltipStyle.top = `${rect.bottom + 10}px`
    settingsTooltipStyle.left = `${rect.left + rect.width / 2}px`
    showSettingsTip.value = true
  }
}

function hideSettingsTooltip() {
  showSettingsTip.value = false
}
</script>

<style scoped>
/* ========================================
   TopBar Styles
   ======================================== */

.top-bar-wrapper {
  pointer-events: auto;
}

/* ========================================
   Status Group - 简洁文本样式
   ======================================== */

.status-group {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.status-group:hover {
  background: var(--bg-hover);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}

.status-dot.connected {
  background: var(--color-success);
  box-shadow: 0 0 8px var(--color-success);
}

.status-dot.unconfigured {
  background: var(--color-warning);
  box-shadow: 0 0 8px var(--color-warning);
}

.status-dot.invalid-key,
.status-dot.disconnected {
  background: var(--color-error);
  box-shadow: 0 0 8px var(--color-error);
}

.status-label {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  font-weight: 500;
}

/* ========================================
   Status Panel - 点击弹出面板
   ======================================== */

.status-panel {
  position: fixed;
  width: 220px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  backdrop-filter: blur(20px);
  z-index: 99999;
  overflow: hidden;
  pointer-events: auto;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-subtle);
}

.panel-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.panel-close {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-tertiary);
  cursor: pointer;
  font-size: 12px;
  transition: all var(--transition-fast);
}

.panel-close:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.panel-body {
  padding: 10px 14px;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-subtle);
}

.status-row:last-child {
  border-bottom: none;
}

.row-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.row-value {
  font-size: var(--text-xs);
  font-weight: 600;
  font-family: var(--font-mono);
}

.row-value.success {
  color: var(--color-success);
}

.row-value.warning {
  color: var(--color-warning);
}

.row-value.error {
  color: var(--color-error);
}

.row-value.model {
  color: var(--text-primary);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Panel 过渡动画 */
.panel-fade-enter-active,
.panel-fade-leave-active {
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.panel-fade-enter-from,
.panel-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* ========================================
   Settings Tooltip
   ======================================== */

.settings-tooltip {
  position: fixed;
  transform: translateX(-50%);
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.15) 0%, var(--bg-elevated) 100%);
  border: 1px solid rgba(245, 158, 11, 0.4);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  z-index: 99999;
  box-shadow: var(--shadow-lg), 0 0 20px rgba(245, 158, 11, 0.1);
  backdrop-filter: blur(16px);
  pointer-events: none;
  animation: tooltipIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  text-align: center;
}

.settings-tooltip::before {
  content: '';
  position: absolute;
  top: -6px;
  left: 50%;
  transform: translateX(-50%);
  border-width: 0 6px 6px 6px;
  border-style: solid;
  border-color: transparent transparent rgba(245, 158, 11, 0.4) transparent;
}

.tooltip-warning {
  color: var(--color-warning);
  font-size: var(--text-sm);
  line-height: 1.6;
  font-weight: 600;
}

@keyframes tooltipIn {
  from {
    opacity: 0;
    transform: translate(-50%, -6px);
  }
  to {
    opacity: 1;
    transform: translate(-50%, 0);
  }
}
</style>
