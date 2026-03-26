/**
 * SettingsPage Component Tests
 *
 * Tests the settings page including:
 * - Page rendering with Layout
 * - UserPreferences component (always shown)
 * - Admin-only sections (ScannerSection, UsersTable)
 * - VersionInfo component (always shown)
 * - Conditional rendering based on isAdmin
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import SettingsPage, { SectionTitle, InputLabelTitle, InputLabelDescription } from './SettingsPage'

// Mock Layout dependencies
vi.mock('../../components/layout/Layout', () => ({
  default: ({ children, title }: { children: React.ReactNode; title: string }) => (
    <div data-testid="Layout" data-title={title}>
      {children}
    </div>
  ),
}))

// Mock child components
vi.mock('./UserPreferences', () => ({
  default: () => <div data-testid="UserPreferences">User Preferences</div>,
}))

vi.mock('./ScannerSection', () => ({
  default: () => <div data-testid="ScannerSection">Scanner Section</div>,
}))

vi.mock('./Users/UsersTable', () => ({
  default: () => <div data-testid="UsersTable">Users Table</div>,
}))

vi.mock('./VersionInfo', () => ({
  default: () => <div data-testid="VersionInfo">Version Info</div>,
}))

// Mock useIsAdmin - returns true by default
const mockUseIsAdmin = vi.fn(() => true)

vi.mock('../../components/routes/AuthorizedRoute', () => ({
  useIsAdmin: () => mockUseIsAdmin(),
}))

describe('SettingsPage Component', () => {
  beforeEach(() => {
    // Reset to admin (true) before each test
    mockUseIsAdmin.mockReturnValue(true)
  })

  it('renders the page with Layout', () => {
    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    )

    expect(screen.getByTestId('Layout')).toBeInTheDocument()
  })

  it('always renders UserPreferences component', () => {
    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    )

    expect(screen.getByTestId('UserPreferences')).toBeInTheDocument()
  })

  it('always renders VersionInfo component', () => {
    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    )

    expect(screen.getByTestId('VersionInfo')).toBeInTheDocument()
  })

  it('renders admin-only sections when user is admin', () => {
    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    )

    expect(screen.getByTestId('ScannerSection')).toBeInTheDocument()
    expect(screen.getByTestId('UsersTable')).toBeInTheDocument()
  })

  it('does not render admin-only sections when user is not admin', () => {
    mockUseIsAdmin.mockReturnValue(false)

    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    )

    expect(screen.queryByTestId('ScannerSection')).not.toBeInTheDocument()
    expect(screen.queryByTestId('UsersTable')).not.toBeInTheDocument()

    // Non-admin sections should still be visible
    expect(screen.getByTestId('UserPreferences')).toBeInTheDocument()
    expect(screen.getByTestId('VersionInfo')).toBeInTheDocument()
  })

  it('renders all components in correct order for admin user', () => {
    render(
      <MemoryRouter>
        <SettingsPage />
      </MemoryRouter>
    )

    const container = screen.getByTestId('Layout').parentElement
    const children = container?.querySelectorAll('[data-testid]') || []

    // Check that UserPreferences comes before ScannerSection
    const userPrefsIndex = Array.from(children).findIndex(
      (el) => el.getAttribute('data-testid') === 'UserPreferences'
    )
    const scannerIndex = Array.from(children).findIndex(
      (el) => el.getAttribute('data-testid') === 'ScannerSection'
    )

    expect(userPrefsIndex).toBeGreaterThanOrEqual(0)
    expect(scannerIndex).toBeGreaterThanOrEqual(0)
  })
})

describe('SettingsPage Styled Components', () => {
  it('SectionTitle renders with correct classes', () => {
    render(
      <MemoryRouter>
        <SectionTitle>Test Section</SectionTitle>
      </MemoryRouter>
    )

    const title = screen.getByText('Test Section')
    expect(title).toBeInTheDocument()
    expect(title.tagName).toBe('H2')
  })

  it('SectionTitle removes margin-top when nospace prop is true', () => {
    render(
      <MemoryRouter>
        <SectionTitle nospace>No Space Section</SectionTitle>
      </MemoryRouter>
    )

    const title = screen.getByText('No Space Section')
    expect(title).toBeInTheDocument()
    // nospace removes the mt-6 class
    expect(title.className).not.toContain('mt-6')
  })

  it('SectionTitle has margin-top when nospace prop is false', () => {
    render(
      <MemoryRouter>
        <SectionTitle nospace={false}>Space Section</SectionTitle>
      </MemoryRouter>
    )

    const title = screen.getByText('Space Section')
    expect(title).toBeInTheDocument()
    // should have mt-6 class
    expect(title.className).toContain('mt-6')
  })

  it('InputLabelTitle renders with correct classes', () => {
    render(
      <MemoryRouter>
        <InputLabelTitle>Input Label</InputLabelTitle>
      </MemoryRouter>
    )

    const label = screen.getByText('Input Label')
    expect(label).toBeInTheDocument()
    expect(label.tagName).toBe('H3')
    expect(label.className).toContain('font-semibold')
  })

  it('InputLabelDescription renders with correct classes', () => {
    render(
      <MemoryRouter>
        <InputLabelDescription>Input Description</InputLabelDescription>
      </MemoryRouter>
    )

    const description = screen.getByText('Input Description')
    expect(description).toBeInTheDocument()
    expect(description.tagName).toBe('P')
    expect(description.className).toContain('text-sm')
  })
})
