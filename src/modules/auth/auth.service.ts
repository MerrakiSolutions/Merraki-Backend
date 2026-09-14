import bcrypt from "bcryptjs";
import { eq, and, gt, or, ilike, count } from "drizzle-orm";
import crypto from "crypto";
import { v4 as uuidv4 } from "uuid";
import { db } from "../../db/index.js";
import {
  users,
  refreshTokens,
  adminActivityLogs,
} from "../../db/schema/users.js";
import {
  signAccessToken,
  signRefreshToken,
  verifyRefreshToken,
} from "../../lib/jwt.js";
import { sendEmail } from "../../lib/resend.js";
import { uploadImageToCloudinary } from "../../lib/cloudinary.js";
import {
  AppError,
  ConflictError,
  NotFoundError,
  UnauthorizedError,
} from "../../lib/errors.js";
import { paginate, getPaginationOffset } from "../../lib/pagination.js";
import { env } from "../../config/env.js";
import { renderEmail } from "../../emails/index.js";

// ── Internal helpers ──────────────────────────────────────────────

const generateToken = (): string => crypto.randomBytes(32).toString("hex");

const hashToken = (token: string): string =>
  crypto.createHash("sha256").update(token).digest("hex");

const safeFields = {
  id: users.id,
  name: users.name,
  email: users.email,
  avatarUrl: users.avatarUrl,
  role: users.role,
  isActive: users.isActive,
  lastLoginAt: users.lastLoginAt,
  createdAt: users.createdAt,
  updatedAt: users.updatedAt,
};

const issueTokenPair = async (
  userId: string,
  email: string,
  role: "admin" | "superadmin",
) => {
  const tokenId = uuidv4();
  const accessToken = signAccessToken({ userId, email, role });
  const refreshToken = signRefreshToken({ userId, tokenId });

  const expiresAt = new Date();
  expiresAt.setDate(expiresAt.getDate() + 7);

  await db.insert(refreshTokens).values({
    id: tokenId,
    userId,
    tokenHash: hashToken(refreshToken),
    expiresAt,
  });

  return { accessToken, refreshToken };
};

const logActivity = async (
  userId: string,
  action:
    | "login"
    | "logout"
    | "password_change"
    | "profile_update"
    | "avatar_update",
  ipAddress?: string,
  userAgent?: string,
): Promise<void> => {
  await db.insert(adminActivityLogs).values({
    userId,
    action,
    ipAddress: ipAddress as any,
    userAgent,
  });
};

// ── Login ─────────────────────────────────────────────────────────

export const login = async (
  email: string,
  password: string,
  ipAddress?: string,
  userAgent?: string,
) => {
  const [user] = await db
    .select()
    .from(users)
    .where(eq(users.email, email))
    .limit(1);

  if (!user) throw new UnauthorizedError("Invalid email or password.");
  if (!user.isActive)
    throw new AppError("Account deactivated. Contact your superadmin.", 403);

  const valid = await bcrypt.compare(password, user.passwordHash);
  if (!valid) throw new UnauthorizedError("Invalid email or password.");

  const tokens = await issueTokenPair(user.id, user.email, user.role);

  await db
    .update(users)
    .set({ lastLoginAt: new Date(), updatedAt: new Date() })
    .where(eq(users.id, user.id));

  await logActivity(user.id, "login", ipAddress, userAgent);

  return {
    accessToken: tokens.accessToken,
    refreshToken: tokens.refreshToken,
    user: {
      id: user.id,
      name: user.name,
      email: user.email,
      role: user.role,
      avatarUrl: user.avatarUrl,
    },
  };
};

// ── Refresh ───────────────────────────────────────────────────────

export const refreshAccessToken = async (token: string) => {
  let payload: { userId: string; tokenId: string };

  try {
    payload = verifyRefreshToken(token);
  } catch {
    throw new UnauthorizedError("Invalid or expired refresh token.");
  }

  const [stored] = await db
    .select()
    .from(refreshTokens)
    .where(
      and(
        eq(refreshTokens.id, payload.tokenId),
        eq(refreshTokens.tokenHash, hashToken(token)),
        gt(refreshTokens.expiresAt, new Date()),
      ),
    )
    .limit(1);

  if (!stored)
    throw new UnauthorizedError("Session expired. Please log in again.");

  const [user] = await db
    .select({
      id: users.id,
      email: users.email,
      role: users.role,
      isActive: users.isActive,
    })
    .from(users)
    .where(eq(users.id, payload.userId))
    .limit(1);

  if (!user || !user.isActive)
    throw new UnauthorizedError("Account not found or deactivated.");

  await db.delete(refreshTokens).where(eq(refreshTokens.id, payload.tokenId));

  return issueTokenPair(user.id, user.email, user.role);
};

// ── Logout ────────────────────────────────────────────────────────

