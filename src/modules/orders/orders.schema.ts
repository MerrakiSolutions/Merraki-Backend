import { z } from 'zod'

export const adminOrdersQuerySchema = z.object({
    page: z.coerce.number().int().min(1).default(1),
    limit: z.coerce.number().int().min(1).max(100).default(15),
    search: z.string().trim().max(100).optional(),
    status: z.enum(['pending', 'paid', 'failed', 'refunded']).optional(),
    sort: z.enum(['newest', 'oldest', 'amount_high', 'amount_low']).default('newest'),
})

export const trackQuerySchema = z.object({
    order_id: z.string().uuid().optional(),
    email: z.string().optional(),
})

export const trackRequestSchema = z.object({ email: z.string().trim().toLowerCase().email().max(254) })
export const idParamSchema = z.object({ id: z.string().uuid() })
export const tokenParamSchema = z.object({ token: z.string().max(64) })