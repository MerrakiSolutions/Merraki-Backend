export interface RegisterBody {
  name: string
  email: string
  password: string
}

export interface LoginBody {
  email: string
  password: string
}

export interface VerifyEmailBody {
  token: string
}

export interface ForgotPasswordBody {
  email: string
}

export interface ResetPasswordBody {
  token: string
  password: string
}

export interface RefreshTokenBody {
  // token comes from httpOnly cookie, no body needed
}

export interface AcceptInviteBody {
  token: string
  password: string
}

export interface InviteAdminBody {
  name: string
  email: string
}