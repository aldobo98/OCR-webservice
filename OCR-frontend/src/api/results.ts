//Eredmények lekéréséért felelős API réteg

export interface ObjectDownload {
    url: string;
    method: string;
    fields: Record<string, string[]>;
}

//Unstructured dokumentum: a felismert szavak összefűzve
export interface TextResult {
    text: string;
}

//Structured dokumentum (számla): a layout-engine dto.go struktúráit tükrözi
export interface InvoicePerson {
    name: string;
    address: string[]; //Több soros cím
    tax_id: string;
}

export interface InvoiceSeller extends InvoicePerson {
    iban: string;
}

export interface InvoiceHeader {
    invoice_no: string;
    date_of_issue: string;
}

export interface InvoiceItem {
    item_no: number;
    item_description: string;
    item_qty: string;
    item_um: string;
    item_net_price: string;
    item_net_worth: string;
    item_vat: string;
    item_gross_worth: string;
}

export interface InvoiceSummary {
    net_worth: string;
    vat: string;
    gross_worth: string;
    vat_percent: string;
}

export interface InvoiceResult {
    header: InvoiceHeader;
    seller: InvoiceSeller;
    client: InvoicePerson;
    items: InvoiceItem[];
    summary: InvoiceSummary;
}

interface ResultsBase {
    jobId: string;
    download: ObjectDownload;
}

//A documentType alapján dől el, milyen alakú a result (discriminated union)
export type GetResultsResponse =
    | (ResultsBase & { documentType: "unstructured"; result: TextResult })
    | (ResultsBase & { documentType: "structured"; result: InvoiceResult });

export async function getResults(jobID: string): Promise<GetResultsResponse> {
    const response = await fetch(`/api/jobs/${jobID}/results`);

    if (!response.ok) {
        throw new Error(`Nem sikerült lekérni az eredményeket (HTTP ${response.status}).`);
    }

    return response.json();
}
