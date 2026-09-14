import { FastifyInstance } from "fastify";
import { z } from "zod";
import {
  processSuccessfulPayment,
  markOrderFailed,
  getUsdToInrRate,
} from "./payments.service.js";
import {
  verifyRazorpaySignature,
  verifyWebhookSignature,
} from "../../lib/razorpay.js";
import { AppError } from "../../lib/errors.js";

const verifySchema = z.object({
  razorpay_order_id: z.string().min(1),
  razorpay_payment_id: z.string().min(1),
  razorpay_signature: z.string().min(1),
});

export const paymentsRoutes = async (app: FastifyInstance) => {
  // ── GET /api/payments/exchange-rate ──────────────
  // Public — frontend calls this to show INR equivalent
  app.get("/exchange-rate", async (_request, reply) => {
    const rate = await getUsdToInrRate();
    return reply.send({
      success: true,
      data: { usdToInr: rate, updatedAt: new Date().toISOString() },
    });
  });

  // ── POST /api/payments/webhook ────────────────────
  // Razorpay fires this automatically after payment
  // Signature verified via HMAC-SHA256
  app.post("/webhook", async (request, reply) => {
    const signature = request.headers["x-razorpay-signature"] as string;

    if (!signature) {
      return reply
        .status(400)
        .send({ success: false, error: "Missing signature" });
    }

    const rawBody = (request as any).rawBody as string;

    if (!rawBody) {
      return reply
        .status(400)
        .send({ success: false, error: "Missing raw body" });
    }

    // verify webhook signature
    const isValid = verifyWebhookSignature(rawBody, signature);
    if (!isValid) {
      return reply
        .status(400)
        .send({ success: false, error: "Invalid signature" });
    }

    const event = request.body as {
      event: string;
      payload: {
        payment: {
          entity: {
            order_id: string;
            id: string;
          };
        };
      };
    };

    if (event.event === "payment.captured") {
      const { order_id, id: payment_id } = event.payload.payment.entity;
      await processSuccessfulPayment(order_id, payment_id);
    }

    if (event.event === "payment.failed") {
      const { order_id } = event.payload.payment.entity;
      await markOrderFailed(order_id);
    }

    // always return 200 to Razorpay
    return reply.status(200).send({ success: true });
  });

  // ── POST /api/payments/verify ─────────────────────
  // Manual verify — called by frontend after Razorpay checkout success
  // Use this for testing + as production fallback if webhook is delayed
  app.post("/verify", async (request, reply) => {
    const body = verifySchema.parse(request.body);

    const isValid = verifyRazorpaySignature(
      body.razorpay_order_id,
      body.razorpay_payment_id,
      body.razorpay_signature,
    );

    if (!isValid) {
      throw new AppError(
        "Payment verification failed. Invalid signature.",
        400,
      );
    }

    await processSuccessfulPayment(
      body.razorpay_order_id,
      body.razorpay_payment_id,
    );

    return reply.send({
      success: true,
      message: "Payment verified successfully.",
    });
  });
};
