import { z } from 'zod'

export const createOrderSchema = z.object({
  // Only ids are accepted — prices always come from the database. `quantity` is ignored (digital goods).
  items: z.array(z.object({ templateId: z.string().uuid(), quantity: z.number().int().optional() })).min(1).max(10),
  guestName: z.string().trim().min(2).max(100),
  guestEmail: z.string().trim().toLowerCase().email().max(254),
  billingAddress: z.object({
    line1: z.string().trim().min(3).max(150),
    line2: z.string().trim().max(150).optional(),
    city: z.string().trim().min(2).max(80),
    state: z.string().trim().min(2).max(80),
    country: z.string().trim().min(2).max(60),
    zip: z.string().trim().min(3).max(12),
    company: z.string().trim().max(120).optional(),
  }),
  paymentMethod: z.enum(['card', 'upi']),
})

export type CreateOrderInput = z.infer<typeof createOrderSchema>