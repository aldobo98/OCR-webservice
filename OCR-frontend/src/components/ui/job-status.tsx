import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { useJobEvents } from "@/hooks/useJobEvents";
import { Check, X, LoaderCircle } from "lucide-react";

type JobStatusProps = {
    jobID: string
}

export default function JobStatus({jobID}: JobStatusProps) {
    const stages = useJobEvents(jobID);    
    return (
        <Card className="w-full max-w-sm">
            <CardHeader>
                <CardTitle>Dokumentum feldolgozása</CardTitle>
            </CardHeader>
            <CardContent>
                {stages.map((stage) => (
                    <div key={stage.step} className="flex items-center gap-2">
                        {stage.status === "completed" ? (
                        <Check className="text-green-600 stroke-[3]" />
                        ) : stage.status === "failed" ? (
                        <X className="text-red-600 stroke-[3]" />
                        ) : (
                        <LoaderCircle className="animate-spin stroke-[3]" />
                        )}
                        <span>{stage.step}</span>
                    </div>
                    ))}
            </CardContent>
            </Card>
  )
}
