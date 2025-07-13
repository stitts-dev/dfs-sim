import { supabase } from '@/store/supabaseAuth'

// Types matching backend models
export interface SupabaseUser {
  id: string
  phone_number: string
  first_name?: string
  last_name?: string
  subscription_tier: string
  subscription_status: string
  subscription_expires_at?: string
  stripe_customer_id?: string
  monthly_optimizations_used: number
  monthly_simulations_used: number
  usage_reset_date: string
  is_active: boolean
  created_at: string
  updated_at: string
  preferences?: SupabaseUserPreferences
}

export interface SupabaseUserPreferences {
  id: string
  user_id: string
  sport_preferences: string[]
  platform_preferences: string[]
  contest_type_preferences: string[]
  theme: string
  language: string
  notifications_enabled: boolean
  tutorial_completed: boolean
  beginner_mode: boolean
  tooltips_enabled: boolean
  created_at: string
  updated_at: string
}

export interface SupabaseSubscriptionTier {
  id: string
  name: string
  price_cents: number
  currency: string
  monthly_optimizations: number
  monthly_simulations: number
  ai_recommendations: boolean
  bank_verification: boolean
  priority_support: boolean
  created_at: string
  updated_at: string
}

export interface UsageStats {
  subscription_tier: string
  optimizations_used: number
  optimizations_limit: number
  optimizations_remaining: number
  simulations_used: number
  simulations_limit: number
  simulations_remaining: number
  usage_reset_date: string
  can_optimize: boolean
  can_simulate: boolean
  ai_recommendations_enabled: boolean
}

export interface UpdateUserRequest {
  first_name?: string
  last_name?: string
  subscription_tier?: string
}

export interface UpdatePreferencesRequest {
  sport_preferences?: string[]
  platform_preferences?: string[]
  contest_type_preferences?: string[]
  theme?: string
  language?: string
  notifications_enabled?: boolean
  tutorial_completed?: boolean
  beginner_mode?: boolean
  tooltips_enabled?: boolean
}

export class SupabaseUserService {
  private static async getAuthHeaders() {
    const { data: { session } } = await supabase.auth.getSession()
    if (!session?.access_token) {
      throw new Error('Not authenticated')
    }
    
    return {
      'Authorization': `Bearer ${session.access_token}`,
      'Content-Type': 'application/json'
    }
  }

  private static async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Network error' }))
      throw new Error(error.message || `HTTP ${response.status}`)
    }
    
    const data = await response.json()
    return data.data || data
  }

  // Get current user profile with preferences
  static async getCurrentUser(): Promise<SupabaseUser> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/me', {
      method: 'GET',
      headers
    })

    return this.handleResponse<SupabaseUser>(response)
  }

  // Update user profile
  static async updateUser(updates: UpdateUserRequest): Promise<SupabaseUser> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/me', {
      method: 'PUT',
      headers,
      body: JSON.stringify(updates)
    })

    return this.handleResponse<SupabaseUser>(response)
  }

  // Get user preferences
  static async getPreferences(): Promise<SupabaseUserPreferences> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/preferences', {
      method: 'GET',
      headers
    })

    return this.handleResponse<SupabaseUserPreferences>(response)
  }

  // Update user preferences with real-time sync
  static async updatePreferences(preferences: UpdatePreferencesRequest): Promise<SupabaseUserPreferences> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/preferences', {
      method: 'PUT',
      headers,
      body: JSON.stringify(preferences)
    })

    return this.handleResponse<SupabaseUserPreferences>(response)
  }

  // Reset preferences to defaults
  static async resetPreferences(): Promise<SupabaseUserPreferences> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/preferences/reset', {
      method: 'POST',
      headers
    })

    return this.handleResponse<SupabaseUserPreferences>(response)
  }

  // Get subscription tiers
  static async getSubscriptionTiers(): Promise<SupabaseSubscriptionTier[]> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/subscription-tiers', {
      method: 'GET',
      headers
    })

    return this.handleResponse<SupabaseSubscriptionTier[]>(response)
  }

  // Get usage statistics
  static async getUsageStats(): Promise<UsageStats> {
    const headers = await this.getAuthHeaders()
    
    const response = await fetch('/api/v1/users/usage', {
      method: 'GET',
      headers
    })

    return this.handleResponse<UsageStats>(response)
  }

  // Subscribe to user data changes via Supabase Realtime
  static subscribeToUserUpdates(userId: string, callback: (data: any) => void) {
    return supabase
      .channel(`user_updates:${userId}`)
      .on('postgres_changes', {
        event: '*',
        schema: 'public',
        table: 'users',
        filter: `id=eq.${userId}`
      }, (payload) => {
        console.log('User data changed:', payload)
        callback({ type: 'user', data: payload.new })
      })
      .on('postgres_changes', {
        event: '*',
        schema: 'public',
        table: 'user_preferences',
        filter: `user_id=eq.${userId}`
      }, (payload) => {
        console.log('User preferences changed:', payload)
        callback({ type: 'preferences', data: payload.new })
      })
      .subscribe()
  }

  // Validation helpers
  static validateSportPreferences(sports: string[]): boolean {
    const validSports = ['nfl', 'nba', 'mlb', 'nhl', 'golf', 'pga', 'nascar', 'mma', 'soccer']
    return sports.every(sport => validSports.includes(sport))
  }

  static validatePlatformPreferences(platforms: string[]): boolean {
    const validPlatforms = ['draftkings', 'fanduel', 'superdraft', 'prizepicks']
    return platforms.every(platform => validPlatforms.includes(platform))
  }

  static validateContestTypePreferences(types: string[]): boolean {
    const validTypes = ['gpp', 'cash', 'tournament', 'head2head', 'multiplier']
    return types.every(type => validTypes.includes(type))
  }

  static validateTheme(theme: string): boolean {
    return ['light', 'dark', 'auto'].includes(theme)
  }

  static validateLanguage(language: string): boolean {
    return ['en', 'es', 'fr', 'de'].includes(language)
  }
}

export default SupabaseUserService