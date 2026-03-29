import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TextInput from '../TextInput.vue'
import BookAppointmentButton from '../BookAppointmentButton.vue'
import ResultDisplay from '../ResultDisplay.vue'

describe('TextInput', () => {
  it('renders a textarea element', () => {
    const wrapper = mount(TextInput, { props: { modelValue: '' } })
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('textarea has min-height: 120px style', () => {
    const wrapper = mount(TextInput, { props: { modelValue: '' } })
    const textarea = wrapper.find('textarea')
    expect(textarea.attributes('style')).toContain('min-height: 120px')
  })

  it('emits update:modelValue when input changes', async () => {
    const wrapper = mount(TextInput, { props: { modelValue: '' } })
    await wrapper.find('textarea').setValue('hello')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual(['hello'])
  })

  it('is disabled when disabled prop is true', () => {
    const wrapper = mount(TextInput, { props: { modelValue: '', disabled: true } })
    expect(wrapper.find('textarea').attributes('disabled')).toBeDefined()
  })
})

describe('BookAppointmentButton', () => {
  it('is not rendered when visible=false', () => {
    const wrapper = mount(BookAppointmentButton, { props: { visible: false } })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('is rendered when visible=true', () => {
    const wrapper = mount(BookAppointmentButton, { props: { visible: true } })
    expect(wrapper.find('button').exists()).toBe(true)
  })

  it('has min-width: 200px style when visible', () => {
    const wrapper = mount(BookAppointmentButton, { props: { visible: true } })
    expect(wrapper.find('button').attributes('style')).toContain('min-width: 200px')
  })

  it('has min-height: 48px style when visible', () => {
    const wrapper = mount(BookAppointmentButton, { props: { visible: true } })
    expect(wrapper.find('button').attributes('style')).toContain('min-height: 48px')
  })

  it('has red background when visible', () => {
    const wrapper = mount(BookAppointmentButton, { props: { visible: true } })
    expect(wrapper.find('button').attributes('style')).toContain('background-color: red')
  })
})

describe('ResultDisplay', () => {
  it('renders score value', () => {
    const wrapper = mount(ResultDisplay, { props: { score: 7, advice: 'Take it easy' } })
    expect(wrapper.text()).toContain('7')
  })

  it('renders advice text', () => {
    const wrapper = mount(ResultDisplay, { props: { score: 7, advice: 'Take it easy' } })
    expect(wrapper.text()).toContain('Take it easy')
  })
})
