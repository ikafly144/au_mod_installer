/** @vitest-environment jsdom */
import { render, screen, cleanup } from '@testing-library/react'
import { CreateModPage } from '../CreateModPage'
import { describe, it, expect, afterEach, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'

import * as matchers from '@testing-library/jest-dom/matchers'

expect.extend(matchers)

afterEach(() => {
  cleanup()
})

// Mock the API and toast
vi.mock('@/api', () => ({
  createMod: vi.fn(),
}))
vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}))

describe('CreateModPage', () => {
  it('renders create mod form', () => {
        render(
      <MemoryRouter>
        <CreateModPage />
      </MemoryRouter>
    )

    expect(screen.getByText('Create New Mod')).toBeInTheDocument()
    expect(screen.getByLabelText('ID')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
  })
})
