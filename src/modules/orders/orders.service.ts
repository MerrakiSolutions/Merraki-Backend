import { eq, or, ilike, and, desc, asc, count, sql } from "drizzle-orm";
import { db } from "../../db/index.js";
import { orders } from "../../db/schema/orders.js";
import { getSignedDownloadUrl } from "../../lib/r2.js";
import { AppError, NotFoundError } from "../../lib/errors.js";
import { paginate, getPaginationOffset } from "../../lib/pagination.js";

interface OrderItem {
  templateId: string;
  title: string;
  priceUsd: string;
  r2Key: string;
}

// ── PUBLIC: Track order by email or order ID ──────────────────────

export const trackOrder = async (params: {
  email?: string;
  orderId?: string;
}) => {
  const { email, orderId } = params;

  if (!email && !orderId) {
    throw new AppError("Provide either an email or order ID.", 400);
  }

  if (orderId) {
    // single order by ID
    const [order] = await db
      .select({
        id: orders.id,
        guestName: orders.guestName,
        guestEmail: orders.guestEmail,
        items: orders.items,
        totalUsd: orders.totalUsd,
        currencyCharged: orders.currencyCharged,
        amountCharged: orders.amountCharged,
        status: orders.status,
        createdAt: orders.createdAt,
      })
      .from(orders)
      .where(eq(orders.id, orderId))
      .limit(1);

    if (!order) throw new NotFoundError("Order not found.");

    return [order];
  }

  // all orders by email
  const data = await db
    .select({
      id: orders.id,
      guestName: orders.guestName,
      guestEmail: orders.guestEmail,
      items: orders.items,
      totalUsd: orders.totalUsd,
      currencyCharged: orders.currencyCharged,
      amountCharged: orders.amountCharged,
      status: orders.status,
      createdAt: orders.createdAt,
    })
    .from(orders)
    .where(eq(orders.guestEmail, email!))
    .orderBy(desc(orders.createdAt));

  return data;
};

// ── PUBLIC: Get download links for paid order ─────────────────────

export const getDownloadLinks = async (downloadToken: string) => {
  const [order] = await db
    .select()
    .from(orders)
    .where(eq(orders.downloadToken, downloadToken as any))
    .limit(1);

  if (!order) throw new NotFoundError("Order not found.");
  if (order.status !== "paid") {
    throw new AppError(
      "Downloads are only available for completed orders.",
      403,
    );
  }

  const items = order.items as OrderItem[];

  // generate fresh signed URLs on every call — never stored
  const downloadItems = await Promise.all(
    items.map(async (item) => ({
      title: item.title,
      downloadUrl: await getSignedDownloadUrl(item.r2Key, 3600), // 1hr
    })),
  );

  return {
    orderId: order.id,
    guestName: order.guestName,
    items: downloadItems,
    expiresIn: "1 hour",
  };
};

// ── ADMIN: List all orders ────────────────────────────────────────

export const getAdminOrders = async (query: {
  page: number;
  limit: number;
  search?: string;
  status?: string;
  sort: string;
}) => {
  const { page, limit, search, status, sort } = query;
  const offset = getPaginationOffset(page, limit);

  const conditions = [];

  if (status) {
    conditions.push(eq(orders.status, status as any));
  }

  if (search) {
    conditions.push(
      or(
        ilike(orders.guestEmail, `%${search}%`),
        ilike(orders.guestName, `%${search}%`),
        ilike(orders.razorpayOrderId, `%${search}%`),
        ilike(orders.razorpayPaymentId, `%${search}%`),
      ),
    );
  }

  const where = conditions.length > 0 ? and(...conditions) : undefined;

  const [{ total }] = await db
    .select({ total: count() })
    .from(orders)
    .where(where);

  const orderBy =
    sort === "oldest"
      ? asc(orders.createdAt)
      : sort === "amount_high"
        ? desc(orders.totalUsd)
        : sort === "amount_low"
          ? asc(orders.totalUsd)
          : desc(orders.createdAt);

  const data = await db
    .select()
    .from(orders)
    .where(where)
    .orderBy(orderBy)
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

// ── ADMIN: Single order ───────────────────────────────────────────

export const getAdminOrderById = async (id: string) => {
  const [order] = await db
    .select()
    .from(orders)
    .where(eq(orders.id, id))
    .limit(1);

  if (!order) throw new NotFoundError("Order not found.");
  return order;
};

// ── ADMIN: Delete order (hard delete) ────────────────────────────

export const deleteOrder = async (id: string) => {
  const [order] = await db
    .select({ id: orders.id })
    .from(orders)
    .where(eq(orders.id, id))
    .limit(1);

  if (!order) throw new NotFoundError("Order not found.");

  await db.delete(orders).where(eq(orders.id, id));
  return { message: "Order deleted successfully." };
};

// ── ADMIN: Export orders CSV ──────────────────────────────────────

export const exportOrdersCSV = async () => {
  const data = await db.select().from(orders).orderBy(desc(orders.createdAt));

  const headers = [
    "Order ID",
    "Name",
    "Email",
    "Total USD",
    "Currency",
    "Amount Charged",
    "Status",
    "Razorpay Order ID",
    "Razorpay Payment ID",
    "Created At",
  ];

  const rows = data.map((o) => [
    o.id,
    o.guestName,
    o.guestEmail,
    o.totalUsd,
    o.currencyCharged,
    o.amountCharged,
    o.status,
    o.razorpayOrderId || "",
    o.razorpayPaymentId || "",
    o.createdAt.toISOString(),
  ]);

  return [headers, ...rows]
    .map((row) => row.map((v) => `"${v}"`).join(","))
    .join("\n");
};
