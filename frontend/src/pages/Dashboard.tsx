import { useState } from 'react'
import { useQuery } from 'react-query'
import { Link } from 'react-router-dom'
import { formatCurrency, formatNumber, formatDate, cn } from '@/lib/utils'
import { getContests } from '@/services/api'
import { Contest } from '@/types/contest'
import { usePreferencesStore } from '@/store/preferences'

export default function Dashboard() {
  const { beginnerMode } = usePreferencesStore()
  const [selectedSport, setSelectedSport] = useState<string>('all')
  const [selectedPlatform, setSelectedPlatform] = useState<string>('all')

  const { data: contests, isLoading } = useQuery(
    ['contests', selectedSport, selectedPlatform],
    () => getContests({
      sport: selectedSport !== 'all' ? selectedSport : undefined,
      platform: selectedPlatform !== 'all' ? selectedPlatform : undefined,
      active: 'true',
    })
  )

  const sports = [
    { value: 'all', label: 'All Sports', icon: '🏆' },
    { value: 'nba', label: 'NBA', icon: '🏀' },
    { value: 'nfl', label: 'NFL', icon: '🏈' },
    { value: 'mlb', label: 'MLB', icon: '⚾' },
    { value: 'nhl', label: 'NHL', icon: '🏒' },
    { value: 'golf', label: 'Golf', icon: '⛳' },
  ]

  const platforms = [
    { value: 'all', label: 'All Platforms' },
    { value: 'draftkings', label: 'DraftKings' },
    { value: 'fanduel', label: 'FanDuel' },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
          Contest Dashboard
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Select a contest to start building lineups
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        {beginnerMode && (
          <div className="w-full p-3 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-700 mb-2">
            <p className="text-sm text-blue-800 dark:text-blue-200">
              🎯 <strong>Quick Start:</strong> Select a sport below, then click on any contest to start building lineups!
            </p>
          </div>
        )}
        <div className="flex gap-2">
          {sports.map((sport) => (
            <button
              key={sport.value}
              onClick={() => setSelectedSport(sport.value)}
              className={cn(
                `flex items-center rounded-lg px-4 py-2 text-sm font-medium transition-all duration-200`,
                selectedSport === sport.value
                  ? 'bg-blue-600 text-white shadow-lg transform scale-105'
                  : 'bg-white text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700',
                beginnerMode && selectedSport === sport.value && 'ring-2 ring-blue-400 ring-offset-2'
              )}
            >
              <span className="mr-2">{sport.icon}</span>
              {sport.label}
            </button>
          ))}
        </div>

        <select
          value={selectedPlatform}
          onChange={(e) => setSelectedPlatform(e.target.value)}
          className="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
        >
          {platforms.map((platform) => (
            <option key={platform.value} value={platform.value}>
              {platform.label}
            </option>
          ))}
        </select>
      </div>

      {/* Contest Grid */}
      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <div
              key={i}
              className="h-48 animate-pulse rounded-lg bg-gray-200 dark:bg-gray-700"
            />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {contests?.map((contest: Contest) => (
            <ContestCard key={contest.id} contest={contest} />
          ))}
        </div>
      )}

      {contests?.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500 dark:text-gray-400">
            No active contests found
          </p>
        </div>
      )}
    </div>
  )
}

function ContestCard({ contest }: { contest: Contest }) {
  const { beginnerMode } = usePreferencesStore()
  const sportIcons: Record<string, string> = {
    nba: '🏀',
    nfl: '🏈',
    mlb: '⚾',
    nhl: '🏒',
    golf: '⛳',
  }

  const platformColors: Record<string, string> = {
    draftkings: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    fanduel: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
  }

  return (
    <Link
      to={`/optimizer?contest=${contest.id}`}
      className={cn(
        "block rounded-lg bg-white p-6 shadow transition-all duration-200 hover:shadow-lg dark:bg-gray-800",
        beginnerMode && "hover:scale-[1.02] hover:ring-2 hover:ring-blue-400"
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center">
          <span className="text-2xl">{sportIcons[contest.sport]}</span>
          <div className="ml-3">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              {contest.name}
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {formatDate(contest.start_time)}
            </p>
          </div>
        </div>
        <span
          className={`rounded-full px-2 py-1 text-xs font-medium ${
            platformColors[contest.platform]
          }`}
        >
          {contest.platform.toUpperCase()}
        </span>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-4 text-sm">
        <div>
          <p className="text-gray-500 dark:text-gray-400">Entry Fee</p>
          <p className="font-semibold text-gray-900 dark:text-white">
            {formatCurrency(contest.entry_fee)}
          </p>
        </div>
        <div>
          <p className="text-gray-500 dark:text-gray-400">Prize Pool</p>
          <p className="font-semibold text-gray-900 dark:text-white">
            {formatCurrency(contest.prize_pool)}
          </p>
        </div>
        <div>
          <p className="text-gray-500 dark:text-gray-400">Entries</p>
          <p className="font-semibold text-gray-900 dark:text-white">
            {formatNumber(contest.total_entries, 0)} / {formatNumber(contest.max_entries, 0)}
          </p>
        </div>
        <div>
          <p className="text-gray-500 dark:text-gray-400">Type</p>
          <p className="font-semibold uppercase text-gray-900 dark:text-white">
            {contest.contest_type}
            {beginnerMode && contest.contest_type === 'gpp' && (
              <span className="ml-1 text-xs normal-case text-gray-500 dark:text-gray-400" title="Guaranteed Prize Pool">
                (Large Tournament)
              </span>
            )}
          </p>
        </div>
      </div>

      <div className="mt-4">
        <div className="text-xs text-gray-500 dark:text-gray-400">
          Salary Cap: {formatCurrency(contest.salary_cap)}
        </div>
        {beginnerMode && (
          <div className="mt-2 text-xs font-medium text-blue-600 dark:text-blue-400">
            Click to start building lineups →
          </div>
        )}
      </div>
    </Link>
  )
}