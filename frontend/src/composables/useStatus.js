import { ref, watch } from 'vue'

export function useStatus(settings) {
  const statusText = ref('就绪')
  const statusIcon = ref('📝')

  function resetStatus() {
    if (!settings.apiKey) {
      statusText.value = '未配置'
      statusIcon.value = '⚠️'
      return
    }

    // 有 API Key 时显示已连接
    statusText.value = '已连接'
    statusIcon.value = '✅'
  }
  
  function setConnected() {
    statusText.value = '已连接'
    statusIcon.value = '✅'
  }
  
  function setDisconnected() {
    statusText.value = '连接失败'
    statusIcon.value = '❌'
  }
  
  function setInvalidKey() {
    statusText.value = 'Key无效'
    statusIcon.value = '🚫'
  }

  // 监听 settings.apiKey 变化，自动更新状态
  watch(() => settings.apiKey, (newVal) => {
    resetStatus()
  }, { immediate: true })

  return {
    statusText,
    statusIcon,
    resetStatus,
    setConnected,
    setDisconnected,
    setInvalidKey
  }
}
