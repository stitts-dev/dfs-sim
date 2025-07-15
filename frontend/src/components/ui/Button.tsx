import React from 'react'
import clsx from 'clsx'

interface ButtonInnerProps {
  arrow?: boolean
  children: React.ReactNode
}

function ButtonInner({ arrow = false, children }: ButtonInnerProps) {
  return (
    <>
      <span className="absolute inset-0 rounded-md bg-gradient-to-b from-white/80 to-white opacity-10 transition-opacity group-hover:opacity-15" />
      <span className="absolute inset-0 rounded-md opacity-7.5 shadow-[inset_0_1px_1px_white] transition-opacity group-hover:opacity-10" />
      {children} {arrow ? <span aria-hidden="true">&rarr;</span> : null}
    </>
  )
}

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  arrow?: boolean
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  children: React.ReactNode
}

export function Button({
  className,
  arrow,
  variant = 'primary',
  size = 'md',
  loading = false,
  children,
  disabled,
  ...props
}: ButtonProps) {
  const baseClasses = 'group relative isolate flex-none rounded-md font-semibold transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-sky-300/50 disabled:opacity-50 disabled:cursor-not-allowed'
  
  const variantClasses = {
    primary: 'bg-sky-600 hover:bg-sky-500 text-white shadow-lg hover:shadow-xl',
    secondary: 'bg-gray-600 hover:bg-gray-500 text-white shadow-lg hover:shadow-xl',
    outline: 'border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100',
    ghost: 'hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300'
  }
  
  const sizeClasses = {
    sm: 'py-1.5 text-[0.75rem]/5',
    md: 'py-2 text-[0.8125rem]/6',
    lg: 'py-2.5 text-[0.875rem]/6'
  }
  
  const paddingClasses = arrow ? 'pl-3 pr-[calc(9/16*1rem)]' : 'px-3'
  
  const finalClassName = clsx(
    baseClasses,
    variantClasses[variant],
    sizeClasses[size],
    paddingClasses,
    className
  )

  return (
    <button 
      className={finalClassName} 
      disabled={disabled || loading}
      {...props}
    >
      {variant === 'primary' || variant === 'secondary' ? (
        <ButtonInner arrow={arrow}>
          {loading ? (
            <div className="flex items-center space-x-2">
              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              <span>{children}</span>
            </div>
          ) : (
            children
          )}
        </ButtonInner>
      ) : (
        loading ? (
          <div className="flex items-center space-x-2">
            <div className="w-4 h-4 border-2 border-current/30 border-t-current rounded-full animate-spin" />
            <span>{children}</span>
          </div>
        ) : (
          <>
            {children} {arrow ? <span aria-hidden="true">&rarr;</span> : null}
          </>
        )
      )}
    </button>
  )
}