<template>
  <svg
    :class="['base-icon', customClass]"
    :width="size"
    :height="size"
    :viewBox="iconDef.viewBox || '0 0 24 24'"
    :fill="iconDef.fill || 'none'"
    :stroke="iconDef.stroke || 'currentColor'"
    :stroke-width="iconDef.strokeWidth ?? 2"
    stroke-linecap="round"
    stroke-linejoin="round"
    :aria-hidden="!ariaLabel"
    :aria-label="ariaLabel"
    :role="ariaLabel ? 'img' : undefined"
    xmlns="http://www.w3.org/2000/svg"
    v-html="renderedInner"
  />
</template>

<script setup>
import { computed } from 'vue'
import { icons } from '../assets/icons/index.js'

const props = defineProps({
  name: {
    type: String,
    required: true,
  },
  size: {
    type: [Number, String],
    default: 16,
  },
  ariaLabel: {
    type: String,
    default: undefined,
  },
  customClass: {
    type: String,
    default: '',
  },
})

const iconDef = computed(() => {
  return (
    icons[props.name] || {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      strokeWidth: 2,
      paths: ['<circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>'],
    }
  )
})

const renderedInner = computed(() => {
  return iconDef.value.paths.join('')
})
</script>

<style scoped>
.base-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
}
</style>
