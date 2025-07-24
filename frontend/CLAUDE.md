# Frontend CLAUDE.md

This file provides specific guidance to Claude Code when working with the PubDataHub frontend application.

## Frontend Overview

The frontend is a modern React application built from scratch using **Shadcn UI with Vite**, implementing a responsive application layout with navigation.

## Current Status

**Implementation**: ✅ **COMPLETE** 
- Modern React 19+ application with TypeScript 5.8+
- Shadcn UI design system with Tailwind CSS 4.1+
- Responsive 2-column layout with collapsible navigation
- React Router 7+ for client-side routing
- Full application structure with pages and components

**Last Updated**: Feature branch `feature/app-layout` (commit `0dc9194`)

## Technology Stack

### Core Framework
- **React**: 19.1.0 with TypeScript
- **Vite**: 7.0.4 (build tool and dev server) 
- **TypeScript**: 5.8.3 with strict mode enabled

### UI Framework
- **Shadcn UI**: Latest with "new-york" style
- **Tailwind CSS**: 4.1.11 with CSS variables
- **Radix UI**: Primitives for accessible components
- **Lucide React**: Icon library

### Routing & State
- **React Router**: 7.7.0 for client-side routing
- **No global state management yet** (add Redux/Zustand when needed)

### Development Tools
- **ESLint**: 9.30.1 with TypeScript and React plugins
- **Vite Dev Server**: Hot module replacement
- **TypeScript Compiler**: Strict type checking

## Project Structure

```
frontend/
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── layout/          # Layout-specific components
│   │   │   ├── AppLayout.tsx      # ✅ Main 2-column layout
│   │   │   ├── TopNavigation.tsx  # ✅ Header with branding
│   │   │   └── LeftNavigation.tsx # ✅ Collapsible sidebar
│   │   └── ui/              # Shadcn UI components
│   │       ├── button.tsx         # ✅ Button variants
│   │       └── collapsible.tsx    # ✅ Collapsible primitive
│   ├── pages/               # Route-level page components
│   │   ├── Home.tsx               # ✅ Dashboard/landing page
│   │   ├── DataSources.tsx        # ✅ Data source management
│   │   ├── Downloads.tsx          # ✅ Download history
│   │   └── Settings.tsx           # ✅ App configuration
│   ├── lib/                 # Utility functions
│   │   └── utils.ts               # ✅ Shadcn utilities (cn, etc.)
│   ├── hooks/               # Custom React hooks (empty, ready for use)
│   ├── App.tsx              # ✅ Router setup and main app
│   ├── main.tsx             # ✅ React app entry point
│   └── index.css            # ✅ Tailwind CSS + Shadcn variables
├── public/                  # Static assets
├── components.json          # ✅ Shadcn UI configuration
├── package.json            # ✅ Dependencies and scripts
├── vite.config.ts          # ✅ Vite config with path aliases
├── tsconfig.json           # ✅ TypeScript config with paths
└── eslint.config.js        # ✅ ESLint configuration
```

## Development Commands

### From Frontend Directory
```bash
# Development server (runs on http://localhost:5173)
npm run dev

# Production build
npm run build

# Preview production build
npm run preview

# Linting
npm run lint

# Type checking (no output = success)
npx tsc --noEmit
```

### From Root Directory (Recommended)
```bash
# Start both backend and frontend
make dev

# Frontend-only development
make dev-frontend

# Quick validation (formatting, linting, type checking)
make quick-check

# Full CI simulation (tests, security, build)
make ci-check

# Frontend-specific commands
make lint-frontend
make build-frontend
make test-frontend
```

## Architecture & Patterns

### Component Organization
- **Layout Components**: `src/components/layout/` - Application shell components
- **UI Components**: `src/components/ui/` - Shadcn UI primitives (auto-generated)
- **Page Components**: `src/pages/` - Route-level components
- **Custom Hooks**: `src/hooks/` - Reusable React logic (add as needed)

### Routing Structure
- **Base Route**: `/` (AppLayout wrapper)
  - **Home**: `/` (index route)
  - **Data Sources**: `/data-sources`
  - **Downloads**: `/downloads` 
  - **Settings**: `/settings`

### Styling Approach
- **Tailwind CSS**: Utility-first styling with custom design tokens
- **CSS Variables**: Shadcn UI color system with light/dark mode support
- **Component Variants**: Using `class-variance-authority` for component APIs
- **Responsive Design**: Mobile-first approach with Tailwind breakpoints

## Code Standards

### TypeScript
- **Strict Mode**: Enabled with `noUnusedLocals` and `noUnusedParameters`
- **Path Aliases**: Use `@/` prefix for imports (`@/components`, `@/lib`, etc.)
- **Component Props**: Define interfaces for all component props
- **Type Safety**: Prefer explicit types over `any`

### React Patterns
- **Functional Components**: Use function declarations, not arrow functions for components
- **Hooks**: Follow Rules of Hooks, use custom hooks for complex logic
- **Props Drilling**: Keep props shallow, consider context for deeply nested state
- **Component Composition**: Prefer composition over complex prop interfaces

### Import Organization
```typescript
// External libraries
import { useState } from "react"
import { Link, useLocation } from "react-router-dom"

// UI components (Shadcn)
import { Button } from "@/components/ui/button"
import { Collapsible } from "@/components/ui/collapsible"

// Internal components
import { TopNavigation } from "./TopNavigation"

// Utilities
import { cn } from "@/lib/utils"
```

## Shadcn UI Integration

### Configuration
- **Style**: "new-york" (clean, minimal aesthetic)
- **Base Color**: "neutral" (gray-based color palette)
- **CSS Variables**: Enabled for theme customization
- **Icon Library**: Lucide React

