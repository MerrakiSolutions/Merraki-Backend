import { PDFDocument, rgb, StandardFonts } from "pdf-lib";

export interface InvoiceItem {
  title: string;
  priceUsd: string;
}

export interface InvoiceData {
  orderId: string;
  guestName: string;
  guestEmail: string;
  billingAddress: {
    line1: string;
    line2?: string;
    city: string;
    state: string;
    country: string;
    zip: string;
    company?: string;
  };
  items: InvoiceItem[];
  subtotalUsd: string;
  totalUsd: string;
  currencyCharged: "USD" | "INR";
  amountCharged: string;
  exchangeRate?: string;
  createdAt: Date;
}

export const generateInvoicePdf = async (
  data: InvoiceData,
): Promise<Buffer> => {
  const doc = await PDFDocument.create();
  const page = doc.addPage([595, 842]); // A4

  const regular = await doc.embedFont(StandardFonts.Helvetica);
  const bold = await doc.embedFont(StandardFonts.HelveticaBold);

  const { width, height } = page.getSize();
  const margin = 50;
  const black = rgb(0.06, 0.09, 0.16);
  const gray = rgb(0.58, 0.63, 0.69);
  const lightGray = rgb(0.95, 0.97, 0.98);
  const white = rgb(1, 1, 1);

  let y = height - margin;

  // ── Header background ───────────────────────────
  page.drawRectangle({
    x: 0,
    y: height - 80,
    width,
    height: 80,
    color: black,
  });

  // Company name
  page.drawText("MerrakiSolutions", {
    x: margin,
    y: height - 52,
    size: 22,
    font: bold,
    color: white,
  });

  // Invoice label
  page.drawText("INVOICE", {
    x: width - margin - 70,
    y: height - 52,
    size: 14,
    font: bold,
    color: white,
  });

  y = height - 110;

  // ── Invoice meta ─────────────────────────────────
  page.drawText(`Invoice #: ${data.orderId.slice(0, 8).toUpperCase()}`, {
    x: margin,
    y,
    size: 10,
    font: bold,
    color: black,
  });

  page.drawText(
    `Date: ${data.createdAt.toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    })}`,
    {
      x: width - margin - 160,
      y,
      size: 10,
      font: regular,
      color: gray,
    },
  );

  y -= 30;

  // ── Bill To ──────────────────────────────────────
  page.drawText("BILL TO", {
    x: margin,
    y,
    size: 8,
    font: bold,
    color: gray,
  });

  y -= 16;

  const billLines = [
    data.guestName,
    data.guestEmail,
    data.billingAddress.company,
    data.billingAddress.line1,
    data.billingAddress.line2,
    `${data.billingAddress.city}, ${data.billingAddress.state} ${data.billingAddress.zip}`,
    data.billingAddress.country,
  ].filter(Boolean) as string[];

  for (const line of billLines) {
    page.drawText(line, {
      x: margin,
      y,
      size: 10,
      font: regular,
      color: black,
    });
    y -= 15;
  }

  y -= 20;

  // ── Items table header ───────────────────────────
  page.drawRectangle({
    x: margin,
    y: y - 4,
    width: width - margin * 2,
    height: 22,
    color: black,
  });

  page.drawText("Description", {
    x: margin + 8,
    y: y + 4,
    size: 9,
    font: bold,
    color: white,
  });

  page.drawText("Amount (USD)", {
    x: width - margin - 90,
    y: y + 4,
    size: 9,
    font: bold,
    color: white,
  });

  y -= 20;

  // ── Items rows ───────────────────────────────────
  let rowAlt = false;
  for (const item of data.items) {
    if (rowAlt) {
      page.drawRectangle({
        x: margin,
        y: y - 4,
        width: width - margin * 2,
        height: 20,
        color: lightGray,
      });
    }

    page.drawText(item.title, {
      x: margin + 8,
      y: y + 3,
      size: 10,
      font: regular,
      color: black,
      maxWidth: width - margin * 2 - 110,
    });

    page.drawText(`$${item.priceUsd}`, {
      x: width - margin - 90,
      y: y + 3,
      size: 10,
      font: regular,
      color: black,
    });

    y -= 22;
    rowAlt = !rowAlt;
  }

  y -= 10;

  // ── Divider ──────────────────────────────────────
  page.drawLine({
    start: { x: margin, y },
    end: { x: width - margin, y },
    thickness: 1,
    color: lightGray,
  });

  y -= 20;

  // ── Totals ───────────────────────────────────────
  const drawTotal = (label: string, value: string, isBold = false) => {
    page.drawText(label, {
      x: width - margin - 200,
      y,
      size: isBold ? 11 : 10,
      font: isBold ? bold : regular,
      color: isBold ? black : gray,
    });
    page.drawText(value, {
      x: width - margin - 90,
      y,
      size: isBold ? 11 : 10,
      font: isBold ? bold : regular,
      color: isBold ? black : gray,
    });
    y -= 18;
  };

  drawTotal("Subtotal", `$${data.subtotalUsd} USD`);
  drawTotal("Total", `$${data.totalUsd} USD`, true);

  if (data.currencyCharged === "INR") {
    y -= 4;
    drawTotal(
      `Exchange Rate (1 USD = ₹${data.exchangeRate})`,
      `₹${data.amountCharged} INR`,
    );
  }

  // ── Footer ────────────────────────────────────────
  page.drawText("Thank you for your purchase.", {
    x: margin,
    y: margin + 30,
    size: 10,
    font: regular,
    color: gray,
  });

  page.drawText("© MerrakiSolutions — merrakisolutions.com", {
    x: margin,
    y: margin + 14,
    size: 8,
    font: regular,
    color: gray,
  });

  const pdfBytes = await doc.save();
  return Buffer.from(pdfBytes);
};
