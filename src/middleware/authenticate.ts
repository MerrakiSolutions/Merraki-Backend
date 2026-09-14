import { FastifyRequest, FastifyReply } from "fastify";
import { verifyAccessToken } from "../lib/jwt.js";
import { UnauthorizedError } from "../lib/errors.js";

export const authenticate = async (
  request: FastifyRequest,
  _reply: FastifyReply,
): Promise<void> => {
  const header = request.headers.authorization;
  if (!header?.startsWith("Bearer ")) {
    throw new UnauthorizedError("No token provided");
  }

  const token = header.split(" ")[1];
  try {
    request.user = verifyAccessToken(token);
  } catch {
    throw new UnauthorizedError("Invalid or expired token");
  }
};
