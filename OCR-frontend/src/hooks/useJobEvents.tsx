import { useEffect } from "react";
import { useState } from 'react'

interface JobProgress{
    step: string;
    status: string;
}

export function useJobEvents(jobID: string) {
    const [stages, setStages] = useState<JobProgress[]>([]);
    useEffect(() => {
        if (!jobID) return;

        const eventSource = new EventSource(
            `/api/jobs/${jobID}/events`
        );

        eventSource.onmessage = (event) => {
            console.log("SSE raw event:", event.data);
            const data:JobProgress = JSON.parse(event.data);
            console.log("Parsed SSE:", data);
            setStages((prev) => {
            const existing = prev.find((item) => item.step === data.step);

            if (existing) {
                return prev.map((item) =>
                    item.step === data.step
                        ? { ...item, status: data.status }
                        : item
                );
            }

            return [...prev, data];
        });
            console.log("SSE received:", data);
        };

        eventSource.onerror = (error) => {
            console.error("SSE error:", error);
        };

        return () => {
            eventSource.close();
        };
    }, [jobID]);
    return stages;
}