import { useState } from "react";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";

import { useJobEvents } from "@/hooks/useJobEvents";
import { getResults, type GetResultsResponse } from "@/api/results";
import { startProcessing } from "@/api/upload";
import { Check, X, LoaderCircle } from "lucide-react";

const stepLabels: Record<string, string> = {
  uploading: "Feltöltés",
  ocr_processing: "Szövegfelismerés",
  layout_processing: "Elrendezés elemzése",
};

type JobStatusProps = {
  jobID: string;
  onResults: (results: GetResultsResponse) => void;
};

export default function JobStatus({ jobID, onResults }: JobStatusProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [startError, setStartError] = useState<string | null>(null);

  //A feldolgozást csak az SSE feliratkozás után indítjuk, hogy ne vesszen el esemény
  const { stages, done } = useJobEvents(jobID, () => {
    startProcessing(jobID).catch((err) => {
      console.error("Failed to start processing:", err);
      setStartError(err instanceof Error ? err.message : "Nem sikerült elindítani a feldolgozást.");
    });
  });

  const handleShowResults = async () => {
    setLoading(true);
    setError(null);
    try {
      const results = await getResults(jobID);
      onResults(results);
    } catch (err) {
      console.error("Failed to fetch results:", err);
      setError(err instanceof Error ? err.message : "Ismeretlen hiba történt.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="w-72 min-h-64 shrink-0">
      <CardHeader>
        <CardTitle>Dokumentum feldolgozása</CardTitle>
      </CardHeader>

      <CardContent className="flex flex-1 flex-col gap-3">
        {stages.map((stage) => (
          <div key={stage.step} className="flex items-center gap-2">
            {stage.status === "completed" ? (
              <Check className="text-green-600 stroke-[3]" />
            ) : stage.status === "failed" ? (
              <X className="text-red-600 stroke-[3]" />
            ) : (
              <LoaderCircle className="animate-spin stroke-[3]" />
            )}

            <span>{stepLabels[stage.step] ?? stage.step}</span>
          </div>
        ))}
        {startError && <p className="text-sm text-red-600">{startError}</p>}
      </CardContent>

      {done && (
        <CardFooter className="flex-col gap-2">
          <Button onClick={handleShowResults} disabled={loading}>
            {loading && <LoaderCircle className="animate-spin" />}
            Eredmények megjelenítése
          </Button>
          {error && <p className="text-sm text-red-600">{error}</p>}
        </CardFooter>
      )}
    </Card>
  );
}
