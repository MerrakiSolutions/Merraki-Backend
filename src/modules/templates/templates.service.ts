import {
  eq,
  and,
  or,
  ilike,
  gte,
  lte,
  desc,
  asc,
  sql,
  count,
} from "drizzle-orm";
import slugify from "slugify";
import { v4 as uuidv4 } from "uuid";
import { db } from "../../db/index.js";
import { templates, templateCategories } from "../../db/schema/templates.js";
import { uploadToR2, deleteFromR2 } from "../../lib/r2.js";
import {
  uploadImageToCloudinary,
  deleteFromCloudinary,
} from "../../lib/cloudinary.js";
import { AppError, NotFoundError, ConflictError } from "../../lib/errors.js";
import { paginate, getPaginationOffset } from "../../lib/pagination.js";

// ── Helpers ───────────────────────────────────────────────────────

const generateSlug = (title: string): string =>
  slugify(title, { lower: true, strict: true, trim: true });

const buildTemplateWhere = (query: {
  search?: string;
  category?: string;
  tag?: string;
  featured?: string;
  minPrice?: number;
  maxPrice?: number;
  status?: string;
  publishedOnly?: boolean;
}) => {
  const conditions = [];

  if (query.publishedOnly) {
    conditions.push(eq(templates.status, "published"));
  } else if (query.status) {
    conditions.push(eq(templates.status, query.status as any));
  }

  if (query.search) {
    conditions.push(
      or(
        ilike(templates.title, `%${query.search}%`),
        ilike(templates.description, `%${query.search}%`),
      ),
    );
  }

  if (query.category) {
    conditions.push(
      sql`EXISTS (
        SELECT 1 FROM ${templateCategories}
        WHERE ${templateCategories.id} = ${templates.categoryId}
        AND ${templateCategories.slug} = ${query.category}
      )`,
    );
  }

  if (query.tag) {
    conditions.push(sql`${query.tag} = ANY(${templates.tags})`);
  }

  if (query.featured === "true") {
    conditions.push(eq(templates.featured, true));
  }

  if (query.minPrice !== undefined) {
    conditions.push(gte(templates.priceUsd, String(query.minPrice)));
  }

  if (query.maxPrice !== undefined) {
    conditions.push(lte(templates.priceUsd, String(query.maxPrice)));
  }

  return conditions.length > 0 ? and(...conditions) : undefined;
};

const buildOrderBy = (sort: string) => {
  switch (sort) {
    case "oldest":
      return asc(templates.createdAt);
    case "price_asc":
      return asc(templates.priceUsd);
    case "price_desc":
      return desc(templates.priceUsd);
    default:
      return desc(templates.createdAt);
  }
};

// ── CATEGORIES ────────────────────────────────────────────────────

export const getAllCategories = async () => {
  return db
    .select()
    .from(templateCategories)
    .orderBy(asc(templateCategories.name));
};

export const createCategory = async (data: {
  name: string;
  slug?: string;
  description?: string;
}) => {
  const slug = data.slug || generateSlug(data.name);

  const [existing] = await db
    .select({ id: templateCategories.id })
    .from(templateCategories)
    .where(eq(templateCategories.slug, slug))
    .limit(1);

  if (existing)
    throw new ConflictError("A category with this slug already exists.");

  const [created] = await db
    .insert(templateCategories)
    .values({ name: data.name, slug, description: data.description })
    .returning();

  return created;
};

