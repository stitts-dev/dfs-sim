import React, { useState, useRef, cloneElement, isValidElement, useEffect, ReactElement } from 'react'
import {
  useFloating,
  autoUpdate,
  offset,
  flip,
  shift,
  useHover,
  useFocus,
  useClick,
  useDismiss,
  useRole,
  useInteractions,
  FloatingPortal,
  arrow,
  FloatingArrow,
  Placement,
} from '@floating-ui/react'
import { cn } from '@/lib/utils'

interface TooltipProps {
  children: React.ReactElement
  content: React.ReactNode | (() => Promise<React.ReactNode>)
  placement?: Placement
  delay?: number
  interactive?: boolean
  maxWidth?: number
  showArrow?: boolean
  trigger?: 'hover' | 'click' | 'focus' | 'manual'
  open?: boolean
  onOpenChange?: (open: boolean) => void
  className?: string
  disabled?: boolean
}

export default function Tooltip({
  children,
  content,
  placement = 'top',
  delay = 300,
  interactive: _interactive = false, // Prefix with _ to indicate it's for future use
  maxWidth = 300,
  showArrow = true,
  trigger = 'hover',
  open: controlledOpen,
  onOpenChange,
  className,
  disabled = false,
}: TooltipProps) {
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false)
  const [asyncContent, setAsyncContent] = useState<React.ReactNode>(null)
  const [isLoading, setIsLoading] = useState(false)
  const arrowRef = useRef(null)

  const open = controlledOpen ?? uncontrolledOpen
  const setOpen = onOpenChange ?? setUncontrolledOpen

  const { x, y, strategy, refs, context } = useFloating({
    open: open && !disabled,
    onOpenChange: setOpen,
    placement,
    middleware: [
      offset(showArrow ? 10 : 4),
      flip({
        fallbackAxisSideDirection: 'start',
        crossAxis: false,
      }),
      shift({ padding: 8 }),
      arrow({ element: arrowRef }),
    ],
    whileElementsMounted: autoUpdate,
  })

  const hover = useHover(context, {
    enabled: trigger === 'hover',
    delay: {
      open: delay,
      close: 100,
    },
    restMs: 40,
  })

  const focus = useFocus(context, {
    enabled: trigger === 'focus',
  })

  const click = useClick(context, {
    enabled: trigger === 'click',
  })

  const dismiss = useDismiss(context, {
    ancestorScroll: true,
  })

  const role = useRole(context, { role: 'tooltip' })

  const { getReferenceProps, getFloatingProps } = useInteractions([
    hover,
    focus,
    click,
    dismiss,
    role,
  ])

  // Load async content
  useEffect(() => {
    if (open && typeof content === 'function') {
      setIsLoading(true)
      content()
        .then(setAsyncContent)
        .catch((error) => {
          console.error('Failed to load tooltip content:', error)
          setAsyncContent('Failed to load content')
        })
        .finally(() => setIsLoading(false))
    }
  }, [open, content])

  const tooltipContent = typeof content === 'function' ? asyncContent : content

  // Add keyboard shortcut (F1) to toggle all tooltips
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      if (e.key === 'F1') {
        e.preventDefault()
        setOpen(!open)
      }
    }

    if (trigger === 'hover') {
      window.addEventListener('keydown', handleKeyPress)
      return () => window.removeEventListener('keydown', handleKeyPress)
    }
  }, [open, setOpen, trigger])

  if (!isValidElement(children)) {
    console.warn('Tooltip requires a valid React element as children')
    return <>{children}</>
  }

  return (
    <>
      {cloneElement(children as ReactElement<any>, {
        ref: refs.setReference,
        ...getReferenceProps(),
      })}
      
      {open && !disabled && (
        <FloatingPortal>
          <div
            ref={refs.setFloating}
            style={{
              position: strategy,
              top: y ?? 0,
              left: x ?? 0,
              maxWidth,
              zIndex: 10000,
            }}
            className={cn(
              'tooltip',
              'animate-fade-in',
              className
            )}
            {...getFloatingProps()}
          >
            <div className="rounded-lg bg-gray-900 px-3 py-2 text-sm text-white shadow-lg dark:bg-gray-800">
              {isLoading ? (
                <div className="flex items-center space-x-2">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  <span>Loading...</span>
                </div>
              ) : (
                tooltipContent
              )}
              
              {showArrow && (
                <FloatingArrow
                  ref={arrowRef}
                  context={context}
                  className="fill-gray-900 dark:fill-gray-800"
                />
              )}
            </div>
          </div>
        </FloatingPortal>
      )}
    </>
  )
}

// Compound component for tooltip content sections
export function TooltipSection({ title, children }: { title?: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      {title && <div className="font-semibold text-white/90">{title}</div>}
      <div className="text-white/70">{children}</div>
    </div>
  )
}

// Loading skeleton for async content
export function TooltipSkeleton() {
  return (
    <div className="space-y-2">
      <div className="h-4 w-32 animate-pulse rounded bg-white/20" />
      <div className="h-3 w-24 animate-pulse rounded bg-white/20" />
      <div className="h-3 w-40 animate-pulse rounded bg-white/20" />
    </div>
  )
}