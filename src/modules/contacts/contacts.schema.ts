import { z } from 'zod'

export const createContactSchema = z.object({
    name: z.string().min(2).max(255),
    email: z.string().email('Valid email is required'),
    phone: z.string().max(50).optional(),
    message: z.string().min(10, 'Message must be at least 10 characters').max(5000),
})

export const updateContactStatusSchema = z.object({
    status: z.enum(['new', 'read', 'replied', 'archived']),
})

export const contactQuerySchema = z.object({
    page: z.coerce.number().int().min(1).default(1),
    limit: z.coerce.number().int().min(1).max(100).default(20),
    search: z.string().optional(),
    status: z.enum(['new', 'read', 'replied', 'archived']).optional(),
    sort: z.enum(['newest', 'oldest']).default('newest'),
})