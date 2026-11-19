//Strukturált (számla) eredmény megjelenítése
import type { InvoicePerson, InvoiceResult } from "@/api/results"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { downloadInvoiceCsv, downloadJson } from "@/lib/export"
import { cn } from "@/lib/utils"

type Props = {
  invoice: InvoiceResult
  jobID: string
  className?: string
}

//Hiányzó mezőknél gondolatjel, hogy ne maradjon üres hely
const show = (value?: string) => (value && value.trim() !== "" ? value : "–")

function PersonBlock({ title, person, iban }: { title: string; person: InvoicePerson; iban?: string }) {
  return (
    <div className="flex-1 min-w-[200px] rounded-base border-2 border-border p-3">
      <div className="font-heading mb-1">{title}</div>
      <div>{show(person.name)}</div>
      {(person.address ?? []).map((line, idx) => (
        <div key={idx} className="text-sm">{line}</div>
      ))}
      <div className="text-sm mt-1">Adószám: {show(person.tax_id)}</div>
      {iban !== undefined && <div className="text-sm">IBAN: {show(iban)}</div>}
    </div>
  )
}

export default function InvoiceCard({ invoice, jobID, className }: Props) {
  const { header, seller, client, items, summary } = invoice

  return (
    <Card className={cn("w-[760px] max-w-full", className)}>
      <CardHeader>
        <CardTitle className="text-xl">Számla: {show(header?.invoice_no)}</CardTitle>
        <div className="text-sm">Kiállítás dátuma: {show(header?.date_of_issue)}</div>
      </CardHeader>

      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-wrap gap-4">
          <PersonBlock title="Eladó" person={seller} iban={seller?.iban ?? ""} />
          <PersonBlock title="Vevő" person={client} />
        </div>

        {/* Tételek: széles táblázat, kis képernyőn vízszintesen görgethető */}
        <div className="overflow-x-auto rounded-base border-2 border-border">
          <table className="w-full text-sm border-collapse">
            <thead>
              <tr className="bg-main text-main-foreground text-left">
                <th className="p-2">#</th>
                <th className="p-2">Megnevezés</th>
                <th className="p-2 text-right">Menny.</th>
                <th className="p-2">Me.</th>
                <th className="p-2 text-right">Nettó egységár</th>
                <th className="p-2 text-right">Nettó érték</th>
                <th className="p-2 text-right">ÁFA</th>
                <th className="p-2 text-right">Bruttó érték</th>
              </tr>
            </thead>
            <tbody>
              {(items ?? []).length === 0 ? (
                <tr>
                  <td colSpan={8} className="p-2 text-center">Nem található tétel.</td>
                </tr>
              ) : (
                items.map((item, idx) => (
                  <tr key={`${item.item_no}-${idx}`} className="border-t-2 border-border">
                    <td className="p-2">{item.item_no}</td>
                    <td className="p-2">{show(item.item_description)}</td>
                    <td className="p-2 text-right">{show(item.item_qty)}</td>
                    <td className="p-2">{show(item.item_um)}</td>
                    <td className="p-2 text-right">{show(item.item_net_price)}</td>
                    <td className="p-2 text-right">{show(item.item_net_worth)}</td>
                    <td className="p-2 text-right">{show(item.item_vat)}</td>
                    <td className="p-2 text-right">{show(item.item_gross_worth)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Összesítő */}
        <div className="self-end grid grid-cols-[auto_auto] gap-x-6 gap-y-1 text-sm">
          <span>Nettó összesen:</span>
          <span className="text-right">{show(summary?.net_worth)}</span>
          <span>ÁFA ({show(summary?.vat_percent)}):</span>
          <span className="text-right">{show(summary?.vat)}</span>
          <span className="font-heading">Bruttó összesen:</span>
          <span className="text-right font-heading">{show(summary?.gross_worth)}</span>
        </div>
      </CardContent>

      <CardFooter className="gap-3">
        <Button variant="neutral" onClick={() => downloadJson(invoice, `invoice-${jobID}.json`)}>
          Export JSON
        </Button>
        <Button variant="neutral" onClick={() => downloadInvoiceCsv(invoice, `invoice-${jobID}.csv`)}>
          Export CSV
        </Button>
      </CardFooter>
    </Card>
  )
}
