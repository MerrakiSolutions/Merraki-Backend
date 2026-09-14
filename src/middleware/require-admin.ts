import { FastifyRequest, FastifyReply } from "fastify";
import { ForbiddenError, UnauthorizedError } from "../lib/errors.js";

export const requireAdmin = async (
  request: FastifyRequest,
  _reply: FastifyReply,
): Promise<void> => {
  if (!request.user) throw new UnauthorizedError();
  if (!["admin", "superadmin"].includes(request.user.role)) {
    throw new ForbiddenError("Admin access required");
  }
};

export const requireSuperAdmin = async (
  request: FastifyRequest,
  _reply: FastifyReply,
): Promise<void> => {
  if (!request.user) throw new UnauthorizedError();
  if (request.user.role !== "superadmin") {
    throw new ForbiddenError("Superadmin access required");
  }
};
