export interface OrderItem {
  title: string;
  priceUsd: string;
}

export interface OrderConfirmationProps {
  guestName: string;
  orderId: string;
  items: OrderItem[];
  totalUsd: string;
  currencyCharged: "USD" | "INR";
  amountCharged: string;
  exchangeRate?: string;
  createdAt: string;
}

export const OrderConfirmation = ({
  guestName,
  orderId,
  items,
  totalUsd,
  currencyCharged,
  amountCharged,
  exchangeRate,
  createdAt,
}: OrderConfirmationProps): string => `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
  <title>Order Confirmed</title>
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
              <h1 style="margin:0 0 8px;font-size:22px;font-weight:700;color:#0f172a">Payment confirmed ✓</h1>
              <p style="margin:0 0 32px;font-size:15px;color:#475569;line-height:1.6">
                Hi ${guestName}, thank you for your purchase. Your download links are included in this email.
              </p>
              <table width="100%" cellpadding="0" cellspacing="0" style="background:#f8fafc;border-radius:6px;margin-bottom:32px">
                <tr>
                  <td style="padding:20px 24px">
                    <p style="margin:0 0 6px;font-size:12px;font-weight:600;color:#94a3b8;text-transform:uppercase;letter-spacing:0.5px">Order Details</p>
                    <p style="margin:0 0 4px;font-size:14px;color:#475569">Order ID: <strong style="color:#0f172a">#${orderId.slice(0, 8).toUpperCase()}</strong></p>
                    <p style="margin:0;font-size:14px;color:#475569">Date: ${createdAt}</p>
                  </td>
                </tr>
              </table>
              <table width="100%" cellpadding="0" cellspacing="0" style="margin-bottom:24px">
                <tr>
                  <td style="padding-bottom:12px;font-size:12px;font-weight:600;color:#94a3b8;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #f1f5f9">Item</td>
                  <td align="right" style="padding-bottom:12px;font-size:12px;font-weight:600;color:#94a3b8;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #f1f5f9">Price</td>
                </tr>
                ${items
                  .map(
                    (item) => `
                <tr>
                  <td style="padding:12px 0;font-size:14px;color:#0f172a;border-bottom:1px solid #f1f5f9">${item.title}</td>
                  <td align="right" style="padding:12px 0;font-size:14px;color:#0f172a;border-bottom:1px solid #f1f5f9">$${item.priceUsd}</td>
                </tr>`,
                  )
                  .join("")}
                <tr>
                  <td style="padding:16px 0 0;font-size:15px;font-weight:700;color:#0f172a">Total</td>
                  <td align="right" style="padding:16px 0 0;font-size:15px;font-weight:700;color:#0f172a">$${totalUsd} USD</td>
                </tr>
                ${
                  currencyCharged === "INR"
                    ? `
                <tr>
                  <td style="padding:4px 0 0;font-size:13px;color:#94a3b8">Charged in INR (rate: 1 USD = ₹${exchangeRate})</td>
                  <td align="right" style="padding:4px 0 0;font-size:13px;color:#94a3b8">₹${amountCharged}</td>
                </tr>`
                    : ""
                }
              </table>
              <p style="margin:0;font-size:13px;color:#94a3b8;line-height:1.6">Your invoice is attached to this email as a PDF.</p>
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
