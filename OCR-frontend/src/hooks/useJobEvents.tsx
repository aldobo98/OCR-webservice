import { useEffect, useEffectEvent, useState } from "react";

export interface JobProgress {
    step: string;
    status: string;
}

//A backend ServerSentDone üzenete: nincs benne "step", van benne "jobId"
interface JobDone {
    jobId: string;
    status: string;
}

function isDoneMessage(data: unknown): data is JobDone {
    return (
        typeof data === "object" &&
        data !== null &&
        "jobId" in data &&
        !("step" in data)
    );
}

//Ha a backend nem küld fejlécet az első eseményig, az onopen nem sül el.
//Ennyi idő után feltételezzük, hogy a feliratkozás már létrejött.
const CONNECT_FALLBACK_MS = 1500;

/**
 * Feliratkozik a job SSE eseményeire, és az onConnected callbacket pontosan egyszer
 * meghívja, amikor a kapcsolat létrejött (vagy a fallback idő letelt).
 * A feldolgozást (/start) itt kell elindítani, így nem veszünk el korai eseményt.
 */
export function useJobEvents(jobID: string, onConnected?: () => void) {
    const [stages, setStages] = useState<JobProgress[]>([]);
    const [done, setDone] = useState(false);

    const handleConnected = useEffectEvent(() => {
        onConnected?.();
    });

    useEffect(() => {
        if (!jobID) return;

        const eventSource = new EventSource(`/api/jobs/${jobID}/events`);

        let connected = false;
        const markConnected = () => {
            if (connected) return;
            connected = true;
            clearTimeout(fallbackTimer);
            handleConnected();
        };
        const fallbackTimer = setTimeout(markConnected, CONNECT_FALLBACK_MS);

        eventSource.onopen = markConnected;

        eventSource.onmessage = (event) => {
            let data: unknown;
            try {
                data = JSON.parse(event.data);
            } catch (error) {
                console.error("Failed to parse SSE message:", event.data, error);
                return;
            }

            if (isDoneMessage(data)) {
                console.log("SSE done:", data);
                setDone(data.status === "completed");
                //Több esemény nem jön, a kapcsolatot lezárjuk
                eventSource.close();
                return;
            }

            const progress = data as JobProgress;
            console.log("SSE received:", progress);
            setStages((prev) => {
                const existing = prev.find((item) => item.step === progress.step);
                if (existing) {
                    return prev.map((item) =>
                        item.step === progress.step
                            ? { ...item, status: progress.status }
                            : item
                    );
                }
                return [...prev, progress];
            });
        };

        eventSource.onerror = (error) => {
            console.error("SSE error:", error);
        };

        return () => {
            clearTimeout(fallbackTimer);
            eventSource.close();
        };
    }, [jobID]);

    return { stages, done };
}
