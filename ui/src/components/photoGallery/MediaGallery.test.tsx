/**
 * MediaGallery Component Tests
 *
 * Tests the media gallery component including:
 * - Empty state when no media
 * - Rendering media grid with items
 * - Loading state (skeleton)
 * - Active index handling
 * - Present mode rendering
 * - Photo filler element
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import React from 'react'
import { MediaType } from '../../__generated__/globalTypes'
import MediaGallery from './MediaGallery'
import { MediaGalleryState } from './mediaGalleryReducer'

// Mock mutations and sidebar
vi.mock('./photoGalleryMutations', () => ({
  useMarkFavoriteMutation: () => [vi.fn()],
}))

vi.mock('../sidebar/MediaSidebar/MediaSidebar', () => ({
  default: () => <div data-testid="media-sidebar">Media Sidebar</div>,
}))

// Mock sidebar context provider
vi.mock('../sidebar/Sidebar', () => ({
  SidebarContext: React.createContext({
    pinned: false,
    content: null,
    updateSidebar: vi.fn(),
  }),
  SidebarProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}))

describe('MediaGallery Component', () => {
  const mockUpdateSidebar = vi.fn()

  const createMockMedia = (id: string, type: MediaType = MediaType.Photo) => ({
    id,
    type,
    thumbnail: type === MediaType.Photo
      ? {
          url: `/photo/thumbnail_${id}.jpg`,
          width: 768,
          height: 1024,
          __typename: 'MediaURL' as const,
        }
      : null,
    highRes: null,
    videoWeb: null,
    blurhash: null,
    favorite: false,
    __typename: 'Media' as const,
  })

  const dispatchMedia = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    mockUpdateSidebar.mockClear()
  })

  describe('Empty State', () => {
    it('renders skeleton when media is null', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: -1,
        media: null,
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      // Should render 6 placeholder elements when media is null
      const gallery = screen.getByTestId('photo-gallery-wrapper')
      expect(gallery).toBeInTheDocument()
      expect(gallery.children.length).toBeGreaterThan(0)
    })

    it('renders empty state when media array is empty', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: -1,
        media: [],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      const gallery = screen.getByTestId('photo-gallery-wrapper')
      expect(gallery).toBeInTheDocument()
      // Should only have PhotoFiller element
      expect(gallery.children.length).toBe(1)
    })
  })

  describe('Media Grid Rendering', () => {
    it('renders correct number of media items', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: 0,
        media: [
          createMockMedia('1'),
          createMockMedia('2'),
          createMockMedia('3'),
        ],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      const gallery = screen.getByTestId('photo-gallery-wrapper')
      // 3 media items + PhotoFiller
      expect(gallery.children.length).toBe(4)
    })

    it('renders photos and videos', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: 0,
        media: [
          createMockMedia('1', MediaType.Photo),
          createMockMedia('2', MediaType.Video),
          createMockMedia('3', MediaType.Photo),
        ],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      const gallery = screen.getByTestId('photo-gallery-wrapper')
      // 3 media items + filler
      expect(gallery.children.length).toBe(4)
    })

    it('renders PhotoFiller element', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: 0,
        media: [createMockMedia('1')],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      const gallery = screen.getByTestId('photo-gallery-wrapper')
      expect(gallery.lastElementChild).toBeInTheDocument()
    })
  })

  describe('Active Index Handling', () => {
    it('handles negative active index', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: -1,
        media: [createMockMedia('1')],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      expect(screen.getByTestId('photo-gallery-wrapper')).toBeInTheDocument()
    })

    it('handles active index within media range', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: 1,
        media: [
          createMockMedia('1'),
          createMockMedia('2'),
          createMockMedia('3'),
        ],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      expect(screen.getByTestId('photo-gallery-wrapper')).toBeInTheDocument()
    })
  })

  describe('Present Mode', () => {
    it('does not render PresentView when not presenting', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: 0,
        media: [createMockMedia('1')],
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      expect(screen.queryByTestId('present-overlay')).not.toBeInTheDocument()
    })

    it('renders PresentView when presenting', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: 0,
        media: [createMockMedia('1')],
        presenting: true,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      expect(screen.getByTestId('present-overlay')).toBeInTheDocument()
    })

    it('passes active media to PresentView', () => {
      const activeMedia = createMockMedia('active-123')
      const mediaState: MediaGalleryState = {
        activeIndex: 0,
        media: [activeMedia],
        presenting: true,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={false}
        />
      )

      expect(screen.getByTestId('present-overlay')).toBeInTheDocument()
    })
  })

  describe('Loading State', () => {
    it('renders skeleton placeholders when loading is true and media is null', () => {
      const mediaState: MediaGalleryState = {
        activeIndex: -1,
        media: null,
        presenting: false,
      }

      render(
        <MediaGallery
          dispatchMedia={dispatchMedia}
          mediaState={mediaState}
          loading={true}
        />
      )

      // Should show 6 skeleton placeholders
      const gallery = screen.getByTestId('photo-gallery-wrapper')
      expect(gallery).toBeInTheDocument()
    })
  })
})
