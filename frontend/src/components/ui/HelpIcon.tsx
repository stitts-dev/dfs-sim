import { cn } from '@/lib/utils'

interface HelpIconProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

export default function HelpIcon({ className, size = 'sm' }: HelpIconProps) {
  const sizeClasses = {
    sm: 'h-3 w-3',
    md: 'h-4 w-4',
    lg: 'h-5 w-5',
  }

  return (
    <svg
      className={cn(
        'inline-block text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors',
        sizeClasses[size],
        className
      )}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  )
}