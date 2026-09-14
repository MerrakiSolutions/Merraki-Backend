import { eq, and, or, ilike, desc, asc, count, sql } from "drizzle-orm";
import slugify from "slugify";
import { db } from "../../db/index.js";
import {
  blogPosts,
  blogCategories,
  blogAuthors,
} from "../../db/schema/blog.js";
import {
  uploadImageToCloudinary,
  deleteFromCloudinary,
} from "../../lib/cloudinary.js";
import { AppError, NotFoundError, ConflictError } from "../../lib/errors.js";
import { paginate, getPaginationOffset } from "../../lib/pagination.js";

// ── Helpers ───────────────────────────────────────────────────────

const generateSlug = (title: string): string =>
  slugify(title, { lower: true, strict: true, trim: true });

const buildPostWhere = (query: {
  search?: string;
  category?: string;
  author?: string;
  tag?: string;
  status?: string;
  publishedOnly?: boolean;
}) => {
  const conditions = [];

  if (query.publishedOnly) {
    conditions.push(eq(blogPosts.status, "published"));
  } else if (query.status) {
    conditions.push(eq(blogPosts.status, query.status as any));
  }

  if (query.search) {
    conditions.push(
      or(
        ilike(blogPosts.title, `%${query.search}%`),
        ilike(blogPosts.excerpt, `%${query.search}%`),
      ),
    );
  }

  if (query.category) {
    conditions.push(
      sql`EXISTS (
        SELECT 1 FROM ${blogCategories}
        WHERE ${blogCategories.id} = ${blogPosts.categoryId}
        AND ${blogCategories.slug} = ${query.category}
      )`,
    );
  }

  if (query.author) {
    conditions.push(eq(blogPosts.authorId, query.author));
  }

  if (query.tag) {
    conditions.push(sql`${query.tag} = ANY(${blogPosts.tags})`);
  }

  return conditions.length > 0 ? and(...conditions) : undefined;
};

// ── Safe public post fields (no internal IDs exposed unnecessarily) ─

const publicPostFields = {
  id: blogPosts.id,
  title: blogPosts.title,
  slug: blogPosts.slug,
  excerpt: blogPosts.excerpt,
  coverImageUrl: blogPosts.coverImageUrl,
  authorId: blogPosts.authorId,
  categoryId: blogPosts.categoryId,
  tags: blogPosts.tags,
  status: blogPosts.status,
  seoTitle: blogPosts.seoTitle,
  seoDescription: blogPosts.seoDescription,
  publishedAt: blogPosts.publishedAt,
  createdAt: blogPosts.createdAt,
  updatedAt: blogPosts.updatedAt,
  // content excluded from list — only returned on single post
};

// ════════════════════════════════════════════════
// CATEGORIES
// ════════════════════════════════════════════════

export const getAllCategories = async () =>
  db.select().from(blogCategories).orderBy(asc(blogCategories.name));

export const createCategory = async (data: { name: string; slug?: string }) => {
  const slug = data.slug || generateSlug(data.name);

  const [existing] = await db
    .select({ id: blogCategories.id })
    .from(blogCategories)
    .where(eq(blogCategories.slug, slug))
    .limit(1);

  if (existing)
    throw new ConflictError("A category with this slug already exists.");

  const [created] = await db
    .insert(blogCategories)
    .values({ name: data.name, slug })
    .returning();

  return created;
};

