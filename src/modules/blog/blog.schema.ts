import { z } from "zod";

// ── Categories ────────────────────────────────────────────────────

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

// ── Authors ───────────────────────────────────────────────────────

export const createAuthorSchema = z.object({
  name: z.string().min(2).max(255),
  bio: z.string().optional(),
  userId: z.string().uuid().optional(),
});

export const updateAuthorSchema = createAuthorSchema.partial();

// ── Posts ─────────────────────────────────────────────────────────

export const createPostSchema = z.object({
  title: z.string().min(2).max(500),
  excerpt: z.string().optional(),
  content: z.any(), // TipTap JSON — any shape
  authorId: z.string().uuid().optional().nullable(),
  categoryId: z.string().uuid().optional().nullable(),
  tags: z.array(z.string()).default([]),
  status: z.enum(["draft", "published", "archived"]).default("draft"),
  seoTitle: z.string().max(500).optional(),
  seoDescription: z.string().optional(),
});

export const updatePostSchema = createPostSchema.partial();

// ── Queries ───────────────────────────────────────────────────────

export const publicPostQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(10),
  search: z.string().optional(),
  category: z.string().optional(), // category slug
  author: z.string().uuid().optional(),
  tag: z.string().optional(),
  sort: z.enum(["newest", "oldest"]).default("newest"),
});

export const adminPostQuerySchema = publicPostQuerySchema.extend({
  status: z.enum(["draft", "published", "archived"]).optional(),
});