### Adding Components
```bash
# Add new Shadcn components
npx shadcn@latest add [component-name]

# Examples
npx shadcn@latest add card
npx shadcn@latest add dialog
npx shadcn@latest add form
```

### Component Customization
- **Generated components** in `src/components/ui/` should not be heavily modified
- **Extend functionality** by wrapping in custom components
- **Add ESLint disable comment** for React Refresh if exporting variants: `/* eslint-disable react-refresh/only-export-components */`

## Development Workflow

### Feature Development
1. **Work from feature branches** created from main
2. **Component Development**:
   ```bash
   # Add Shadcn components first
   npx shadcn@latest add [needed-components]
   
   # Create feature components in appropriate directories
   # Update routes in App.tsx if needed
   # Test locally with npm run dev
   ```
3. **Validation before commit**:
   ```bash
   # From root directory
   make quick-check    # Fast validation
   make ci-check      # Full validation
   ```

### Code Quality Checks
- **TypeScript**: All code must pass `npx tsc --noEmit`
- **ESLint**: All code must pass `npm run lint`
- **Build**: Production build must succeed
- **Dev Server**: Must start without errors

### Testing Strategy (Future)
- **Unit Tests**: Add Vitest + React Testing Library
- **E2E Tests**: Consider Playwright for critical user flows
- **Visual Tests**: Consider Chromatic for component stories

## Integration with Backend

### API Communication (Future Implementation)
- **Backend URL**: `http://localhost:8080` (development)
- **API Endpoints**: `/api/*` routes
- **Type Safety**: Copy generated types from `backend/api-types.ts` to `frontend/src/types/api.ts`
- **HTTP Client**: Add axios or fetch abstraction in `src/services/`

### Type Generation Workflow
```bash
# When backend types change
cp backend/api-types.ts frontend/src/types/api.ts

# Or run backend type generation
cd backend && ./scripts/generate-types.sh
```

## Common Development Tasks

### Adding a New Page
1. Create component in `src/pages/NewPage.tsx`
2. Add route to `src/App.tsx`
3. Add navigation link to `src/components/layout/LeftNavigation.tsx`
4. Test navigation and responsive behavior

### Adding a New UI Component
```bash
# Check if Shadcn has the component
npx shadcn@latest add [component-name]

# If not, create custom component
touch src/components/NewComponent.tsx
```

### Customizing Theme
- Modify CSS variables in `src/index.css`
- Update Shadcn config in `components.json` if needed
- Use Tailwind utilities for component-specific styling

### Debugging Common Issues

#### TypeScript Path Resolution
- Ensure `tsconfig.json` and `vite.config.ts` both have `@/*` aliases
- Check import paths use `@/` prefix correctly

#### Shadcn Component Issues  
- Add ESLint disable comment for components exporting variants
- Check `components.json` configuration is correct
- Verify Tailwind CSS is properly configured

#### Build Failures
- Run `npx tsc --noEmit` to check TypeScript errors
- Check for unused imports with ESLint
- Verify all dependencies are installed

## Performance Considerations

### Current State
- **Bundle Size**: ~265KB gzipped (reasonable for React + UI library)
- **Code Splitting**: Not implemented (add React.lazy when needed)
- **Tree Shaking**: Working correctly with Vite

### Future Optimizations
- **Route-based code splitting** for larger applications
- **Image optimization** for assets
- **Bundle analysis** with `vite-bundle-analyzer`

## Responsive Design

### Breakpoints (Tailwind CSS)
- **sm**: 640px and up (tablet)
- **md**: 768px and up (small desktop)
- **lg**: 1024px and up (desktop)
- **xl**: 1280px and up (large desktop)

### Current Responsive Features
- **Collapsible Navigation**: Works on all screen sizes
- **2-Column Layout**: Responsive with proper overflow handling
- **Button Sizing**: Adapts to screen size
- **Typography**: Scales appropriately

## Next Steps & Roadmap

### Immediate Priorities
1. **Backend Integration**: Connect to Go API endpoints
2. **State Management**: Add Redux Toolkit or Zustand for complex state
3. **Error Handling**: Add error boundaries and user feedback
4. **Loading States**: Add skeleton components and spinners

### Feature Enhancements
1. **Data Source Connections**: Implement actual API integrations
2. **Download Management**: File upload/download functionality
3. **User Settings**: Persistent configuration storage
4. **Search & Filtering**: Add search capabilities to data views

### Technical Improvements
1. **Testing Suite**: Add comprehensive test coverage
2. **Accessibility**: ARIA labels and keyboard navigation
3. **Internationalization**: i18n support for multiple languages
4. **PWA Features**: Service worker and offline capabilities

## Troubleshooting

### Port Conflicts
```bash
# If port 5173 is in use
npm run dev -- --port 3000

# Or kill existing Vite processes
pkill -f "vite"
```

### Type Errors
```bash
# Clear TypeScript cache
rm -rf node_modules/.cache/
npx tsc --noEmit

# Verify TypeScript config
cat tsconfig.json | grep -A 5 "paths"
```

### Build Issues
```bash
# Clear all caches and reinstall
rm -rf node_modules/ dist/
npm ci
npm run build
```

## Contributing Guidelines

### Before Making Changes
- Check if similar functionality exists
- Consider responsive design implications  
- Follow established component patterns
- Run validation checks locally

### Component Naming
- **PascalCase** for component files and exports
- **camelCase** for props and variables
- **kebab-case** for route paths
- **SCREAMING_SNAKE_CASE** for constants

### Commit Messages (from root)
Follow conventional commits format:
```
feat(frontend): add user authentication flow
fix(frontend): resolve navigation state bug
style(frontend): update button component styling
```

This frontend is ready for further development and can serve as a solid foundation for the PubDataHub application.