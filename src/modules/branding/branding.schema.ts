import { z } from 'zod'

// ── Logos ─────────────────────────────────────────────────────────

export const createLogoSchema = z.object({
    altText: z.string().max(255).optional(),
    type: z.enum(['primary', 'secondary', 'favicon']),
    isActive: z.boolean().default(true),
})

export const updateLogoSchema = z.object({
    altText: z.string().max(255).optional(),
    type: z.enum(['primary', 'secondary', 'favicon']).optional(),
    isActive: z.boolean().optional(),
})

// ── Headings ──────────────────────────────────────────────────────

export const createHeadingSchema = z.object({
    page: z.string().min(1).max(100),
    heading: z.string().min(1).max(500),
    subheading: z.string().optional(),
    isActive: z.boolean().default(true),
})

export const updateHeadingSchema = createHeadingSchema.partial()

// ── Social Links ──────────────────────────────────────────────────

export const createSocialLinkSchema = z.object({
    platform: z.enum([
        'instagram',
        'twitter',
        'linkedin',
        'youtube',
        'facebook',
        'tiktok',
        'github',
        'website',
    ]),
    url: z.string().url('Must be a valid URL'),
    displayOrder: z.number().int().min(0).default(0),
    isActive: z.boolean().default(true),
})

export const updateSocialLinkSchema = createSocialLinkSchema.partial()

export const reorderSocialLinksSchema = z.object({
    // array of { id, displayOrder } to bulk update ordering
    items: z.array(
        z.object({
            id: z.string().uuid(),
            displayOrder: z.number().int().min(0),
        })
    ).min(1),
})