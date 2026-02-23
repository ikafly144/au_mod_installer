import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ManualVersionForm } from './ManualVersionForm'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { createVersion } from '../../api'
import { useToast } from '../../hooks/use-toast'

// Mock react-router-dom useParams
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useParams: () => ({ id: 'test-mod-id' }),
    useNavigate: () => vi.fn(),
  }
})

// Mock the API call
vi.mock('../../api', () => ({
  createVersion: vi.fn(),
}))

// Mock useToast
vi.mock('../../hooks/use-toast', () => ({
  useToast: () => ({
    toast: vi.fn(),
  }),
}))

describe('ManualVersionForm', () => {
  const mockCreateVersion = createVersion as vi.Mock
  const mockToast = useToast().toast as vi.Mock

  beforeEach(() => {
    vi.clearAllMocks()
    mockCreateVersion.mockResolvedValue({}) // Default success
  })

  it('renders correctly with initial fields', () => {
    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<ManualVersionForm />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(screen.getByLabelText(/Version ID/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/File URL/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/File Type/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Compatible Game Versions/i)).toBeInTheDocument()
    expect(screen.getByText(/Add File/i)).toBeInTheDocument()
    expect(screen.getByText(/Add Dependency/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Create Version/i })).toBeInTheDocument()
  })

  it('allows adding and removing mod files', () => {
    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<ManualVersionForm />} />
        </Routes>
      </MemoryRouter>,
    )

    const addFileButton = screen.getByText(/Add File/i)
    fireEvent.click(addFileButton)
    expect(screen.getAllByLabelText(/File URL/i)).toHaveLength(2)

    const removeFileButtons = screen.getAllByRole('button', { name: /Minus Circle/i })
    fireEvent.click(removeFileButtons[0]) // Remove the first file
    expect(screen.getAllByLabelText(/File URL/i)).toHaveLength(1)
  })

  it('allows adding and removing dependencies', () => {
    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<ManualVersionForm />} />
        </Routes>
      </MemoryRouter>,
    )

    const addDependencyButton = screen.getByText(/Add Dependency/i)
    fireEvent.click(addDependencyButton)
    expect(screen.getAllByLabelText(/Mod ID/i)).toHaveLength(1)

    const removeDependencyButtons = screen.getAllByRole('button', { name: /Minus Circle/i })
    fireEvent.click(removeDependencyButtons[0]) // Remove the first dependency
    expect(screen.queryByLabelText(/Mod ID/i)).not.toBeInTheDocument()
  })

  it('submits form with correct data on success', async () => {
    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<ManualVersionForm />} />
        </Routes>
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText(/Version ID/i), { target: { value: 'v1.0.0' } })
    fireEvent.change(screen.getByLabelText(/File URL/i), { target: { value: 'http://example.com/mod.zip' } })
    fireEvent.change(screen.getByLabelText(/Compatible Game Versions/i), { target: { value: '2023.1.1' } })

    // Add a dependency
    fireEvent.click(screen.getByText(/Add Dependency/i))
    fireEvent.change(screen.getByLabelText(/Mod ID/i), { target: { value: 'dep-mod' } })
    fireEvent.change(screen.getByLabelText(/Required Version \(optional\)/i), { target: { value: 'v0.5.0' } })

    fireEvent.click(screen.getByRole('button', { name: /Create Version/i }))

    await waitFor(() => {
      expect(mockCreateVersion).toHaveBeenCalledWith('test-mod-id', {
        id: 'v1.0.0',
        mod_id: 'test-mod-id',
        files: [
          { url: 'http://example.com/mod.zip', file_type: 'normal' },
        ],
        game_versions: ['2023.1.1'],
        dependencies: [{ id: 'dep-mod', version: 'v0.5.0' }],
      })
    })
    expect(mockToast).toHaveBeenCalledWith({
      title: 'Success',
      description: 'Version v1.0.0 created successfully.',
    })
  })

  it('shows error toast on form submission failure', async () => {
    mockCreateVersion.mockRejectedValue(new Error('API error'))

    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<ManualVersionForm />} />
        </Routes>
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText(/Version ID/i), { target: { value: 'v1.0.0' } })
    fireEvent.change(screen.getByLabelText(/File URL/i), { target: { value: 'http://example.com/mod.zip' } })

    fireEvent.click(screen.getByRole('button', { name: /Create Version/i }))

    await waitFor(() => {
      expect(mockToast).toHaveBeenCalledWith({
        variant: 'destructive',
        title: 'Error',
        description: 'Failed to create version: API error',
      })
    })
  })

  it('displays validation errors', async () => {
    render(
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<ManualVersionForm />} />
        </Routes>
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText(/File URL/i), { target: { value: 'invalid-url' } })
    fireEvent.click(screen.getByRole('button', { name: /Create Version/i }))

    await waitFor(() => {
      expect(screen.getByText(/Must be a valid URL/i)).toBeInTheDocument()
    })
  })
})
