/**
 * Catalyst UI Kit configuration for DFS Lineup Optimizer
 * 
 * This file provides project-specific configurations, color mappings,
 * and helper functions for integrating Catalyst components with our
 * existing design system and beginner mode features.
 */

import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// Re-export the existing cn utility for compatibility
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Color mapping from our existing CSS variables to Catalyst color props
 */
export const colorMap = {
  // Primary actions (Catalyst 'blue' variants)
  primary: 'blue',
  secondary: 'gray',
  
  // Status colors
  success: 'green',
  warning: 'amber', 
  error: 'red',
  info: 'blue',
  
  // Sport-specific colors
  nba: 'blue',
  nfl: 'green', 
  mlb: 'yellow',
  nhl: 'purple',
  golf: 'emerald',
} as const

/**
 * Beginner mode helper to add highlighting classes to Catalyst components
 */
export function withBeginnerMode(
  baseClasses: string, 
  isBeginnerMode: boolean, 
  isHighlighted?: boolean
): string {
  if (!isBeginnerMode || !isHighlighted) {
    return baseClasses
  }
  
  return cn(
    baseClasses,
    'ring-2 ring-blue-400 ring-offset-2 transition-all duration-200'
  )
}

/**
 * Enhanced Catalyst component props with beginner mode support
 */
export interface CatalystPropsWithBeginnerMode {
  /** Whether beginner mode highlighting should be applied */
  'data-beginner-highlight'?: boolean
  /** Custom class name for additional styling */
  className?: string
}

/**
 * Default Catalyst component configurations
 */
export const catalystDefaults = {
  button: {
    // Default to 'solid' appearance for primary actions
    color: colorMap.primary,
  },
  dialog: {
    // Add backdrop blur by default
    className: 'backdrop-blur-sm',
  },
  input: {
    // Consistent sizing
    className: 'min-h-[2.5rem]',
  },
  select: {
    // Consistent styling with inputs
    className: 'min-h-[2.5rem]',
  },
} as const

/**
 * Position colors for DFS positions - maps to Catalyst badge colors
 */
export function getPositionCatalystColor(position: string, sport?: string): 'red' | 'orange' | 'amber' | 'yellow' | 'lime' | 'green' | 'emerald' | 'teal' | 'cyan' | 'sky' | 'blue' | 'indigo' | 'violet' | 'purple' | 'fuchsia' | 'pink' | 'rose' | 'gray' | 'white' | 'dark' {
  // Handle golf positions specially since 'G' conflicts with NBA Guard
  if (sport === 'golf' && (position === 'G' || position.startsWith('G'))) {
    return 'emerald'
  }
  
  const positionColorMap: Record<string, typeof colorMap[keyof typeof colorMap] | 'red' | 'orange' | 'amber' | 'yellow' | 'lime' | 'green' | 'emerald' | 'teal' | 'cyan' | 'sky' | 'blue' | 'indigo' | 'violet' | 'purple' | 'fuchsia' | 'pink' | 'rose' | 'gray' | 'white' | 'dark'> = {
    // NBA
    PG: 'blue',
    SG: 'green', 
    SF: 'yellow',
    PF: 'orange',
    C: 'red',
    G: 'sky',
    F: 'lime',
    UTIL: 'gray',
    // NFL
    QB: 'red',
    RB: 'blue',
    WR: 'green',
    TE: 'purple',
    FLEX: 'orange',
    DST: 'gray',
    'D/ST': 'gray',
    // MLB
    P: 'red',
    '1B': 'blue',
    '2B': 'green', 
    '3B': 'yellow',
    SS: 'purple',
    OF: 'orange',
    'C/1B': 'indigo',
    // NHL
    W: 'blue',
    D: 'green',
    // Golf (handled separately above for 'G' conflict resolution)
  }
  
  return positionColorMap[position] || 'gray'
}

/**
 * Theme configuration for dark mode support
 */
export const themeConfig = {
  // Preserve existing CSS variable system
  colors: {
    // These will work with our existing index.css variables
    background: 'hsl(var(--background))',
    foreground: 'hsl(var(--foreground))',
    primary: 'hsl(var(--primary))',
    secondary: 'hsl(var(--secondary))',
    muted: 'hsl(var(--muted))',
    accent: 'hsl(var(--accent))',
    destructive: 'hsl(var(--destructive))',
    border: 'hsl(var(--border))',
    input: 'hsl(var(--input))',
    ring: 'hsl(var(--ring))',
  }
}

/**
 * Accessibility helpers for screen readers and keyboard navigation
 */
export const a11yHelpers = {
  /**
   * Generate ARIA label for position badges
   */
  getPositionAriaLabel: (position: string, sport?: string): string => {
    // Handle golf positions specially since 'G' conflicts with NBA Guard
    if (sport === 'golf' && (position === 'G' || position.startsWith('G'))) {
      return 'Golfer'
    }
    
    const positionNames: Record<string, string> = {
      // NBA
      PG: 'Point Guard',
      SG: 'Shooting Guard',
      SF: 'Small Forward', 
      PF: 'Power Forward',
      C: 'Center',
      G: 'Guard',
      F: 'Forward',
      UTIL: 'Utility',
      // NFL
      QB: 'Quarterback',
      RB: 'Running Back', 
      WR: 'Wide Receiver',
      TE: 'Tight End',
      FLEX: 'Flex Position',
      DST: 'Defense Special Teams',
      'D/ST': 'Defense Special Teams',
      // MLB
      P: 'Pitcher',
      '1B': 'First Base',
      '2B': 'Second Base',
      '3B': 'Third Base', 
      SS: 'Shortstop',
      OF: 'Outfield',
      'C/1B': 'Catcher First Base',
      // NHL
      W: 'Winger',
      D: 'Defense',
      // Golf handled separately above
    }
    
    return positionNames[position] || position
  },
  
  /**
   * Generate descriptive text for optimization status
   */
  getOptimizationStatusLabel: (status: string): string => {
    const statusLabels: Record<string, string> = {
      idle: 'Optimization not started',
      running: 'Optimization in progress',
      completed: 'Optimization completed successfully',
      error: 'Optimization failed with error',
    }
    
    return statusLabels[status] || status
  }
}