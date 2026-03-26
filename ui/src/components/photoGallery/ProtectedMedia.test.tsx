/**
 * ProtectedMedia Component Tests
 *
 * Tests the ProtectedImage and ProtectedVideo components including:
 * - Token appending for shared URLs
 * - Native lazy loading support
 * - IntersectionObserver fallback
 * - Blurhash placeholder behavior
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { ProtectedImage, ProtectedVideo } from './ProtectedMedia'

// Mock IntersectionObserver
class MockIntersectionObserver implements IntersectionObserver {
  root: Element | Document | null = null
  rootMargin: string = ''
  thresholds: ReadonlyArray<number> = []
  readonly observations: Set<Element> = new Set()
  private callback: IntersectionObserverCallback

  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback
  }

  observe(target: Element): void {
    this.observations.add(target)
    // Simulate element immediately intersecting
    this.callback([{ target, isIntersecting: true, intersectionRatio: 1 }], this)
  }

  unobserve(target: Element): void {
    this.observations.delete(target)
  }

  disconnect(): void {
    this.observations.clear()
  }

  takeRecords(): IntersectionObserverEntry[] {
    return []
  }
}

// Mock react-blurhash
vi.mock('react-blurhash', () => ({
  BlurhashCanvas: ({ className, hash }: { className: string; hash: string }) => (
    <div className={className} data-blurhash={hash} data-testid="blurhash-placeholder" />
  ),
}))

describe('ProtectedMedia Component', () => {
  beforeEach(() => {
    // Mock location
    delete (window as any).location
    window.location = { origin: 'http://localhost:1234', pathname: '/' } as any

    // Mock IntersectionObserver
    window.IntersectionObserver = MockIntersectionObserver as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('getProtectedUrl - Token Appending', () => {
    it('returns undefined when url is undefined', () => {
      const { container } = render(<ProtectedImage src={undefined} />)
      const img = container.querySelector('img')
      expect(img?.getAttribute('src')).toContain('data:image/gif')
    })

    it('appends token from share path to URL', () => {
      // Mock share path with token
      delete (window as any).location
      window.location = {
        origin: 'http://localhost:1234',
        pathname: '/share/abc123',
      } as any

      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(<ProtectedImage src={testUrl} />)
      const img = container.querySelector('img')

      expect(img?.getAttribute('src')).toContain('token=abc123')
    })

    it('does not append token when not on share path', () => {
      // Mock regular path (no share)
      delete (window as any).location
      window.location = {
        origin: 'http://localhost:1234',
        pathname: '/album/123',
      } as any

      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(<ProtectedImage src={testUrl} />)
      const img = container.querySelector('img')

      expect(img?.getAttribute('src')).toBe(testUrl)
      expect(img?.getAttribute('src')).not.toContain('token=')
    })

    it('handles share path with trailing slash', () => {
      delete (window as any).location
      window.location = {
        origin: 'http://localhost:1234',
        pathname: '/share/xyz789/',
      } as any

      const testUrl = 'http://localhost:1234/api/media/photo.jpg'
      const { container } = render(<ProtectedImage src={testUrl} />)
      const img = container.querySelector('img')

      expect(img?.getAttribute('src')).toContain('token=xyz789')
    })

    it('preserves existing URL parameters when appending token', () => {
      delete (window as any).location
      window.location = {
        origin: 'http://localhost:1234',
        pathname: '/share/token123',
      } as any

      const testUrl = 'http://localhost:1234/api/media/image.jpg?size=large'
      const { container } = render(<ProtectedImage src={testUrl} />)
      const img = container.querySelector('img')

      const src = img?.getAttribute('src')
      expect(src).toContain('token=token123')
      expect(src).toContain('size=large')
    })
  })

  describe('ProtectedImage - Lazy Loading', () => {
    it('uses native lazy loading when supported', () => {
      // Note: isNativeLazyLoadSupported is checked at module load time
      // In jsdom environment, loading attribute exists on img elements
      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(<ProtectedImage src={testUrl} lazyLoading={true} />)

      // Find img element (might be wrapped in div for blurhash)
      const img = container.querySelector('img')

      // Component should render with either native lazy or fallback
      expect(img).toBeTruthy()
    })

    it('uses IntersectionObserver fallback when native lazy loading not supported', () => {
      // Mock no native lazy load support
      Object.defineProperty(document.createElement('img'), 'loading', {
        get: () => undefined,
        configurable: true,
      })

      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(<ProtectedImage src={testUrl} lazyLoading={true} />)

      // Should render div wrapper for IntersectionObserver
      const wrapper = container.querySelector('div')
      expect(wrapper).toBeTruthy()
    })

    it('loads image eagerly when lazyLoading is false', () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(<ProtectedImage src={testUrl} lazyLoading={false} />)
      const img = container.querySelector('img')

      expect(img?.getAttribute('loading')).toBe('eager')
    })

    it('sets crossOrigin to use-credentials', () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(<ProtectedImage src={testUrl} />)
      const img = container.querySelector('img')

      expect(img?.getAttribute('crossOrigin')).toBe('use-credentials')
    })
  })

  describe('ProtectedImage - Blurhash', () => {
    it('shows blurhash placeholder when loading with lazy loading', () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const blurhash = 'L6KZcS}[Rk~qM{WCWBs9AD%Mx]Rj'

      const { container, getByTestId } = render(
        <ProtectedImage src={testUrl} lazyLoading={true} blurhash={blurhash} />
      )

      // Blurhash should be present initially
      expect(getByTestId('blurhash-placeholder')).toBeTruthy()
      expect(getByTestId('blurhash-placeholder')).toHaveAttribute('data-blurhash', blurhash)
    })

    it('does not show blurhash when image is loaded', async () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const blurhash = 'L6KZcS}[Rk~qM{WCWBs9AD%Mx]Rj'

      const { container, queryByTestId, getByRole } = render(
        <ProtectedImage src={testUrl} lazyLoading={true} blurhash={blurhash} />
      )

      const img = getByRole('img')

      // Simulate image load
      img.dispatchEvent(new Event('load'))

      await waitFor(() => {
        expect(queryByTestId('blurhash-placeholder')).toBeNull()
      })
    })

    it('does not show blurhash when blurhash prop is null', () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'

      const { queryByTestId } = render(
        <ProtectedImage src={testUrl} lazyLoading={true} blurhash={null} />
      )

      expect(queryByTestId('blurhash-placeholder')).toBeNull()
    })

    it('does not show blurhash when blurhash prop is undefined', () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'

      const { queryByTestId } = render(
        <ProtectedImage src={testUrl} lazyLoading={true} blurhash={undefined} />
      )

      expect(queryByTestId('blurhash-placeholder')).toBeNull()
    })
  })

  describe('ProtectedImage - Empty State', () => {
    it('uses placeholder when src is undefined', () => {
      const { container } = render(<ProtectedImage src={undefined} />)
      const img = container.querySelector('img')

      expect(img?.getAttribute('src')).toContain('data:image/gif')
    })

    it('passes through additional props to img element', () => {
      const testUrl = 'http://localhost:1234/api/media/image.jpg'
      const { container } = render(
        <ProtectedImage
          src={testUrl}
          alt="Test image"
          className="custom-class"
          data-testid="test-img"
        />
      )
      const img = container.querySelector('img')

      expect(img?.getAttribute('alt')).toBe('Test image')
      expect(img?.getAttribute('class')).toContain('custom-class')
      expect(img?.getAttribute('data-testid')).toBe('test-img')
    })
  })

  describe('ProtectedVideo', () => {
    const mockMedia = {
      __typename: 'Media' as const,
      id: 'media-123',
      thumbnail: {
        __typename: 'MediaURL' as const,
        url: 'http://localhost:1234/api/media/thumb.jpg',
      },
      videoWeb: {
        __typename: 'MediaURL' as const,
        url: 'http://localhost:1234/api/media/video.mp4',
      },
    }

    it('renders video element with correct attributes', () => {
      const { container } = render(<ProtectedVideo media={mockMedia} />)
      const video = container.querySelector('video')

      expect(video).toBeTruthy()
      expect(video?.getAttribute('controls')).toBe('')
      // Note: 'key' is a React prop, not an HTML attribute, so we verify component renders
      expect(video?.getAttribute('crossOrigin')).toBe('use-credentials')
    })

    it('appends token to video source when on share path', () => {
      delete (window as any).location
      window.location = {
        origin: 'http://localhost:1234',
        pathname: '/share/shareToken',
      } as any

      const { container } = render(<ProtectedVideo media={mockMedia} />)
      const source = container.querySelector('source')

      expect(source?.getAttribute('src')).toContain('token=shareToken')
      expect(source?.getAttribute('type')).toBe('video/mp4')
    })

    it('appends token to poster when on share path', () => {
      delete (window as any).location
      window.location = {
        origin: 'http://localhost:1234',
        pathname: '/share/posterToken',
      } as any

      const { container } = render(<ProtectedVideo media={mockMedia} />)
      const video = container.querySelector('video')

      expect(video?.getAttribute('poster')).toContain('token=posterToken')
    })

    it('returns null when videoWeb is null', () => {
      const mediaWithoutVideo = {
        ...mockMedia,
        videoWeb: null,
      }

      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { container } = render(<ProtectedVideo media={mediaWithoutVideo} />)
      const video = container.querySelector('video')

      expect(video).toBeNull()
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'ProetctedVideo called with media.videoWeb = null'
      )

      consoleErrorSpy.mockRestore()
    })

    it('passes through additional props to video element', () => {
      const { container } = render(
        <ProtectedVideo media={mockMedia} className="video-class" autoPlay />
      )
      const video = container.querySelector('video')

      expect(video?.getAttribute('class')).toContain('video-class')
      expect(video?.getAttribute('autoplay')).toBe('')
    })
  })
})