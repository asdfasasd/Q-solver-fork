import { reactive, ref } from 'vue'
import { StartRecordingKey } from '../../wailsjs/go/main/App'

export function useShortcuts() {
  const shortcuts = reactive({})
  const tempShortcuts = reactive({})
  const recordingAction = ref(null)
  const recordingText = ref('')

  // 同步检测 macOS - 使用 navigator.platform (可靠且同步)
  const isMacOS = ref(
    typeof navigator !== 'undefined' &&
    (navigator.platform.toLowerCase().includes('mac') ||
      navigator.userAgent.toLowerCase().includes('mac'))
  )

  console.log('[useShortcuts] Platform detection - isMacOS:', isMacOS.value, 'navigator.platform:', navigator?.platform)

  // 快捷键动作列表，包含 Windows 和 macOS 默认值
  const shortcutActions = [
    { action: 'solve', label: '一键解题', default: 'F8', macDefault: '⌘1' },
    { action: 'toggle', label: '隐藏/展示', default: 'F9', macDefault: '⌘2' },
    { action: 'clickthrough', label: '鼠标穿透', default: 'F10', macDefault: '⌘3' },
    { action: 'move_up', label: '向上移动', default: 'Alt+↑', macDefault: '⌘⌥↑' },
    { action: 'move_down', label: '向下移动', default: 'Alt+↓', macDefault: '⌘⌥↓' },
    { action: 'move_left', label: '向左移动', default: 'Alt+←', macDefault: '⌘⌥←' },
    { action: 'move_right', label: '向右移动', default: 'Alt+→', macDefault: '⌘⌥→' },
    { action: 'scroll_up', label: '向上滚动', default: 'Alt+PgUp', macDefault: '⌘⌥⇧↑' },
    { action: 'scroll_down', label: '向下滚动', default: 'Alt+PgDn', macDefault: '⌘⌥⇧↓' },
  ]

  // 获取当前平台的默认快捷键
  function getDefaultKey(action) {
    const item = shortcutActions.find(a => a.action === action)
    if (!item) return ''
    return isMacOS.value ? item.macDefault : item.default
  }

  function recordKey(action) {
    // macOS 上直接显示提示，不触发录制
    if (isMacOS.value) {
      console.log('[useShortcuts] macOS detected, skipping recording')
      // 不设置 recordingAction，不调用后端
      return
    }
    recordingAction.value = action
    recordingText.value = '请按键...'
    StartRecordingKey(action)
  }

  return {
    shortcuts,
    tempShortcuts,
    recordingAction,
    recordingText,
    shortcutActions,
    recordKey,
    isMacOS,
    getDefaultKey
  }
}
