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
            const data:JobProgress = JSON.parse(event.data);
            setStages((prev) => {
                const existingIndex = prev.findIndex(
                    (item) => item.step === data.step
                );

                // Ha már létezik, frissítjük
                if (existingIndex !== -1) {
                    return prev.map((item, index) =>
                    index === existingIndex
                        ? { ...item, status: data.status }
                        : item
                    );
                }

                // Ha még nem létezik, hozzáadjuk
                return [
                    ...prev,
                    {
                    step: data.step,
                    status: data.status,
                    },
                ];
                });
            console.log(data.status);
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