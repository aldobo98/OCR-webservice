import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { useJobEvents } from "@/hooks/useJobEvents";

type JobStatusProps = {
    jobID: string
}

export default function JobStatus({jobID}: JobStatusProps) {
    useJobEvents(jobID);
    return (
        <Card className="w-full max-w-sm">
            <CardHeader>
                <CardTitle>Dokumentum feldolgozása</CardTitle>
            </CardHeader>
            <CardContent>

            </CardContent>
            </Card>
  )
}
