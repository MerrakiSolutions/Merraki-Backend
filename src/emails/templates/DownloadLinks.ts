export interface DownloadItem {
  title: string;
  downloadUrl: string;
}

export interface DownloadLinksProps {
  guestName: string;
  orderId: string;
  items: DownloadItem[];
  trackOrderUrl: string;
  expiresIn: string;
}

export const DownloadLinks = ({
  guestName,
  orderId,
  items,
  trackOrderUrl,
  expiresIn,
}: DownloadLinksProps): string => `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
  <title>Your Downloads</title>
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
              <h1 style="margin:0 0 8px;font-size:22px;font-weight:700;color:#0f172a">Your files are ready</h1>
              <p style="margin:0 0 32px;font-size:15px;color:#475569;line-height:1.6">
                Hi ${guestName}, your purchase for order <strong>#${orderId.slice(0, 8).toUpperCase()}</strong> is complete. Click below to download your files.
              </p>
              ${items
                .map(
                  (item) => `
              <table width="100%" cellpadding="0" cellspacing="0" style="margin-bottom:12px">
                <tr>
                  <td style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:6px;padding:16px 20px">
                    <p style="margin:0 0 12px;font-size:14px;font-weight:600;color:#0f172a">${item.title}</p>
                    <a href="${item.downloadUrl}" style="display:inline-block;padding:10px 20px;background:#0f172a;color:#fff;border-radius:5px;text-decoration:none;font-size:13px;font-weight:600">
                      Download File
                    </a>
                  </td>
                </tr>
              </table>`,
                )
                .join("")}
              <p style="margin:24px 0 0;font-size:13px;color:#94a3b8;line-height:1.6">
                Links expire in <strong>${expiresIn}</strong>. Regenerate anytime from the
                <a href="${trackOrderUrl}" style="color:#0f172a">order tracking page</a>
                using your email or order ID.
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
