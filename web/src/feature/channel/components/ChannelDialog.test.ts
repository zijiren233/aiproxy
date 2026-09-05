import { describe, expect, it } from 'vitest'
import { getChannelFormDefaults } from './channel-form-defaults'
import type { Channel } from '@/types/channel'

describe('getChannelFormDefaults', () => {
    it('preserves the channel name and remark for copy dialogs', () => {
        const channel = {
            id: 42,
            type: 1,
            name: 'Production OpenAI',
            remark: 'Primary route for production traffic',
            key: 'secret',
            models: ['gpt-4.1'],
            model_mapping: null,
            request_count: 0,
            retry_count: 0,
            status: 1,
            created_at: 0,
            accessed_at: 0,
            priority: 10,
        } satisfies Channel

        const defaults = getChannelFormDefaults(channel)

        expect(defaults.name).toBe(channel.name)
        expect(defaults.remark).toBe(channel.remark)
        expect(defaults.models).toEqual(channel.models)
    })

    it('starts an empty form without a source channel', () => {
        const defaults = getChannelFormDefaults(null)

        expect(defaults.name).toBe('')
        expect(defaults.remark).toBe('')
        expect(defaults.models).toEqual([])
    })
})
