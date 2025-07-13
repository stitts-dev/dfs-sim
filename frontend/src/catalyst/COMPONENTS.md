# Catalyst UI Kit Components Reference

Quick reference for all available Catalyst UI Kit components and their standardized import paths.

## Import Quick Reference

| Component | Barrel Import | Individual Import | Original Path |
|-----------|---------------|-------------------|---------------|
| Alert | `import { Alert } from '@/catalyst'` | - | `alert.tsx` |
| Avatar | `import { Avatar } from '@/catalyst'` | - | `avatar.tsx` |
| Badge | `import { Badge } from '@/catalyst'` | `import { Badge } from '@/catalyst/Badge'` | `badge.tsx` |
| Button | `import { Button } from '@/catalyst'` | `import { Button } from '@/catalyst/Button'` | `button.tsx` |
| Checkbox | `import { Checkbox } from '@/catalyst'` | - | `checkbox.tsx` |
| Combobox | `import { Combobox, ComboboxOption } from '@/catalyst'` | - | `combobox.tsx` |
| Description List | `import { DescriptionList, DescriptionTerm, DescriptionDetails } from '@/catalyst'` | - | `description-list.tsx` |
| Dialog | `import { Dialog, DialogTitle, DialogBody, DialogActions } from '@/catalyst'` | `import { Dialog, DialogTitle, DialogBody, DialogActions } from '@/catalyst/Dialog'` | `dialog.tsx` |
| Divider | `import { Divider } from '@/catalyst'` | - | `divider.tsx` |
| Dropdown | `import { Dropdown, DropdownButton, DropdownMenu, DropdownItem } from '@/catalyst'` | - | `dropdown.tsx` |
| Fieldset | `import { Field, Fieldset, Label, ErrorMessage } from '@/catalyst'` | `import { Field, Fieldset, Label, ErrorMessage } from '@/catalyst/Fieldset'` | `fieldset.tsx` |
| Heading | `import { Heading } from '@/catalyst'` | - | `heading.tsx` |
| Input | `import { Input } from '@/catalyst'` | `import { Input } from '@/catalyst/Input'` | `input.tsx` |
| Link | `import { Link } from '@/catalyst'` | - | `link.tsx` |
| Listbox | `import { Listbox, ListboxOption } from '@/catalyst'` | - | `listbox.tsx` |
| Navbar | `import { Navbar, NavbarItem, NavbarSection } from '@/catalyst'` | `import { Navbar, NavbarItem, NavbarSection } from '@/catalyst/Navbar'` | `navbar.tsx` |
| Pagination | `import { Pagination, PaginationPrevious, PaginationNext } from '@/catalyst'` | - | `pagination.tsx` |
| Radio | `import { Radio, RadioGroup } from '@/catalyst'` | - | `radio.tsx` |
| Select | `import { Select } from '@/catalyst'` | `import { Select } from '@/catalyst/Select'` | `select.tsx` |
| Sidebar | `import { Sidebar, SidebarItem, SidebarSection } from '@/catalyst'` | - | `sidebar.tsx` |
| Sidebar Layout | `import { SidebarLayout } from '@/catalyst'` | - | `sidebar-layout.tsx` |
| Stacked Layout | `import { StackedLayout } from '@/catalyst'` | `import { StackedLayout } from '@/catalyst/StackedLayout'` | `stacked-layout.tsx` |
| Switch | `import { Switch, SwitchField, SwitchGroup } from '@/catalyst'` | - | `switch.tsx` |
| Table | `import { Table, TableBody, TableCell, TableHead } from '@/catalyst'` | - | `table.tsx` |
| Text | `import { Text } from '@/catalyst'` | - | `text.tsx` |
| Textarea | `import { Textarea } from '@/catalyst'` | - | `textarea.tsx` |
| Auth Layout | `import { AuthLayout } from '@/catalyst'` | - | `auth-layout.tsx` |

## Component Categories

### 🎨 **UI Elements**
- Alert, Avatar, Badge, Button, Divider, Heading, Link, Text

### 📝 **Form Components**
- Checkbox, Combobox, Input, Radio, Select, Switch, Textarea, Fieldset

### 📊 **Data Display**
- Description List, Table, Pagination

### 🗂️ **Layout & Navigation**
- Navbar, Sidebar, Stacked Layout, Sidebar Layout, Auth Layout

### 🎯 **Interactive**
- Dialog, Dropdown, Listbox

## Usage Examples

### Form with Validation
```tsx
import { 
  Fieldset, 
  Field, 
  Label, 
  Input, 
  ErrorMessage, 
  Button 
} from '@/catalyst'

function ContactForm() {
  return (
    <Fieldset>
      <Field>
        <Label>Email</Label>
        <Input type="email" />
        <ErrorMessage>Please enter a valid email</ErrorMessage>
      </Field>
      <Button type="submit">Submit</Button>
    </Fieldset>
  )
}
```

### Navigation Layout
```tsx
import { StackedLayout } from '@/catalyst/StackedLayout'
import { Navbar, NavbarSection, NavbarItem } from '@/catalyst/Navbar'

function Layout({ children }) {
  return (
    <StackedLayout
      navbar={
        <Navbar>
          <NavbarSection>
            <NavbarItem href="/">Home</NavbarItem>
            <NavbarItem href="/optimizer">Optimizer</NavbarItem>
          </NavbarSection>
        </Navbar>
      }
    >
      {children}
    </StackedLayout>
  )
}
```

### Data Display
```tsx
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from '@/catalyst'
import { Badge } from '@/catalyst/Badge'

function PlayerTable({ players }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableHeader>Name</TableHeader>
          <TableHeader>Position</TableHeader>
          <TableHeader>Salary</TableHeader>
        </TableRow>
      </TableHead>
      <TableBody>
        {players.map(player => (
          <TableRow key={player.id}>
            <TableCell>{player.name}</TableCell>
            <TableCell>
              <Badge color="blue">{player.position}</Badge>
            </TableCell>
            <TableCell>${player.salary}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
```