import { FastifyInstance } from "fastify";
import * as authService from "./auth.service.js";
import {
  loginSchema,
  forgotPasswordSchema,
  resetPasswordSchema,
  acceptInviteSchema,
  inviteAdminSchema,
  updateProfileSchema,
  updatePasswordSchema,
  adminListQuerySchema,
  activityQuerySchema,
  adminUpdateSchema,
} from "./auth.schema.js";
import { authenticate } from "../../middleware/authenticate.js";
import { requireSuperAdmin } from "../../middleware/require-admin.js";
import { AppError } from "../../lib/errors.js";
import { env } from "../../config/env.js";

const REFRESH_COOKIE = "refresh_token";

const cookieOpts = {
  httpOnly: true,
  secure: env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 7 * 24 * 60 * 60,
};

export const authRoutes = async (app: FastifyInstance) => {
  // ════════════════════════════════════════
  // NO AUTH REQUIRED
  // ════════════════════════════════════════

  app.post("/login", async (request, reply) => {
    const body = loginSchema.parse(request.body);
    const result = await authService.login(
      body.email,
      body.password,
      request.ip,
      request.headers["user-agent"],
    );
    reply.setCookie(REFRESH_COOKIE, result.refreshToken, cookieOpts);
    return reply.send({
      success: true,
      accessToken: result.accessToken,
      user: result.user,
    });
  });

  app.post("/refresh", async (request, reply) => {
    const token = request.cookies?.[REFRESH_COOKIE];
    if (!token) throw new AppError("No refresh token.", 401);
    const result = await authService.refreshAccessToken(token);
    reply.setCookie(REFRESH_COOKIE, result.refreshToken, cookieOpts);
    return reply.send({ success: true, accessToken: result.accessToken });
  });

  app.post("/forgot-password", async (request, reply) => {
    const body = forgotPasswordSchema.parse(request.body);
    const result = await authService.forgotPassword(body.email);
    return reply.send({ success: true, ...result });
  });

  app.post("/reset-password", async (request, reply) => {
    const body = resetPasswordSchema.parse(request.body);
    const result = await authService.resetPassword(body.token, body.password);
    return reply.send({ success: true, ...result });
  });

  app.post("/accept-invite", async (request, reply) => {
    const body = acceptInviteSchema.parse(request.body);
    const result = await authService.acceptInvite(body.token, body.password);
    return reply.send({ success: true, ...result });
  });

  // ════════════════════════════════════════
  // AUTH REQUIRED — MY ACCOUNT
  // ════════════════════════════════════════

  app.post(
    "/logout",
    { preHandler: [authenticate] },
    async (request, reply) => {
      const token = request.cookies?.[REFRESH_COOKIE];
      if (token) {
        await authService.logout(
          token,
          request.user!.userId,
          request.ip,
          request.headers["user-agent"],
        );
      }
      reply.clearCookie(REFRESH_COOKIE, { path: "/" });
      return reply.send({ success: true, message: "Logged out successfully." });
    },
  );

  app.get("/me", { preHandler: [authenticate] }, async (request, reply) => {
    const user = await authService.getMe(request.user!.userId);
    return reply.send({ success: true, data: user });
  });

  app.put("/me", { preHandler: [authenticate] }, async (request, reply) => {
    const body = updateProfileSchema.parse(request.body);
    const user = await authService.updateMe(request.user!.userId, body.name);
    return reply.send({ success: true, data: user });
  });

  app.put(
    "/me/avatar",
    { preHandler: [authenticate] },
    async (request, reply) => {
      const file = await request.file();
      if (!file) throw new AppError("No file uploaded.", 400);
      const buffer = await file.toBuffer();
      const user = await authService.updateAvatar(
        request.user!.userId,
        buffer,
        file.mimetype,
      );
      return reply.send({ success: true, data: user });
    },
  );

  app.put(
    "/me/password",
    { preHandler: [authenticate] },
    async (request, reply) => {
      const body = updatePasswordSchema.parse(request.body);
      const result = await authService.updatePassword(
        request.user!.userId,
        body.currentPassword,
        body.newPassword,
      );
      return reply.send({ success: true, ...result });
    },
  );

  // ════════════════════════════════════════
  // SUPERADMIN ONLY — ADMIN MANAGEMENT
  // ════════════════════════════════════════

  app.post(
    "/invite",
    { preHandler: [authenticate, requireSuperAdmin] },
    async (request, reply) => {
      const body = inviteAdminSchema.parse(request.body);
      const result = await authService.inviteAdmin(
        body.name,
        body.email,
        request.user!.email,
      );
      return reply.status(201).send({ success: true, ...result });
    },
  );

  app.get(
    "/admins",
    { preHandler: [authenticate, requireSuperAdmin] },
    async (request, reply) => {
      const query = adminListQuerySchema.parse(request.query);
      const result = await authService.getAllAdmins(query);
      return reply.send({ success: true, ...result });
    },
  );

  app.get(
    "/admins/:id",
    { preHandler: [authenticate, requireSuperAdmin] },
    async (request, reply) => {
      const { id } = request.params as { id: string };
      const user = await authService.getAdminById(id);
      return reply.send({ success: true, data: user });
    },
  );

  app.put(
    "/admins/:id",
    { preHandler: [authenticate, requireSuperAdmin] },
    async (request, reply) => {
      const { id } = request.params as { id: string };
      const body = adminUpdateSchema.parse(request.body);
      const user = await authService.updateAdmin(
        id,
        body,
        request.user!.userId,
      );
      return reply.send({ success: true, data: user });
    },
  );

  app.delete(
    "/admins/:id",
    { preHandler: [authenticate, requireSuperAdmin] },
    async (request, reply) => {
      const { id } = request.params as { id: string };
      const result = await authService.deleteAdmin(id, request.user!.userId);
      return reply.send({ success: true, ...result });
    },
  );

  app.get(
    "/admins/:id/activity",
    { preHandler: [authenticate, requireSuperAdmin] },
    async (request, reply) => {
      const { id } = request.params as { id: string };
      const query = activityQuerySchema.parse(request.query);
      const result = await authService.getAdminActivity(id, query);
      return reply.send({ success: true, ...result });
    },
  );
};
