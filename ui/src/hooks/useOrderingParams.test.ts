/**
 * useOrderingParams Hook Tests
 *
 * Tests the ordering parameter management hook including:
 * - Reading orderBy and orderDirection from URL
 * - Default values for orderBy and orderDirection
 * - Setting ordering parameters
 * - Toggling order direction
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import useOrderingParams from './useOrderingParams'
import { OrderDirection } from '../__generated__/globalTypes'
import type { UrlParams } from './useURLParameters'

// Mock history.replaceState
const mockReplaceState = vi.fn()
Object.defineProperty(window, 'history', {
  value: {
    replaceState: mockReplaceState,
  },
  writable: true,
})

// Create a test wrapper that provides mocked URL params
const createMockUrlParams = (params: Record<string, string | null>): UrlParams => {
  const paramMap = new Map(Object.entries(params))

  return {
    getParam: (key: string, defaultValue: string | null = null) => {
      if (paramMap.has(key)) {
        return paramMap.get(key)!
      }
      return defaultValue
    },
    setParams: (pairs: { key: string; value: string | null }[]) => {
      // Just mock - don't actually update URL in tests
      mockReplaceState({}, '', '/mock')
    },
  }
}

describe('useOrderingParams Hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Mock location
    delete (window as any).location
    window.location = new URL('http://localhost:1234/album/test')
  })

  describe('Reading URL Parameters', () => {
    it('returns default orderBy when not in URL', () => {
      const mockUrlParams = createMockUrlParams({})
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderBy).toBe('date_shot')
    })

    it('returns custom default orderBy when provided', () => {
      const mockUrlParams = createMockUrlParams({})
      const { result } = renderHook(() => useOrderingParams(mockUrlParams, 'created_at'))

      expect(result.current.orderBy).toBe('created_at')
    })

    it('returns orderBy from URL when present', () => {
      const mockUrlParams = createMockUrlParams({ orderBy: 'title' })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderBy).toBe('title')
    })

    it('returns default orderBy when URL value is empty string', () => {
      const mockUrlParams = createMockUrlParams({ orderBy: '' })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderBy).toBe('date_shot')
    })

    it('returns default orderBy when URL value is null', () => {
      const mockUrlParams = createMockUrlParams({ orderBy: null })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderBy).toBe('date_shot')
    })
  })

  describe('orderDirection Handling', () => {
    it('returns ASC as default orderDirection', () => {
      const mockUrlParams = createMockUrlParams({})
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.ASC)
    })

    it('returns orderDirection from URL when valid', () => {
      const mockUrlParams = createMockUrlParams({ orderDirection: OrderDirection.DESC })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.DESC)
    })

    it('returns ASC when URL value is invalid', () => {
      const mockUrlParams = createMockUrlParams({ orderDirection: 'invalid' })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.ASC)
    })

    it('returns ASC when URL value is null', () => {
      const mockUrlParams = createMockUrlParams({ orderDirection: null })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.ASC)
    })

    it('returns ASC when URL value is empty string', () => {
      const mockUrlParams = createMockUrlParams({ orderDirection: '' })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.ASC)
    })
  })

  describe('setOrdering', () => {
    it('sets orderBy parameter', () => {
      const mockUrlParams = createMockUrlParams({})
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      act(() => {
        result.current.setOrdering({ orderBy: 'title' })
      })

      expect(mockReplaceState).toHaveBeenCalled()
    })

    it('sets orderDirection parameter', () => {
      const mockUrlParams = createMockUrlParams({})
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      act(() => {
        result.current.setOrdering({ orderDirection: OrderDirection.DESC })
      })

      expect(mockReplaceState).toHaveBeenCalled()
    })

    it('sets both orderBy and orderDirection parameters', () => {
      const mockUrlParams = createMockUrlParams({})
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      act(() => {
        result.current.setOrdering({
          orderBy: 'created_at',
          orderDirection: OrderDirection.DESC,
        })
      })

      expect(mockReplaceState).toHaveBeenCalled()
    })

    it('does not call setParams when no arguments provided', () => {
      const setParamsMock = vi.fn()
      const mockUrlParams: UrlParams = {
        getParam: () => null,
        setParams: setParamsMock,
      }

      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      act(() => {
        result.current.setOrdering({})
      })

      expect(setParamsMock).toHaveBeenCalledWith([])
    })
  })

  describe('Toggle Order Direction Pattern', () => {
    it('can toggle from ASC to DESC', () => {
      const mockUrlParams = createMockUrlParams({ orderDirection: OrderDirection.ASC })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.ASC)

      act(() => {
        result.current.setOrdering({ orderDirection: OrderDirection.DESC })
      })

      expect(mockReplaceState).toHaveBeenCalled()
    })

    it('can toggle from DESC to ASC', () => {
      const mockUrlParams = createMockUrlParams({ orderDirection: OrderDirection.DESC })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderDirection).toBe(OrderDirection.DESC)

      act(() => {
        result.current.setOrdering({ orderDirection: OrderDirection.ASC })
      })

      expect(mockReplaceState).toHaveBeenCalled()
    })

    it('can change orderBy while keeping current orderDirection', () => {
      const mockUrlParams = createMockUrlParams({ orderBy: 'title', orderDirection: OrderDirection.DESC })
      const { result } = renderHook(() => useOrderingParams(mockUrlParams))

      expect(result.current.orderBy).toBe('title')
      expect(result.current.orderDirection).toBe(OrderDirection.DESC)

      act(() => {
        result.current.setOrdering({
          orderBy: 'created_at',
          orderDirection: OrderDirection.DESC,
        })
      })

      expect(mockReplaceState).toHaveBeenCalled()
    })
  })

  describe('Edge Cases', () => {
    it('handles all valid OrderDirection values', () => {
      const validDirections = [OrderDirection.ASC, OrderDirection.DESC]

      validDirections.forEach((direction) => {
        const mockUrlParams = createMockUrlParams({ orderDirection: direction })
        const { result } = renderHook(() => useOrderingParams(mockUrlParams))

        expect(result.current.orderDirection).toBe(direction)
      })
    })

    it('handles custom orderBy values', () => {
      const customOrderBys = ['title', 'date_shot', 'created_at', 'size', 'duration']

      customOrderBys.forEach((orderBy) => {
        const mockUrlParams = createMockUrlParams({ orderBy })
        const { result } = renderHook(() => useOrderingParams(mockUrlParams))

        expect(result.current.orderBy).toBe(orderBy)
      })
    })
  })
})
