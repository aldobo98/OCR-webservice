import { useState } from 'react'
import ImageForm from './components/ui/image-form'
import JobStatus from './components/ui/job-status';
import ImageCard from './components/ui/image-card';
import InvoiceCard from './components/ui/invoice-card';
import { Button } from './components/ui/button';
import type { GetResultsResponse } from './api/results';
import { downloadText } from './lib/export';

//A documentType alapján választjuk ki a megfelelő nézetet
function ResultView({ results }: { results: GetResultsResponse }) {
  switch (results.documentType) {
    case "unstructured":
      return (
        <div className="flex flex-col gap-3 items-start">
          <ImageCard imageUrl={results.download.url} caption={results.result.text} />
          <Button variant="neutral" onClick={() => downloadText(results.result.text, `ocr-${results.jobId}.txt`)}>
            Export TXT
          </Button>
        </div>
      );
    case "structured":
      return (
        <div className="flex flex-wrap gap-5 items-start">
          <ImageCard imageUrl={results.download.url} className="w-[400px]" />
          <InvoiceCard invoice={results.result} jobID={results.jobId} />
        </div>
      );
  }
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
          {/* Amint megvannak az eredmények, a feldolgozási kártya helyére az eredmény kerül */}
          {results && results.jobId === jobID ? (
            <ResultView results={results} />
          ) : (
            <JobStatus key={jobID} jobID={jobID} onResults={setResults} />
          )}
        </div>
      )}
    </div>
  )
}

export default App
