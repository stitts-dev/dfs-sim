import { useState } from 'react'
import { 
  CheckCircleIcon, 
  ChevronRightIcon, 
  ChevronLeftIcon,
  TrophyIcon,
  ZapIcon,
  BarChart3Icon,
  SettingsIcon 
} from 'lucide-react'
// import { useAuth } from '../hooks/useAuth'
import { usePreferences } from '../hooks/usePreferences'

interface OnboardingWizardProps {
  onComplete: () => void
  onSkip?: () => void
}

export function OnboardingWizard({ onComplete, onSkip }: OnboardingWizardProps) {
  // const {} = useAuth() // Not currently needed but could be used for user info
  const { updatePreferences } = usePreferences()
  const [currentStep, setCurrentStep] = useState(0)
  const [selections, setSelections] = useState({
    sportPreferences: ['golf'],
    platformPreferences: ['draftkings'],
    contestTypePreferences: ['gpp'],
  })

  const steps = [
    {
      title: 'Welcome to DFS Optimizer!',
      subtitle: 'The fastest way to build winning DraftKings and FanDuel lineups',
      content: 'welcome'
    },
    {
      title: 'Choose Your Sports',
      subtitle: 'Select the sports you want to optimize lineups for',
      content: 'sports'
    },
    {
      title: 'Select Your Platforms',
      subtitle: 'Which DFS platforms do you use?',
      content: 'platforms'
    },
    {
      title: 'Contest Types',
      subtitle: 'What types of contests do you usually play?',
      content: 'contests'
    },
    {
      title: 'You\'re All Set!',
      subtitle: 'Start building winning lineups with your trial',
      content: 'complete'
    }
  ]

  const sportOptions = [
    { id: 'golf', name: 'Golf', description: 'PGA Tour tournaments' },
    { id: 'nba', name: 'NBA', description: 'Basketball lineups' },
    { id: 'nfl', name: 'NFL', description: 'Football lineups' },
    { id: 'mlb', name: 'MLB', description: 'Baseball lineups' }
  ]

  const platformOptions = [
    { id: 'draftkings', name: 'DraftKings', description: 'America\'s favorite DFS platform' },
    { id: 'fanduel', name: 'FanDuel', description: 'Simple salary cap contests' },
    { id: 'superdraft', name: 'SuperDraft', description: 'Over/under pick\'em contests' }
  ]

  const contestOptions = [
    { id: 'gpp', name: 'GPP (Tournaments)', description: 'Large field, top-heavy payouts' },
    { id: 'cash', name: 'Cash Games', description: 'Head-to-head, 50/50s, double-ups' },
    { id: 'single', name: 'Single Entry', description: 'One lineup per contest' }
  ]

  const handleSelection = (category: string, value: string) => {
    setSelections(prev => ({
      ...prev,
      [category]: prev[category as keyof typeof prev].includes(value)
        ? prev[category as keyof typeof prev].filter(item => item !== value)
        : [...prev[category as keyof typeof prev], value]
    }))
  }

  const handleNext = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      handleComplete()
    }
  }

  const handleBack = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1)
    }
  }

  const handleComplete = async () => {
    try {
      // Update user preferences
      await updatePreferences({
        preferred_sports: selections.sportPreferences,
        beginner_mode: true,
        show_tooltips: true
      })
      
      onComplete()
    } catch (error) {
      console.error('Failed to save preferences:', error)
      // Still complete onboarding even if preferences fail
      onComplete()
    }
  }

  const renderStepContent = () => {
    const step = steps[currentStep]
    
    switch (step.content) {
      case 'welcome':
        return (
          <div className="text-center py-8">
            <div className="mx-auto w-20 h-20 bg-blue-100 rounded-full flex items-center justify-center mb-6">
              <TrophyIcon className="h-10 w-10 text-blue-600" />
            </div>
            <div className="space-y-6">
              <div className="bg-gray-50 rounded-lg p-4">
                <h4 className="font-medium text-gray-900 mb-2">Your Free Trial Includes:</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                  <div className="flex items-center space-x-2">
                    <ZapIcon className="h-4 w-4 text-blue-500" />
                    <span>10 lineup optimizations</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <BarChart3Icon className="h-4 w-4 text-green-500" />
                    <span>5 Monte Carlo simulations</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <SettingsIcon className="h-4 w-4 text-purple-500" />
                    <span>All contest types</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <CheckCircleIcon className="h-4 w-4 text-green-500" />
                    <span>No credit card required</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )
      
      case 'sports':
        return (
          <div className="space-y-4">
            {sportOptions.map(sport => (
              <div
                key={sport.id}
                onClick={() => handleSelection('sportPreferences', sport.id)}
                className={`p-4 border rounded-lg cursor-pointer transition-all ${
                  selections.sportPreferences.includes(sport.id)
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium text-gray-900">{sport.name}</h3>
                    <p className="text-sm text-gray-600">{sport.description}</p>
                  </div>
                  {selections.sportPreferences.includes(sport.id) && (
                    <CheckCircleIcon className="h-5 w-5 text-blue-500" />
                  )}
                </div>
              </div>
            ))}
          </div>
        )
      
      case 'platforms':
        return (
          <div className="space-y-4">
            {platformOptions.map(platform => (
              <div
                key={platform.id}
                onClick={() => handleSelection('platformPreferences', platform.id)}
                className={`p-4 border rounded-lg cursor-pointer transition-all ${
                  selections.platformPreferences.includes(platform.id)
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium text-gray-900">{platform.name}</h3>
                    <p className="text-sm text-gray-600">{platform.description}</p>
                  </div>
                  {selections.platformPreferences.includes(platform.id) && (
                    <CheckCircleIcon className="h-5 w-5 text-blue-500" />
                  )}
                </div>
              </div>
            ))}
          </div>
        )
      
      case 'contests':
        return (
          <div className="space-y-4">
            {contestOptions.map(contest => (
              <div
                key={contest.id}
                onClick={() => handleSelection('contestTypePreferences', contest.id)}
                className={`p-4 border rounded-lg cursor-pointer transition-all ${
                  selections.contestTypePreferences.includes(contest.id)
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-medium text-gray-900">{contest.name}</h3>
                    <p className="text-sm text-gray-600">{contest.description}</p>
                  </div>
                  {selections.contestTypePreferences.includes(contest.id) && (
                    <CheckCircleIcon className="h-5 w-5 text-blue-500" />
                  )}
                </div>
              </div>
            ))}
          </div>
        )
      
      case 'complete':
        return (
          <div className="text-center py-8">
            <div className="mx-auto w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mb-6">
              <CheckCircleIcon className="h-10 w-10 text-green-600" />
            </div>
            <div className="space-y-4">
              <h4 className="font-medium text-gray-900">Ready to optimize!</h4>
              <p className="text-gray-600">
                Your preferences have been saved. You can always change them later in Settings.
              </p>
              <div className="bg-blue-50 rounded-lg p-4 text-sm">
                <p className="text-blue-800">
                  <strong>Pro tip:</strong> Start with cash games to learn the platform, then move to GPPs as you get comfortable.
                </p>
              </div>
            </div>
          </div>
        )
      
      default:
        return null
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4">
        {/* Header */}
        <div className="px-6 py-4 border-b">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold text-gray-900">{steps[currentStep].title}</h2>
              <p className="text-gray-600">{steps[currentStep].subtitle}</p>
            </div>
            {onSkip && currentStep < steps.length - 1 && (
              <button
                onClick={onSkip}
                className="text-gray-400 hover:text-gray-600 text-sm"
              >
                Skip Setup
              </button>
            )}
          </div>
        </div>

        {/* Progress */}
        <div className="px-6 py-2">
          <div className="flex space-x-2">
            {steps.map((_, index) => (
              <div
                key={index}
                className={`flex-1 h-2 rounded-full ${
                  index <= currentStep ? 'bg-blue-500' : 'bg-gray-200'
                }`}
              />
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="px-6 py-6 min-h-[400px]">
          {renderStepContent()}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t flex items-center justify-between">
          <button
            onClick={handleBack}
            disabled={currentStep === 0}
            className={`flex items-center space-x-2 px-4 py-2 rounded-md transition-colors ${
              currentStep === 0
                ? 'text-gray-400 cursor-not-allowed'
                : 'text-gray-600 hover:text-gray-800'
            }`}
          >
            <ChevronLeftIcon className="h-4 w-4" />
            <span>Back</span>
          </button>

          <span className="text-sm text-gray-500">
            {currentStep + 1} of {steps.length}
          </span>

          <button
            onClick={handleNext}
            className="flex items-center space-x-2 px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
          >
            <span>{currentStep === steps.length - 1 ? 'Get Started' : 'Next'}</span>
            <ChevronRightIcon className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}

export default OnboardingWizard