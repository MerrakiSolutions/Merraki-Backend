import { FastifyInstance } from "fastify";
import { createOrderSchema } from "./checkout.schema.js";
import { createCheckoutOrder } from "./checkout.service.js";

export const checkoutRoutes = async (app: FastifyInstance) => {
  // POST /api/checkout/create-order
  app.post("/create-order", async (request, reply) => {
    const body = createOrderSchema.parse(request.body);

    const result = await createCheckoutOrder({
      ...body,
      ipAddress: request.ip,
    });

    return reply.status(201).send({ success: true, data: result });
  });
};
