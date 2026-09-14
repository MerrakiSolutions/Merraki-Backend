import { eq, ilike, or, and, count, asc, desc, sql } from "drizzle-orm";
import crypto from "crypto";
import slugify from "slugify";
import { db } from "../../db/index.js";
import {
  newsletterSubscribers,
  newsletterCategories,
  newsletterCampaigns,
} from "../../db/schema/newsletter.js";
import { sendEmail } from "../../lib/resend.js";
import { renderEmail } from "../../emails/index.js";
import { AppError, ConflictError, NotFoundError } from "../../lib/errors.js";
import { paginate, getPaginationOffset } from "../../lib/pagination.js";
import { env } from "../../config/env.js";

// ── Helpers ───────────────────────────────────────────────────────

const generateToken = (): string => crypto.randomBytes(32).toString("hex");

const generateSlug = (name: string): string =>
  slugify(name, { lower: true, strict: true, trim: true });

// ════════════════════════════════════════════════
// CATEGORIES
// ════════════════════════════════════════════════

export const getAllCategories = async () =>
  db
    .select()
    .from(newsletterCategories)
    .orderBy(asc(newsletterCategories.name));

export const createCategory = async (data: { name: string; slug?: string }) => {
  const slug = data.slug || generateSlug(data.name);

  const [existing] = await db
    .select({ id: newsletterCategories.id })
    .from(newsletterCategories)
    .where(eq(newsletterCategories.slug, slug))
    .limit(1);

  if (existing)
    throw new ConflictError("A category with this slug already exists.");

  const [created] = await db
    .insert(newsletterCategories)
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
    .from(newsletterCategories)
    .where(eq(newsletterCategories.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Category not found.");

  if (data.slug && data.slug !== existing.slug) {
    const [slugTaken] = await db
      .select({ id: newsletterCategories.id })
      .from(newsletterCategories)
      .where(eq(newsletterCategories.slug, data.slug))
      .limit(1);
    if (slugTaken)
      throw new ConflictError("A category with this slug already exists.");
  }

  const [updated] = await db
    .update(newsletterCategories)
    .set({ ...data, updatedAt: new Date() })
    .where(eq(newsletterCategories.id, id))
    .returning();

  return updated;
};

export const deleteCategory = async (id: string) => {
  const [existing] = await db
    .select({ id: newsletterCategories.id })
    .from(newsletterCategories)
    .where(eq(newsletterCategories.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Category not found.");

  await db.delete(newsletterCategories).where(eq(newsletterCategories.id, id));

  return { message: "Category deleted successfully." };
};

// ════════════════════════════════════════════════
// SUBSCRIBERS — PUBLIC
// ════════════════════════════════════════════════

export const subscribe = async (data: {
  email: string;
  name?: string;
  categoryIds?: string[];
}) => {
  const { email, name, categoryIds = [] } = data;

  const [existing] = await db
    .select()
    .from(newsletterSubscribers)
    .where(eq(newsletterSubscribers.email, email))
    .limit(1);

  if (existing) {
    if (existing.confirmed) {
      throw new ConflictError("This email is already subscribed.");
    }

    // resend confirmation if not yet confirmed
    const confirmToken = generateToken();

    await db
      .update(newsletterSubscribers)
      .set({
        confirmToken,
        categoryIds: categoryIds as any,
        name: name || existing.name,
        updatedAt: new Date(),
      })
      .where(eq(newsletterSubscribers.id, existing.id));

    await sendEmail({
      to: email,
      subject: "Confirm your MerrakiSolutions newsletter subscription",
      html: renderEmail.newsletterConfirm({
        name,
        confirmUrl: `${env.APP_URL}/api/newsletter/confirm/${confirmToken}`,
      }),
    });

    return { message: "Confirmation email resent. Please check your inbox." };
  }

  // new subscriber
  const confirmToken = generateToken();
  const unsubscribeToken = generateToken();

  await db.insert(newsletterSubscribers).values({
    email,
    name,
    categoryIds: categoryIds as any,
    confirmed: false,
    confirmToken,
    unsubscribeToken,
  });

  await sendEmail({
    to: email,
    subject: "Confirm your MerrakiSolutions newsletter subscription",
    html: renderEmail.newsletterConfirm({
      name,
      confirmUrl: `${env.APP_URL}/api/newsletter/confirm/${confirmToken}`,
    }),
  });

  return {
    message:
      "Almost there! Please check your email to confirm your subscription.",
  };
};

export const confirmSubscription = async (token: string) => {
  const [subscriber] = await db
    .select()
    .from(newsletterSubscribers)
    .where(eq(newsletterSubscribers.confirmToken, token))
    .limit(1);

  if (!subscriber)
    throw new NotFoundError("Invalid or expired confirmation token.");
  if (subscriber.confirmed)
    throw new AppError("Subscription already confirmed.", 400);

  await db
    .update(newsletterSubscribers)
    .set({
      confirmed: true,
      confirmToken: null,
      updatedAt: new Date(),
    })
    .where(eq(newsletterSubscribers.id, subscriber.id));

  return {
    message: "Subscription confirmed! Welcome to MerrakiSolutions newsletter.",
  };
};

export const unsubscribe = async (token: string) => {
  const [subscriber] = await db
    .select()
    .from(newsletterSubscribers)
    .where(eq(newsletterSubscribers.unsubscribeToken, token))
    .limit(1);

  if (!subscriber) throw new NotFoundError("Invalid unsubscribe token.");

  await db
    .delete(newsletterSubscribers)
    .where(eq(newsletterSubscribers.id, subscriber.id));

  return { message: "You have been unsubscribed successfully." };
};

// ════════════════════════════════════════════════
// SUBSCRIBERS — ADMIN
// ════════════════════════════════════════════════

export const getAdminSubscribers = async (query: {
  page: number;
  limit: number;
  search?: string;
  confirmed?: string;
  categoryId?: string;
}) => {
  const { page, limit, search, confirmed, categoryId } = query;
  const offset = getPaginationOffset(page, limit);

  const conditions = [];

  if (search) {
    conditions.push(
      or(
        ilike(newsletterSubscribers.email, `%${search}%`),
        ilike(newsletterSubscribers.name, `%${search}%`),
      ),
    );
  }

  if (confirmed !== undefined) {
    conditions.push(eq(newsletterSubscribers.confirmed, confirmed === "true"));
  }

  if (categoryId) {
    conditions.push(
      sql`${categoryId}::uuid = ANY(${newsletterSubscribers.categoryIds})`,
    );
  }

  const where = conditions.length > 0 ? and(...conditions) : undefined;

  const [{ total }] = await db
    .select({ total: count() })
    .from(newsletterSubscribers)
    .where(where);

  const data = await db
    .select()
    .from(newsletterSubscribers)
    .where(where)
    .orderBy(desc(newsletterSubscribers.createdAt))
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

export const deleteSubscriber = async (id: string) => {
  const [existing] = await db
    .select({ id: newsletterSubscribers.id })
    .from(newsletterSubscribers)
    .where(eq(newsletterSubscribers.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Subscriber not found.");

  await db
    .delete(newsletterSubscribers)
    .where(eq(newsletterSubscribers.id, id));

  return { message: "Subscriber deleted successfully." };
};

export const exportSubscribersCSV = async () => {
  const data = await db
    .select()
    .from(newsletterSubscribers)
    .orderBy(desc(newsletterSubscribers.createdAt));

  const headers = ["ID", "Email", "Name", "Confirmed", "Created At"];

  const rows = data.map((s) => [
    s.id,
    s.email,
    s.name || "",
    s.confirmed ? "Yes" : "No",
    s.createdAt.toISOString(),
  ]);

  return [headers, ...rows]
    .map((row) => row.map((v) => `"${v}"`).join(","))
    .join("\n");
};

// ════════════════════════════════════════════════
// CAMPAIGNS
// ════════════════════════════════════════════════

export const getAdminCampaigns = async (query: {
  page: number;
  limit: number;
  search?: string;
  status?: string;
}) => {
  const { page, limit, search, status } = query;
  const offset = getPaginationOffset(page, limit);

  const conditions = [];

  if (search) {
    conditions.push(
      or(
        ilike(newsletterCampaigns.title, `%${search}%`),
        ilike(newsletterCampaigns.subject, `%${search}%`),
      ),
    );
  }

  if (status) {
    conditions.push(eq(newsletterCampaigns.status, status as any));
  }

  const where = conditions.length > 0 ? and(...conditions) : undefined;

  const [{ total }] = await db
    .select({ total: count() })
    .from(newsletterCampaigns)
    .where(where);

  const data = await db
    .select()
    .from(newsletterCampaigns)
    .where(where)
    .orderBy(desc(newsletterCampaigns.createdAt))
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

export const getAdminCampaignById = async (id: string) => {
  const [campaign] = await db
    .select()
    .from(newsletterCampaigns)
    .where(eq(newsletterCampaigns.id, id))
    .limit(1);

  if (!campaign) throw new NotFoundError("Campaign not found.");
  return campaign;
};

export const createCampaign = async (data: {
  title: string;
  subject: string;
  content: any;
  categoryId?: string | null;
}) => {
  const [created] = await db
    .insert(newsletterCampaigns)
    .values({
      title: data.title,
      subject: data.subject,
      content: data.content,
      categoryId: data.categoryId || null,
      status: "draft",
    })
    .returning();

  return created;
};

export const updateCampaign = async (
  id: string,
  data: {
    title?: string;
    subject?: string;
    content?: any;
    categoryId?: string | null;
  },
) => {
  const [existing] = await db
    .select()
    .from(newsletterCampaigns)
    .where(eq(newsletterCampaigns.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Campaign not found.");

  if (existing.status === "sent") {
    throw new AppError(
      "Cannot edit a campaign that has already been sent.",
      400,
    );
  }

  const updateData: Record<string, any> = { updatedAt: new Date() };
  if (data.title !== undefined) updateData.title = data.title;
  if (data.subject !== undefined) updateData.subject = data.subject;
  if (data.content !== undefined) updateData.content = data.content;
  if (data.categoryId !== undefined)
    updateData.categoryId = data.categoryId || null;

  const [updated] = await db
    .update(newsletterCampaigns)
    .set(updateData)
    .where(eq(newsletterCampaigns.id, id))
    .returning();

  return updated;
};

export const deleteCampaign = async (id: string) => {
  const [existing] = await db
    .select({ id: newsletterCampaigns.id, status: newsletterCampaigns.status })
    .from(newsletterCampaigns)
    .where(eq(newsletterCampaigns.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Campaign not found.");
  if (existing.status === "sending") {
    throw new AppError(
      "Cannot delete a campaign that is currently sending.",
      400,
    );
  }

  await db.delete(newsletterCampaigns).where(eq(newsletterCampaigns.id, id));

  return { message: "Campaign deleted successfully." };
};

export const sendCampaign = async (id: string) => {
  // ── 1. Fetch campaign ──────────────────────────
  const [campaign] = await db
    .select()
    .from(newsletterCampaigns)
    .where(eq(newsletterCampaigns.id, id))
    .limit(1);

  if (!campaign) throw new NotFoundError("Campaign not found.");
  if (campaign.status === "sent") {
    throw new AppError("This campaign has already been sent.", 400);
  }
  if (campaign.status === "sending") {
    throw new AppError("This campaign is already being sent.", 400);
  }

  // ── 2. Mark as sending ─────────────────────────
  await db
    .update(newsletterCampaigns)
    .set({ status: "sending", updatedAt: new Date() })
    .where(eq(newsletterCampaigns.id, id));

  // ── 3. Fetch recipients ────────────────────────
  // confirmed subscribers only
  // if categoryId is set — only subscribers of that category
  // if categoryId is null — send to ALL confirmed subscribers
  let subscribers: {
    email: string;
    name: string | null;
    unsubscribeToken: string;
  }[];

  if (campaign.categoryId) {
    subscribers = await db
      .select({
        email: newsletterSubscribers.email,
        name: newsletterSubscribers.name,
        unsubscribeToken: newsletterSubscribers.unsubscribeToken,
      })
      .from(newsletterSubscribers)
      .where(
        and(
          eq(newsletterSubscribers.confirmed, true),
          sql`${campaign.categoryId}::uuid = ANY(${newsletterSubscribers.categoryIds})`,
        ),
      );
  } else {
    subscribers = await db
      .select({
        email: newsletterSubscribers.email,
        name: newsletterSubscribers.name,
        unsubscribeToken: newsletterSubscribers.unsubscribeToken,
      })
      .from(newsletterSubscribers)
      .where(eq(newsletterSubscribers.confirmed, true));
  }

  if (subscribers.length === 0) {
    // no recipients — revert to draft
    await db
      .update(newsletterCampaigns)
      .set({ status: "draft", updatedAt: new Date() })
      .where(eq(newsletterCampaigns.id, id));

    throw new AppError(
      "No confirmed subscribers found for this campaign.",
      400,
    );
  }

  // ── 4. Convert TipTap JSON to plain HTML ───────
  // Simple recursive converter — handles common TipTap nodes
  const tiptapToHtml = (content: any): string => {
    if (!content || !content.content) return "";

    return content.content
      .map((node: any) => {
        switch (node.type) {
          case "paragraph":
            return `<p style="margin:0 0 16px;font-size:15px;color:#475569;line-height:1.7">${node.content ? node.content.map(renderInline).join("") : ""}</p>`;
          case "heading": {
            const level = node.attrs?.level || 2;
            const size = level === 1 ? "24px" : level === 2 ? "20px" : "17px";
            return `<h${level} style="margin:24px 0 12px;font-size:${size};font-weight:700;color:#0f172a">${node.content ? node.content.map(renderInline).join("") : ""}</h${level}>`;
          }
          case "bulletList":
            return `<ul style="margin:0 0 16px;padding-left:24px">${node.content ? node.content.map((li: any) => `<li style="margin-bottom:8px;font-size:15px;color:#475569;line-height:1.7">${li.content ? li.content.map(tiptapToHtml).join("") : ""}</li>`).join("") : ""}</ul>`;
          case "orderedList":
            return `<ol style="margin:0 0 16px;padding-left:24px">${node.content ? node.content.map((li: any) => `<li style="margin-bottom:8px;font-size:15px;color:#475569;line-height:1.7">${li.content ? li.content.map(tiptapToHtml).join("") : ""}</li>`).join("") : ""}</ol>`;
          case "blockquote":
            return `<blockquote style="margin:0 0 16px;padding:12px 20px;border-left:4px solid #0f172a;background:#f8fafc;color:#475569;font-style:italic">${node.content ? node.content.map(tiptapToHtml).join("") : ""}</blockquote>`;
          case "image":
            return `<img src="${node.attrs?.src || ""}" alt="${node.attrs?.alt || ""}" style="max-width:100%;height:auto;border-radius:6px;margin:0 0 16px" />`;
          case "horizontalRule":
            return `<hr style="border:none;border-top:1px solid #f1f5f9;margin:24px 0" />`;
          default:
            return node.content ? node.content.map(tiptapToHtml).join("") : "";
        }
      })
      .join("");
  };

  const renderInline = (node: any): string => {
    if (node.type === "text") {
      let text = node.text || "";
      const marks = node.marks || [];
      for (const mark of marks) {
        if (mark.type === "bold") text = `<strong>${text}</strong>`;
        if (mark.type === "italic") text = `<em>${text}</em>`;
        if (mark.type === "underline") text = `<u>${text}</u>`;
        if (mark.type === "link") {
          text = `<a href="${mark.attrs?.href || "#"}" style="color:#0f172a">${text}</a>`;
        }
      }
      return text;
    }
    return "";
  };

  const htmlContent = tiptapToHtml(campaign.content);

  // ── 5. Send emails in batches ──────────────────
  // Resend allows up to 100 emails/second on paid plan
  // Batch in groups of 50 to be safe on free tier
  const BATCH_SIZE = 50;
  let sentCount = 0;

  for (let i = 0; i < subscribers.length; i += BATCH_SIZE) {
    const batch = subscribers.slice(i, i + BATCH_SIZE);

    await Promise.allSettled(
      batch.map((subscriber) =>
        sendEmail({
          to: subscriber.email,
          subject: campaign.subject,
          html: renderEmail.newsletterCampaign({
            subject: campaign.subject,
            content: htmlContent,
            unsubscribeUrl: `${env.APP_URL}/api/newsletter/unsubscribe/${subscriber.unsubscribeToken}`,
          }),
        }),
      ),
    );

    sentCount += batch.length;

    // small delay between batches to avoid rate limits
    if (i + BATCH_SIZE < subscribers.length) {
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
  }

  // ── 6. Mark as sent ────────────────────────────
  await db
    .update(newsletterCampaigns)
    .set({
      status: "sent",
      sentAt: new Date(),
      recipientCount: String(sentCount),
      updatedAt: new Date(),
    })
    .where(eq(newsletterCampaigns.id, id));

  return {
    message: `Campaign sent successfully to ${sentCount} subscriber${sentCount !== 1 ? "s" : ""}.`,
    recipientCount: sentCount,
  };
};
