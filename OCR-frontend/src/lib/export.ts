//Eredmények exportálása a böngészőből (letöltés generálása kliens oldalon)
import type { InvoiceResult } from "@/api/results";

function downloadFile(content: string, fileName: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export function downloadJson(data: unknown, fileName: string) {
  downloadFile(JSON.stringify(data, null, 2), fileName, "application/json");
}

export function downloadText(text: string, fileName: string) {
  downloadFile(text, fileName, "text/plain;charset=utf-8");
}

//CSV mező escape-elése: idézőjelek duplázása, és idézőjelek közé tétel, ha kell
function csvField(value: string | number): string {
  const s = String(value ?? "");
  return /[";\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

//A tételek CSV-ben; pontosvessző elválasztó és BOM, hogy a magyar Excel jól nyissa meg
export function downloadInvoiceCsv(invoice: InvoiceResult, fileName: string) {
  const header = ["No", "Description", "Qty", "UM", "Net price", "Net worth", "VAT", "Gross worth"];
  const rows = invoice.items.map((i) => [
    i.item_no,
    i.item_description,
    i.item_qty,
    i.item_um,
    i.item_net_price,
    i.item_net_worth,
    i.item_vat,
    i.item_gross_worth,
  ]);
  const s = invoice.summary;
  rows.push(["", "Total", "", "", "", s.net_worth, s.vat, s.gross_worth]);

  const csv = [header, ...rows].map((r) => r.map(csvField).join(";")).join("\r\n");
  downloadFile("﻿" + csv, fileName, "text/csv;charset=utf-8");
}
