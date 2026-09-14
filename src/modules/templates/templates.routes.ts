import { FastifyInstance } from "fastify";
import * as templatesService from "./templates.service.js";
import { authenticate } from "../../middleware/authenticate.js";
import { requireAdmin } from "../../middleware/require-admin.js";
import {
  createCategorySchema,
  updateCategorySchema,
  templateQuerySchema,
  adminTemplateQuerySchema,
  updateTemplateSchema,
} from "./templates.schema.js";
import { AppError } from "../../lib/errors.js";

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/templates  and  /api/template-categories
// ════════════════════════════════════════════════

export const publicTemplateRoutes = async (app: FastifyInstance) => {
  // ── GET /api/template-categories ────────────────
  app.get("/template-categories", async (_request, reply) => {
    const data = await templatesService.getAllCategories();
    return reply.send({ success: true, data });
  });

  // ── GET /api/templates ───────────────────────────
  app.get("/templates", async (request, reply) => {
    const query = templateQuerySchema.parse(request.query);
    const result = await templatesService.getPublicTemplates(query);
    return reply.send({ success: true, ...result });
  });

  // ── GET /api/templates/:slug ─────────────────────
  app.get("/templates/:slug", async (request, reply) => {
    const { slug } = request.params as { slug: string };
    const data = await templatesService.getPublicTemplateBySlug(slug);
    return reply.send({ success: true, data });
  });
};

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/templates
// ════════════════════════════════════════════════

export const adminTemplateRoutes = async (app: FastifyInstance) => {
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireAdmin);

  // ── CATEGORIES ───────────────────────────────────

  // GET /api/admin/templates/categories
  app.get("/categories", async (_request, reply) => {
    const data = await templatesService.getAllCategories();
    return reply.send({ success: true, data });
  });

  // POST /api/admin/templates/categories
  app.post("/categories", async (request, reply) => {
    const body = createCategorySchema.parse(request.body);
    const data = await templatesService.createCategory(body);
    return reply.status(201).send({ success: true, data });
  });

  // PUT /api/admin/templates/categories/:id
  app.put("/categories/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const body = updateCategorySchema.parse(request.body);
    const data = await templatesService.updateCategory(id, body);
    return reply.send({ success: true, data });
  });

  // DELETE /api/admin/templates/categories/:id
  app.delete("/categories/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await templatesService.deleteCategory(id);
    return reply.send({ success: true, ...result });
  });

  // ── TEMPLATES ────────────────────────────────────

  // GET /api/admin/templates
  app.get("/", async (request, reply) => {
    const query = adminTemplateQuerySchema.parse(request.query);
    const result = await templatesService.getAdminTemplates(query);
    return reply.send({ success: true, ...result });
  });

  // GET /api/admin/templates/:id
  app.get("/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await templatesService.getAdminTemplateById(id);
    return reply.send({ success: true, data });
  });

  // POST /api/admin/templates
  // multipart: fields + template_file + preview_images[]
  app.post("/", async (request, reply) => {
    const parts = request.parts();

    const fields: Record<string, string> = {};
    let templateFile:
      | { buffer: Buffer; mimetype: string; filename: string }
      | undefined;
    const previewImageFiles: { buffer: Buffer; mimetype: string }[] = [];

    for await (const part of parts) {
      if (part.type === "file") {
        const buffer = await part.toBuffer();
        if (part.fieldname === "template_file") {
          templateFile = {
            buffer,
            mimetype: part.mimetype,
            filename: part.filename,
          };
        } else if (part.fieldname === "preview_images") {
          previewImageFiles.push({ buffer, mimetype: part.mimetype });
        }
      } else {
        fields[part.fieldname] = part.value as string;
      }
    }

    if (!fields.title) throw new AppError("Title is required.", 400);
    if (!fields.priceUsd) throw new AppError("Price is required.", 400);
    if (!templateFile) throw new AppError("Template file is required.", 400);

    const data = await templatesService.createTemplate({
      title: fields.title,
      description: fields.description,
      longDescription: fields.longDescription
        ? JSON.parse(fields.longDescription)
        : undefined,
      priceUsd: parseFloat(fields.priceUsd),
      categoryId: fields.categoryId || null,
      tags: fields.tags ? JSON.parse(fields.tags) : [],
      featured: fields.featured === "true",
      status: (fields.status as any) || "draft",
      templateFile,
      previewImageFiles,
    });

    return reply.status(201).send({ success: true, data });
  });

  // PUT /api/admin/templates/:id
  // multipart: same as create but all fields optional
  app.put("/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const parts = request.parts();

    const fields: Record<string, string> = {};
    let newTemplateFile:
      | { buffer: Buffer; mimetype: string; filename: string }
      | undefined;
    const newPreviewImageFiles: { buffer: Buffer; mimetype: string }[] = [];

    for await (const part of parts) {
      if (part.type === "file") {
        const buffer = await part.toBuffer();
        if (part.fieldname === "template_file") {
          newTemplateFile = {
            buffer,
            mimetype: part.mimetype,
            filename: part.filename,
          };
        } else if (part.fieldname === "preview_images") {
          newPreviewImageFiles.push({ buffer, mimetype: part.mimetype });
        }
      } else {
        fields[part.fieldname] = part.value as string;
      }
    }

    const data = await templatesService.updateTemplate(id, {
      title: fields.title,
      description: fields.description,
      longDescription: fields.longDescription
        ? JSON.parse(fields.longDescription)
        : undefined,
      priceUsd: fields.priceUsd ? parseFloat(fields.priceUsd) : undefined,
      categoryId:
        fields.categoryId !== undefined ? fields.categoryId || null : undefined,
      tags: fields.tags ? JSON.parse(fields.tags) : undefined,
      featured:
        fields.featured !== undefined ? fields.featured === "true" : undefined,
      status: fields.status as any,
      newTemplateFile,
      newPreviewImageFiles,
    });

    return reply.send({ success: true, data });
  });

  // UPDATE /api/admin/templates/:id/update
  app.patch("/:id/update", async (request, reply) => {
    const { id } = request.params as { id: string };
    const body = updateTemplateSchema.parse(request.body);
    const data = await templatesService.updateTemplate(id, body);
    return reply.send({ success: true, data });
  });

  // DELETE /api/admin/templates/:id
  app.delete("/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await templatesService.deleteTemplate(id);
    return reply.send({ success: true, ...result });
  });

  // POST /api/admin/templates/upload-image
  // standalone image upload for TipTap editor
  app.post("/upload-image", async (request, reply) => {
    const file = await request.file();
    if (!file) throw new AppError("No file uploaded.", 400);
    const buffer = await file.toBuffer();
    const result = await templatesService.uploadTemplateImage(
      buffer,
      file.mimetype,
    );
    return reply.send({ success: true, data: result });
  });
};
