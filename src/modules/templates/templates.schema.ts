import { z } from "zod";

export const createCategorySchema = z.object({
  name: z.string().min(2).max(255),
  slug: z
    .string()
    .min(2)
    .max(255)
    .regex(
      /^[a-z0-9-]+$/,
      "Slug must be lowercase letters, numbers, and hyphens only",
    )
    .optional(),
  description: z.string().optional(),
});

export const updateCategorySchema = createCategorySchema.partial();

export const templateQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(12),
  search: z.string().optional(),
  category: z.string().optional(), // category slug
  tag: z.string().optional(),
  featured: z.enum(["true", "false"]).optional(),
  minPrice: z.coerce.number().min(0).optional(),
  maxPrice: z.coerce.number().min(0).optional(),
  sort: z
    .enum(["newest", "oldest", "price_asc", "price_desc"])
    .default("newest"),
});

export const adminTemplateQuerySchema = templateQuerySchema.extend({
  status: z.enum(["draft", "published", "archived"]).optional(),
});

export const updateTemplateSchema = z.object({
  title: z.string().min(2).max(500).optional(),
  description: z.string().optional(),
  longDescription: z.any().optional(), // TipTap JSON — any shape
  priceUsd: z.coerce.number().min(0).optional(),
  categoryId: z.string().uuid().optional().nullable(),
  tags: z.array(z.string()).optional(),
  featured: z.boolean().optional(),
  status: z.enum(["draft", "published", "archived"]).optional(),
});
