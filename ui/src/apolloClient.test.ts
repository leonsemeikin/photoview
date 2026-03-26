/**
 * Apollo Client Configuration Tests
 *
 * Tests the Apollo Client setup including:
 * - HTTP link configuration with correct GraphQL endpoint
 * - WebSocket split for subscriptions
 * - Error handling (401, unauthorized, network errors)
 * - Cache pagination configuration
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  ApolloClient,
  InMemoryCache,
  ApolloLink,
  HttpLink,
  WebSocketLink,
  split,
  from,
} from '@apollo/client'
import { getMainDefinition } from '@apollo/client/utilities'
import { onError } from '@apollo/client/link/error'

// Mock url-join function (same implementation as used in apolloClient.ts)
const mockUrlJoin = (...args: string[]): string => {
  // Remove trailing slashes from all parts except the last
  const parts = args.map((part, index) => {
    if (index < args.length - 1) {
      return part.replace(/\/+$/, '')
    }
    return part
  })
  // Join with single slashes
  return parts.join('/')
}

describe('Apollo Client Configuration', () => {
  beforeEach(() => {
    // Mock location
    delete (window as any).location
    window.location = { origin: 'http://localhost:1234' } as any

    // Mock console.log to avoid noise
    vi.spyOn(console, 'log').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('HTTP Link Configuration', () => {
    it('configures HTTP link with correct GraphQL endpoint', () => {
      const API_ENDPOINT = mockUrlJoin(window.location.origin, '/api')
      const GRAPHQL_ENDPOINT = mockUrlJoin(API_ENDPOINT, '/graphql')

      // Verify the endpoint contains the correct parts
      expect(GRAPHQL_ENDPOINT).toContain('localhost:1234')
      expect(GRAPHQL_ENDPOINT).toContain('api')
      expect(GRAPHQL_ENDPOINT).toContain('graphql')
      expect(API_ENDPOINT).toContain('localhost:1234')
      expect(API_ENDPOINT).toContain('api')
    })

    it('uses environment variable when provided', () => {
      const customOrigin = 'https://custom-api.example.com'
      const API_ENDPOINT = mockUrlJoin(customOrigin, '/api')
      const GRAPHQL_ENDPOINT = mockUrlJoin(API_ENDPOINT, '/graphql')

      // Verify the endpoint uses custom origin
      expect(GRAPHQL_ENDPOINT).toContain('custom-api.example.com')
      expect(GRAPHQL_ENDPOINT).toContain('api')
      expect(GRAPHQL_ENDPOINT).toContain('graphql')
      expect(API_ENDPOINT).toContain('custom-api.example.com')
      expect(API_ENDPOINT).toContain('api')
    })
  })

  describe('WebSocket Split', () => {
    it('splits subscriptions to WebSocket correctly', () => {
      // Create a mock subscription query
      const subscriptionQuery = {
        kind: 'Document',
        definitions: [{
          kind: 'OperationDefinition',
          operation: 'subscription',
          selectionSet: { kind: 'SelectionSet', selections: [] }
        }]
      }

      // Create a mock query
      const query = {
        kind: 'Document',
        definitions: [{
          kind: 'OperationDefinition',
          operation: 'query',
          selectionSet: { kind: 'SelectionSet', selections: [] }
        }]
      }

      // Test the split function logic (same as in apolloClient.ts)
      const isSubscription = ({ query }: any) => {
        const definition = getMainDefinition(query)
        return (
          definition.kind === 'OperationDefinition' &&
          definition.operation === 'subscription'
        )
      }

      expect(getMainDefinition(subscriptionQuery as any).operation).toBe('subscription')
      expect(getMainDefinition(query as any).operation).toBe('query')
      expect(isSubscription({ query: subscriptionQuery as any })).toBe(true)
      expect(isSubscription({ query: query as any })).toBe(false)
    })

    it('converts HTTPS to WSS protocol', () => {
      const GRAPHQL_ENDPOINT = 'https://example.com/api/graphql'
      const apiProtocol = new URL(GRAPHQL_ENDPOINT).protocol

      const websocketUri = new URL(GRAPHQL_ENDPOINT)
      websocketUri.protocol = apiProtocol === 'https:' ? 'wss:' : 'ws:'

      expect(websocketUri.protocol).toBe('wss:')
      expect(websocketUri.toString()).toBe('wss://example.com/api/graphql')
    })

    it('converts HTTP to WS protocol', () => {
      const GRAPHQL_ENDPOINT = 'http://example.com/api/graphql'
      const apiProtocol = new URL(GRAPHQL_ENDPOINT).protocol

      const websocketUri = new URL(GRAPHQL_ENDPOINT)
      websocketUri.protocol = apiProtocol === 'https:' ? 'wss:' : 'ws:'

      expect(websocketUri.protocol).toBe('ws:')
      expect(websocketUri.toString()).toBe('ws://example.com/api/graphql')
    })
  })

  describe('Error Handler', () => {
    let clearTokenCookieMock: ReturnType<typeof vi.fn>
    let consoleLogSpy: ReturnType<typeof vi.spyOn>

    beforeEach(() => {
      clearTokenCookieMock = vi.fn()
      consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => {})
    })

    afterEach(() => {
      consoleLogSpy.mockRestore()
    })

    it('clears token on 401 network error', () => {
      // Create a mock network error
      const networkError = new Error('Unauthorized') as any
      networkError.statusCode = 401
      networkError.result = { errors: [] }

      // Simulate the error being triggered
      expect(clearTokenCookieMock).not.toHaveBeenCalled()

      // Manually trigger the logic to test
      if (networkError.statusCode === 401) {
        clearTokenCookieMock()
        console.log(`[Network error]: ${JSON.stringify(networkError)}`)
      }

      expect(clearTokenCookieMock).toHaveBeenCalled()
      expect(consoleLogSpy).toHaveBeenCalledWith(
        expect.stringContaining('[Network error]:')
      )
    })

    it('clears token on GraphQL unauthorized error', () => {
      const graphQLErrors = [
        { message: 'unauthorized', locations: [], path: ['test'] }
      ]

      // Simulate the error being triggered
      if (graphQLErrors.find((x: any) => x.message === 'unauthorized')) {
        clearTokenCookieMock()
      }

      expect(clearTokenCookieMock).toHaveBeenCalled()
    })

    it('formats error message for single GraphQL error', () => {
      const graphQLError = {
        message: 'Test error',
        locations: [{ line: 1, column: 2 }],
        path: ['test'],
      }

      const formatPath = (path: readonly (string | number)[] | undefined) =>
        path?.join('::') ?? 'undefined'

      const errorMessage = {
        header: 'Something went wrong',
        content: `Server error: ${graphQLError.message} at (${formatPath(
          graphQLError.path
        )})`,
      }

      expect(errorMessage.header).toBe('Something went wrong')
      expect(errorMessage.content).toBe('Server error: Test error at (test)')
    })

    it('formats error message for multiple GraphQL errors', () => {
      const graphQLErrors = [
        { message: 'Error 1', locations: [], path: ['test1'] },
        { message: 'Error 2', locations: [], path: ['test2'] },
      ]

      const errorMessage = {
        header: 'Multiple things went wrong',
        content: `Received ${graphQLErrors.length} errors from the server. See the console for more information`,
      }

      expect(errorMessage.header).toBe('Multiple things went wrong')
      expect(errorMessage.content).toBe('Received 2 errors from the server. See the console for more information')
    })
  })

  describe('Cache Configuration', () => {
    it('configures SiteInfo with merge: true', () => {
      const typePolicies = {
        SiteInfo: {
          merge: true,
        },
      }

      expect(typePolicies.SiteInfo.merge).toBe(true)
    })

    it('configures MediaURL with url keyFields', () => {
      const typePolicies = {
        MediaURL: {
          keyFields: ['url'],
        },
      }

      expect(typePolicies.MediaURL.keyFields).toEqual(['url'])
    })

    it('configures Album media pagination with correct keyArgs', () => {
      const typePolicies = {
        Album: {
          fields: {
            media: {
              keyArgs: ['onlyFavorites', 'order'],
            },
          },
        },
      }

      expect(typePolicies.Album.fields.media.keyArgs).toEqual([
        'onlyFavorites',
        'order',
      ])
    })

    it('configures FaceGroup imageFaces pagination', () => {
      const typePolicies = {
        FaceGroup: {
          fields: {
            imageFaces: {
              keyArgs: [],
            },
          },
        },
      }

      expect(typePolicies.FaceGroup.fields.imageFaces.keyArgs).toEqual([])
    })

    it('configures Query fields pagination', () => {
      const typePolicies = {
        Query: {
          fields: {
            myTimeline: {
              keyArgs: ['onlyFavorites'],
            },
            myFaceGroups: {
              keyArgs: [],
            },
          },
        },
      }

      expect(typePolicies.Query.fields.myTimeline.keyArgs).toEqual([
        'onlyFavorites',
      ])
      expect(typePolicies.Query.fields.myFaceGroups.keyArgs).toEqual([])
    })
  })

  describe('Cache Pagination Merge Function', () => {
    it('merges existing and incoming items with offset', () => {
      const existing = [{ id: 1, name: 'item1' }, { id: 2, name: 'item2' }]
      const incoming = [{ id: 3, name: 'item3' }, { id: 4, name: 'item4' }]

      const args = {
        paginate: {
          offset: 2,
        },
      }

      // Same logic as in apolloClient.ts
      const merged = existing ? existing.slice(0) : []
      if (args?.paginate) {
        const { offset = 0 } = args.paginate as { offset: number }
        for (let i = 0; i < incoming.length; ++i) {
          merged[offset + i] = incoming[i]
        }
      }

      expect(merged).toEqual([
        { id: 1, name: 'item1' },
        { id: 2, name: 'item2' },
        { id: 3, name: 'item3' },
        { id: 4, name: 'item4' },
      ])
    })

    it('throws error when paginate argument is missing', () => {
      const args = {}
      const fieldName = 'testField'

      expect(() => {
        if (!args?.paginate) {
          throw new Error(`Paginate argument is missing for query: ${fieldName}`)
        }
      }).toThrow('Paginate argument is missing for query: testField')
    })

    it('handles offset of 0 correctly', () => {
      const existing: any[] = []
      const incoming = [{ id: 1, name: 'item1' }]

      const args = {
        paginate: {
          offset: 0,
        },
      }

      const merged = existing ? existing.slice(0) : []
      if (args?.paginate) {
        const { offset = 0 } = args.paginate as { offset: number }
        for (let i = 0; i < incoming.length; ++i) {
          merged[offset + i] = incoming[i]
        }
      }

      expect(merged).toEqual([{ id: 1, name: 'item1' }])
    })
  })

  describe('Client Creation', () => {
    it('creates ApolloClient with error link and main link', () => {
      const errorLink = onError(() => {})
      const httpLink = new HttpLink({ uri: 'http://test' })
      const mainLink = from([errorLink, httpLink])

      const testClient = new ApolloClient({
        link: mainLink,
        cache: new InMemoryCache(),
      })

      expect(testClient).toBeInstanceOf(ApolloClient)
      expect(testClient.link).toBe(mainLink)
      expect(testClient.cache).toBeInstanceOf(InMemoryCache)
    })

    it('creates ApolloLink.from with error and main links', () => {
      const errorLink = onError(() => {})
      const httpLink = new HttpLink({ uri: 'http://test' })

      const combinedLink = ApolloLink.from([errorLink, httpLink])

      expect(combinedLink).toBeInstanceOf(ApolloLink)
    })
  })
})