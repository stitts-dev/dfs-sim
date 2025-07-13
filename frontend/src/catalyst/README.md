# Catalyst UI Kit - Standardized Imports

This directory provides standardized imports for the Catalyst UI Kit (Tailwind Plus) components.

## Original Location
The original Catalyst UI Kit components are located in `../catalyst-ui-kit/typescript/` and are vendorized from Tailwind Plus.

## Import Patterns

### Option 1: Barrel Imports (Recommended for multiple components)
```tsx
import { Button, Dialog, Input, Badge } from '@/catalyst'
```

### Option 2: Individual Component Imports (Recommended for single components)
```tsx
import { Button } from '@/catalyst/Button'
import { Dialog, DialogTitle, DialogBody } from '@/catalyst/Dialog'
import { Input } from '@/catalyst/Input'
```

### Option 3: Direct Imports (Not recommended - use for debugging only)
```tsx
import { Button } from '@/catalyst-ui-kit/typescript/button'
```

## Available Components

### Core Components
- `Alert` - Alert messages and notifications
- `Avatar` - User profile images and placeholders
- `Badge` - Status indicators and labels
- `Button` - Primary and secondary action buttons
- `Checkbox` - Form checkboxes

### Form Components
- `Combobox` - Searchable select inputs
- `Input` - Text inputs
- `Select` - Dropdown selectors
- `Textarea` - Multi-line text inputs
- `Radio` - Radio button groups
- `Switch` - Toggle switches
- `Fieldset` - Form field grouping and labels

### Layout Components
- `Dialog` - Modal dialogs and overlays
- `Divider` - Section separators
- `Dropdown` - Action menus
- `Navbar` - Navigation bars
- `Sidebar` - Side navigation panels
- `StackedLayout` - Main layout wrapper
- `SidebarLayout` - Layout with sidebar

### Data Display
- `DescriptionList` - Key-value data display
- `Heading` - Semantic headings
- `Text` - Styled text content
- `Table` - Data tables
- `Pagination` - Page navigation

### Navigation
- `Link` - Navigation links
- `Listbox` - Selection lists

## Integration with Project

### Beginner Mode Support
All Catalyst components integrate with the project's beginner mode through the `@/lib/catalyst` utility:

```tsx
import { Button } from '@/catalyst/Button'
import { withBeginnerMode } from '@/lib/catalyst'

function MyComponent({ isBeginnerMode }) {
  return (
    <Button 
      className={withBeginnerMode('base-classes', isBeginnerMode, true)}
    >
      Click me
    </Button>
  )
}
```

### Color Mapping
Use the `colorMap` from `@/lib/catalyst` to maintain consistency:

```tsx
import { Badge } from '@/catalyst/Badge'
import { colorMap } from '@/lib/catalyst'

<Badge color={colorMap.success}>Success</Badge>
```

### Position Badge Integration
For DFS positions, use the position color helper:

```tsx
import { Badge } from '@/catalyst/Badge'
import { getPositionCatalystColor } from '@/lib/catalyst'

<Badge color={getPositionCatalystColor('PG', 'nba')}>PG</Badge>
```

## Maintenance

- **Do not edit the original files** in `../catalyst-ui-kit/typescript/`
- **Update this directory** when upgrading Catalyst UI Kit
- **Document any custom modifications** in this README
- **Keep version tracking** in the main Catalyst UI Kit README

## Version
Based on Catalyst UI Kit from Tailwind Plus (vendorized copy)
Last updated: [Current date when last modified]