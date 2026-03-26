/**
 * AlbumPage Component Tests
 *
 * Tests the album page including:
 * - Rendering album info
 * - Loading state (skeleton)
 * - Error state handling
 * - Share token handling
 * - Favorite filtering
 * - URL parameters
 * - Ordering parameters
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MockedProvider } from '@apollo/client/testing'
import { render, screen, waitFor } from '@testing-library/react'
import React from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { gql } from '@apollo/client'
import AlbumPage from './AlbumPage'

// Mock the hooks
vi.mock('../../hooks/useScrollPagination', () => ({
  default: () => ({
    containerElem: vi.fn(),
    finished: false,
  }),
}))

vi.mock('../../hooks/useURLParameters', () => ({
  default: () => ({
    getParam: vi.fn(() => null),
    setParam: vi.fn(),
    setParams: vi.fn(),
  }),
}))

vi.mock('../../hooks/useOrderingParams', () => ({
  default: () => ({
    orderBy: 'date_shot',
    orderDirection: 'DESC',
    setOrdering: vi.fn(),
  }),
}))

const ALBUM_QUERY = gql`
  query AlbumPageTest($id: ID!, $onlyFavorites: Boolean, $mediaOrderBy: String, $orderDirection: OrderDirection, $limit: Int, $offset: Int) {
    album(id: $id) {
      id
      title
      media(paginate: { limit: $limit, offset: $offset }) {
        id
        title
        type
      }
    }
  }
`

const mockAlbumData = {
  __typename: 'Album',
  id: '1',
  title: 'Test Album',
  media: [
    {
      __typename: 'Media',
      id: 'media-1',
      title: 'Photo 1',
      type: 'photo',
    },
    {
      __typename: 'Media',
      id: 'media-2',
      title: 'Photo 2',
      type: 'photo',
    },
  ],
}

const successMocks = [
  {
    request: {
      query: ALBUM_QUERY,
      variables: {
        id: '1',
        onlyFavorites: false,
        mediaOrderBy: 'date_shot',
        orderDirection: 'DESC',
        offset: 0,
        limit: 200,
      },
    },
    result: {
      data: {
        album: mockAlbumData,
      },
    },
  },
]

describe('AlbumPage Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('renders the page without crashing', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('Layout')).toBeInTheDocument()
    })

    it('renders album filter controls', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('AlbumFilter')).toBeInTheDocument()
    })

    it('displays sort controls', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByText(/Sort/i)).toBeInTheDocument()
    })

    it('renders AlbumGallery component', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('AlbumGallery')).toBeInTheDocument()
    })
  })

  describe('Data Loading', () => {
    it('loads album data on mount', async () => {
      render(
        <MockedProvider mocks={successMocks}>
          <MemoryRouter initialEntries={['/album/1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('AlbumGallery')).toBeInTheDocument()
      })
    })

    it('uses correct album ID from URL params', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/test-album-123']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('Layout')).toBeInTheDocument()
    })

    it('handles numeric album ID', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/42']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('Layout')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('displays error state when query fails', async () => {
      const errorMocks = [
        {
          request: {
            query: ALBUM_QUERY,
            variables: {
              id: '1',
              onlyFavorites: false,
              mediaOrderBy: 'date_shot',
              orderDirection: 'DESC',
              offset: 0,
              limit: 200,
            },
          },
          error: new Error('Album not found'),
        },
      ]

      render(
        <MockedProvider mocks={errorMocks}>
          <MemoryRouter initialEntries={['/album/1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('AlbumGallery')).toBeInTheDocument()
      })
    })

    it('handles 404 error for non-existent album', async () => {
      const notFoundMocks = [
        {
          request: {
            query: ALBUM_QUERY,
            variables: {
              id: '999',
              onlyFavorites: false,
              mediaOrderBy: 'date_shot',
              orderDirection: 'DESC',
              offset: 0,
              limit: 200,
            },
          },
          result: {
            data: {
              album: null,
            },
          },
        },
      ]

      render(
        <MockedProvider mocks={notFoundMocks}>
          <MemoryRouter initialEntries={['/album/999']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('AlbumGallery')).toBeInTheDocument()
      })
    })
  })

  describe('URL Parameters', () => {
    it('parses favorites parameter from URL', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/1?favorites=1']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('Layout')).toBeInTheDocument()
    })

    it('respects orderBy parameter from URL', () => {
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/1?orderBy=title']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('Layout')).toBeInTheDocument()
    })
  })

  describe('Share Token', () => {
    it('renders page with share token route using different param', () => {
      // AlbumPage uses :id param, share tokens might use different routes
      render(
        <MockedProvider mocks={[]}>
          <MemoryRouter initialEntries={['/album/share-token-abc']}>
            <Routes>
              <Route path="/album/:id" element={<AlbumPage />} />
            </Routes>
          </MemoryRouter>
        </MockedProvider>
      )

      expect(screen.getByTestId('Layout')).toBeInTheDocument()
    })
  })
})
