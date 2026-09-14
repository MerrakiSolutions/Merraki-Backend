import { eq } from "drizzle-orm";
import { db } from "../../db/index.js";
import { orders } from "../../db/schema/orders.js";
import { getSignedDownloadUrl, uploadToR2 } from "../../lib/r2.js";
import { sendEmail } from "../../lib/resend.js";
import { generateInvoicePdf } from "../../lib/invoice.js";
import { renderEmail } from "../../emails/index.js";
import { AppError } from "../../lib/errors.js";
import { env } from "../../config/env.js";

// ── Types ─────────────────────────────────────────────────────────

interface OrderItem {
  templateId: string;
  title: string;
  priceUsd: string;
  r2Key: string;
}

// ── Core: process a successful payment ───────────────────────────
// Called by both webhook and manual /verify endpoint
// Idempotent — safe to call twice, skips if already paid

export const processSuccessfulPayment = async (
  razorpayOrderId: string,
  razorpayPaymentId: string,
): Promise<void> => {
  // ── 1. Find order ──────────────────────────────
  const [order] = await db
    .select()
    .from(orders)
    .where(eq(orders.razorpayOrderId, razorpayOrderId))
    .limit(1);

  if (!order) throw new AppError("Order not found.", 404);

  // ── 2. Idempotency check ───────────────────────
  if (order.status === "paid") return; // already processed — skip

  // ── 3. Mark as paid ────────────────────────────
  await db
    .update(orders)
    .set({
      status: "paid",
      razorpayPaymentId,
      updatedAt: new Date(),
    })
    .where(eq(orders.id, order.id));

  const items = order.items as OrderItem[];

  // ── 4. Generate signed download URLs (1hr) ─────
  const downloadItems = await Promise.all(
    items.map(async (item) => ({
      title: item.title,
      downloadUrl: await getSignedDownloadUrl(item.r2Key, 3600),
    })),
  );

  // ── 5. Generate PDF invoice ────────────────────
  const invoicePdf = await generateInvoicePdf({
    orderId: order.id,
    guestName: order.guestName,
    guestEmail: order.guestEmail,
    billingAddress: order.billingAddress as any,
    items: items.map((i) => ({ title: i.title, priceUsd: i.priceUsd })),
    subtotalUsd: String(order.subtotalUsd),
    totalUsd: String(order.totalUsd),
    currencyCharged: order.currencyCharged as "USD" | "INR",
    amountCharged: String(order.amountCharged),
    exchangeRate: order.exchangeRate ? String(order.exchangeRate) : undefined,
    createdAt: order.createdAt,
  });

  // ── 6. Upload invoice to R2 ────────────────────
  const invoiceKey = `invoices/${order.id}.pdf`;
  await uploadToR2(invoiceKey, invoicePdf, "application/pdf");

  await db
    .update(orders)
    .set({ invoiceR2Key: invoiceKey, updatedAt: new Date() })
    .where(eq(orders.id, order.id));

  // ── 7. Send confirmation email ─────────────────
  const trackOrderUrl = `${env.FRONTEND_URL}/track-order?order_id=${order.id}`;

  await sendEmail({
    to: order.guestEmail,
    subject: `Your MerrakiSolutions order is confirmed — #${order.id.slice(0, 8).toUpperCase()}`,
    html: renderEmail.orderConfirmation({
      guestName: order.guestName,
      orderId: order.id,
      items: items.map((i) => ({ title: i.title, priceUsd: i.priceUsd })),
      totalUsd: String(order.totalUsd),
      currencyCharged: order.currencyCharged as "USD" | "INR",
      amountCharged: String(order.amountCharged),
      exchangeRate: order.exchangeRate ? String(order.exchangeRate) : undefined,
      createdAt: order.createdAt.toLocaleDateString("en-US", {
        year: "numeric",
        month: "long",
        day: "numeric",
      }),
    }),
    attachments: [
      { filename: `invoice-${order.id.slice(0, 8)}.pdf`, content: invoicePdf },
    ],
  });

  // ── 8. Send download links email ───────────────
  await sendEmail({
    to: order.guestEmail,
    subject: `Your downloads are ready — MerrakiSolutions`,
    html: renderEmail.downloadLinks({
      guestName: order.guestName,
      orderId: order.id,
      items: downloadItems,
      trackOrderUrl,
      expiresIn: "1 hour",
    }),
    attachments: [
      { filename: `invoice-${order.id.slice(0, 8)}.pdf`, content: invoicePdf },
    ],
  });
};

// ── Mark order as failed ─────────────────────────────────────────

export const markOrderFailed = async (
  razorpayOrderId: string,
): Promise<void> => {
  await db
    .update(orders)
    .set({ status: "failed", updatedAt: new Date() })
    .where(eq(orders.razorpayOrderId, razorpayOrderId));
};

// ── Exchange rate (public endpoint) ─────────────────────────────

export { getUsdToInrRate } from "../../lib/exchange-rate.js";
