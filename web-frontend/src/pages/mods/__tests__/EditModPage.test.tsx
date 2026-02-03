/** @vitest-environment jsdom */
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import { EditModPage } from '../EditModPage'
import { describe, it, expect, afterEach, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import * as matchers from '@testing-library/jest-dom/matchers'

expect.extend(matchers)

afterEach(() => {
  cleanup()
})

// Mock the API and toast
vi.mock('@/api', () => ({
  getMod: vi.fn(() => Promise.resolve({
    id: 'test-mod',
    name: 'Test Mod',
    author: 'Test Author',
    description: 'Test Description',
    website: 'https://test.com',
    type: 'mod'
  })),
  updateMod: vi.fn(),
  getModVersions: vi.fn(() => Promise.resolve([])),
}))
vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}))

describe('EditModPage', () => {
  it('renders edit mod form with data', async () => {
    render(
      <MemoryRouter initialEntries={['/mods/test-mod/edit']}>
        <Routes>
          <Route path="/mods/:id/edit" element={<EditModPage />} />
        </Routes>
      </MemoryRouter>
    )
    
        expect(screen.getAllByTestId('skeleton')).toHaveLength(4)
    
    await waitFor(() => {
      expect(screen.getByText('Edit Mod: Test Mod')).toBeInTheDocument()
    })

    
    expect(screen.getByDisplayValue('test-mod')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Test Mod')).toBeInTheDocument()
  })
})
