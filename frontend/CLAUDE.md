# Frontend CLAUDE.md

This file provides guidance to Claude Code when working with the React frontend in this repository.

## Project Overview

This is the React frontend for PubDataHub that displays the user's home directory path by fetching it from the Go backend API.

## Technology Stack

- **Framework**: React 18+ with TypeScript
- **Build Tool**: Vite 7.0+
- **HTTP Client**: Axios
- **Styling**: Inline CSS (minimal styling approach)
- **Type Safety**: TypeScript types imported from Go backend

## Project Structure

```
frontend/
├── public/
│   └── vite.svg                 # Vite logo
├── src/
│   ├── components/
│   │   └── HomePage.tsx         # Main home page component
│   ├── services/
│   │   └── api.ts               # API service with axios
│   ├── types/
│   │   └── api.ts               # TypeScript types (copied from backend)
│   ├── App.tsx                  # Main App component
│   ├── App.css                  # App styles
│   ├── index.css                # Global styles
│   ├── main.tsx                 # React entry point
│   └── vite-env.d.ts            # Vite TypeScript definitions
├── eslint.config.js             # ESLint configuration
├── index.html                   # HTML template
├── package.json                 # Dependencies and scripts
├── tsconfig.json                # TypeScript configuration
├── tsconfig.app.json            # App-specific TypeScript config
├── tsconfig.node.json           # Node-specific TypeScript config
└── vite.config.ts               # Vite configuration
```

## Development Commands

### Starting Development
```bash
# From root directory
cd frontend && npm run dev
```
Runs on: http://localhost:5173

### Installing Dependencies
```bash
cd frontend && npm install
```

### Building for Production
```bash
cd frontend && npm run build
```

### Preview Production Build
```bash
cd frontend && npm run preview
```

## API Integration

### Backend Communication
- **Base URL**: http://localhost:8080
- **Primary Endpoint**: `GET /api/home`
- **HTTP Client**: Axios with error handling
- **CORS**: Configured on backend for frontend integration

### API Service Structure
```typescript
// src/services/api.ts
import axios from 'axios';
import { HomeResponse } from '../types/api';

const api = axios.create({
  baseURL: 'http://localhost:8080',
  headers: { 'Content-Type': 'application/json' }
});

export const homeAPI = {
  getHomeDirectory: async (): Promise<HomeResponse> => {
    // Implementation with error handling
  }
};
```

### TypeScript Types
Types are imported from the backend's generated TypeScript definitions:
```typescript
// src/types/api.ts (copied from backend/api-types.ts)
export interface HomeResponse {
  homePath: string;
}

export interface ErrorResponse {
  error: string;
  message: string;
}
```

## Component Architecture

### HomePage Component
- **Purpose**: Main component that displays user's home directory
- **State Management**: React hooks (useState, useEffect)
- **Features**: 
  - Loading state while fetching data
  - Error state for API failures
  - Success state showing home directory path
- **Styling**: Inline CSS for simplicity

### App Component
- **Purpose**: Root component that renders HomePage
- **Minimal Structure**: Just imports and renders HomePage

## Development Guidelines

### Adding New Components
1. Create components in `src/components/`
2. Use TypeScript with proper type definitions
3. Follow React functional component pattern with hooks
4. Include loading and error states for async operations

### API Integration
1. Add new API calls to `src/services/api.ts`
2. Import and use TypeScript types from `src/types/api.ts`
3. Handle errors consistently using try/catch blocks
4. Provide user feedback for loading and error states

### Type Safety
1. Always import types from `src/types/api.ts`
2. Update types when backend API changes
3. Copy latest types from backend: `cp ../backend/api-types.ts src/types/api.ts`
4. Use TypeScript strict mode for better type checking

## Common Development Tasks

### Adding a New API Endpoint
1. **Backend**: Add endpoint to Go backend
2. **Types**: Regenerate types in backend: `cd backend && ./scripts/generate-types.sh`
3. **Frontend**: Copy new types: `cp ../backend/api-types.ts src/types/api.ts`
4. **Service**: Add API call to `src/services/api.ts`
5. **Component**: Use new API in React components

### Updating Styling
- **Current Approach**: Inline CSS for simplicity
- **Future**: Consider adding Tailwind CSS or styled-components
- **Global Styles**: Modify `src/index.css` for app-wide styles
- **Component Styles**: Use `src/App.css` or inline styles

### Error Handling Best Practices
```typescript
// In components
const [error, setError] = useState<string>('');

try {
  const response = await homeAPI.getHomeDirectory();
  // Handle success
} catch (err) {
  const errorMessage = err instanceof Error ? err.message : 'Unknown error';
  setError(errorMessage);
}
```

## Git Workflow (Frontend-Specific)

### Before Starting Work
```bash
# Always start from main and get latest changes
git checkout main
git pull origin main
git checkout -b feature/frontend-feature-name

# If you need backend code that's already merged
git merge main
```

### Common Issues and Solutions

#### Missing Backend Types
**Problem**: TypeScript errors about missing types
**Solution**: 
```bash
cp ../backend/api-types.ts src/types/api.ts
```

#### Backend Server Not Running
**Problem**: API calls failing with network errors
**Solution**:
```bash
# Start backend server
cd ../backend && go run cmd/server/main.go
```

#### Port Conflicts
**Problem**: Frontend dev server won't start on 5173
**Solution**: Vite will automatically use next available port, or specify:
```bash
npm run dev -- --port 3000
```

## Testing the Application

### Manual Testing Checklist
- [ ] Frontend starts without errors: `npm run dev`
- [ ] Backend API is accessible: `curl http://localhost:8080/api/home`
- [ ] Page loads and shows "PubDataHub" title
- [ ] Loading state appears briefly when fetching data
- [ ] Home directory path displays correctly
- [ ] Error state shows if backend is down
- [ ] Responsive design works on different screen sizes

### Integration Testing
1. Start backend: `cd backend && go run cmd/server/main.go`
2. Start frontend: `cd frontend && npm run dev`
3. Open http://localhost:5173
4. Verify home directory path displays correctly
5. Test error handling by stopping backend server

## Deployment Notes

### Production Build
```bash
npm run build
```
Creates `dist/` directory with optimized static files.

### Environment Variables
For different environments, create `.env` files:
```bash
# .env.development
VITE_API_BASE_URL=http://localhost:8080

# .env.production  
VITE_API_BASE_URL=https://your-production-api.com
```

Update API service to use environment variables:
```typescript
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
```