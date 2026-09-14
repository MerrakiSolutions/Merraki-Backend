import { FastifyInstance } from "fastify";
import * as blogService from "./blog.service.js";
import { authenticate } from "../../middleware/authenticate.js";
import { requireAdmin } from "../../middleware/require-admin.js";
import {
  createCategorySchema,
  updateCategorySchema,
  createAuthorSchema,
  updateAuthorSchema,
  createPostSchema,
  updatePostSchema,
  publicPostQuerySchema,
  adminPostQuerySchema,
} from "./blog.schema.js";
import { AppError } from "../../lib/errors.js";

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/blog
// ════════════════════════════════════════════════

export const publicBlogRoutes = async (app: FastifyInstance) => {
  // ── GET /api/blog/categories ─────────────────────
  app.get("/categories", async (_request, reply) => {
    const data = await blogService.getAllCategories();
    return reply.send({ success: true, data });
  });

  // ── GET /api/blog/authors ────────────────────────
  app.get("/authors", async (_request, reply) => {
    const data = await blogService.getAllAuthors();
    return reply.send({ success: true, data });
  });

  // ── GET /api/blog/authors/:id ────────────────────
  app.get("/authors/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await blogService.getAuthorById(id);
    return reply.send({ success: true, data });
  });

  // ── GET /api/blog/posts ──────────────────────────
  app.get("/posts", async (request, reply) => {
    const query = publicPostQuerySchema.parse(request.query);
    const result = await blogService.getPublicPosts(query);
    return reply.send({ success: true, ...result });
  });

  // ── GET /api/blog/posts/:slug ────────────────────
  app.get("/posts/:slug", async (request, reply) => {
    const { slug } = request.params as { slug: string };
    const data = await blogService.getPublicPostBySlug(slug);
    return reply.send({ success: true, data });
  });
};

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/blog
// ════════════════════════════════════════════════

