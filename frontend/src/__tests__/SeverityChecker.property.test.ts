/**
 * Property-Based Tests for SeverityChecker
 *
 * Feature: severity-checker, Property 1: Non-empty input triggers POST request
 * Validates: Requirements 1.4
 */

import { describe, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import * as fc from 'fast-check'
import SeverityChecker from '../views/SeverityChecker.vue'

describe('SeverityChecker property tests', () => {
  /**
   * Property 1: Non-empty input triggers POST request
   * Validates: Requirements 1.4
   * Tag: Feature: severity-checker, Property 1: Non-empty input triggers POST request
   */
  it('TestNonEmptyInputTriggersPost', async () => {
    await fc.assert(
      fc.asyncProperty(fc.string({ minLength: 1 }), async (text) => {
        // Set up fetch mock that returns a successful response
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          status: 200,
          json: async () => ({ score: 5, advice: 'ok' }),
        })
        vi.stubGlobal('fetch', mockFetch)

        const wrapper = mount(SeverityChecker)

        // Set textarea value and trigger input event
        const textarea = wrapper.find('textarea')
        await textarea.setValue(text)
        await nextTick()

        // Click the "Check Severity" button
        const button = wrapper.find('button')
        await button.trigger('click')
        await nextTick()

        // Assert fetch was called exactly once with the correct arguments
        const calls = mockFetch.mock.calls
        const wasCalled = calls.length === 1
        const firstCall = calls[0]
        const url = firstCall?.[0]
        const options = firstCall?.[1]
        const body = options?.body ? JSON.parse(options.body) : null

        const result =
          wasCalled &&
          url === '/analyze' &&
          options?.method === 'POST' &&
          body?.text === text

        // Clean up
        wrapper.unmount()
        vi.unstubAllGlobals()

        return result
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Property 9: Book Appointment button visibility matches score threshold (score >= 8)
   * Validates: Requirements 5.1, 5.2
   * Tag: Feature: severity-checker, Property 9: Book Appointment button visibility matches score threshold
   */
  it('TestBookAppointmentVisibleForHighScore', async () => {
    await fc.assert(
      fc.asyncProperty(fc.integer({ min: 8, max: 10 }), async (score) => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          status: 200,
          json: async () => ({ score, advice: 'some advice' }),
        })
        vi.stubGlobal('fetch', mockFetch)

        const wrapper = mount(SeverityChecker)

        const textarea = wrapper.find('textarea')
        await textarea.setValue('some symptom text')
        await nextTick()

        const button = wrapper.find('button')
        await button.trigger('click')

        // Wait for async operations to complete
        await nextTick()
        await nextTick()
        await nextTick()
        await nextTick()
        await nextTick()

        const bookButton = wrapper.find('button[style*="background-color: red"]')
        const isVisible = bookButton.exists()

        wrapper.unmount()
        vi.unstubAllGlobals()

        return isVisible
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Property 9: Book Appointment button visibility matches score threshold (score < 8)
   * Validates: Requirements 5.1, 5.2
   * Tag: Feature: severity-checker, Property 9: Book Appointment button visibility matches score threshold
   */
  it('TestBookAppointmentHiddenForLowScore', async () => {
    await fc.assert(
      fc.asyncProperty(fc.integer({ min: 1, max: 7 }), async (score) => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          status: 200,
          json: async () => ({ score, advice: 'some advice' }),
        })
        vi.stubGlobal('fetch', mockFetch)

        const wrapper = mount(SeverityChecker)

        const textarea = wrapper.find('textarea')
        await textarea.setValue('some symptom text')
        await nextTick()

        const button = wrapper.find('button')
        await button.trigger('click')

        // Wait for async operations to complete
        await nextTick()
        await nextTick()
        await nextTick()
        await nextTick()
        await nextTick()

        const bookButton = wrapper.find('button[style*="background-color: red"]')
        const isHidden = !bookButton.exists()

        wrapper.unmount()
        vi.unstubAllGlobals()

        return isHidden
      }),
      { numRuns: 100 }
    )
  })

  /**
   * Property 10: Frontend displays score and advice for any valid response
   * Validates: Requirements 4.1
   * Tag: Feature: severity-checker, Property 10: Frontend displays score and advice for any valid response
   */
  it('TestFrontendDisplaysScoreAndAdvice', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 1, max: 10 }),
        fc.string({ minLength: 1, maxLength: 100 }).filter((s) =>
          // Keep only printable ASCII, non-whitespace-only strings to avoid text matching issues
          /^[\x20-\x7E]+$/.test(s) && s.trim().length > 0
        ),
        async (score, advice) => {
          const mockFetch = vi.fn().mockResolvedValue({
            ok: true,
            status: 200,
            json: async () => ({ score, advice }),
          })
          vi.stubGlobal('fetch', mockFetch)

          const wrapper = mount(SeverityChecker)

          const textarea = wrapper.find('textarea')
          await textarea.setValue('some symptom text')
          await nextTick()

          const button = wrapper.find('button')
          await button.trigger('click')

          // Wait for async operations to complete
          await nextTick()
          await nextTick()
          await nextTick()
          await nextTick()
          await nextTick()

          const text = wrapper.text()
          const hasScore = text.includes(String(score))
          const hasAdvice = text.includes(advice.trim())

          wrapper.unmount()
          vi.unstubAllGlobals()

          return hasScore && hasAdvice
        }
      ),
      { numRuns: 100 }
    )
  })

  /**
   * Property 11: Frontend displays error message for any error response
   * Validates: Requirements 4.2
   * Tag: Feature: severity-checker, Property 11: Frontend displays error message for any error response
   */
  it('TestFrontendDisplaysErrorForErrorResponse', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.oneof(fc.integer({ min: 400, max: 499 }), fc.integer({ min: 500, max: 599 })),
        async (status) => {
          const mockFetch = vi.fn().mockResolvedValue({
            ok: false,
            status,
            json: async () => ({}),
          })
          vi.stubGlobal('fetch', mockFetch)

          const wrapper = mount(SeverityChecker)

          const textarea = wrapper.find('textarea')
          await textarea.setValue('some symptom text')
          await nextTick()

          const button = wrapper.find('button')
          await button.trigger('click')

          // Wait for async operations to complete
          await nextTick()
          await nextTick()
          await nextTick()
          await nextTick()
          await nextTick()

          // Check that a non-empty error message is shown (via error-box div)
          const errorDiv = wrapper.find('.error-box')
          const hasError = errorDiv.exists() && errorDiv.text().length > 0

          // Check that no result is displayed (ResultDisplay component absent)
          const resultDisplay = wrapper.findComponent({ name: 'ResultDisplay' })
          const noResult = !resultDisplay.exists()

          wrapper.unmount()
          vi.unstubAllGlobals()

          return hasError && noResult
        }
      ),
      { numRuns: 100 }
    )
  })
})
