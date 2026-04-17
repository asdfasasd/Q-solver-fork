<template>
  <div class="error-container">
    <div class="error-icon-large">{{ errorState.icon }}</div>
    <div class="error-title-large">{{ errorState.title }}</div>
    <div class="error-desc">{{ errorState.desc }}</div>
    
    <!-- 极简模式：只保留快捷键提示 -->
    <div class="error-shortcut-hint" style="margin-top: 20px;">
      按下 <span class="shortcut-tag">{{ solveShortcut }}</span> 重试
    </div>
  </div>
</template>

<script setup>
defineProps({
  errorState: Object,
  solveShortcut: String
})
</script>

<style scoped>
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: var(--space-5);
  animation: fadeInError 0.3s ease-out;
}

@keyframes fadeInError {
  from { opacity: 0; }
  to { opacity: 1; }
}

.error-icon-large {
  font-size: 64px;
  margin-bottom: var(--space-4);
  filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.3));
}

.error-title-large {
  font-size: var(--text-xl);
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: var(--space-2);
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.5);
}

.error-desc {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin-bottom: var(--space-6);
  max-width: 80%;
  line-height: 1.5;
}

.error-shortcut-hint {
  font-size: var(--text-sm);
  color: var(--text-tertiary);
  margin-bottom: var(--space-5);
  animation: pulseOpacity 2s infinite;
}

@keyframes pulseOpacity {
  0% { opacity: 0.6; }
  50% { opacity: 1; }
  100% { opacity: 0.6; }
}

.shortcut-tag {
  background: var(--color-primary-light);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-weight: bold;
  color: var(--color-primary);
  border: 1px solid rgba(16, 185, 129, 0.3);
}
</style>
