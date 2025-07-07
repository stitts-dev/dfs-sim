import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import toast from 'react-hot-toast'
import { formatCurrency, formatNumber, formatDate } from '@/lib/utils'
import { getLineups, deleteLineup, submitLineup, exportLineups } from '@/services/api'
import { Lineup } from '@/types/lineup'

export default function Lineups() {
  const queryClient = useQueryClient()
  const [selectedLineups, setSelectedLineups] = useState<Set<number>>(new Set())
  const [page, setPage] = useState(1)
  const [contestFilter] = useState<number | null>(null)

  const { data: lineupsData, isLoading } = useQuery(
    ['lineups', page, contestFilter],
    () => getLineups({
      page,
      perPage: 20,
      contest_id: contestFilter || undefined,
    })
  )

  const deleteMutation = useMutation(deleteLineup, {
    onSuccess: () => {
      queryClient.invalidateQueries(['lineups'])
      toast.success('Lineup deleted successfully')
    },
    onError: () => {
      toast.error('Failed to delete lineup')
    },
  })

  const submitMutation = useMutation(submitLineup, {
    onSuccess: () => {
      queryClient.invalidateQueries(['lineups'])
      toast.success('Lineup submitted successfully')
    },
    onError: () => {
      toast.error('Failed to submit lineup')
    },
  })

  const handleExport = async () => {
    if (selectedLineups.size === 0) {
      toast.error('Please select lineups to export')
      return
    }

    try {
      // Get the first selected lineup to determine format
      const firstLineupId = Array.from(selectedLineups)[0]
      const lineup = lineupsData?.data.find((l: Lineup) => l.id === firstLineupId)
      
      if (!lineup?.contest) {
        toast.error('Contest information not available')
        return
      }

      const format = `${lineup.contest.platform.substring(0, 2)}_${lineup.contest.sport}`
      await exportLineups(Array.from(selectedLineups), format)
      toast.success('Lineups exported successfully')
    } catch (error) {
      toast.error('Failed to export lineups')
    }
  }

  const handleSelectAll = () => {
    if (selectedLineups.size === lineupsData?.data.length) {
      setSelectedLineups(new Set())
    } else {
      setSelectedLineups(new Set(lineupsData?.data.map((l: Lineup) => l.id)))
    }
  }

  const handleSelectLineup = (lineupId: number) => {
    const newSelected = new Set(selectedLineups)
    if (newSelected.has(lineupId)) {
      newSelected.delete(lineupId)
    } else {
      newSelected.add(lineupId)
    }
    setSelectedLineups(newSelected)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            My Lineups
          </h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage and export your saved lineups
          </p>
        </div>
        
        <div className="flex gap-2">
          {selectedLineups.size > 0 && (
            <button
              onClick={handleExport}
              className="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"
            >
              Export Selected ({selectedLineups.size})
            </button>
          )}
        </div>
      </div>

      {/* Lineup Table */}
      <div className="overflow-hidden rounded-lg bg-white shadow dark:bg-gray-800">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left">
                <input
                  type="checkbox"
                  checked={selectedLineups.size === lineupsData?.data.length && lineupsData?.data.length > 0}
                  onChange={handleSelectAll}
                  className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-300">
                Name
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-300">
                Contest
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-300">
                Salary
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-300">
                Projected
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-300">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-300">
                Created
              </th>
              <th className="relative px-6 py-3">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-800">
            {isLoading ? (
              <tr>
                <td colSpan={8} className="px-6 py-4 text-center">
                  <div className="animate-pulse">Loading lineups...</div>
                </td>
              </tr>
            ) : lineupsData?.data.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-6 py-4 text-center text-gray-500">
                  No lineups found
                </td>
              </tr>
            ) : (
              lineupsData?.data.map((lineup: Lineup) => (
                <tr key={lineup.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                  <td className="px-6 py-4">
                    <input
                      type="checkbox"
                      checked={selectedLineups.has(lineup.id)}
                      onChange={() => handleSelectLineup(lineup.id)}
                      className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-900 dark:text-white">
                    {lineup.name || `Lineup #${lineup.id}`}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                    {lineup.contest?.name || 'Unknown Contest'}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                    {formatCurrency(lineup.total_salary)}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                    {formatNumber(lineup.projected_points)}
                  </td>
                  <td className="px-6 py-4">
                    {lineup.is_submitted ? (
                      <span className="inline-flex rounded-full bg-green-100 px-2 text-xs font-semibold leading-5 text-green-800">
                        Submitted
                      </span>
                    ) : (
                      <span className="inline-flex rounded-full bg-yellow-100 px-2 text-xs font-semibold leading-5 text-yellow-800">
                        Draft
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                    {formatDate(lineup.created_at)}
                  </td>
                  <td className="px-6 py-4 text-right text-sm font-medium">
                    <div className="flex justify-end gap-2">
                      {!lineup.is_submitted && (
                        <>
                          <button
                            onClick={() => submitMutation.mutate(lineup.id)}
                            className="text-blue-600 hover:text-blue-900"
                          >
                            Submit
                          </button>
                          <button
                            onClick={() => {
                              if (confirm('Are you sure you want to delete this lineup?')) {
                                deleteMutation.mutate(lineup.id)
                              }
                            }}
                            className="text-red-600 hover:text-red-900"
                          >
                            Delete
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {lineupsData?.meta && lineupsData.meta.total_pages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-gray-700 dark:text-gray-300">
            Showing {((page - 1) * 20) + 1} to {Math.min(page * 20, lineupsData.meta.total)} of{' '}
            {lineupsData.meta.total} results
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(page - 1)}
              disabled={page === 1}
              className="rounded-lg bg-white px-3 py-1 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              Previous
            </button>
            <button
              onClick={() => setPage(page + 1)}
              disabled={page === lineupsData.meta.total_pages}
              className="rounded-lg bg-white px-3 py-1 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  )
}