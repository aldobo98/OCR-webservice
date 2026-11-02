import { useEffect } from "react";

export function useJobEvents(jobID: string) {
    useEffect(() => {
        if (!jobID) return;

        const eventSource = new EventSource(
            `http://localhost:8080/api/jobs/${jobID}/events`
        );

        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);

            console.log(data.status);
        };

        eventSource.onerror = (error) => {
            console.error("SSE error:", error);
        };

        return () => {
            eventSource.close();
        };
    }, [jobID]);
}