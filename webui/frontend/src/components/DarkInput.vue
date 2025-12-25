<script setup lang="ts">
interface Props {
  modelValue?: string
  placeholder?: string
  type?: string
  disabled?: boolean
  size?: 'large' | 'default' | 'small'
  showPassword?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  size: 'default',
  showPassword: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const updateValue = (value: string) => {
  emit('update:modelValue', value)
}
</script>

<template>
  <el-input
    :model-value="modelValue"
    @update:model-value="updateValue"
    :placeholder="placeholder"
    :type="type"
    :disabled="disabled"
    :size="size"
    :show-password="showPassword"
    class="dark-input"
  >
    <template v-for="(_, name) in $slots" #[name]="slotProps">
      <slot :name="name" v-bind="slotProps || {}" />
    </template>
  </el-input>
</template>

<style scoped>
.dark-input :deep(.el-input__wrapper) {
  background: var(--bg-glass-weak);
  border: 1px solid var(--border-glass-normal);
  box-shadow: none;
  transition: all var(--transition-base);
}

.dark-input :deep(.el-input__wrapper:hover) {
  border-color: var(--border-glass-strong);
}

.dark-input :deep(.el-input__wrapper.is-focus) {
  background: var(--bg-glass-normal);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.dark-input :deep(.el-input__inner) {
  color: var(--text-primary);
}

.dark-input :deep(.el-input__inner::placeholder) {
  color: var(--text-muted);
}

.dark-input :deep(.el-input__prefix),
.dark-input :deep(.el-input__suffix) {
  color: var(--text-quaternary);
}

.dark-input :deep(.el-input__disabled) {
  opacity: 0.5;
}
</style>