export const logout = async (
  token: string,
  userId: string,
  ipAddress?: string,
  userAgent?: string,
): Promise<void> => {
  try {
    const payload = verifyRefreshToken(token);
    await db.delete(refreshTokens).where(eq(refreshTokens.id, payload.tokenId));
  } catch {
    // already invalid — fine
  }
  await logActivity(userId, "logout", ipAddress, userAgent);
};

// ── Forgot password ───────────────────────────────────────────────

export const forgotPassword = async (email: string) => {
  const [user] = await db
    .select({ id: users.id, name: users.name })
    .from(users)
    .where(eq(users.email, email))
    .limit(1);

  if (!user)
    return { message: "If that email exists, a reset link has been sent." };

  const token = generateToken();
  const expires = new Date(Date.now() + 60 * 60 * 1000);

  await db
    .update(users)
    .set({
      resetToken: hashToken(token),
      resetTokenExpires: expires,
      updatedAt: new Date(),
    })
    .where(eq(users.id, user.id));

  await sendEmail({
    to: email,
    subject: "Reset your MerrakiSolutions admin password",
    html: renderEmail.passwordReset({
      name: user.name,
      resetUrl: `${env.FRONTEND_URL}/admin/reset-password?token=${token}`,
    }),
  });

  return { message: "If that email exists, a reset link has been sent." };
};

// ── Reset password ────────────────────────────────────────────────

export const resetPassword = async (token: string, newPassword: string) => {
  const [user] = await db
    .select({ id: users.id, resetTokenExpires: users.resetTokenExpires })
    .from(users)
    .where(eq(users.resetToken, hashToken(token)))
    .limit(1);

  if (!user) throw new AppError("Invalid or expired reset token.", 400);
  if (!user.resetTokenExpires || user.resetTokenExpires < new Date()) {
    throw new AppError(
      "Reset token has expired. Please request a new one.",
      400,
    );
  }

  await db
    .update(users)
    .set({
      passwordHash: await bcrypt.hash(newPassword, 12),
      resetToken: null,
      resetTokenExpires: null,
      updatedAt: new Date(),
    })
    .where(eq(users.id, user.id));

  await db.delete(refreshTokens).where(eq(refreshTokens.userId, user.id));
  await logActivity(user.id, "password_change");

  return { message: "Password reset successful. Please log in." };
};

// ── Get my profile ────────────────────────────────────────────────

export const getMe = async (userId: string) => {
  const [user] = await db
    .select(safeFields)
    .from(users)
    .where(eq(users.id, userId))
    .limit(1);

  if (!user) throw new NotFoundError("User not found.");
  return user;
};

// ── Update my profile ─────────────────────────────────────────────

export const updateMe = async (userId: string, name: string) => {
  const [updated] = await db
    .update(users)
    .set({ name, updatedAt: new Date() })
    .where(eq(users.id, userId))
    .returning(safeFields);

  await logActivity(userId, "profile_update");
  return updated;
};

// ── Update my avatar ──────────────────────────────────────────────

export const updateAvatar = async (
  userId: string,
  fileBuffer: Buffer,
  mimetype: string,
) => {
  const allowed = ["image/jpeg", "image/png", "image/webp"];
  if (!allowed.includes(mimetype)) {
    throw new AppError("Only JPEG, PNG, and WebP images are allowed.", 400);
  }

  const { url } = await uploadImageToCloudinary(
    fileBuffer,
    "merraki/admin-avatars",
    `admin_${userId}`,
  );

  const [updated] = await db
    .update(users)
    .set({ avatarUrl: url, updatedAt: new Date() })
    .where(eq(users.id, userId))
    .returning(safeFields);

  await logActivity(userId, "avatar_update");
  return updated;
};

// ── Update my password ────────────────────────────────────────────

export const updatePassword = async (
  userId: string,
  currentPassword: string,
  newPassword: string,
) => {
  const [user] = await db
    .select({ id: users.id, passwordHash: users.passwordHash })
    .from(users)
    .where(eq(users.id, userId))
    .limit(1);

  if (!user) throw new NotFoundError("User not found.");

  const valid = await bcrypt.compare(currentPassword, user.passwordHash);
  if (!valid) throw new UnauthorizedError("Current password is incorrect.");

  await db
    .update(users)
    .set({
      passwordHash: await bcrypt.hash(newPassword, 12),
      updatedAt: new Date(),
    })
    .where(eq(users.id, userId));

  await logActivity(userId, "password_change");
  return { message: "Password updated successfully." };
};

// ── Invite admin ──────────────────────────────────────────────────

