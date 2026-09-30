import { z } from 'zod'

// ── Public submit — fully dynamic, no hardcoded question keys ────────
// Accepts any string key with a string value; the service validates
// against the DB-defined active questions, so an unknown q-key or
// invalid option value gets caught there with a specific message.

export const submitTestSchema = z.object({
    leadName: z.string().min(2).max(255),
    leadEmail: z.string().email('Valid email is required'),
    leadCompany: z.string().max(255).optional(),
    answers: z.record(z.string(), z.string().min(1).max(50)),
})

export const founderTestQuerySchema = z.object({
    page: z.coerce.number().int().min(1).default(1),
    limit: z.coerce.number().int().min(1).max(100).default(20),
    search: z.string().optional(),
    resultType: z.string().optional(),
    sort: z.enum(['newest', 'oldest']).default('newest'),
})

// ── Admin: question CRUD ──────────────────────────────────────────────

const optionSchema = z.object({
    value: z.string().min(1).max(20),
    label: z.string().min(1).max(300),
    score: z.number().int().min(0).max(10),
})

export const createQuestionSchema = z.object({
    questionKey: z.string().min(1).max(50).regex(/^[a-z0-9_]+$/, 'Use lowercase letters, numbers, underscores only'),
    section: z.string().min(1).max(100),
    sectionLabel: z.string().min(1).max(150),
    category: z.string().min(1).max(200),
    question: z.string().min(1),
    description: z.string().optional(),
    type: z.enum(['single', 'scale']),
    options: z.array(optionSchema).min(2),
    displayOrder: z.number().int().min(0).optional(),
    isActive: z.boolean().optional(),
})

export const updateQuestionSchema = createQuestionSchema.partial().omit({ questionKey: true })

export const reorderQuestionsSchema = z.object({
    items: z.array(z.object({ id: z.string().uuid(), displayOrder: z.number().int().min(0) })),
})

// ── Admin: archetype CRUD ─────────────────────────────────────────────

export const createArchetypeSchema = z.object({
    archetypeKey: z.string().min(1).max(50).regex(/^[a-z0-9_]+$/, 'Use lowercase letters, numbers, underscores only'),
    title: z.string().min(1).max(150),
    badge: z.string().max(10).optional(),
    color: z.string().max(20).optional(),
    description: z.string().min(1),
    message: z.string().min(1),
    traits: z.array(z.string()).optional(),
    strengths: z.array(z.string()).optional(),
    growthSuggestions: z.array(z.string()).optional(),
    riskAreas: z.array(z.string()).optional(),
    minScore: z.number().int().min(0),
    maxScore: z.number().int().min(0),
    displayOrder: z.number().int().min(0).optional(),
    isActive: z.boolean().optional(),
})

export const updateArchetypeSchema = createArchetypeSchema.partial().omit({ archetypeKey: true })