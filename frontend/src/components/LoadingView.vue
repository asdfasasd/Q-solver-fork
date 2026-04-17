<template>
  <div class="loading-container">
    <div class="loader-content">
      <div class="orb-container">
        <div class="orb"></div>
        <div class="ring ring-1"></div>
        <div class="ring ring-2"></div>
      </div>
      <div class="loading-text">
        <span class="text">深度思考中</span>
        <span class="timer">{{ formattedTime }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'

const startTime = ref(Date.now())
const currentTime = ref(Date.now())
let timerInterval = null

const formattedTime = computed(() => {
  const diff = currentTime.value - startTime.value
  const seconds = Math.floor(diff / 1000)
  const ms = Math.floor((diff % 1000) / 10)
  return `${seconds}.${ms.toString().padStart(2, '0')}s`
})

onMounted(() => {
  timerInterval = setInterval(() => {
    currentTime.value = Date.now()
  }, 30)
})

onUnmounted(() => {
  if (timerInterval) clearInterval(timerInterval)
})
</script>

<style scoped>
/* ========================================
   Loading Container
   ======================================== */

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  width: 100%;
  background: transparent;
  animation: containerFadeIn 0.5s ease-out;
}

.loader-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-6);
}

/* ========================================
   Orb Animation
   ======================================== */

.orb-container {
  position: relative;
  width: 90px;
  height: 90px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.orb {
  width: 36px;
  height: 36px;
  background: radial-gradient(circle at 30% 30%, #a7f3d0, var(--color-primary));
  border-radius: 50%;
  box-shadow: 0 0 30px rgba(16, 185, 129, 0.6), 
              0 0 60px rgba(16, 185, 129, 0.3);
  animation: orbBreathe 3s ease-in-out infinite;
  z-index: 2;
}

.ring {
  position: absolute;
  border-radius: 50%;
  border: 2px solid transparent;
  border-top-color: rgba(16, 185, 129, 0.5);
  border-right-color: rgba(16, 185, 129, 0.2);
}

.ring-1 {
  width: 60px;
  height: 60px;
  animation: ringSpin 2.5s linear infinite;
}

.ring-2 {
  width: 84px;
  height: 84px;
  border-top-color: rgba(16, 185, 129, 0.3);
  border-left-color: rgba(16, 185, 129, 0.5);
  animation: ringSpin 3.5s linear infinite reverse;
}

/* ========================================
   Loading Text
   ======================================== */

.loading-text {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-family: var(--font-sans);
}

.text {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.5px;
}

.timer {
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-primary);
  background: var(--color-primary-light);
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-sm);
  border: 1px solid rgba(16, 185, 129, 0.25);
}

/* ========================================
   Animations
   ======================================== */

@keyframes containerFadeIn {
  from { 
    opacity: 0; 
    transform: translateY(10px); 
  }
  to { 
    opacity: 1; 
    transform: translateY(0); 
  }
}

@keyframes orbBreathe {
  0%, 100% { 
    transform: scale(0.92); 
    box-shadow: 0 0 25px rgba(16, 185, 129, 0.5), 
                0 0 50px rgba(16, 185, 129, 0.2);
  }
  50% { 
    transform: scale(1.08); 
    box-shadow: 0 0 40px rgba(16, 185, 129, 0.7), 
                0 0 80px rgba(16, 185, 129, 0.35);
  }
}

@keyframes ringSpin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
