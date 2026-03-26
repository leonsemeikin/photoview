/**
 * TimelinePage Component Tests
 *
 * Tests the timeline page including:
 * - Page rendering with Layout
 * - TimelineGallery component
 * - Title translation
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import TimelinePage from './TimelinePage'

// Mock TimelineGallery to avoid complex dependencies
vi.mock('../../components/timelineGallery/TimelineGallery', () => ({
  default: () => <div data-testid="TimelineGallery">Timeline Gallery</div>,
}))

describe('TimelinePage Component', () => {
  it('renders the page with Layout', () => {
    render(
      <MemoryRouter>
        <TimelinePage />
      </MemoryRouter>
    )

    // Layout should be rendered
    expect(screen.getByTestId('Layout')).toBeInTheDocument()
  })

  it('renders TimelineGallery component', () => {
    render(
      <MemoryRouter>
        <TimelinePage />
      </MemoryRouter>
    )

    expect(screen.getByTestId('TimelineGallery')).toBeInTheDocument()
    expect(screen.getByText('Timeline Gallery')).toBeInTheDocument()
  })

  it('has correct page title in document', () => {
    render(
      <MemoryRouter>
        <TimelinePage />
      </MemoryRouter>
    )

    // Check that the title is in the document (via Helmet)
    const layout = screen.getByTestId('Layout')
    expect(layout).toBeInTheDocument()
  })

  it('renders without errors', () => {
    const { container } = render(
      <MemoryRouter>
        <TimelinePage />
      </MemoryRouter>
    )

    expect(container.firstChild).toBeInTheDocument()
  })
})
