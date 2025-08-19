import { describe, it, expect, vi } from 'vitest'
import { render, screen } from 'solid-testing-library'
import Settings from './Settings'

// Mock the fetch API
global.fetch = vi.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve([]),
    headers: new Headers({ 'content-type': 'application/json' }),
  } as Response)
) as any

describe('Settings Component', () => {
  it('renders loading state initially', () => {
    render(<Settings />)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('renders empty state when no jobs', async () => {
    // Mock fetch to return empty array
    vi.mocked(global.fetch).mockImplementationOnce(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve([]),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response)
    )

    render(<Settings />)
    
    // Wait for async operations
    await new Promise(resolve => setTimeout(resolve, 100))
    
    expect(screen.getByText(/no active downloads/i)).toBeInTheDocument()
  })
})