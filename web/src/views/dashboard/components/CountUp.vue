<template>
  <span ref="elRef">{{ displayValue }}</span>
</template>

<script>
import { ref, onMounted, watch } from 'vue'

export default {
  name: 'CountUp',
  props: {
    value: { type: [Number, String], default: 0 },
    duration: { type: Number, default: 2 }
  },
  setup(props) {
    const elRef = ref(null)
    const displayValue = ref(0)

    let startTime = null
    let rafId = null
    const target = parseFloat(props.value) || 0

    const animate = (timestamp) => {
      if (!startTime) startTime = timestamp
      const progress = Math.min((timestamp - startTime) / (props.duration * 1000), 1)
      // easeOut
      const eased = 1 - Math.pow(1 - progress, 3)
      displayValue.value = Math.round(eased * target)
      if (progress < 1) {
        rafId = requestAnimationFrame(animate)
      } else {
        displayValue.value = target
      }
    }

    onMounted(() => {
      rafId = requestAnimationFrame(animate)
    })

    watch(() => props.value, () => {
      if (rafId) cancelAnimationFrame(rafId)
      startTime = null
      rafId = requestAnimationFrame(animate)
    })

    return { elRef, displayValue }
  }
}
</script>
