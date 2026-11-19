//Eredmények lekéréséért felelős API réteg

export interface ObjectDownload {
    url: string;
    method: string;
    fields: Record<string, string[]>;
}

export interface GetResultsResponse {
    jobId: string;
    download: ObjectDownload;
    //Raw JSON: unstructured esetben jelenleg egy string (a felismert szavak összefűzve)
    result: unknown;
}

export async function getResults(jobID: string): Promise<GetResultsResponse> {
    const response = await fetch(`/api/jobs/${jobID}/results`);

    if (!response.ok) {
        throw new Error(`Nem sikerült lekérni az eredményeket (HTTP ${response.status}).`);
    }

    return response.json();
}