export const inviteAdmin = async (
  name: string,
  email: string,
  invitedByName: string,
) => {
  const [existing] = await db
    .select({ id: users.id })
    .from(users)
    .where(eq(users.email, email))
    .limit(1);

  if (existing)
    throw new ConflictError("A user with this email already exists.");

  const token = generateToken();
  const expires = new Date(Date.now() + 48 * 60 * 60 * 1000);

  await db.insert(users).values({
    name,
    email,
    passwordHash: "",
    role: "admin",
    isActive: false,
    inviteToken: hashToken(token),
    inviteTokenExpires: expires,
  });

  await sendEmail({
    to: email,
    subject: "You have been invited to MerrakiSolutions Admin",
    html: renderEmail.adminInvite({
      name,
      invitedBy: invitedByName,
      inviteUrl: `${env.FRONTEND_URL}/admin/accept-invite?token=${token}`,
    }),
  });

  return { message: `Invite sent to ${email}.` };
};

// ── Accept invite ─────────────────────────────────────────────────

export const acceptInvite = async (token: string, password: string) => {
  const [user] = await db
    .select()
    .from(users)
    .where(eq(users.inviteToken, hashToken(token)))
    .limit(1);

  if (!user) throw new AppError("Invalid or expired invite token.", 400);
  if (!user.inviteTokenExpires || user.inviteTokenExpires < new Date()) {
    throw new AppError(
      "Invite has expired. Ask your superadmin to send a new invite.",
      400,
    );
  }

  await db
    .update(users)
    .set({
      passwordHash: await bcrypt.hash(password, 12),
      isActive: true,
      inviteToken: null,
      inviteTokenExpires: null,
      updatedAt: new Date(),
    })
    .where(eq(users.id, user.id));

  return { message: "Account activated. You can now log in." };
};

// ── Superadmin: list all admins ───────────────────────────────────

export const getAllAdmins = async (query: {
  page: number;
  limit: number;
  search?: string;
}) => {
  const { page, limit, search } = query;
  const offset = getPaginationOffset(page, limit);

  const where = search
    ? or(ilike(users.name, `%${search}%`), ilike(users.email, `%${search}%`))
    : undefined;

  const [{ total }] = await db
    .select({ total: count() })
    .from(users)
    .where(where);

  const data = await db
    .select(safeFields)
    .from(users)
    .where(where)
    .orderBy(users.createdAt)
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};

// ── Superadmin: get single admin ──────────────────────────────────

export const getAdminById = async (id: string) => {
  const [user] = await db
    .select(safeFields)
    .from(users)
    .where(eq(users.id, id))
    .limit(1);

  if (!user) throw new NotFoundError("Admin not found.");
  return user;
};

// ── Superadmin: update admin ──────────────────────────────────────

export const updateAdmin = async (
  id: string,
  data: { name?: string; isActive?: boolean },
  requestingUserId: string,
) => {
  if (id === requestingUserId && data.isActive === false) {
    throw new AppError("You cannot deactivate your own account.", 400);
  }

  const [existing] = await db
    .select({ id: users.id, role: users.role })
    .from(users)
    .where(eq(users.id, id))
    .limit(1);

  if (!existing) throw new NotFoundError("Admin not found.");
  if (existing.role === "superadmin" && id !== requestingUserId) {
    throw new AppError("Cannot modify another superadmin account.", 403);
  }

  const updateData: Record<string, unknown> = { updatedAt: new Date() };
  if (data.name !== undefined) updateData.name = data.name;
  if (data.isActive !== undefined) updateData.isActive = data.isActive;

  const [updated] = await db
    .update(users)
    .set(updateData)
    .where(eq(users.id, id))
    .returning(safeFields);

  return updated;
};

// ── Superadmin: delete admin ──────────────────────────────────────

export const deleteAdmin = async (id: string, requestingUserId: string) => {
  if (id === requestingUserId)
    throw new AppError("You cannot delete your own account.", 400);

  const [user] = await db
    .select({ id: users.id, role: users.role })
    .from(users)
    .where(eq(users.id, id))
    .limit(1);

  if (!user) throw new NotFoundError("Admin not found.");
  if (user.role === "superadmin")
    throw new AppError("Cannot delete a superadmin account.", 403);

  await db.delete(users).where(eq(users.id, id));
  return { message: "Admin deleted successfully." };
};

// ── Superadmin: get admin activity ────────────────────────────────

export const getAdminActivity = async (
  userId: string,
  query: { page: number; limit: number },
) => {
  const { page, limit } = query;
  const offset = getPaginationOffset(page, limit);

  const [existing] = await db
    .select({ id: users.id })
    .from(users)
    .where(eq(users.id, userId))
    .limit(1);

  if (!existing) throw new NotFoundError("Admin not found.");

  const [{ total }] = await db
    .select({ total: count() })
    .from(adminActivityLogs)
    .where(eq(adminActivityLogs.userId, userId));

  const data = await db
    .select()
    .from(adminActivityLogs)
    .where(eq(adminActivityLogs.userId, userId))
    .orderBy(adminActivityLogs.createdAt)
    .limit(limit)
    .offset(offset);

  return { data, pagination: paginate(page, limit, Number(total)) };
};
