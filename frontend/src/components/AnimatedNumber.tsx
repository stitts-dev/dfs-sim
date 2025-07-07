import { useEffect, useRef, useState } from 'react'

interface AnimatedNumberProps {
  value: number
  duration?: number
  formatter?: (value: number) => string
  className?: string
}

export default function AnimatedNumber({ 
  value, 
  duration = 500, 
  formatter = (val) => val.toString(),
  className = ''
}: AnimatedNumberProps) {
  const [displayValue, setDisplayValue] = useState(value)
  const previousValue = useRef(value)
  const animationRef = useRef<number>()

  useEffect(() => {
    const startValue = previousValue.current
    const endValue = value
    const startTime = Date.now()

    const animate = () => {
      const now = Date.now()
      const progress = Math.min((now - startTime) / duration, 1)
      
      // Easing function
      const easeOutQuart = 1 - Math.pow(1 - progress, 4)
      
      const currentValue = startValue + (endValue - startValue) * easeOutQuart
      setDisplayValue(currentValue)

      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate)
      } else {
        previousValue.current = endValue
      }
    }

    animate()

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [value, duration])

  return (
    <span className={className}>
      {formatter(displayValue)}
    </span>
  )
}