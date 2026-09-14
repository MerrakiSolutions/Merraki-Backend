import { z } from "zod";

export const subscribeSchema = z.object({
  email: z.string().email("Valid email is required"),
  name: z.string().min(2).max(255).optional(),
  categoryIds: z.array(z.string().uuid()).default([]),
});

export const createCategorySchema = z.object({
  name: z.string().min(2).max(255),
  slug: z
    .string()
    .min(2)
    .max(255)
    .regex(
      /^[a-z0-9-]+$/,
      "Slug must be lowercase letters, numbers, hyphens only",
    )
    .optional(),
});

export const updateCategorySchema = createCategorySchema.partial();

export const createCampaignSchema = z.object({
  title: z.string().min(2).max(500),
  subject: z.string().min(2).max(500),
  content: z.any(), // TipTap JSON
  categoryId: z.string().uuid().optional().nullable(),
});

export const updateCampaignSchema = createCampaignSchema.partial();

export const subscriberQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(20),
  search: z.string().optional(),
  confirmed: z.enum(["true", "false"]).optional(),
  categoryId: z.string().uuid().optional(),
});

export const campaignQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(20),
  search: z.string().optional(),
  status: z.enum(["draft", "sending", "sent"]).optional(),
});
