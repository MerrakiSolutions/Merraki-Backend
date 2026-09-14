import { FastifyInstance } from "fastify";
import * as newsletterService from "./newsletter.service.js";
import { authenticate } from "../../middleware/authenticate.js";
import { requireAdmin } from "../../middleware/require-admin.js";
import {
  subscribeSchema,
  createCategorySchema,
  updateCategorySchema,
  createCampaignSchema,
  updateCampaignSchema,
  subscriberQuerySchema,
  campaignQuerySchema,
} from "./newsletter.schema.js";

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/newsletter
// ════════════════════════════════════════════════

export const publicNewsletterRoutes = async (app: FastifyInstance) => {
  // ── POST /api/newsletter/subscribe ───────────────
  app.post("/subscribe", async (request, reply) => {
    const body = subscribeSchema.parse(request.body);
    const result = await newsletterService.subscribe(body);
    return reply.status(201).send({ success: true, ...result });
  });

  // ── GET /api/newsletter/confirm/:token ───────────
  // Called when subscriber clicks confirmation email link
  app.get("/confirm/:token", async (request, reply) => {
    const { token } = request.params as { token: string };
    const result = await newsletterService.confirmSubscription(token);

    // redirect to frontend with success message
    return reply.redirect(`${process.env.FRONTEND_URL}/newsletter/confirmed`);
  });

  // ── GET /api/newsletter/unsubscribe/:token ───────
  // Called when subscriber clicks unsubscribe link in email
  app.get("/unsubscribe/:token", async (request, reply) => {
    const { token } = request.params as { token: string };
    await newsletterService.unsubscribe(token);

    // redirect to frontend with goodbye message
    return reply.redirect(
      `${process.env.FRONTEND_URL}/newsletter/unsubscribed`,
    );
  });

  // ── GET /api/newsletter/categories ──────────────
  // Public — frontend uses this to show subscription options
  app.get("/categories", async (_request, reply) => {
    const data = await newsletterService.getAllCategories();
    return reply.send({ success: true, data });
  });
};

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/newsletter
// ════════════════════════════════════════════════

export const adminNewsletterRoutes = async (app: FastifyInstance) => {
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireAdmin);

  // ── CATEGORIES ───────────────────────────────────

  app.get("/categories", async (_request, reply) => {
    const data = await newsletterService.getAllCategories();
    return reply.send({ success: true, data });
  });

  app.post("/categories", async (request, reply) => {
    const body = createCategorySchema.parse(request.body);
    const data = await newsletterService.createCategory(body);
    return reply.status(201).send({ success: true, data });
  });

  app.put("/categories/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const body = updateCategorySchema.parse(request.body);
    const data = await newsletterService.updateCategory(id, body);
    return reply.send({ success: true, data });
  });

  app.delete("/categories/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await newsletterService.deleteCategory(id);
    return reply.send({ success: true, ...result });
  });

  // ── SUBSCRIBERS ──────────────────────────────────

  app.get("/subscribers", async (request, reply) => {
    const query = subscriberQuerySchema.parse(request.query);
    const result = await newsletterService.getAdminSubscribers(query);
    return reply.send({ success: true, ...result });
  });

  app.delete("/subscribers/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await newsletterService.deleteSubscriber(id);
    return reply.send({ success: true, ...result });
  });

  app.get("/subscribers/export", async (_request, reply) => {
    const csv = await newsletterService.exportSubscribersCSV();
    return reply
      .header("Content-Type", "text/csv")
      .header("Content-Disposition", 'attachment; filename="subscribers.csv"')
      .send(csv);
  });

  // ── CAMPAIGNS ────────────────────────────────────

  app.get("/campaigns", async (request, reply) => {
    const query = campaignQuerySchema.parse(request.query);
    const result = await newsletterService.getAdminCampaigns(query);
    return reply.send({ success: true, ...result });
  });

  app.get("/campaigns/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await newsletterService.getAdminCampaignById(id);
    return reply.send({ success: true, data });
  });

  app.post("/campaigns", async (request, reply) => {
    const body = createCampaignSchema.parse(request.body);
    const data = await newsletterService.createCampaign(body);
    return reply.status(201).send({ success: true, data });
  });

  app.put("/campaigns/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const body = updateCampaignSchema.parse(request.body);
    const data = await newsletterService.updateCampaign(id, body);
    return reply.send({ success: true, data });
  });

  app.delete("/campaigns/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await newsletterService.deleteCampaign(id);
    return reply.send({ success: true, ...result });
  });

  // POST /api/admin/newsletter/campaigns/:id/send
  app.post("/campaigns/:id/send", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await newsletterService.sendCampaign(id);
    return reply.send({ success: true, ...result });
  });
};
