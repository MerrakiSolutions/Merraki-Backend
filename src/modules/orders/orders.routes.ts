import { FastifyInstance } from "fastify";
import { z } from "zod";
import * as ordersService from "./orders.service.js";
import { authenticate } from "../../middleware/authenticate.js";
import { requireAdmin } from "../../middleware/require-admin.js";

const trackQuerySchema = z
  .object({
    email: z.string().email().optional(),
    order_id: z.string().uuid().optional(),
  })
  .refine((d) => d.email || d.order_id, {
    message: "Provide either email or order_id",
  });

const adminOrderQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(20),
  search: z.string().optional(),
  status: z.enum(["pending", "paid", "failed", "refunded"]).optional(),
  sort: z
    .enum(["newest", "oldest", "amount_high", "amount_low"])
    .default("newest"),
});

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/orders
// ════════════════════════════════════════════════

export const publicOrderRoutes = async (app: FastifyInstance) => {
  // GET /api/orders/track?email= or ?order_id=
  app.get("/track", async (request, reply) => {
    const query = trackQuerySchema.parse(request.query);
    const data = await ordersService.trackOrder({
      email: query.email,
      orderId: query.order_id,
    });
    return reply.send({ success: true, data });
  });

  // GET /api/orders/download/:token
  // Regenerates fresh signed R2 URLs on every call
  app.get("/download/:token", async (request, reply) => {
    const { token } = request.params as { token: string };
    const data = await ordersService.getDownloadLinks(token);
    return reply.send({ success: true, data });
  });
};

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/orders
// ════════════════════════════════════════════════

export const adminOrderRoutes = async (app: FastifyInstance) => {
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireAdmin);

  // GET /api/admin/orders
  app.get("/", async (request, reply) => {
    const query = adminOrderQuerySchema.parse(request.query);
    const result = await ordersService.getAdminOrders(query);
    return reply.send({ success: true, ...result });
  });

  // GET /api/admin/orders/export
  app.get("/export", async (_request, reply) => {
    const csv = await ordersService.exportOrdersCSV();
    return reply
      .header("Content-Type", "text/csv")
      .header("Content-Disposition", 'attachment; filename="orders.csv"')
      .send(csv);
  });

  // GET /api/admin/orders/:id
  app.get("/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const data = await ordersService.getAdminOrderById(id);
    return reply.send({ success: true, data });
  });

  // DELETE /api/admin/orders/:id
  app.delete("/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const result = await ordersService.deleteOrder(id);
    return reply.send({ success: true, ...result });
  });
};
