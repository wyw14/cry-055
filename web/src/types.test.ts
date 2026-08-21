import { describe, expect, it } from 'vitest'
import type { InstrumentStatus } from './types'

describe('instrument status contract', () => {
  it('keeps the seven server states available to the UI', () => {
    const statuses: InstrumentStatus[] = ['pending', 'qualified', 'due_soon', 'overdue', 'unqualified', 'disabled', 'reinspection']
    expect(new Set(statuses).size).toBe(7)
  })
})