export const updateCategory = async (
  id: string,
  data: { name?: string; slug?: string; description?: string },
) => {
  const [existing] = await db
    .select()
    .from(templateCategories)
    .where(eq(templateCategories.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Category not found.");

  if (data.slug && data.slug !== existing.slug) {
    const [slugTaken] = await db
      .select({ id: templateCategories.id })
      .from(templateCategories)
      .where(eq(templateCategories.slug, data.slug))
      .limit(1);
    if (slugTaken)
      throw new ConflictError("A category with this slug already exists.");
  }

  const [updated] = await db
    .update(templateCategories)
    .set({ ...data, updatedAt: new Date() })
    .where(eq(templateCategories.id, id))
    .returning();

  return updated;
};

export const deleteCategory = async (id: string) => {
  const [existing] = await db
    .select({ id: templateCategories.id })
    .from(templateCategories)
    .where(eq(templateCategories.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Category not found.");

  await db.delete(templateCategories).where(eq(templateCategories.id, id));
  return { message: "Category deleted successfully." };
};

// ── PUBLIC: List templates ─────────────────────────────────────────

export const getPublicTemplates = async (query: {
  page: number;
  limit: number;
  search?: string;
  category?: string;
  tag?: string;
  featured?: string;
  minPrice?: number;
  maxPrice?: number;
  sort: string;
}) => {
  const { page, limit, sort } = query;
  const offset = getPaginationOffset(page, limit);
  const where = buildTemplateWhere({ ...query, publishedOnly: true });

  const [{ total }] = await db
    .select({ total: count() })
    .from(templates)
    .where(where);

  const data = await db
    .select({
      id: templates.id,
      title: templates.title,
      slug: templates.slug,
      description: templates.description,
      priceUsd: templates.priceUsd,
      previewImages: templates.previewImages,
      categoryId: templates.categoryId,
      tags: templates.tags,
      featured: templates.featured,
      createdAt: templates.createdAt,
    })
    .from(templates)
    .where(where)
    .orderBy(buildOrderBy(sort))
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

// ── PUBLIC: Single template ────────────────────────────────────────

export const getPublicTemplateBySlug = async (slug: string) => {
  const [template] = await db
    .select({
      id: templates.id,
      title: templates.title,
      slug: templates.slug,
      description: templates.description,
      longDescription: templates.longDescription,
      priceUsd: templates.priceUsd,
      previewImages: templates.previewImages,
      categoryId: templates.categoryId,
      tags: templates.tags,
      featured: templates.featured,
      createdAt: templates.createdAt,
      // note: r2Key intentionally excluded from public response
    })
    .from(templates)
    .where(and(eq(templates.slug, slug), eq(templates.status, "published")))
    .limit(1);

  if (!template) throw new NotFoundError("Template not found.");
  return template;
};

// ── ADMIN: List templates ──────────────────────────────────────────

export const getAdminTemplates = async (query: {
  page: number;
  limit: number;
  search?: string;
  category?: string;
  tag?: string;
  featured?: string;
  minPrice?: number;
  maxPrice?: number;
  sort: string;
  status?: string;
}) => {
  const { page, limit, sort } = query;
  const offset = getPaginationOffset(page, limit);
  const where = buildTemplateWhere(query);

  const [{ total }] = await db
    .select({ total: count() })
    .from(templates)
    .where(where);

  const data = await db
    .select()
    .from(templates)
    .where(where)
    .orderBy(buildOrderBy(sort))
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

// ── ADMIN: Single template ─────────────────────────────────────────

export const getAdminTemplateById = async (id: string) => {
  const [template] = await db
    .select()
    .from(templates)
    .where(eq(templates.id, id))
    .limit(1);

  if (!template) throw new NotFoundError("Template not found.");
  return template;
};

// ── ADMIN: Create template ─────────────────────────────────────────

export const createTemplate = async (data: {
  title: string;
  description?: string;
  longDescription?: any;
  priceUsd: number;
  categoryId?: string | null;
  tags?: string[];
  featured?: boolean;
  status?: "draft" | "published" | "archived";
  templateFile: { buffer: Buffer; mimetype: string; filename: string };
  previewImageFiles?: { buffer: Buffer; mimetype: string }[];
}) => {
  const slug = generateSlug(data.title);

  // check slug uniqueness
  const [existing] = await db
    .select({ id: templates.id })
    .from(templates)
    .where(eq(templates.slug, slug))
    .limit(1);

  if (existing)
    throw new ConflictError("A template with this title already exists.");

  // upload template file to R2
  const fileExt = data.templateFile.filename.split(".").pop() || "bin";
  const r2Key = `templates/${uuidv4()}.${fileExt}`;

  await uploadToR2(r2Key, data.templateFile.buffer, data.templateFile.mimetype);

  // upload preview images to Cloudinary
  const previewImages: { url: string; alt: string }[] = [];

  if (data.previewImageFiles && data.previewImageFiles.length > 0) {
    for (const img of data.previewImageFiles) {
      const { url } = await uploadImageToCloudinary(
        img.buffer,
        "merraki/template-previews",
      );
      previewImages.push({ url, alt: data.title });
    }
  }

  const [created] = await db
    .insert(templates)
    .values({
      title: data.title,
      slug,
      description: data.description,
      longDescription: data.longDescription,
      priceUsd: String(data.priceUsd),
      r2Key,
      previewImages,
      categoryId: data.categoryId || null,
      tags: data.tags || [],
      featured: data.featured ?? false,
      status: data.status ?? "draft",
    })
    .returning();

  return created;
};

// ── ADMIN: Update template ─────────────────────────────────────────

export const updateTemplate = async (
  id: string,
  data: {
    title?: string;
    description?: string;
    longDescription?: any;
    priceUsd?: number;
    categoryId?: string | null;
    tags?: string[];
    featured?: boolean;
    status?: "draft" | "published" | "archived";
    newTemplateFile?: { buffer: Buffer; mimetype: string; filename: string };
    newPreviewImageFiles?: { buffer: Buffer; mimetype: string }[];
  },
) => {
  const [existing] = await db
    .select()
    .from(templates)
    .where(eq(templates.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Template not found.");

  const updateData: Record<string, any> = { updatedAt: new Date() };

  if (data.title && data.title !== existing.title) {
    const newSlug = generateSlug(data.title);
    const [slugTaken] = await db
      .select({ id: templates.id })
      .from(templates)
      .where(and(eq(templates.slug, newSlug), sql`${templates.id} != ${id}`))
      .limit(1);
    if (slugTaken)
      throw new ConflictError("A template with this title already exists.");
    updateData.title = data.title;
    updateData.slug = newSlug;
  }

  if (data.description !== undefined) updateData.description = data.description;
  if (data.longDescription !== undefined)
    updateData.longDescription = data.longDescription;
  if (data.priceUsd !== undefined) updateData.priceUsd = String(data.priceUsd);
  if (data.categoryId !== undefined) updateData.categoryId = data.categoryId;
  if (data.tags !== undefined) updateData.tags = data.tags;
  if (data.featured !== undefined) updateData.featured = data.featured;
  if (data.status !== undefined) updateData.status = data.status;

  // replace template file if new one uploaded
  if (data.newTemplateFile) {
    await deleteFromR2(existing.r2Key);
    const fileExt = data.newTemplateFile.filename.split(".").pop() || "bin";
    const newR2Key = `templates/${uuidv4()}.${fileExt}`;
    await uploadToR2(
      newR2Key,
      data.newTemplateFile.buffer,
      data.newTemplateFile.mimetype,
    );
    updateData.r2Key = newR2Key;
  }

  // append new preview images if uploaded
  if (data.newPreviewImageFiles && data.newPreviewImageFiles.length > 0) {
    const existing_images = (existing.previewImages as any[]) || [];
    const newImages: { url: string; alt: string }[] = [];

    for (const img of data.newPreviewImageFiles) {
      const { url } = await uploadImageToCloudinary(
        img.buffer,
        "merraki/template-previews",
      );
      newImages.push({ url, alt: updateData.title || existing.title });
    }

    updateData.previewImages = [...existing_images, ...newImages];
  }

  const [updated] = await db
    .update(templates)
    .set(updateData)
    .where(eq(templates.id, id))
    .returning();

  return updated;
};

// ── ADMIN: Delete template ─────────────────────────────────────────

export const deleteTemplate = async (id: string) => {
  const [template] = await db
    .select({ id: templates.id, r2Key: templates.r2Key })
    .from(templates)
    .where(eq(templates.id, id))
    .limit(1);

  if (!template) throw new NotFoundError("Template not found.");

  // delete file from R2
  await deleteFromR2(template.r2Key);

  await db.delete(templates).where(eq(templates.id, id));
  return { message: "Template deleted successfully." };
};

// ── ADMIN: Upload standalone image (for TipTap editor) ────────────

export const uploadTemplateImage = async (
  buffer: Buffer,
  mimetype: string,
): Promise<{ url: string }> => {
  const allowed = ["image/jpeg", "image/png", "image/webp"];
  if (!allowed.includes(mimetype)) {
    throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
  }
  const { url } = await uploadImageToCloudinary(
    buffer,
    "merraki/template-content",
  );
  return { url };
};

// ── ADMIN: Delete standalone image (for TipTap editor) ────────────

export const deleteTemplateImage = async (url: string) => {
  await deleteFromCloudinary(url);
  return { message: "Image deleted successfully." };
};
