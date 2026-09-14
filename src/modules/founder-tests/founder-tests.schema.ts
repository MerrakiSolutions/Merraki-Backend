import { z } from 'zod'
import { FOUNDER_TEST_QUESTIONS } from '../../config/founder-test.js'

// dynamically build valid answer shape from question config
const answersShape = Object.fromEntries(
    FOUNDER_TEST_QUESTIONS.map((q) => [
        q.id,
        z.enum(
            q.options.map((o) => o.value) as [string, ...string[]],
            { error: () => ({ message: `Invalid answer for ${q.id}` }) }
        ),
    ])
)

export const submitTestSchema = z.object({
    leadName: z.string().min(2).max(255),
    leadEmail: z.string().email('Valid email is required'),
    leadCompany: z.string().max(255).optional(),
    answers: z.object(answersShape),
})

export const founderTestQuerySchema = z.object({
    page: z.coerce.number().int().min(1).default(1),
    limit: z.coerce.number().int().min(1).max(100).default(20),
    search: z.string().optional(),
    resultType: z.string().optional(),
    sort: z.enum(['newest', 'oldest']).default('newest'),
})