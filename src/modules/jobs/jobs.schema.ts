import { z } from 'zod'

export const idParamSchema = z.object({ id: z.string().uuid('Invalid id') })
export const slugParamSchema = z.object({ slug: z.string().min(1).max(220) })

export const createJobSchema = z.object({
    title: z.string().trim().min(2).max(200),
    team: z.string().trim().min(1).max(100),
    type: z.string().trim().min(1).max(50),
    location: z.string().trim().min(1).max(150),
    description: z.unknown().optional(),
    requirements: z.array(z.string().trim().min(1).max(300)).max(30).optional(),
    applyUrl: z.string().trim().url('Apply link must be a valid URL').max(500).nullish(),
    isActive: z.boolean().optional(),
    displayOrder: z.number().int().min(0).optional(),
})

export const updateJobSchema = createJobSchema.partial()

export const applyLinkSettingsSchema = z.object({
    companyApplyUrl: z.string().trim().url('Must be a valid URL').max(500).nullable(),
})

export type CreateJobInput = z.infer<typeof createJobSchema>
export type UpdateJobInput = z.infer<typeof updateJobSchema>