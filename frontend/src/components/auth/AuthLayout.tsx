import React from 'react'
import { StarField } from '@/components/ui/StarField'
import { Glow } from '@/components/ui/Glow'
import { SparkleIcon } from '@/components/ui/SparkleIcon'

interface AuthLayoutProps {
  children: React.ReactNode
  title?: string
  subtitle?: string
  showBranding?: boolean
  className?: string
}

export function AuthLayout({ 
  children, 
  title = "DFS Lineup Optimizer",
  subtitle = "Professional Daily Fantasy Sports optimization with advanced algorithms",
  showBranding = true,
  className = ""
}: AuthLayoutProps) {
  return (
    <div className={`min-h-screen flex ${className}`}>
      {/* Left Side - Branding & Visual Effects */}
      {showBranding && (
        <div className="hidden lg:flex lg:w-1/2 relative overflow-hidden bg-gray-950">
          <Glow />
          <StarField className="top-14 -right-44" />
          
          <div className="relative z-10 flex flex-col justify-center px-12">
            {/* Logo placeholder */}
            <div className="flex items-center mb-8">
              <div className="w-10 h-10 bg-gradient-to-br from-sky-400 to-sky-600 rounded-lg flex items-center justify-center">
                <SparkleIcon className="w-6 h-6" />
              </div>
              <span className="ml-3 text-xl font-semibold text-white">DFS Pro</span>
            </div>
            
            <h1 className="text-4xl font-light text-white leading-tight">
              {title.split(' ').map((word, index) => (
                <span key={index}>
                  {word === 'Optimizer' ? (
                    <span className="text-sky-300">{word}</span>
                  ) : (
                    word
                  )}
                  {index < title.split(' ').length - 1 && ' '}
                </span>
              ))}
            </h1>
            
            <p className="mt-4 text-sm/6 text-gray-300 max-w-md">
              {subtitle}
            </p>
            
            {/* Feature highlights */}
            <div className="mt-8 space-y-3">
              <div className="flex items-center text-sm text-gray-300">
                <div className="w-2 h-2 bg-sky-400 rounded-full mr-3" />
                Monte Carlo simulations with correlation matrices
              </div>
              <div className="flex items-center text-sm text-gray-300">
                <div className="w-2 h-2 bg-sky-400 rounded-full mr-3" />
                Advanced stacking and lineup optimization
              </div>
              <div className="flex items-center text-sm text-gray-300">
                <div className="w-2 h-2 bg-sky-400 rounded-full mr-3" />
                Real-time data from multiple providers
              </div>
            </div>
          </div>
          
          {/* Bottom decoration */}
          <div className="absolute bottom-8 left-12 right-12">
            <div className="h-px bg-gradient-to-r from-transparent via-sky-500/30 to-transparent" />
            <div className="mt-4 text-xs text-gray-500 text-center">
              Trusted by professional DFS players
            </div>
          </div>
        </div>
      )}
      
      {/* Right Side - Auth Form */}
      <div className={`flex-1 flex items-center justify-center px-6 py-12 bg-white dark:bg-gray-900 ${
        showBranding ? '' : 'w-full'
      }`}>
        <div className="w-full max-w-md">
          {children}
        </div>
      </div>
    </div>
  )
}

interface AuthCardProps {
  children: React.ReactNode
  title?: string
  subtitle?: string
  className?: string
}

export function AuthCard({ 
  children, 
  title, 
  subtitle, 
  className = "" 
}: AuthCardProps) {
  return (
    <div className={`relative ${className}`}>
      {/* Simplified card background with better contrast */}
      <div className="bg-white dark:bg-gray-800 shadow-xl rounded-2xl border border-gray-200 dark:border-gray-700 p-8">
        {title && (
          <div className="text-center mb-6">
            <h2 className="text-2xl font-semibold text-gray-900 dark:text-white">
              {title}
            </h2>
            {subtitle && (
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                {subtitle}
              </p>
            )}
          </div>
        )}
        
        {children}
      </div>
    </div>
  )
}

interface AuthStepIndicatorProps {
  steps: string[]
  currentStep: string
  completedSteps?: string[]
  className?: string
}

export function AuthStepIndicator({ 
  steps, 
  currentStep, 
  completedSteps = [], 
  className = "" 
}: AuthStepIndicatorProps) {
  return (
    <div className={`flex justify-center space-x-2 ${className}`}>
      {steps.map((step) => {
        const isCompleted = completedSteps.includes(step)
        const isCurrent = step === currentStep
        
        return (
          <div
            key={step}
            className={`w-2 h-2 rounded-full transition-all duration-300 ${
              isCompleted
                ? 'bg-green-500 shadow-lg shadow-green-500/30'
                : isCurrent
                ? 'bg-sky-500 shadow-lg shadow-sky-500/30 scale-125'
                : 'bg-gray-300 dark:bg-gray-600'
            }`}
          />
        )
      })}
    </div>
  )
}