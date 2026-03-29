/**
 * Unit Tests for SeverityChecker view behaviour
 * Validates: Requirements 1.5, 4.3
 */

import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import SeverityChecker from '../views/SeverityChecker.vue'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('SeverityChecker unit tests', () => {
  /**
   * Validates: Requirements 1.5
   * While loading, button is disabled and loading indicator is present
   */
  it('while loading, button is disabled and loading indicator is present', async () => {
    // Mock fetch to return a promise that never resolves
    const mockFetch = vi.fn().mockReturnValue(new Promise(() => {}))
    vi.stubGlobal('fetch', mockFetch)

    const wrapper = mount(SeverityChecker)

    // Set non-empty text
    const textarea = wrapper.find('textarea')
    await textarea.setValue('some symptom text')
    await nextTick()

    // Click submit
    const button = wrapper.find('button[disabled]').exists()
      ? wrapper.find('button')
      : wrapper.find('button')
    await button.trigger('click')
    await nextTick()

    // Button should be disabled
    const submitButton = wrapper.find('button')
    expect(submitButton.attributes('disabled')).toBeDefined()

    // Loading indicator should be present
    const loadingSpan = wrapper.findAll('span').find((s) => s.text() === 'Loading...')
    expect(loadingSpan).toBeDefined()
    expect(loadingSpan!.exists()).toBe(true)

    wrapper.unmount()
  })

  /**
   * Validates: Requirements 4.3
   * New request clears previous result
   */
  it('new request clears previous result', async () => {
    // First fetch returns a valid response
    let resolvePending: ((value: unknown) => void) | null = null

    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ score: 5, advice: 'first advice' }),
      })
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolvePending = resolve
        })
      )

    vi.stubGlobal('fetch', mockFetch)

    const wrapper = mount(SeverityChecker)

    // Set text and submit
    const textarea = wrapper.find('textarea')
    await textarea.setValue('some symptom text')
    await nextTick()

    const button = wrapper.find('button')
    await button.trigger('click')

    // Wait for first response to resolve
    await nextTick()
    await nextTick()
    await nextTick()
    await nextTick()
    await nextTick()

    // Assert first result is shown
    expect(wrapper.text()).toContain('first advice')

    // Click submit again (second fetch is pending)
    await button.trigger('click')
    await nextTick()

    // Before second response resolves, previous result should be cleared
    expect(wrapper.text()).not.toContain('first advice')

    // Resolve pending to clean up
    if (resolvePending) {
      ;(resolvePending as (value: unknown) => void)({
        ok: true,
        status: 200,
        json: async () => ({ score: 3, advice: 'second advice' }),
      })
    }

    wrapper.unmount()
  })

  /**
   * Validates: Requirements 1.3
   * Empty input shows validation message and does not call fetch
   */
  it('empty input shows validation message and does not call fetch', async () => {
    const mockFetch = vi.fn()
    vi.stubGlobal('fetch', mockFetch)

    const wrapper = mount(SeverityChecker)

    // Ensure input is empty (default state)
    const textarea = wrapper.find('textarea')
    await textarea.setValue('')
    await nextTick()

    // Click submit
    const button = wrapper.find('button')
    await button.trigger('click')
    await nextTick()

    // Fetch should NOT have been called
    expect(mockFetch).not.toHaveBeenCalled()

    // Validation message should be shown
    const allSpans = wrapper.findAll('span')
    const validationSpan = allSpans.find((s) => s.text().includes('Text is required'))
    expect(validationSpan).toBeDefined()
    expect(validationSpan!.exists()).toBe(true)

    wrapper.unmount()
  })
})
