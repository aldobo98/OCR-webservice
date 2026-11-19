import { useState } from 'react'
import ImageForm from './components/ui/image-form'
import JobStatus from './components/ui/job-status';
import ImageCard from './components/ui/image-card';
import type { GetResultsResponse } from './api/results';

//A raw JSON eredményből megjeleníthető szöveg
function resultToText(result: unknown): string {
  if (typeof result === "string") return result.trim();
  return JSON.stringify(result, null, 2);
}

export function App() {
  const [jobID, setJobID] = useState<string | null>(null);
  const [results, setResults] = useState<GetResultsResponse | null>(null);

  const handleNewJob = (id: string) => {
    setResults(null);
    setJobID(id);
  };

  return (
    <div className="flex place-content-center gap-5">
      <div>
        <title>Karakterfelismerés</title>
        <h1 className="text-2xl pb-8 pt-8">Karakterfelismerés EasyOCR használatával</h1>
        <ImageForm setJobID={handleNewJob} />
      </div>
      {jobID && (
        <div className="pt-24">
          <JobStatus key={jobID} jobID={jobID} onResults={setResults} />
        </div>
      )}
      {results && results.jobId === jobID && (
        <div className="pt-24">
          <ImageCard imageUrl={results.download.url} caption={resultToText(results.result)} />
        </div>
      )}
    </div>
  )
}

export default App
