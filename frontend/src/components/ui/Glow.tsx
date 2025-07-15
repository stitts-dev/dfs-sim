import { useId } from 'react'

interface GlowProps {
  className?: string
  variant?: 'desktop' | 'mobile' | 'both'
}

export function Glow({ className = '', variant = 'both' }: GlowProps) {
  let id = useId()

  return (
    <div className={`absolute inset-0 -z-10 overflow-hidden ${className}`}>
      <svg
        className="absolute -bottom-48 left-[-40%] h-320 w-[180%] lg:top-[-40%] lg:-right-40 lg:bottom-auto lg:left-auto lg:h-[180%] lg:w-7xl"
        aria-hidden="true"
      >
        <defs>
          <radialGradient id={`${id}-desktop`} cx="100%">
            <stop offset="0%" stopColor="rgba(56, 189, 248, 0.3)" />
            <stop offset="53.95%" stopColor="rgba(0, 71, 255, 0.09)" />
            <stop offset="100%" stopColor="rgba(10, 14, 23, 0)" />
          </radialGradient>
          <radialGradient id={`${id}-mobile`} cy="100%">
            <stop offset="0%" stopColor="rgba(56, 189, 248, 0.3)" />
            <stop offset="53.95%" stopColor="rgba(0, 71, 255, 0.09)" />
            <stop offset="100%" stopColor="rgba(10, 14, 23, 0)" />
          </radialGradient>
        </defs>
        {(variant === 'desktop' || variant === 'both') && (
          <rect
            width="100%"
            height="100%"
            fill={`url(#${id}-desktop)`}
            className="hidden lg:block"
          />
        )}
        {(variant === 'mobile' || variant === 'both') && (
          <rect
            width="100%"
            height="100%"
            fill={`url(#${id}-mobile)`}
            className="lg:hidden"
          />
        )}
      </svg>
      <div className="absolute inset-x-0 right-0 bottom-0 h-px bg-white mix-blend-overlay lg:top-0 lg:left-auto lg:h-auto lg:w-px" />
    </div>
  )
}