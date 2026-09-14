export interface AdminInviteProps {
  name: string;
  inviteUrl: string;
  invitedBy: string;
}

export const AdminInvite = ({
  name,
  inviteUrl,
  invitedBy,
}: AdminInviteProps): string => `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
  <title>You're Invited</title>
</head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 20px">
    <tr>
      <td align="center">
        <table width="560" cellpadding="0" cellspacing="0" style="background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1)">
          <tr>
            <td style="background:#0f172a;padding:32px 40px">
              <p style="margin:0;font-size:20px;font-weight:700;color:#fff;letter-spacing:-0.3px">MerrakiSolutions</p>
            </td>
          </tr>
          <tr>
            <td style="padding:40px">
              <h1 style="margin:0 0 16px;font-size:22px;font-weight:700;color:#0f172a">You're invited to the admin panel</h1>
              <p style="margin:0 0 8px;font-size:15px;color:#475569;line-height:1.6">Hi ${name},</p>
              <p style="margin:0 0 32px;font-size:15px;color:#475569;line-height:1.6">
                <strong>${invitedBy}</strong> has invited you to join the MerrakiSolutions admin panel. Click below to set your password and activate your account.
              </p>
              <a href="${inviteUrl}" style="display:inline-block;padding:14px 28px;background:#0f172a;color:#fff;border-radius:6px;text-decoration:none;font-size:15px;font-weight:600">
                Accept Invite & Set Password
              </a>
              <p style="margin:32px 0 0;font-size:13px;color:#94a3b8;line-height:1.6">
                This invite expires in <strong>48 hours</strong>. If you weren't expecting this, you can safely ignore this email.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:24px 40px;border-top:1px solid #f1f5f9">
              <p style="margin:0;font-size:12px;color:#94a3b8">© ${new Date().getFullYear()} MerrakiSolutions. All rights reserved.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`;
