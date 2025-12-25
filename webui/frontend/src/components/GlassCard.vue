<script setup lang="ts">
interface Props {
  padding?: 'none' | 'sm' | 'md' | 'lg'
  hover?: boolean
}

withDefaults(defineProps<Props>(), {
  padding: 'md',
  hover: true
})
</script>

<template>
  <div
    class="glass-card"
    :class="[`padding-${padding}`, { 'has-hover': hover }]"
  >
    <slot />
  </div>
</template>

<style scoped>
.glass-card {
  background: var(--bg-glass-weak);
  backdrop-filter: var(--blur-glass);
  -webkit-backdrop-filter: var(--blur-glass);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-glass-normal);
  transition: all var(--transition-base);
}

.glass-card.has-hover:hover {
  background: var(--bg-glass-normal);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
  border-color: var(--border-glass-strong);
}

.glass-card.padding-none {
  padding: 0;
}

.glass-card.padding-sm {
  padding: var(--spacing-md);
}

.glass-card.padding-md {
  padding: var(--spacing-lg);
}

.glass-card.padding-lg {
  padding: var(--spacing-xl);
}
</style>