export const adminBlogRoutes = async (app: FastifyInstance) => {
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireAdmin);

  // ── CATEGORIES ───────────────────────────────────

  app.get("/categories", async (_request, reply) => {
    const data = await blogService.getAllCategories();
    return reply.send({ success: true, data });
  });

  app.post("/categories", async (request, reply) => {
    const body = createCategorySchema.parse(request.body);
    const data = await blogService.createCategory(body);
    return reply.status(201).send({ success: true, data });
  });

  app.put("/categories/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const body = updateCategorySchema.parse(request.body);
    const data = await blogService.updateCategory(id, body);
    return reply.send({ success: true, data });
  });

  app.delete("/categories/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await blogService.deleteCategory(id);
    return reply.send({ success: true, ...result });
  });

  // ── AUTHORS ──────────────────────────────────────
  // Author create/update uses multipart because of avatar upload

  app.get("/authors", async (_request, reply) => {
    const data = await blogService.getAllAuthors();
    return reply.send({ success: true, data });
  });

  app.get("/authors/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await blogService.getAuthorById(id);
    return reply.send({ success: true, data });
  });

  app.post("/authors", async (request, reply) => {
    const parts = request.parts();
    const fields: Record<string, string> = {};
    let avatarFile: { buffer: Buffer; mimetype: string } | undefined;

    for await (const part of parts) {
      if (part.type === "file" && part.fieldname === "avatar") {
        avatarFile = { buffer: await part.toBuffer(), mimetype: part.mimetype };
      } else if (part.type !== "file") {
        fields[part.fieldname] = part.value as string;
      }
    }

    const body = createAuthorSchema.parse({
      name: fields.name,
      bio: fields.bio,
      userId: fields.userId,
    });

    const data = await blogService.createAuthor({ ...body, avatarFile });
    return reply.status(201).send({ success: true, data });
  });

  app.put("/authors/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const parts = request.parts();
    const fields: Record<string, string> = {};
    let avatarFile: { buffer: Buffer; mimetype: string } | undefined;

    for await (const part of parts) {
      if (part.type === "file" && part.fieldname === "avatar") {
        avatarFile = { buffer: await part.toBuffer(), mimetype: part.mimetype };
      } else if (part.type !== "file") {
        fields[part.fieldname] = part.value as string;
      }
    }

    const body = updateAuthorSchema.parse({
      name: fields.name,
      bio: fields.bio,
    });

    const data = await blogService.updateAuthor(id, { ...body, avatarFile });
    return reply.send({ success: true, data });
  });

  app.delete("/authors/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await blogService.deleteAuthor(id);
    return reply.send({ success: true, ...result });
  });

  // ── POSTS ────────────────────────────────────────
  // Posts use multipart because of cover image upload

  app.get("/posts", async (request, reply) => {
    const query = adminPostQuerySchema.parse(request.query);
    const result = await blogService.getAdminPosts(query);
    return reply.send({ success: true, ...result });
  });

  app.get("/posts/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await blogService.getAdminPostById(id);
    return reply.send({ success: true, data });
  });

  app.post("/posts", async (request, reply) => {
    const parts = request.parts();
    const fields: Record<string, string> = {};
    let coverImageFile: { buffer: Buffer; mimetype: string } | undefined;

    for await (const part of parts) {
      if (part.type === "file" && part.fieldname === "cover_image") {
        coverImageFile = {
          buffer: await part.toBuffer(),
          mimetype: part.mimetype,
        };
      } else if (part.type !== "file") {
        fields[part.fieldname] = part.value as string;
      }
    }

    const body = createPostSchema.parse({
      title: fields.title,
      excerpt: fields.excerpt,
      content: fields.content ? JSON.parse(fields.content) : undefined,
      authorId: fields.authorId || null,
      categoryId: fields.categoryId || null,
      tags: fields.tags ? JSON.parse(fields.tags) : [],
      status: fields.status,
      seoTitle: fields.seoTitle,
      seoDescription: fields.seoDescription,
    });

    if (!body.content) throw new AppError("Content is required.", 400);

    const data = await blogService.createPost({ ...body, coverImageFile });
    return reply.status(201).send({ success: true, data });
  });

  app.put("/posts/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const parts = request.parts();
    const fields: Record<string, string> = {};
    let coverImageFile: { buffer: Buffer; mimetype: string } | undefined;

    for await (const part of parts) {
      if (part.type === "file" && part.fieldname === "cover_image") {
        coverImageFile = {
          buffer: await part.toBuffer(),
          mimetype: part.mimetype,
        };
      } else if (part.type !== "file") {
        fields[part.fieldname] = part.value as string;
      }
    }

    const body = updatePostSchema.parse({
      title: fields.title,
      excerpt: fields.excerpt,
      content: fields.content ? JSON.parse(fields.content) : undefined,
      authorId:
        fields.authorId !== undefined ? fields.authorId || null : undefined,
      categoryId:
        fields.categoryId !== undefined ? fields.categoryId || null : undefined,
      tags: fields.tags ? JSON.parse(fields.tags) : undefined,
      status: fields.status,
      seoTitle: fields.seoTitle,
      seoDescription: fields.seoDescription,
    });

    const data = await blogService.updatePost(id, { ...body, coverImageFile });
    return reply.send({ success: true, data });
  });

  // POST /api/admin/blog/posts/:id/publish
  // Dedicated publish action — cleaner than passing status in update
  app.post("/posts/:id/publish", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await blogService.publishPost(id);
    return reply.send({ success: true, data });
  });

  app.delete("/posts/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await blogService.deletePost(id);
    return reply.send({ success: true, ...result });
  });

  // POST /api/admin/blog/upload-image
  // Standalone image upload for TipTap editor
  app.post("/upload-image", async (request, reply) => {
    const file = await request.file();
    if (!file) throw new AppError("No file uploaded.", 400);
    const buffer = await file.toBuffer();
    const result = await blogService.uploadBlogImage(buffer, file.mimetype);
    return reply.send({ success: true, data: result });
  });
};
