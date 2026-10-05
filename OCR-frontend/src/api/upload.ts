//Kép feltöltéséért felelős API réteg

export interface GetUploadUrlRequest {
    contentType: string;
}

export interface GetUploadUrlResponse {
    jobId: string;
    upload: {
        url: string;
        fields: Record<string, string>;
    }
}


export async function uploadImage(request: GetUploadUrlRequest, file: File): Promise<string> {
    //Presigned POST linket elkérem a backendtől
    const response = await fetch("/api/uploadurl", {
        method: "POST",
        headers: {
            "Content-type": "application/json",
        },
        body: JSON.stringify(request),
    });

    if (!response.ok) {
        throw new Error("Failed to get presigned upload URL from backend.");
    }

    const data: GetUploadUrlResponse = await response.json();

    //Feltöltjük S3-ba az imaget
    //Kitöltjük a POST-hoz a megfelelő dolgokat
    const formData = new FormData();
    for (const [key, value] of Object.entries(data.upload.fields)) {
        formData.append(key, value);
    }

    formData.append("file", file);
    const uploadResponse = await fetch(data.upload.url, {
        method: "POST",
        body: formData,
    });

    if (!uploadResponse.ok) {
        throw new Error("Failed to upload image to S3.");
    }

    return data.jobId;
}