<template>
  <div 
    class="resize-handle" 
    @mousedown="startResize"
    title="拖动调整窗口大小"
  >
    <svg class="resize-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M22 22L12 22M22 22L22 12M22 22L12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </div>
</template>

<script setup>
import { WindowGetSize, WindowSetSize } from '../../wailsjs/runtime/runtime'

const MIN_WIDTH = 840
const MIN_HEIGHT = 700
const MAX_WIDTH = 1024
const MAX_HEIGHT = 768

let startX = 0
let startY = 0
let startWidth = 0
let startHeight = 0

async function startResize(e) {
  e.preventDefault()
  
  // 获取当前窗口大小
  const size = await WindowGetSize()
  startWidth = size.w
  startHeight = size.h
  startX = e.screenX
  startY = e.screenY
  
  // 添加全局事件监听
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  
  // 添加拖动时的样式
  document.body.style.cursor = 'nwse-resize'
  document.body.style.userSelect = 'none'
}

function onResize(e) {
  const deltaX = e.screenX - startX
  const deltaY = e.screenY - startY
  
  // 计算新尺寸，确保在最小值和最大值之间
  const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth + deltaX))
  const newHeight = Math.min(MAX_HEIGHT, Math.max(MIN_HEIGHT, startHeight + deltaY))
  
  // 设置窗口大小
  WindowSetSize(newWidth, newHeight)
}

import { UpdateSettings, GetSettings } from '../../wailsjs/go/main/App'

async function stopResize() {
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  
  // 恢复样式
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  
  // 保存窗口尺寸到配置
  try {
    const size = await WindowGetSize()
    const settings = await GetSettings()
    settings.windowWidth = size.w
    settings.windowHeight = size.h
    await UpdateSettings(JSON.stringify(settings))
  } catch (e) {
    console.error('保存窗口尺寸失败:', e)
  }
}
</script>

<style scoped>
.resize-handle {
  position: fixed;
  bottom: 4px;
  right: 4px;
  width: 20px;
  height: 20px;
  cursor: nwse-resize;
  z-index: 99999;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
  pointer-events: auto;
  opacity: 0.4;
}

.resize-handle:hover {
  opacity: 1;
  background: var(--bg-hover);
}

.resize-handle:active {
  opacity: 1;
  background: var(--bg-active);
}

.resize-icon {
  width: 14px;
  height: 14px;
  color: var(--text-tertiary);
  transition: color var(--transition-fast);
}

.resize-handle:hover .resize-icon {
  color: var(--text-secondary);
}
</style>