export const updateCategory = async (
  id: string,
  data: { name?: string; slug?: string },
) => {
  const [existing] = await db
    .select()
    .from(blogCategories)
    .where(eq(blogCategories.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Category not found.");

  if (data.slug && data.slug !== existing.slug) {
    const [slugTaken] = await db
      .select({ id: blogCategories.id })
      .from(blogCategories)
      .where(eq(blogCategories.slug, data.slug))
      .limit(1);
    if (slugTaken)
      throw new ConflictError("A category with this slug already exists.");
  }

  const [updated] = await db
    .update(blogCategories)
    .set({ ...data, updatedAt: new Date() })
    .where(eq(blogCategories.id, id))
    .returning();

  return updated;
};

export const deleteCategory = async (id: string) => {
  const [existing] = await db
    .select({ id: blogCategories.id })
    .from(blogCategories)
    .where(eq(blogCategories.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Category not found.");

  await db.delete(blogCategories).where(eq(blogCategories.id, id));
  return { message: "Category deleted successfully." };
};

// ════════════════════════════════════════════════
// AUTHORS
// ════════════════════════════════════════════════

export const getAllAuthors = async () =>
  db.select().from(blogAuthors).orderBy(asc(blogAuthors.name));

export const getAuthorById = async (id: string) => {
  const [author] = await db
    .select()
    .from(blogAuthors)
    .where(eq(blogAuthors.id, id))
    .limit(1);

  if (!author) throw new NotFoundError("Author not found.");
  return author;
};

export const createAuthor = async (data: {
  name: string;
  bio?: string;
  userId?: string;
  avatarFile?: { buffer: Buffer; mimetype: string };
}) => {
  let avatarUrl: string | undefined;

  if (data.avatarFile) {
    const allowed = ["image/jpeg", "image/png", "image/webp"];
    if (!allowed.includes(data.avatarFile.mimetype)) {
      throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
    }
    const { url } = await uploadImageToCloudinary(
      data.avatarFile.buffer,
      "merraki/blog-authors",
    );
    avatarUrl = url;
  }

  const [created] = await db
    .insert(blogAuthors)
    .values({
      name: data.name,
      bio: data.bio,
      userId: data.userId || null,
      avatarUrl,
    })
    .returning();

  return created;
};

export const updateAuthor = async (
  id: string,
  data: {
    name?: string;
    bio?: string;
    avatarFile?: { buffer: Buffer; mimetype: string };
  },
) => {
  const [existing] = await db
    .select()
    .from(blogAuthors)
    .where(eq(blogAuthors.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Author not found.");

  const updateData: Record<string, any> = { updatedAt: new Date() };
  if (data.name !== undefined) updateData.name = data.name;
  if (data.bio !== undefined) updateData.bio = data.bio;

  if (data.avatarFile) {
    const allowed = ["image/jpeg", "image/png", "image/webp"];
    if (!allowed.includes(data.avatarFile.mimetype)) {
      throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
    }
    const { url } = await uploadImageToCloudinary(
      data.avatarFile.buffer,
      "merraki/blog-authors",
      `author_${id}`,
    );
    updateData.avatarUrl = url;
  }

  const [updated] = await db
    .update(blogAuthors)
    .set(updateData)
    .where(eq(blogAuthors.id, id))
    .returning();

  return updated;
};

export const deleteAuthor = async (id: string) => {
  const [existing] = await db
    .select({ id: blogAuthors.id })
    .from(blogAuthors)
    .where(eq(blogAuthors.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Author not found.");

  await db.delete(blogAuthors).where(eq(blogAuthors.id, id));
  return { message: "Author deleted successfully." };
};

// ════════════════════════════════════════════════
// POSTS — PUBLIC
// ════════════════════════════════════════════════

export const getPublicPosts = async (query: {
  page: number;
  limit: number;
  search?: string;
  category?: string;
  author?: string;
  tag?: string;
  sort: string;
}) => {
  const { page, limit, sort } = query;
  const offset = getPaginationOffset(page, limit);
  const where = buildPostWhere({ ...query, publishedOnly: true });
  const orderBy =
    sort === "oldest"
      ? asc(blogPosts.publishedAt)
      : desc(blogPosts.publishedAt);

  const [{ total }] = await db
    .select({ total: count() })
    .from(blogPosts)
    .where(where);

  const data = await db
    .select(publicPostFields)
    .from(blogPosts)
    .where(where)
    .orderBy(orderBy)
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

export const getPublicPostBySlug = async (slug: string) => {
  const [post] = await db
    .select()
    .from(blogPosts)
    .where(and(eq(blogPosts.slug, slug), eq(blogPosts.status, "published")))
    .limit(1);

  if (!post) throw new NotFoundError("Post not found.");

  // fetch author and category alongside
  const [author] = post.authorId
    ? await db
        .select()
        .from(blogAuthors)
        .where(eq(blogAuthors.id, post.authorId))
        .limit(1)
    : [null];

  const [category] = post.categoryId
    ? await db
        .select()
        .from(blogCategories)
        .where(eq(blogCategories.id, post.categoryId))
        .limit(1)
    : [null];

  return { ...post, author, category };
};

// ════════════════════════════════════════════════
// POSTS — ADMIN
// ════════════════════════════════════════════════

export const getAdminPosts = async (query: {
  page: number;
  limit: number;
  search?: string;
  category?: string;
  author?: string;
  tag?: string;
  sort: string;
  status?: string;
}) => {
  const { page, limit, sort } = query;
  const offset = getPaginationOffset(page, limit);
  const where = buildPostWhere(query);
  const orderBy =
    sort === "oldest" ? asc(blogPosts.createdAt) : desc(blogPosts.createdAt);

  const [{ total }] = await db
    .select({ total: count() })
    .from(blogPosts)
    .where(where);

  const data = await db
    .select(publicPostFields)
    .from(blogPosts)
    .where(where)
    .orderBy(orderBy)
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

export const getAdminPostById = async (id: string) => {
  const [post] = await db
    .select()
    .from(blogPosts)
    .where(eq(blogPosts.id, id))
    .limit(1);

  if (!post) throw new NotFoundError("Post not found.");
  return post;
};

export const createPost = async (data: {
  title: string;
  excerpt?: string;
  content: any;
  authorId?: string | null;
  categoryId?: string | null;
  tags?: string[];
  status?: "draft" | "published" | "archived";
  seoTitle?: string;
  seoDescription?: string;
  coverImageFile?: { buffer: Buffer; mimetype: string };
}) => {
  const slug = generateSlug(data.title);

  const [existing] = await db
    .select({ id: blogPosts.id })
    .from(blogPosts)
    .where(eq(blogPosts.slug, slug))
    .limit(1);

  if (existing)
    throw new ConflictError("A post with this title already exists.");

  let coverImageUrl: string | undefined;

  if (data.coverImageFile) {
    const allowed = ["image/jpeg", "image/png", "image/webp"];
    if (!allowed.includes(data.coverImageFile.mimetype)) {
      throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
    }
    const { url } = await uploadImageToCloudinary(
      data.coverImageFile.buffer,
      "merraki/blog-covers",
    );
    coverImageUrl = url;
  }

  const publishedAt = data.status === "published" ? new Date() : null;

  const [created] = await db
    .insert(blogPosts)
    .values({
      title: data.title,
      slug,
      excerpt: data.excerpt,
      coverImageUrl,
      content: data.content,
      authorId: data.authorId || null,
      categoryId: data.categoryId || null,
      tags: data.tags || [],
      status: data.status || "draft",
      seoTitle: data.seoTitle,
      seoDescription: data.seoDescription,
      publishedAt,
    })
    .returning();

  return created;
};

export const updatePost = async (
  id: string,
  data: {
    title?: string;
    excerpt?: string;
    content?: any;
    authorId?: string | null;
    categoryId?: string | null;
    tags?: string[];
    status?: "draft" | "published" | "archived";
    seoTitle?: string;
    seoDescription?: string;
    coverImageFile?: { buffer: Buffer; mimetype: string };
  },
) => {
  const [existing] = await db
    .select()
    .from(blogPosts)
    .where(eq(blogPosts.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Post not found.");

  const updateData: Record<string, any> = { updatedAt: new Date() };

  if (data.title && data.title !== existing.title) {
    const newSlug = generateSlug(data.title);
    const [slugTaken] = await db
      .select({ id: blogPosts.id })
      .from(blogPosts)
      .where(and(eq(blogPosts.slug, newSlug), sql`${blogPosts.id} != ${id}`))
      .limit(1);
    if (slugTaken)
      throw new ConflictError("A post with this title already exists.");
    updateData.title = data.title;
    updateData.slug = newSlug;
  }

  if (data.excerpt !== undefined) updateData.excerpt = data.excerpt;
  if (data.content !== undefined) updateData.content = data.content;
  if (data.authorId !== undefined) updateData.authorId = data.authorId;
  if (data.categoryId !== undefined) updateData.categoryId = data.categoryId;
  if (data.tags !== undefined) updateData.tags = data.tags;
  if (data.seoTitle !== undefined) updateData.seoTitle = data.seoTitle;
  if (data.seoDescription !== undefined)
    updateData.seoDescription = data.seoDescription;

  // set publishedAt when publishing for the first time
  if (data.status !== undefined) {
    updateData.status = data.status;
    if (data.status === "published" && !existing.publishedAt) {
      updateData.publishedAt = new Date();
    }
  }

  if (data.coverImageFile) {
    const allowed = ["image/jpeg", "image/png", "image/webp"];
    if (!allowed.includes(data.coverImageFile.mimetype)) {
      throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
    }
    const { url } = await uploadImageToCloudinary(
      data.coverImageFile.buffer,
      "merraki/blog-covers",
      `cover_${id}`,
    );
    updateData.coverImageUrl = url;
  }

  const [updated] = await db
    .update(blogPosts)
    .set(updateData)
    .where(eq(blogPosts.id, id))
    .returning();

  return updated;
};

export const publishPost = async (id: string) => {
  const [existing] = await db
    .select({
      id: blogPosts.id,
      status: blogPosts.status,
      publishedAt: blogPosts.publishedAt,
    })
    .from(blogPosts)
    .where(eq(blogPosts.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Post not found.");
  if (existing.status === "published") {
    throw new AppError("Post is already published.", 400);
  }

  const [updated] = await db
    .update(blogPosts)
    .set({
      status: "published",
      publishedAt: existing.publishedAt ?? new Date(),
      updatedAt: new Date(),
    })
    .where(eq(blogPosts.id, id))
    .returning();

  return updated;
};

export const deletePost = async (id: string) => {
  const [existing] = await db
    .select({ id: blogPosts.id })
    .from(blogPosts)
    .where(eq(blogPosts.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Post not found.");

  await db.delete(blogPosts).where(eq(blogPosts.id, id));
  return { message: "Post deleted successfully." };
};

// ── Standalone image upload for TipTap editor ─────────────────────

export const uploadBlogImage = async (
  buffer: Buffer,
  mimetype: string,
): Promise<{ url: string }> => {
  const allowed = ["image/jpeg", "image/png", "image/webp"];
  if (!allowed.includes(mimetype)) {
    throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
  }
  const { url } = await uploadImageToCloudinary(buffer, "merraki/blog-content");
  return { url };
};

// Standalone image deletion for TipTap editor (if needed) ─────────────

export const deleteBlogImage = async (publicId: string) => {
  await deleteFromCloudinary(publicId);
};
