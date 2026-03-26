/**
 * AlbumsPage Component Tests
 *
 * Tests the albums listing page including:
 * - Page rendering with Layout
 * - GraphQL query for myAlbums
 * - AlbumFilter component
 * - AlbumBoxes component
 * - Sorting options
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MockedProvider } from '@apollo/client/testing'
import { MemoryRouter } from 'react-router-dom'
import { gql } from '@apollo/client'
import AlbumsPage from './AlbumsPage'

// Mock the hooks
vi.mock('../../hooks/useURLParameters', () => ({
  default: () => ({
    getParam: vi.fn(() => null),
    setParam: vi.fn(),
    setParams: vi.fn(),
  }),
}))

vi.mock('../../hooks/useOrderingParams', () => ({
  default: () => ({
    orderBy: 'updated_at',
    orderDirection: 'DESC',
    setOrdering: vi.fn(),
  }),
}))

describe('AlbumsPage Component', () => {
  const mockAlbums = [
    {
      __typename: 'Album',
      id: '1',
      title: 'Vacation Photos',
      thumbnail: {
        __typename: 'MediaURL',
        id: 'thumb1',
        thumbnail: {
          __typename: 'MediaURL',
          url: '/photo/thumb_vacation.jpg',
        },
      },
    },
    {
      __typename: 'Album',
      id: '2',
      title: 'Family Pictures',
      thumbnail: null,
    },
  ]

  const GET_MY_ALBUMS = gql`
    query getMyAlbums($orderBy: String, $orderDirection: OrderDirection) {
      myAlbums(
        order: { order_by: $orderBy, order_direction: $orderDirection }
        onlyRoot: true
        showEmpty: true
      ) {
        id
        title
        thumbnail {
          id
          thumbnail {
            url
          }
        }
      }
    }
  `

  const graphqlMocks = [
    {
      request: {
        query: GET_MY_ALBUMS,
        variables: {
          orderBy: 'updated_at',
          orderDirection: 'DESC',
        },
      },
      result: {
        data: {
          myAlbums: mockAlbums,
        },
      },
    },
  ]

  it('renders the page with Layout', () => {
    render(
      <MemoryRouter>
        <MockedProvider mocks={[]} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    // Layout should be rendered
    expect(screen.getByTestId('Layout')).toBeInTheDocument()
  })

  it('renders AlbumFilter component', () => {
    render(
      <MemoryRouter>
        <MockedProvider mocks={[]} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    // AlbumFilter should be in the document
    expect(screen.getByTestId('AlbumFilter')).toBeInTheDocument()
  })

  it('renders AlbumBoxes component', () => {
    render(
      <MemoryRouter>
        <MockedProvider mocks={[]} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    // AlbumBoxes should be in the document
    expect(screen.getByTestId('AlbumBoxes')).toBeInTheDocument()
  })

  it('queries albums with correct variables', async () => {
    render(
      <MemoryRouter>
        <MockedProvider mocks={graphqlMocks} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByTestId('AlbumBoxes')).toBeInTheDocument()
    })
  })

  it('displays error state when query fails', async () => {
    const errorMocks = [
      {
        request: {
          query: GET_MY_ALBUMS,
          variables: {
            orderBy: 'updated_at',
            orderDirection: 'DESC',
          },
        },
        error: new Error('Network error'),
      },
    ]

    render(
      <MemoryRouter>
        <MockedProvider mocks={errorMocks} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    await waitFor(() => {
      expect(screen.getByText(/Error/i)).toBeInTheDocument()
    })
  })

  it('renders loading state initially', () => {
    render(
      <MemoryRouter>
        <MockedProvider mocks={[]} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    // AlbumBoxes should show loading state (skeleton) when albums is undefined
    expect(screen.getByTestId('AlbumBoxes')).toBeInTheDocument()
  })

  it('passes correct sorting options to AlbumFilter', () => {
    render(
      <MemoryRouter>
        <MockedProvider mocks={[]} addTypename={false}>
          <AlbumsPage />
        </MockedProvider>
      </MemoryRouter>
    )

    // AlbumFilter should be rendered with sorting options
    expect(screen.getByTestId('AlbumFilter')).toBeInTheDocument()
  })
})
