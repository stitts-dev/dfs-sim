import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount)
}

export function formatNumber(num: number, decimals = 1): string {
  return num.toFixed(decimals)
}

export function formatPercentage(value: number): string {
  return `${value.toFixed(1)}%`
}

export function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(d)
}

export function getPositionColor(position: string, sport?: string): string {
  // Handle golf positions specially since 'G' conflicts with NBA Guard
  if (sport === 'golf' && (position === 'G' || position.startsWith('G'))) {
    return 'bg-emerald-500'
  }
  
  const colors: Record<string, string> = {
    // NBA
    PG: 'bg-blue-500',
    SG: 'bg-green-500',
    SF: 'bg-yellow-500',
    PF: 'bg-orange-500',
    C: 'bg-red-500',
    G: 'bg-blue-400',
    F: 'bg-yellow-400',
    UTIL: 'bg-gray-500',
    // NFL
    QB: 'bg-red-500',
    RB: 'bg-blue-500',
    WR: 'bg-green-500',
    TE: 'bg-purple-500',
    FLEX: 'bg-orange-500',
    DST: 'bg-gray-600',
    'D/ST': 'bg-gray-600',
    // MLB
    P: 'bg-red-500',
    '1B': 'bg-blue-500',
    '2B': 'bg-green-500',
    '3B': 'bg-yellow-500',
    SS: 'bg-purple-500',
    OF: 'bg-orange-500',
    'C/1B': 'bg-indigo-500',
    // NHL
    W: 'bg-blue-500',
    D: 'bg-green-500',
  }
  return colors[position] || 'bg-gray-400'
}